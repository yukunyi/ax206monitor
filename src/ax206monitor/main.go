package main

import (
	"flag"
	"fmt"
	"image"
	"os"
	"os/signal"
	"sort"
	"strings"
	"syscall"
	"time"
)

var (
	Version   = "unknown"
	BuildTime = "unknown"
)

const (
	RefreshInterval = 1 * time.Second
	RetryInterval   = 3 * time.Second
	RepositoryURL   = "https://github.com/yukunyi/ax206monitor"
)

func main() {
	initLogger()

	logInfo("AX206 Monitor - Repository: %s", RepositoryURL)

	listMonitorsFlag := flag.Bool("list-monitors", false, "List all available monitor items and exit")
	portFlag := flag.Int("port", 18086, "Web UI listen port (tray/env web mode)")
	addUdevRuleFlag := flag.Bool("add-udev-rule", false, "Install AX206 USB udev rule for current user and reload udev")
	// New: dump all monitor values for N seconds and exit
	dumpSecondsFlag := flag.Int("dump", 0, "Dump all monitor values for N seconds and exit (0 to disable)")

	flag.Parse()

	if *portFlag < 1 || *portFlag > 65535 {
		logFatal("Invalid --port value: %d", *portFlag)
	}

	if *addUdevRuleFlag && (*listMonitorsFlag || *dumpSecondsFlag > 0) {
		logFatal("--add-udev-rule cannot be used with other execution flags")
	}

	webModeEnabled, webDevEnabled, devViteURL := resolveWebModeFromEnv()

	if *addUdevRuleFlag {
		if err := InstallAX206UdevRule(); err != nil {
			logFatal("Failed to add udev rule: %v", err)
		}
		return
	}

	if *listMonitorsFlag {
		listAllMonitors()
		return
	}

	if webModeEnabled {
		logInfoModule("web", "Web mode enabled on port %d", *portFlag)
		if err := RunWebServer(WebServerOptions{
			Addr:    fmt.Sprintf("127.0.0.1:%d", *portFlag),
			DevMode: webDevEnabled,
			ViteURL: devViteURL,
		}); err != nil {
			logFatal("Web server failed: %v", err)
		}
		return
	}

	userConfigPath, pathErr := getUserConfigPath()
	if pathErr != nil {
		logFatal("Failed to resolve user config path: %v", pathErr)
	}
	config, err := loadUserConfigOrDefault(userConfigPath)
	if err != nil {
		logFatal("Config load failed '%s': %v", userConfigPath, err)
	}
	configSource := userConfigPath

	// Set global config for monitor system
	SetGlobalCollectorConfig(config)

	// Initialize system information cache and print details
	initializeCache()

	networkInterface := config.GetNetworkInterface()
	requiredMonitors := getRequiredMonitors(config)
	registry := GetCollectorManagerWithConfig(requiredMonitors, networkInterface)

	// New: dump mode - print all monitors and exit
	if *dumpSecondsFlag > 0 {
		interval := config.GetCollectTickDuration()
		waitMax := config.GetRenderWaitMaxDuration()
		end := time.Now().Add(time.Duration(*dumpSecondsFlag) * time.Second)
		logInfo("Dumping all monitor values for %d seconds...", *dumpSecondsFlag)

		// build stable, sorted name list
		names := make([]string, 0)
		for n := range registry.GetAll() {
			names = append(names, n)
		}
		sort.Strings(names)

		lastEpoch := int64(0)
		for frame := 0; time.Now().Before(end); frame++ {
			noteRenderAccess()
			start := time.Now()
			epochID, completed, waitDuration := registry.WaitForNextEpoch(lastEpoch, waitMax)
			if epochID <= lastEpoch {
				time.Sleep(10 * time.Millisecond)
				frame--
				continue
			}
			lastEpoch = epochID
			items := registry.GetAll()

			// print
			logInfoModule(
				"dump",
				"frame=%d epoch=%d complete=%v wait=%v time=%s",
				frame,
				epochID,
				completed,
				waitDuration,
				time.Now().Format("15:04:05"),
			)
			for _, name := range names {
				it := items[name]
				val := "-"
				if it != nil && it.IsAvailable() {
					if mv := it.GetValue(); mv != nil {
						val = FormatCollectValue(mv, true, "")
					}
				}
				logInfoModule("dump", "%-28s = %s", name, val)
			}

			// pacing
			elapsed := time.Since(start)
			if elapsed < interval {
				time.Sleep(interval - elapsed)
			}
		}
		return
	}

	fontCache, err := loadFontCache()
	if err != nil {
		logFatal("Font initialization failed: %v", err)
	}

	renderManager := NewRenderManager(fontCache, registry)
	outputManager, outputTypes := buildOutputManager(config, false)
	webProcessController := NewWebServerProcess(*portFlag, webDevEnabled, devViteURL)
	if webDevEnabled {
		if err := webProcessController.Start(); err != nil {
			logFatal("Web server auto-start failed in dev mode: %v", err)
		}
	}
	trayHandle, trayErr := StartTray(webProcessController)
	if trayErr != nil || trayHandle == nil {
		logFatal("Tray startup failed: %v", trayErr)
	}
	defer trayHandle.Close()
	defer func() {
		if err := webProcessController.Stop(); err != nil {
			logWarnModule("tray", "Web server stop failed on shutdown: %v", err)
		}
	}()

	refreshInterval := config.GetCollectTickDuration()
	renderWaitMax := config.GetRenderWaitMaxDuration()

	logInfo("started, pid is %d", os.Getpid())
	logInfo("AX206 Monitor v%s", Version)
	logInfo(
		"Config: %s | Output: %s | Tick: %v | RenderWaitMax: %v",
		configSource,
		strings.Join(outputTypes, ","),
		refreshInterval,
		renderWaitMax,
	)

	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, os.Interrupt, syscall.SIGTERM)

	logInfo("Monitoring %d items", len(requiredMonitors))

	// Keep at most one pending frame and always prefer the newest frame.
	outputChan := make(chan outputFrame, 1)
	go func() {
		for frame := range outputChan {
			outputStart := time.Now()
			if err := outputManager.Output(frame.img); err != nil {
				logWarn("Output failed: %v", err)
			} else {
				outputDuration := time.Since(outputStart)
				queueDelay := outputStart.Sub(frame.enqueuedAt)
				logDebug("Output time: %v | Queue delay: %v", outputDuration, queueDelay)
			}
		}
	}()

	lastEpoch := int64(0)
	for {
		select {
		case <-signalChan:
			logInfo("Shutdown initiated")
			close(outputChan)
			outputManager.Close()
			return
		default:
		}

		cycleStart := time.Now()
		noteRenderAccess()

		epochID, waitComplete, waitDuration := registry.WaitForNextEpoch(lastEpoch, renderWaitMax)
		if epochID <= lastEpoch {
			time.Sleep(10 * time.Millisecond)
			continue
		}
		lastEpoch = epochID

		// Render timing
		renderStart := time.Now()
		img, err := renderManager.Render(config)
		if err != nil {
			logError("Render failed: %v", err)
			continue
		}
		renderDuration := time.Since(renderStart)
		recordRenderDuration(renderDuration)

		// Async output (non-blocking), drop stale pending frame first.
		replaced, ok := enqueueLatestFrame(outputChan, outputFrame{
			img:        img,
			enqueuedAt: time.Now(),
		})
		if !ok {
			logWarn("Output queue busy, skipping frame")
		} else if replaced {
			logDebug("Output queue replaced stale frame")
		}

		cycleDuration := time.Since(cycleStart)
		logDebug(
			"Cycle: %v | Epoch: %d | CollectWait: %v (complete=%v) | Render: %v",
			cycleDuration,
			epochID,
			waitDuration,
			waitComplete,
			renderDuration,
		)
	}
}

