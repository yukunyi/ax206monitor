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

const (
	defaultCoolerControlURL        = "http://127.0.0.1:11987"
	defaultLibreHardwareMonitorURL = "http://127.0.0.1:8085"
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

type ItemTypeDefaults struct {
	Style          map[string]interface{} `json:"style,omitempty"`
	RenderAttrsMap map[string]interface{} `json:"render_attrs_map,omitempty"`
}

type MonitorConfig struct {
	Name                    string                      `json:"name"`
	Width                   int                         `json:"width"`
	Height                  int                         `json:"height"`
	LayoutPadding           int                         `json:"layout_padding,omitempty"`
	MonitorUpdateWorkers    int                         `json:"monitor_update_workers,omitempty"`
	MonitorUpdateQueueSize  int                         `json:"monitor_update_queue_size,omitempty"`
	DefaultFont             string                      `json:"default_font,omitempty"`
	StyleBase               map[string]interface{}      `json:"style_base,omitempty"`
	AllowCustomStyle        bool                        `json:"allow_custom_style,omitempty"`
	FontFamilies            []string                    `json:"font_families"`
	Outputs                 []OutputConfig              `json:"outputs"`
	OutputTypes             []string                    `json:"output_types"`
	PauseCollectOnLock      bool                        `json:"pause_collect_on_lock,omitempty"`
	RefreshInterval         int                         `json:"refresh_interval"`
	CollectWarnMS           int                         `json:"collect_warn_ms,omitempty"`
	RenderWaitMaxMS         int                         `json:"render_wait_max_ms,omitempty"`
	HistorySize             int                         `json:"history_size,omitempty"`
	DefaultHistoryPoints    int                         `json:"default_history_points,omitempty"`
	NetworkInterface        string                      `json:"network_interface,omitempty"`
	EnableRTSSCollect       bool                        `json:"enable_rtss_collect,omitempty"`
	LibreHardwareMonitorURL string                      `json:"libre_hardware_monitor_url,omitempty"`
	CoolerControlURL        string                      `json:"coolercontrol_url,omitempty"`
	CoolerControlUsername   string                      `json:"coolercontrol_username,omitempty"`
	CoolerControlPassword   string                      `json:"coolercontrol_password,omitempty"`
	CollectorConfig         map[string]CollectorConfig  `json:"collector_config,omitempty"`
	TypeDefaults            map[string]ItemTypeDefaults `json:"type_defaults,omitempty"`
	CustomMonitors          []CustomMonitorConfig       `json:"custom_monitors,omitempty"`
	Items                   []ItemConfig                `json:"items"`
}

type ItemConfig struct {
	ID             string                 `json:"id,omitempty"`
	Type           string                 `json:"type"`
	EditUIName     string                 `json:"edit_ui_name,omitempty"`
	CustomStyle    bool                   `json:"custom_style,omitempty"`
	Monitor        string                 `json:"monitor,omitempty"`
	Unit           string                 `json:"unit,omitempty"`
	X              int                    `json:"x"`
	Y              int                    `json:"y"`
	Width          int                    `json:"width"`
	Height         int                    `json:"height"`
	Max            float64                `json:"max,omitempty"`
	MaxValue       *float64               `json:"max_value,omitempty"`
	MinValue       *float64               `json:"min_value,omitempty"`
	Text           string                 `json:"text,omitempty"`
	Style          map[string]interface{} `json:"style,omitempty"`
	RenderAttrsMap map[string]interface{} `json:"render_attrs_map,omitempty"`
	runtime        renderItemRuntime
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
	switch runtime.GOOS {
	case "windows":
		return []string{
			"Microsoft YaHei UI",
			"Microsoft YaHei",
			"Segoe UI",
			"Consolas",
			"Arial",
			"SimSun",
			"Courier New",
			"monospace",
		}
	case "darwin":
		return []string{
			"SF Mono",
			"PingFang SC",
			"Menlo",
			"Monaco",
			"Helvetica",
			"Courier New",
			"monospace",
		}
	default:
		return []string{
			"Noto Sans CJK SC",
			"WenQuanYi Micro Hei",
			"DejaVu Sans Mono",
			"Liberation Mono",
			"Ubuntu Mono",
			"Courier New",
			"monospace",
		}
	}
}

func getDefaultFontName() string {
	families := getDefaultFontFamilies()
	if len(families) == 0 {
		return ""
	}
	return families[0]
}

func (config *MonitorConfig) GetDefaultFontName() string {
	if styleFont := strings.TrimSpace(resolveStyleString(nil, config, "font_family", "")); styleFont != "" {
		return styleFont
	}
	if strings.TrimSpace(config.DefaultFont) != "" {
		return strings.TrimSpace(config.DefaultFont)
	}
	if len(config.FontFamilies) > 0 && strings.TrimSpace(config.FontFamilies[0]) != "" {
		return strings.TrimSpace(config.FontFamilies[0])
	}
	return getDefaultFontName()
}

func (config *MonitorConfig) GetDefaultFontSize() int {
	size := resolveStyleInt(nil, config, "text_font_size", 0)
	if size > 0 {
		return size
	}
	return 16
}

func (config *MonitorConfig) GetDefaultValueFontSize() int {
	size := resolveStyleInt(nil, config, "value_font_size", 0)
	if size > 0 {
		return size
	}
	return 18
}

func (config *MonitorConfig) GetDefaultLabelFontSize() int {
	size := resolveStyleInt(nil, config, "text_font_size", 0)
	if size > 0 {
		return size
	}
	return 16
}

func (config *MonitorConfig) GetDefaultUnitFontSize() int {
	size := resolveStyleInt(nil, config, "unit_font_size", 0)
	if size > 0 {
		return size
	}
	return 14
}

func (config *MonitorConfig) GetDefaultHistoryPoints() int {
	points := config.DefaultHistoryPoints
	if points <= 0 {
		points = config.HistorySize
	}
	if points <= 0 {
		points = 150
	}
	if points < 10 {
		points = 10
	}
	if points > 5000 {
		points = 5000
	}
	return points
}

func (config *MonitorConfig) GetDefaultTextColor() string {
	if color := strings.TrimSpace(resolveStyleColor(nil, config, "color", "")); color != "" {
		return color
	}
	return "#f8fafc"
}

func (config *MonitorConfig) GetDefaultBackgroundColor() string {
	return "#0b1220"
}

func (config *MonitorConfig) GetTypeDefaults(itemType string) ItemTypeDefaults {
	if config == nil {
		return ItemTypeDefaults{}
	}
	normalizedType := normalizeItemTypeName(itemType)
	if config.TypeDefaults == nil {
		return ItemTypeDefaults{}
	}
	defaults, exists := config.TypeDefaults[normalizedType]
	if !exists {
		return ItemTypeDefaults{}
	}
	if defaults.RenderAttrsMap == nil {
		defaults.RenderAttrsMap = map[string]interface{}{}
	}
	return defaults
}

func normalizeThresholds(raw []float64, minValue, maxValue float64) []float64 {
	percentages := make([]float64, 0, 4)
	for _, value := range raw {
		if value < 0 {
			value = 0
		}
		if value > 100 {
			value = 100
		}
		percentages = append(percentages, value)
		if len(percentages) == 4 {
			break
		}
	}
	if len(percentages) == 0 {
		return nil
	}
	for len(percentages) < 4 {
		percentages = append(percentages, percentages[len(percentages)-1])
	}
	sort.Float64s(percentages)

	if maxValue <= minValue {
		minValue = 0
		maxValue = 100
	}
	span := maxValue - minValue
	thresholds := make([]float64, 4)
	for i := 0; i < 4; i++ {
		thresholds[i] = minValue + span*(percentages[i]/100.0)
	}
	for i := 1; i < len(thresholds); i++ {
		if thresholds[i] < thresholds[i-1] {
			thresholds[i] = thresholds[i-1]
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
	if enabled := config.IsCollectorEnabled(collectorRTSS, false); enabled {
		return true
	}
	return config.EnableRTSSCollect
}

func (config *MonitorConfig) GetCollectTickDuration() time.Duration {
	intervalMS := config.RefreshInterval
	if intervalMS <= 0 {
		intervalMS = 1000
	}
	if intervalMS < 100 {
		intervalMS = 100
	}
	if intervalMS > 10_000 {
		intervalMS = 10_000
	}
	return time.Duration(intervalMS) * time.Millisecond
}

func (config *MonitorConfig) GetCollectWarnDuration() time.Duration {
	warnMS := config.CollectWarnMS
	if warnMS <= 0 {
		warnMS = 100
	}
	if warnMS < 10 {
		warnMS = 10
	}
	if warnMS > 10_000 {
		warnMS = 10_000
	}
	return time.Duration(warnMS) * time.Millisecond
}

func (config *MonitorConfig) GetRenderWaitMaxDuration() time.Duration {
	waitMS := config.RenderWaitMaxMS
	if waitMS <= 0 {
		waitMS = 300
	}
	if waitMS < 0 {
		waitMS = 0
	}
	tick := config.GetCollectTickDuration()
	maxByTick := int(tick / time.Millisecond)
	if waitMS > maxByTick {
		waitMS = maxByTick
	}
	return time.Duration(waitMS) * time.Millisecond
}

func (config *MonitorConfig) IsPauseCollectOnLockEnabled() bool {
	if config == nil {
		return false
	}
	return config.PauseCollectOnLock
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

func (config *MonitorConfig) GetCoolerControlURL() string {
	if url := config.GetCollectorStringOption(collectorCoolerControl, "url", ""); url != "" {
		return normalizeEndpointURL(url)
	}
	if url := normalizeEndpointURL(config.CoolerControlURL); url != "" {
		return url
	}
	return normalizeEndpointURL(defaultCoolerControlURL)
}

func (config *MonitorConfig) GetCoolerControlUsername() string {
	if username := strings.TrimSpace(config.GetCollectorStringOption(collectorCoolerControl, "username", "")); username != "" {
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
	if password := config.GetCollectorStringOption(collectorCoolerControl, "password", ""); password != "" {
		return password
	}
	return config.CoolerControlPassword
}

func (config *MonitorConfig) GetLibreHardwareMonitorURL() string {
	if url := config.GetCollectorStringOption(collectorLibreHardwareMonitor, "url", ""); url != "" {
		return normalizeEndpointURL(url)
	}
	if url := normalizeEndpointURL(config.LibreHardwareMonitorURL); url != "" {
		return url
	}
	return normalizeEndpointURL(defaultLibreHardwareMonitorURL)
}

func (config *MonitorConfig) GetLibreHardwareMonitorUsername() string {
	return strings.TrimSpace(config.GetCollectorStringOption(collectorLibreHardwareMonitor, "username", ""))
}

func (config *MonitorConfig) GetLibreHardwareMonitorPassword() string {
	return config.GetCollectorStringOption(collectorLibreHardwareMonitor, "password", "")
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
