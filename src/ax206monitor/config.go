package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"time"
)

type CustomMonitorConfig struct {
	Name      string   `json:"name"`
	Label     string   `json:"label,omitempty"`
	Type      string   `json:"type"`
	Unit      string   `json:"unit,omitempty"`
	Precision *int     `json:"precision,omitempty"`
	Min       *float64 `json:"min,omitempty"`
	Max       *float64 `json:"max,omitempty"`

	// File monitor
	Path   string   `json:"path,omitempty"`
	Scale  *float64 `json:"scale,omitempty"`
	Offset float64  `json:"offset,omitempty"`

	// Mixed monitor
	Sources   []string `json:"sources,omitempty"`
	Aggregate string   `json:"aggregate,omitempty"`

	// CoolerControl monitor
	Source string `json:"source,omitempty"`

	// LibreHardwareMonitor sensor
	// Reuse Source field.
}

type CollectorConfig struct {
	Enabled *bool                  `json:"enabled,omitempty"`
	Options map[string]interface{} `json:"options,omitempty"`
}

type MonitorConfig struct {
	Name                    string                     `json:"name"`
	Width                   int                        `json:"width"`
	Height                  int                        `json:"height"`
	LayoutPadding           int                        `json:"layout_padding,omitempty"`
	MonitorUpdateWorkers    int                        `json:"monitor_update_workers,omitempty"`
	MonitorUpdateQueueSize  int                        `json:"monitor_update_queue_size,omitempty"`
	MonitorAutoTune         *bool                      `json:"monitor_auto_tune,omitempty"`
	MonitorAutoTuneInterval int                        `json:"monitor_auto_tune_interval_sec,omitempty"`
	MonitorAutoTuneSlowRate float64                    `json:"monitor_auto_tune_slow_rate,omitempty"`
	MonitorAutoTuneStable   int                        `json:"monitor_auto_tune_stable_runs,omitempty"`
	MonitorAutoTuneMaxScale int                        `json:"monitor_auto_tune_max_scale,omitempty"`
	DefaultFont             string                     `json:"default_font,omitempty"`
	DefaultFontSize         int                        `json:"default_font_size,omitempty"`
	DefaultColor            string                     `json:"default_color,omitempty"`
	DefaultBackground       string                     `json:"default_background,omitempty"`
	LevelColors             []string                   `json:"level_colors,omitempty"`
	DefaultThresholds       []float64                  `json:"default_thresholds,omitempty"`
	FontFamilies            []string                   `json:"font_families"`
	OutputTypes             []string                   `json:"output_types"`
	RefreshInterval         int                        `json:"refresh_interval"`
	HistorySize             int                        `json:"history_size,omitempty"`
	NetworkInterface        string                     `json:"network_interface,omitempty"`
	EnableRTSSCollect       bool                       `json:"enable_rtss_collect,omitempty"`
	LibreHardwareMonitorURL string                     `json:"libre_hardware_monitor_url,omitempty"`
	CoolerControlURL        string                     `json:"coolercontrol_url,omitempty"`
	CoolerControlUsername   string                     `json:"coolercontrol_username,omitempty"`
	CoolerControlPassword   string                     `json:"coolercontrol_password,omitempty"`
	CollectorConfig         map[string]CollectorConfig `json:"collector_config,omitempty"`
	CustomMonitors          []CustomMonitorConfig      `json:"custom_monitors,omitempty"`
	Items                   []ItemConfig               `json:"items"`
}

type ItemConfig struct {
	Type           string                 `json:"type"`
	EditUIName     string                 `json:"edit_ui_name,omitempty"`
	Monitor        string                 `json:"monitor,omitempty"`
	Unit           string                 `json:"unit,omitempty"`
	UnitColor      string                 `json:"unit_color,omitempty"`
	UnitFontSize   int                    `json:"unit_font_size,omitempty"`
	X              int                    `json:"x"`
	Y              int                    `json:"y"`
	Width          int                    `json:"width"`
	Height         int                    `json:"height"`
	FontSize       int                    `json:"font_size,omitempty"`
	Color          string                 `json:"color,omitempty"`
	Background     string                 `json:"bg,omitempty"`
	History        bool                   `json:"history,omitempty"`
	PointSize      int                    `json:"point_size,omitempty"`
	Max            float64                `json:"max,omitempty"`
	MaxValue       *float64               `json:"max_value,omitempty"`
	MinValue       *float64               `json:"min_value,omitempty"`
	Text           string                 `json:"text,omitempty"`
	Thresholds     []float64              `json:"thresholds,omitempty"`
	LevelColors    []string               `json:"level_colors,omitempty"`
	BorderColor    string                 `json:"border_color,omitempty"`
	BorderWidth    float64                `json:"border_width,omitempty"`
	Radius         int                    `json:"radius,omitempty"`
	RenderAttrsMap map[string]interface{} `json:"render_attrs_map,omitempty"`
}

