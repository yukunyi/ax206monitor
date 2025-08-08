package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

type FontSizes struct {
	MinLabel int `json:"min_label"`
	DefLabel int `json:"def_label"`
	Default  int `json:"default"`
	Small    int `json:"small"`
	Large    int `json:"large"`
}

type ColorThresholds struct {
	LowThreshold  float64 `json:"low_threshold"`
	HighThreshold float64 `json:"high_threshold"`
	LowColor      string  `json:"low_color"`
	MediumColor   string  `json:"medium_color"`
	HighColor     string  `json:"high_color"`
}

type MonitorConfig struct {
	Name                    string                     `json:"name"`
	Width                   int                        `json:"width"`
	Height                  int                        `json:"height"`
	FontSizes               FontSizes                  `json:"font_sizes"`
	FontFamilies            []string                   `json:"font_families"`
	OutputType              string                     `json:"output_type"`
	OutputFile              string                     `json:"output_file,omitempty"`
	RefreshInterval         int                        `json:"refresh_interval"`
	HistorySize             int                        `json:"history_size,omitempty"`
	NetworkInterface        string                     `json:"network_interface,omitempty"`
	LibreHardwareMonitorURL string                     `json:"libre_hardware_monitor_url,omitempty"`
	Colors                  map[string]string          `json:"colors"`
	ColorThresholds         map[string]ColorThresholds `json:"color_thresholds,omitempty"`
	Items                   []ItemConfig               `json:"items"`
	Labels                  map[string]string          `json:"labels,omitempty"`
	Units                   map[string]string          `json:"units,omitempty"`
}

type ItemConfig struct {
	Type          string   `json:"type"`
	Monitor       string   `json:"monitor"`
	X             int      `json:"x"`
	Y             int      `json:"y"`
	Width         int      `json:"width"`
	Height        int      `json:"height"`
	FontSize      int      `json:"font_size,omitempty"`
	LabelFontSize int      `json:"label_font_size,omitempty"`
	ValueFontSize int      `json:"value_font_size,omitempty"`
	Color         string   `json:"color,omitempty"`
	Background    string   `json:"bg,omitempty"`
	ShowUnit      *bool    `json:"unit,omitempty"`
	ShowLabel     *bool    `json:"label,omitempty"`
	ShowValue     *bool    `json:"value,omitempty"`
	ShowHeader    *bool    `json:"header,omitempty"`
	History       bool     `json:"history,omitempty"`
	Max           float64  `json:"max,omitempty"`
	MaxValue      *float64 `json:"max_value,omitempty"`
	MinValue      *float64 `json:"min_value,omitempty"`
	Text          string   `json:"text,omitempty"`
	LabelText     string   `json:"label_text,omitempty"`
	UnitText      string   `json:"unit_text,omitempty"`
}

func (item *ItemConfig) GetShowUnit() bool {
	if item.ShowUnit == nil {
		return true
	}
	return *item.ShowUnit
}

func (item *ItemConfig) GetShowLabel() bool {
	if item.ShowLabel == nil {
		return true
	}
	return *item.ShowLabel
}

func (item *ItemConfig) GetShowValue() bool {
	if item.ShowValue == nil {
		return true
	}
	return *item.ShowValue
}

func (item *ItemConfig) GetShowHeader() bool {
	if item.ShowHeader == nil {
		return true
	}
	return *item.ShowHeader
}

type ConfigManager struct {
	configDir string
	configs   map[string]*MonitorConfig
}

func NewConfigManager(configDir string) *ConfigManager {
	return &ConfigManager{
		configDir: configDir,
		configs:   make(map[string]*MonitorConfig),
	}
}

func (cm *ConfigManager) LoadConfig(configName string) (*MonitorConfig, error) {
	if config, exists := cm.configs[configName]; exists {
		return config, nil
	}

	configFile := filepath.Join(cm.configDir, configName+".json")

	if _, err := os.Stat(configFile); os.IsNotExist(err) {
		return nil, fmt.Errorf("config file not found: %s", configFile)
	}

	data, err := os.ReadFile(configFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %v", err)
	}

	var config MonitorConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config: %v", err)
	}

	cm.configs[configName] = &config
	return &config, nil
}

func (cm *ConfigManager) ListConfigs() ([]string, error) {
	files, err := os.ReadDir(cm.configDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read config directory: %v", err)
	}

	var configs []string
	for _, file := range files {
		if !file.IsDir() && filepath.Ext(file.Name()) == ".json" {
			name := file.Name()[:len(file.Name())-5]
			configs = append(configs, name)
		}
	}

	return configs, nil
}

