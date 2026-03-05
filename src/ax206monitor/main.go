package main

import (
	"context"
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
	webModeFlag := flag.Bool("web", false, "Start web configuration UI")
	portFlag := flag.Int("port", 18086, "Web UI listen port")
	webDevFlag := flag.Bool("web-dev", false, "Enable web frontend development proxy mode")
	viteURLFlag := flag.String("vite-url", "http://127.0.0.1:18087", "Vite dev server URL for web-dev mode")
	installFlag := flag.Bool("install", false, "Install as systemd service")
	uninstallFlag := flag.Bool("uninstall", false, "Uninstall systemd service")
	addUdevRuleFlag := flag.Bool("add-udev-rule", false, "Install AX206 USB udev rule for current user and reload udev")
	// New: dump all monitor values for N seconds and exit
	dumpSecondsFlag := flag.Int("dump", 0, "Dump all monitor values for N seconds and exit (0 to disable)")

	flag.Parse()

	if *portFlag < 1 || *portFlag > 65535 {
		logFatal("Invalid --port value: %d", *portFlag)
	}

	if *installFlag && *uninstallFlag {
		logFatal("--install and --uninstall cannot be used together")
	}
	if *addUdevRuleFlag && (*installFlag || *uninstallFlag || *webModeFlag || *listMonitorsFlag || *dumpSecondsFlag > 0) {
		logFatal("--add-udev-rule cannot be used with other execution flags")
	}

	if *installFlag {
		if err := InstallService(ServiceInstallOptions{
			WebMode: *webModeFlag,
			Port:    *portFlag,
		}); err != nil {
			logFatal("Service install failed: %v", err)
		}
		return
	}

	if *uninstallFlag {
		if err := UninstallService(); err != nil {
			logFatal("Service uninstall failed: %v", err)
		}
		return
	}

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

	if *webModeFlag {
		logInfoModule("web", "Web mode enabled on port %d", *portFlag)
		if err := RunWebServer(WebServerOptions{
			Addr:    fmt.Sprintf("127.0.0.1:%d", *portFlag),
			DevMode: *webDevFlag,
			ViteURL: *viteURLFlag,
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
		interval := time.Second
		end := time.Now().Add(time.Duration(*dumpSecondsFlag) * time.Second)
		logInfo("Dumping all monitor values for %d seconds...", *dumpSecondsFlag)

		// build stable, sorted name list
		names := make([]string, 0)
		for n := range registry.GetAll() {
			names = append(names, n)
		}
		sort.Strings(names)

		for frame := 0; time.Now().Before(end); frame++ {
			noteRenderAccess()
			start := time.Now()
			items := registry.GetAll()

			// concurrent updates with context timeout per frame
			ctx, cancel := context.WithTimeout(context.Background(), 300*time.Millisecond)
			doneCh := make(chan struct{}, len(items))
			for _, it := range items {
				go func(m *CollectItem) {
					defer func() { doneCh <- struct{}{} }()
					select {
					case <-ctx.Done():
						return
					default:
						_ = m.Update()
					}
				}(it)
			}
			// wait for either all or timeout
			waitCount := 0
			for waitCount < len(items) {
				select {
				case <-doneCh:
					waitCount++
				case <-ctx.Done():
					waitCount = len(items)
				}
			}
			cancel()

			// print
			logInfoModule("dump", "frame=%d time=%s", frame, time.Now().Format("15:04:05"))
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

	refreshInterval := time.Second

	logInfo("started, pid is %d", os.Getpid())
	logInfo("AX206 Monitor v%s", Version)
	logInfo("Config: %s | Output: %s | Refresh: %v", configSource, strings.Join(outputTypes, ","), refreshInterval)

	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, os.Interrupt, syscall.SIGTERM)

	logInfo("Monitoring %d items", len(requiredMonitors))

	ticker := time.NewTicker(refreshInterval)
	defer ticker.Stop()

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

	for {
		select {
		case <-ticker.C:
			cycleStart := time.Now()
			noteRenderAccess()

			// Monitor update timing
			updateStart := time.Now()
			if err := registry.Update(requiredMonitors); err != nil {
				logWarn("Monitor update failed: %v", err)
				continue
			}
			updateDuration := time.Since(updateStart)

			// Render timing
			renderStart := time.Now()
			img, err := renderManager.Render(config)
			if err != nil {
				logError("Render failed: %v", err)
				continue
			}
			renderDuration := time.Since(renderStart)

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
			logDebug("Cycle: %v | Update: %v | Render: %v", cycleDuration, updateDuration, renderDuration)

		case <-signalChan:
			logInfo("Shutdown initiated")
			close(outputChan)
			outputManager.Close()
			return
		}
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
	registry.UpdateAll()

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