func resolveWebModeFromEnv() (bool, bool, string) {
	devURL := strings.TrimSpace(os.Getenv("AX206_MONITOR_DEV_URL"))
	webEnabled := parseEnvBool(os.Getenv("AX206_MONITOR_WEB"))
	webDevEnabled := devURL != ""
	return webEnabled, webDevEnabled, devURL
}

func parseEnvBool(raw string) bool {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "1", "true", "yes", "y", "on":
		return true
	default:
		return false
	}
}

type outputFrame struct {
	img        image.Image
	enqueuedAt time.Time
}

func enqueueLatestFrame(ch chan outputFrame, frame outputFrame) (bool, bool) {
	select {
	case ch <- frame:
		return false, true
	default:
	}

	replaced := false
	select {
	case <-ch:
		replaced = true
	default:
	}

	select {
	case ch <- frame:
		return replaced, true
	default:
		return replaced, false
	}
}

func getRequiredMonitors(config *MonitorConfig) []string {
	monitors := make(map[string]bool)
	queue := make([]string, 0, len(config.Items))
	for _, item := range config.Items {
		name := normalizeMonitorAlias(item.Monitor)
		if name == "" {
			continue
		}
		if !monitors[name] {
			queue = append(queue, name)
		}
		monitors[name] = true
	}

	customByName := make(map[string]CustomMonitorConfig)
	for _, custom := range config.CustomMonitors {
		if custom.Name == "" {
			continue
		}
		customByName[custom.Name] = custom
	}

	for len(queue) > 0 {
		name := queue[0]
		queue = queue[1:]

		custom, exists := customByName[name]
		if !exists {
			continue
		}

		if normalizeCustomMonitorType(custom.Type) != "mixed" {
			continue
		}

		for _, source := range custom.Sources {
			source = normalizeMonitorAlias(source)
			if source == "" || monitors[source] {
				continue
			}
			monitors[source] = true
			queue = append(queue, source)
		}
	}

	result := make([]string, 0, len(monitors))
	for monitor := range monitors {
		result = append(result, monitor)
	}
	return result
}