func getDefaultFontFamilies() []string {
	return []string{
		"DejaVu Sans Mono",
		"Liberation Mono",
		"Consolas",
		"Monaco",
		"Menlo",
		"Ubuntu Mono",
		"Courier New",
		"monospace",
	}
}

func (config *MonitorConfig) GetMinLabelFontSize() int {
	if config.FontSizes.MinLabel > 0 {
		return config.FontSizes.MinLabel
	}
	return 14
}

func (config *MonitorConfig) GetDefLabelFontSize() int {
	if config.FontSizes.DefLabel > 0 {
		return config.FontSizes.DefLabel
	}
	return 16
}

func (config *MonitorConfig) GetDefaultFontSize() int {
	if config.FontSizes.Default > 0 {
		return config.FontSizes.Default
	}
	return 14
}

func (config *MonitorConfig) GetSmallFontSize() int {
	if config.FontSizes.Small > 0 {
		return config.FontSizes.Small
	}
	return 12
}

func (config *MonitorConfig) GetLargeFontSize() int {
	if config.FontSizes.Large > 0 {
		return config.FontSizes.Large
	}
	return 18
}

func (config *MonitorConfig) GetLabelText(monitorName string, defaultLabel string) string {
	if config.Labels != nil {
		if label, exists := config.Labels[monitorName]; exists {
			return label
		}
	}
	return defaultLabel
}

// GetDynamicColor returns color based on value and thresholds
func (config *MonitorConfig) GetDynamicColor(monitorName string, value float64) string {
	// Check if there are specific thresholds for this monitor
	if config.ColorThresholds != nil {
		if thresholds, exists := config.ColorThresholds[monitorName]; exists {
			if value <= thresholds.LowThreshold {
				return thresholds.LowColor
			} else if value <= thresholds.HighThreshold {
				return thresholds.MediumColor
			} else {
				return thresholds.HighColor
			}
		}
	}

	// Default thresholds based on monitor type
	return config.getDefaultDynamicColor(monitorName, value)
}

// GetDynamicColorForNetworkSpeed returns color based on display value and unit for network speed
func (config *MonitorConfig) GetDynamicColorForNetworkSpeed(monitorName string, displayValue float64, unit string) string {
	// Check if there are specific thresholds for this monitor
	if config.ColorThresholds != nil {
		if thresholds, exists := config.ColorThresholds[monitorName]; exists {
			// Convert thresholds to display unit for comparison
			lowThreshold := config.convertSpeedToDisplayUnit(thresholds.LowThreshold, unit)
			highThreshold := config.convertSpeedToDisplayUnit(thresholds.HighThreshold, unit)

			if displayValue <= lowThreshold {
				return thresholds.LowColor
			} else if displayValue <= highThreshold {
				return thresholds.MediumColor
			} else {
				return thresholds.HighColor
			}
		}
	}

	// Default thresholds based on display unit
	return config.getDefaultNetworkSpeedColor(displayValue, unit)
}

// GetDynamicColorForDiskSpeed returns color based on display value and unit for disk speed
func (config *MonitorConfig) GetDynamicColorForDiskSpeed(monitorName string, displayValue float64, unit string) string {
	// Check if there are specific thresholds for this monitor
	if config.ColorThresholds != nil {
		if thresholds, exists := config.ColorThresholds[monitorName]; exists {
			// Convert thresholds to display unit for comparison
			lowThreshold := config.convertSpeedToDisplayUnit(thresholds.LowThreshold, unit)
			highThreshold := config.convertSpeedToDisplayUnit(thresholds.HighThreshold, unit)

			if displayValue <= lowThreshold {
				return thresholds.LowColor
			} else if displayValue <= highThreshold {
				return thresholds.MediumColor
			} else {
				return thresholds.HighColor
			}
		}
	}

	// Default thresholds based on display unit (same logic as network speed)
	return config.getDefaultNetworkSpeedColor(displayValue, unit)
}

// convertSpeedToDisplayUnit converts MB/s threshold to the display unit
func (config *MonitorConfig) convertSpeedToDisplayUnit(mbpsValue float64, displayUnit string) float64 {
	switch displayUnit {
	case " MiB/s":
		return mbpsValue // Already in MB/s
	case " KiB/s":
		return mbpsValue * 1024 // Convert MB/s to KB/s
	case " B/s":
		return mbpsValue * 1024 * 1024 // Convert MB/s to B/s
	default:
		return mbpsValue // Fallback
	}
}