func (item *ItemConfig) UnmarshalJSON(data []byte) error {
	type alias ItemConfig
	var decoded alias
	if err := json.Unmarshal(data, &decoded); err != nil {
		return err
	}
	*item = ItemConfig(decoded)

	raw := make(map[string]json.RawMessage)
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil
	}

	mergeAttrs := func(key string) {
		payload, exists := raw[key]
		if !exists {
			return
		}
		attrs := make(map[string]interface{})
		if err := json.Unmarshal(payload, &attrs); err != nil {
			return
		}
		if len(attrs) == 0 {
			return
		}
		if item.RenderAttrsMap == nil {
			item.RenderAttrsMap = make(map[string]interface{}, len(attrs))
		}
		for attrKey, attrValue := range attrs {
			if _, exists := item.RenderAttrsMap[attrKey]; !exists {
				item.RenderAttrsMap[attrKey] = attrValue
			}
		}
	}

	mergeAttrs("render_attrs_map")
	mergeAttrs("renderAttrsMap")
	return nil
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
	normalizeMonitorConfig(&config)

	cm.configs[configName] = &config
	return &config, nil
}

func (cm *ConfigManager) ListConfigs() ([]string, error) {
	files, err := os.ReadDir(cm.configDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read config directory: %v", err)
	}

	configs := make([]string, 0, len(files))
	for _, file := range files {
		if file.IsDir() || filepath.Ext(file.Name()) != ".json" {
			continue
		}
		configs = append(configs, file.Name()[:len(file.Name())-5])
	}
	sort.Strings(configs)
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

func defaultLevelColors() []string {
	return []string{"#22c55e", "#eab308", "#f97316", "#ef4444"}
}

func (config *MonitorConfig) GetDefaultFontName() string {
	if strings.TrimSpace(config.DefaultFont) != "" {
		return strings.TrimSpace(config.DefaultFont)
	}
	if len(config.FontFamilies) > 0 && strings.TrimSpace(config.FontFamilies[0]) != "" {
		return strings.TrimSpace(config.FontFamilies[0])
	}
	return getDefaultFontFamilies()[0]
}

func (config *MonitorConfig) GetDefaultFontSize() int {
	if config.DefaultFontSize > 0 {
		return config.DefaultFontSize
	}
	return 16
}

func (config *MonitorConfig) GetDefaultTextColor() string {
	if strings.TrimSpace(config.DefaultColor) != "" {
		return strings.TrimSpace(config.DefaultColor)
	}
	return "#f8fafc"
}

func (config *MonitorConfig) GetDefaultBackgroundColor() string {
	if strings.TrimSpace(config.DefaultBackground) != "" {
		return strings.TrimSpace(config.DefaultBackground)
	}
	return "#0b1220"
}

func (config *MonitorConfig) GetLevelColors() []string {
	colors := make([]string, 0, 4)
	for _, color := range config.LevelColors {
		trimmed := strings.TrimSpace(color)
		if trimmed != "" {
			colors = append(colors, trimmed)
		}
	}
	if len(colors) == 0 {
		colors = append(colors, defaultLevelColors()...)
	}
	for len(colors) < 4 {
		colors = append(colors, colors[len(colors)-1])
	}
	if len(colors) > 4 {
		colors = colors[:4]
	}
	return colors
}

func (config *MonitorConfig) GetResolvedThresholds(minValue, maxValue float64) []float64 {
	thresholds := normalizeThresholds(config.DefaultThresholds, minValue, maxValue)
	if len(thresholds) == 4 {
		return thresholds
	}
	return buildAverageThresholds(minValue, maxValue)
}

func normalizeThresholds(raw []float64, minValue, maxValue float64) []float64 {
	thresholds := make([]float64, 0, 4)
	for _, value := range raw {
		thresholds = append(thresholds, value)
		if len(thresholds) == 4 {
			break
		}
	}
	if len(thresholds) == 0 {
		return nil
	}
	for len(thresholds) < 4 {
		thresholds = append(thresholds, thresholds[len(thresholds)-1])
	}
	sort.Float64s(thresholds)

	if maxValue > minValue {
		if thresholds[0] < minValue {
			thresholds[0] = minValue
		}
		for i := 1; i < len(thresholds); i++ {
			if thresholds[i] < thresholds[i-1] {
				thresholds[i] = thresholds[i-1]
			}
		}
		if thresholds[3] > maxValue {
			thresholds[3] = maxValue
		}
	}
	return thresholds
}

func buildAverageThresholds(minValue, maxValue float64) []float64 {
	if maxValue <= minValue {
		minValue = 0
		maxValue = 100
	}
	step := (maxValue - minValue) / 4.0
	return []float64{
		minValue + step,
		minValue + 2*step,
		minValue + 3*step,
		maxValue,
	}
}

func (config *MonitorConfig) GetNetworkInterface() string {
	if value := config.GetCollectorStringOption("go_native.network", "interface", ""); strings.TrimSpace(value) != "" {
		if strings.EqualFold(strings.TrimSpace(value), "auto") {
			return ""
		}
		return strings.TrimSpace(value)
	}
	if strings.EqualFold(strings.TrimSpace(config.NetworkInterface), "auto") {
		return ""
	}
	return strings.TrimSpace(config.NetworkInterface)
}

func (config *MonitorConfig) IsRTSSCollectEnabled() bool {
	if runtime.GOOS != "windows" {
		return false
	}
	if enabled := config.IsCollectorEnabled("external.rtss", false); enabled {
		return true
	}
	return config.EnableRTSSCollect
}

func (config *MonitorConfig) GetMonitorUpdateWorkers() int {
	workers := config.MonitorUpdateWorkers
	if workers <= 0 {
		return defaultMonitorWorkerCount()
	}
	if workers > 64 {
		return 64
	}
	return workers
}

func (config *MonitorConfig) GetMonitorUpdateQueueSize(workers int) int {
	if workers <= 0 {
		workers = config.GetMonitorUpdateWorkers()
	}
	queueSize := config.MonitorUpdateQueueSize
	if queueSize <= 0 {
		return defaultMonitorQueueSize(workers)
	}
	if queueSize < workers {
		queueSize = workers
	}
	if queueSize > 4096 {
		queueSize = 4096
	}
	return queueSize
}

func (config *MonitorConfig) GetMonitorAutoTune() bool {
	if config.MonitorAutoTune == nil {
		return true
	}
	return *config.MonitorAutoTune
}

func (config *MonitorConfig) GetMonitorAutoTuneInterval() time.Duration {
	seconds := config.MonitorAutoTuneInterval
	if seconds <= 0 {
		return 5 * time.Second
	}
	if seconds > 60 {
		seconds = 60
	}
	return time.Duration(seconds) * time.Second
}

func (config *MonitorConfig) GetMonitorAutoTuneSlowRate() float64 {
	rate := config.MonitorAutoTuneSlowRate
	if rate <= 0 {
		return 0.20
	}
	if rate > 1 {
		return 1
	}
	if rate < 0.01 {
		return 0.01
	}
	return rate
}

func (config *MonitorConfig) GetMonitorAutoTuneStableRuns() int {
	runs := config.MonitorAutoTuneStable
	if runs <= 0 {
		return 3
	}
	if runs > 20 {
		return 20
	}
	return runs
}

func (config *MonitorConfig) GetMonitorAutoTuneMaxScale() int {
	scale := config.MonitorAutoTuneMaxScale
	if scale <= 0 {
		return 8
	}
	if scale > 64 {
		return 64
	}
	return scale
}

func (config *MonitorConfig) GetCoolerControlURL() string {
	if url := config.GetCollectorStringOption("external.coolercontrol", "url", ""); url != "" {
		return normalizeEndpointURL(url)
	}
	return normalizeEndpointURL(config.CoolerControlURL)
}

func (config *MonitorConfig) GetCoolerControlUsername() string {
	if username := strings.TrimSpace(config.GetCollectorStringOption("external.coolercontrol", "username", "")); username != "" {
		return username
	}
	if config.CoolerControlUsername != "" {
		return config.CoolerControlUsername
	}
	if config.GetCoolerControlPassword() != "" {
		return "CCAdmin"
	}
	return ""
}

func (config *MonitorConfig) GetCoolerControlPassword() string {
	if password := config.GetCollectorStringOption("external.coolercontrol", "password", ""); password != "" {
		return password
	}
	return config.CoolerControlPassword
}

func (config *MonitorConfig) GetLibreHardwareMonitorURL() string {
	if url := config.GetCollectorStringOption("external.librehardwaremonitor", "url", ""); url != "" {
		return normalizeEndpointURL(url)
	}
	return normalizeEndpointURL(config.LibreHardwareMonitorURL)
}

func (config *MonitorConfig) GetCollectorConfig(name string) CollectorConfig {
	if config == nil || config.CollectorConfig == nil {
		return CollectorConfig{}
	}
	return config.CollectorConfig[strings.TrimSpace(name)]
}

func (config *MonitorConfig) IsCollectorEnabled(name string, defaultValue bool) bool {
	collector := config.GetCollectorConfig(name)
	if collector.Enabled == nil {
		return defaultValue
	}
	return *collector.Enabled
}

func (config *MonitorConfig) GetCollectorStringOption(name, key, defaultValue string) string {
	collector := config.GetCollectorConfig(name)
	if collector.Options == nil {
		return defaultValue
	}
	value, exists := collector.Options[strings.TrimSpace(key)]
	if !exists || value == nil {
		return defaultValue
	}
	switch typed := value.(type) {
	case string:
		if strings.TrimSpace(typed) == "" {
			return defaultValue
		}
		return typed
	default:
		text := strings.TrimSpace(fmt.Sprintf("%v", typed))
		if text == "" {
			return defaultValue
		}
		return text
	}
}

func normalizeEndpointURL(raw string) string {
	url := strings.TrimSpace(raw)
	if url == "" {
		return ""
	}
	if !strings.Contains(url, "://") {
		url = "http://" + url
	}
	return strings.TrimRight(url, "/")
}
