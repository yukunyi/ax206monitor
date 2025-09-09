package main

import (
	"context"
	"flag"
	"fmt"
	"image"
	"os"
	"os/signal"
	"runtime"
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

	// Default to config directory in current working directory
	defaultConfigDir := "./config"
	defaultConfigName := ""

	// Check if config directory exists in current directory first
	if _, err := os.Stat("./config"); err == nil {
		defaultConfigDir = "./config"
	} else if runtime.GOOS == "linux" {
		// Fallback to system config directory on Linux
		if _, err := os.Stat("/etc/ax206monitor"); err == nil {
			defaultConfigDir = "/etc/ax206monitor"
		}
	}

	if runtime.GOOS == "windows" {
		defaultConfigName = "windows"
	}

	configFlag := flag.String("config", defaultConfigName, "Configuration file name (without .json extension)")
	configDirFlag := flag.String("config-dir", defaultConfigDir, "Configuration directory")
	listConfigsFlag := flag.Bool("list-configs", false, "List available configuration files")
	listMonitorsFlag := flag.Bool("list-monitors", false, "List all available monitor items and exit")
	// New: dump all monitor values for N seconds and exit
	dumpSecondsFlag := flag.Int("dump", 0, "Dump all monitor values for N seconds and exit (0 to disable)")

	flag.Parse()

	configManager := NewConfigManager(*configDirFlag)

	if *listConfigsFlag {
		configs, err := configManager.ListConfigs()
		if err != nil {
			logFatal("Config enumeration failed: %v", err)
		}
		fmt.Println("Available configurations:")
		for _, config := range configs {
			fmt.Printf("  %s\n", config)
		}
		return
	}

	if *listMonitorsFlag {
		listAllMonitors()
		return
	}

	if *configFlag == "" {
		logError("Configuration name required")
		fmt.Println("Usage: ax206monitor -config <name>")
		fmt.Println("Use -list-configs to enumerate available configurations")
		fmt.Println("Use -list-monitors to see all available monitor items")
		return
	}

	config, err := configManager.LoadConfig(*configFlag)
	if err != nil {
		logFatal("Config load failed '%s': %v", *configFlag, err)
	}

	// Set global config for monitor system
	SetGlobalMonitorConfig(config)

	// Initialize system information cache and print details
	initializeCache()

	// Initialize network interface manager early to ensure proper detection
	networkInterface := config.GetNetworkInterface()
	if networkInterface == "" || networkInterface == "auto" {
		logInfo("Initializing network interface detection...")
		manager := GetNetworkInterfaceManager()
		manager.TryRefreshAsync()
		for i := 0; i < 5; i++ {
			time.Sleep(2 * time.Second)
			if def := manager.GetDefaultInterface(); def != "" {
				logInfo("Network interface detected: %s", def)
				break
			}
		}
	}

	requiredMonitors := getRequiredMonitors(config)
	registry := GetMonitorRegistryWithConfig(requiredMonitors, networkInterface)

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
				go func(m MonitorItem) {
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
						val = FormatMonitorValue(mv, true, "")
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
	outputManager := NewOutputManager()

	outputMode := strings.ToLower(config.OutputType)
	if outputMode == "" {
		outputMode = "file"
	}

	outputFile := config.OutputFile
	if outputFile == "" {
		outputFile = "monitor.png"
	}

	needDevice := (outputMode == "ax206usb" || outputMode == "both")

	if needDevice {
		logInfoModule("ax206usb", "Initializing handler")
		handler, err := NewAX206USBOutputHandler()
		if err != nil {
			logErrorModule("ax206usb", "Handler creation failed: %v", err)
		} else {
			logInfoModule("ax206usb", "Handler ready")
			outputManager.AddHandler(handler)
		}
	}

	if outputMode == "file" || outputMode == "both" {
		outputManager.AddHandler(NewFileOutputHandler(outputFile))
	}

	refreshInterval := time.Duration(config.RefreshInterval) * time.Millisecond
	if refreshInterval == 0 {
		refreshInterval = RefreshInterval
	}

	logInfo("started, pid is %d", os.Getpid())
	logInfo("AX206 Monitor v%s", Version)
	logInfo("Config: %s | Output: %s | Refresh: %v", *configFlag, outputMode, refreshInterval)

	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, os.Interrupt, syscall.SIGTERM)

	logInfo("Monitoring %d items", len(requiredMonitors))

	ticker := time.NewTicker(refreshInterval)
	defer ticker.Stop()

	// Channel for async output
	outputChan := make(chan image.Image, 1)
	go func() {
		for img := range outputChan {
			outputStart := time.Now()
			if err := outputManager.Output(img); err != nil {
				logWarn("Output failed: %v", err)
			} else {
				outputDuration := time.Since(outputStart)
				logDebug("Output time: %v", outputDuration)
			}
		}
	}()

	for {
		select {
		case <-ticker.C:
			cycleStart := time.Now()
			cache := GetMonitorCache()
			noteRenderAccess()
			cache.StartRender()

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

			// Async output (non-blocking)
			select {
			case outputChan <- img:
				// ok
			default:
				logWarn("Output queue full, skipping frame")
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

func getRequiredMonitors(config *MonitorConfig) []string {
	monitors := make(map[string]bool)
	for _, item := range config.Items {
		monitors[item.Monitor] = true
	}

	result := make([]string, 0, len(monitors))
	for monitor := range monitors {
		result = append(result, monitor)
	}
	return result
}

func listAllMonitors() {
	fmt.Println("Initializing system monitoring...")

	// Initialize system cache first
	initializeCache()

	registry := GetMonitorRegistry()

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
				value = FormatMonitorValue(monitorValue, true, "")
			}
		}

		fmt.Printf("%-30s %-20s %s\n", name, label, value)
	}
}