func normalizeCustomMonitorType(t string) string {
	switch strings.ToLower(strings.TrimSpace(t)) {
	case "mixed", "mix":
		return "mixed"
	case "file":
		return "file"
	case "coolercontrol":
		return "coolercontrol"
	case "librehardwaremonitor", "libre", "lhm":
		return "librehardwaremonitor"
	default:
		return strings.ToLower(strings.TrimSpace(t))
	}
}

func listAllMonitors() {
	fmt.Println("Initializing system monitoring...")

	// Initialize system cache first
	initializeCache()

	registry := GetCollectorManager()

	// Wait 2 seconds for data to stabilize
	fmt.Println("Waiting for data to stabilize...")
	time.Sleep(2 * time.Second)

	// Update all monitors to get current values
	fmt.Println("Updating monitor values...")
	_, _, _ = registry.WaitForNextEpoch(0, 500*time.Millisecond)

	registry.mutex.RLock()
	defer registry.mutex.RUnlock()

	// Collect and sort monitor names
	var names []string
	for name := range registry.items {
		names = append(names, name)
	}

	// Sort names
	for i := 0; i < len(names)-1; i++ {
		for j := i + 1; j < len(names); j++ {
			if names[i] > names[j] {
				names[i], names[j] = names[j], names[i]
			}
		}
	}

	fmt.Println("\n=== System Information ===")
	printSystemInfo()

	fmt.Println("\n=== Available Monitor Items ===")
	fmt.Printf("%-30s %-20s %s\n", "Name", "Label", "Current Value")
	fmt.Printf("%-30s %-20s %s\n", "----", "-----", "-------------")

	for _, name := range names {
		monitor := registry.items[name]
		label := monitor.GetLabel()
		if label == "" {
			label = "-"
		}

		value := "-"
		if monitor.IsAvailable() {
			monitorValue := monitor.GetValue()
			if monitorValue != nil {
				value = FormatCollectValue(monitorValue, true, "")
			}
		}

		fmt.Printf("%-30s %-20s %s\n", name, label, value)
	}
}