// getDefaultNetworkSpeedColor provides default color logic for network speed based on display unit
func (config *MonitorConfig) getDefaultNetworkSpeedColor(displayValue float64, unit string) string {
	switch unit {
	case " MiB/s":
		// For MiB/s display
		if displayValue <= 10 {
			return "#22c55e" // Green - Normal/Low speed
		} else if displayValue <= 50 {
			return "#eab308" // Yellow - Medium speed
		} else {
			return "#ef4444" // Red - High speed
		}
	case " KiB/s":
		// For KiB/s display
		if displayValue <= 10240 { // 10 MB/s = 10240 KB/s
			return "#22c55e" // Green - Normal/Low speed
		} else if displayValue <= 51200 { // 50 MB/s = 51200 KB/s
			return "#eab308" // Yellow - Medium speed
		} else {
			return "#ef4444" // Red - High speed
		}
	case " B/s":
		// For B/s display
		if displayValue <= 10485760 { // 10 MB/s = 10485760 B/s
			return "#22c55e" // Green - Normal/Low speed
		} else if displayValue <= 52428800 { // 50 MB/s = 52428800 B/s
			return "#eab308" // Yellow - Medium speed
		} else {
			return "#ef4444" // Red - High speed
		}
	default:
		// Fallback to default color
		if color, exists := config.Colors["default_text"]; exists {
			return color
		}
		return "#f8fafc"
	}
}

// getDefaultDynamicColor provides default color logic for different monitor types
func (config *MonitorConfig) getDefaultDynamicColor(monitorName string, value float64) string {
	// Temperature monitors (CPU, GPU, Disk)
	if isTemperatureMonitor(monitorName) {
		if value <= 60 {
			return "#22c55e" // Green - Safe
		} else if value <= 75 {
			return "#eab308" // Yellow - Warning
		} else {
			return "#ef4444" // Red - Critical
		}
	}

	// Usage monitors (CPU, Memory, GPU usage)
	if isUsageMonitor(monitorName) {
		if value <= 60 {
			return "#22c55e" // Green - Normal
		} else if value <= 75 {
			return "#eab308" // Yellow - High
		} else {
			return "#ef4444" // Red - Critical
		}
	}

	// Network speed monitors (using original MB/s values for backward compatibility)
	if isNetworkMonitor(monitorName) {
		// For network speed, low is normal (green), high might indicate issues (red)
		if value <= 10 { // MB/s
			return "#22c55e" // Green - Normal/Low speed
		} else if value <= 50 {
			return "#eab308" // Yellow - Medium speed
		} else {
			return "#ef4444" // Red - High speed (potential issue)
		}
	}

	// Default fallback color
	if color, exists := config.Colors["default_text"]; exists {
		return color
	}
	return "#f8fafc"
}

// Helper functions to identify monitor types
func isTemperatureMonitor(monitorName string) bool {
	tempMonitors := []string{"cpu_temp", "gpu_temp", "disk_temp", "disk1_temp"}
	for _, temp := range tempMonitors {
		if monitorName == temp {
			return true
		}
	}
	return false
}

func isUsageMonitor(monitorName string) bool {
	usageMonitors := []string{"cpu_usage", "memory_usage", "gpu_usage"}
	for _, usage := range usageMonitors {
		if monitorName == usage {
			return true
		}
	}
	return false
}

func isNetworkMonitor(monitorName string) bool {
	networkMonitors := []string{"net_upload", "net_download", "net1_upload", "net1_download"}
	for _, network := range networkMonitors {
		if monitorName == network {
			return true
		}
	}
	return false
}

func isDiskSpeedMonitor(monitorName string) bool {
	diskSpeedMonitors := []string{"disk_total_read_speed", "disk_total_write_speed"}
	for _, diskSpeed := range diskSpeedMonitors {
		if monitorName == diskSpeed {
			return true
		}
	}
	return false
}

func (config *MonitorConfig) GetUnitText(monitorName string, defaultUnit string) string {
	if config.Units != nil {
		if unit, exists := config.Units[monitorName]; exists {
			return unit
		}
	}
	return defaultUnit
}

func (config *MonitorConfig) GetNetworkInterface() string {
	if config.NetworkInterface == "" {
		return "auto"
	}
	return config.NetworkInterface
}
