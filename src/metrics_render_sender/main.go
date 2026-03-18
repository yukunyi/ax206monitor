package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
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
	RepositoryURL   = "https://github.com/yukunyi/metrics_render_sender"
)

func main() {
	initLogger()

	logInfo("MetricsRenderSender - Repository: %s", RepositoryURL)

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
	profileManager, config, err := InitializeGlobalProfileManager(userConfigPath, config)
	if err != nil {
		logFatal("Profile initialization failed: %v", err)
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
		names := registry.AllNames()

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

	runtimeAPI, err := AcquireSharedWebAPI(config)
	if err != nil {
		logFatal("Runtime initialization failed: %v", err)
	}
	defer ReleaseSharedWebAPI(runtimeAPI)
	runtimeAPI.SetIdleConfigProvider(func() (*MonitorConfig, error) {
		activeName := strings.TrimSpace(profileManager.ActiveName())
		if activeName == "" {
			return loadUserConfigOrDefault(userConfigPath)
		}
		cfg, err := profileManager.LoadProfile(activeName)
		if err != nil {
			return nil, err
		}
		return cfg, nil
	})

	outputTypes := resolveOutputConfigSummaryFromList(config.Outputs, false).Types
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
	logInfo("MetricsRenderSender v%s", Version)
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
	for {
		select {
		case <-signalChan:
			logInfo("Shutdown initiated")
			return
		default:
			time.Sleep(200 * time.Millisecond)
		}
	}
}

func resolveWebModeFromEnv() (bool, bool, string) {
	devURL := firstNonEmptyEnv("METRICS_RENDER_SENDER_DEV_URL", "AX206_MONITOR_DEV_URL")
	webEnabled := parseEnvBool(firstNonEmptyEnv("METRICS_RENDER_SENDER_WEB", "AX206_MONITOR_WEB"))
	webDevEnabled := devURL != ""
	return webEnabled, webDevEnabled, devURL
}

func firstNonEmptyEnv(keys ...string) string {
	for _, key := range keys {
		if value := strings.TrimSpace(os.Getenv(key)); value != "" {
			return value
		}
	}
	return ""
}

func parseEnvBool(raw string) bool {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "1", "true", "yes", "y", "on":
		return true
	default:
		return false
	}
}

func getRequiredMonitors(config *MonitorConfig) []string {
	if config == nil {
		return nil
	}
	monitors := make(map[string]struct{})
	queue := make([]string, 0, len(config.Items))
	for _, item := range config.Items {
		refs := collectItemMonitorRefs(&item)
		queue = appendUniqueMonitorRefs(queue, monitors, refs)
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
			if source == "" {
				continue
			}
			if _, exists := monitors[source]; exists {
				continue
			}
			monitors[source] = struct{}{}
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

	// Collect and sort monitor names
	items := registry.GetAll()
	names := registry.AllNames()

	fmt.Println("\n=== System Information ===")
	printSystemInfo()

	fmt.Println("\n=== Available Monitor Items ===")
	fmt.Printf("%-30s %-20s %s\n", "Name", "Label", "Current Value")
	fmt.Printf("%-30s %-20s %s\n", "----", "-----", "-------------")

	for _, name := range names {
		monitor := items[name]
		if monitor == nil {
			continue
		}
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
