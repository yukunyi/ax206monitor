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

type MonitorConfig struct {
	Name            string            `json:"name"`
	Width           int               `json:"width"`
	Height          int               `json:"height"`
	FontSizes       FontSizes         `json:"font_sizes"`
	FontFamilies    []string          `json:"font_families"`
	OutputType      string            `json:"output_type"`
	OutputFile      string            `json:"output_file,omitempty"`
	RefreshInterval int               `json:"refresh_interval"`
	HistorySize     int               `json:"history_size,omitempty"`
	Colors          map[string]string `json:"colors"`
	Items           []ItemConfig      `json:"items"`
	Labels          map[string]string `json:"labels,omitempty"`
	Units           map[string]string `json:"units,omitempty"`
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

func (config *MonitorConfig) GetUnitText(monitorName string, defaultUnit string) string {
	if config.Units != nil {
		if unit, exists := config.Units[monitorName]; exists {
			return unit
		}
	}
	return defaultUnit
}
