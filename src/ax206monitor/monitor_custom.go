package main

import (
	"math"
	"strconv"
	"strings"
)

type CustomFileMonitor struct {
	*BaseMonitorItem
	path   string
	scale  *float64
	offset float64
}

func NewCustomFileMonitor(cfg *CustomMonitorConfig) MonitorItem {
	return &CustomFileMonitor{
		BaseMonitorItem: buildCustomMonitorBase(cfg, cfg.Name, "°C", 0, 0, 120),
		path:            cfg.Path,
		scale:           cfg.Scale,
		offset:          cfg.Offset,
	}
}

func (m *CustomFileMonitor) Update() error {
	content, err := readSysFile(m.path)
	if err != nil {
		m.SetAvailable(false)
		return nil
	}

	rawValue, err := strconv.ParseFloat(content, 64)
	if err != nil {
		m.SetAvailable(false)
		return nil
	}

	value := rawValue
	if m.scale != nil {
		value = value * (*m.scale)
	} else if math.Abs(value) > 500 {
		// Most temp sensors in sysfs use milli-degree Celsius.
		value = value / 1000.0
	}
	value += m.offset

	m.SetValue(value)
	m.SetAvailable(true)
	return nil
}

type CustomMixedMonitor struct {
	*BaseMonitorItem
	registry  *MonitorRegistry
	sources   []string
	aggregate string
}

func NewCustomMixedMonitor(cfg *CustomMonitorConfig, registry *MonitorRegistry) MonitorItem {
	return &CustomMixedMonitor{
		BaseMonitorItem: buildCustomMonitorBase(cfg, cfg.Name, "°C", 0, 0, 120),
		registry:        registry,
		sources:         cfg.Sources,
		aggregate:       normalizeAggregateMethod(cfg.Aggregate),
	}
}

func (m *CustomMixedMonitor) Update() error {
	if m.registry == nil || len(m.sources) == 0 {
		m.SetAvailable(false)
		return nil
	}

	values := make([]float64, 0, len(m.sources))
	for _, sourceName := range m.sources {
		source := m.registry.Get(sourceName)
		if source == nil || !source.IsAvailable() {
			continue
		}

		monitorValue := source.GetValue()
		if monitorValue == nil {
			continue
		}

		switch v := monitorValue.Value.(type) {
		case float64, float32, int, int64, uint64:
			values = append(values, getFloat64Value(v))
		}
	}

	if len(values) == 0 {
		m.SetAvailable(false)
		return nil
	}

	result := values[0]
	switch m.aggregate {
	case "min":
		for _, value := range values[1:] {
			if value < result {
				result = value
			}
		}
	case "avg":
		sum := 0.0
		for _, value := range values {
			sum += value
		}
		result = sum / float64(len(values))
	default:
		for _, value := range values[1:] {
			if value > result {
				result = value
			}
		}
	}

	m.SetValue(result)
	m.SetAvailable(true)
	return nil
}

type CoolerControlMonitor struct {
	*BaseMonitorItem
	registry      *MonitorRegistry
	source        string
	specifiedUnit string
}

func NewCoolerControlMonitor(cfg *CustomMonitorConfig, registry *MonitorRegistry) MonitorItem {
	defaultUnit := strings.TrimSpace(cfg.Unit)
	return &CoolerControlMonitor{
		BaseMonitorItem: buildCustomMonitorBase(cfg, cfg.Name, defaultUnit, 2, 0, 0),
		registry:        registry,
		source:          strings.TrimSpace(cfg.Source),
		specifiedUnit:   defaultUnit,
	}
}

func (m *CoolerControlMonitor) Update() error {
	if m.registry == nil || strings.TrimSpace(m.source) == "" {
		m.SetAvailable(false)
		return nil
	}
	sourceItem := m.registry.Get(m.source)
	if sourceItem == nil || !sourceItem.IsAvailable() {
		m.SetAvailable(false)
		return nil
	}
	value := sourceItem.GetValue()
	if value == nil {
		m.SetAvailable(false)
		return nil
	}
	if strings.TrimSpace(m.specifiedUnit) == "" && strings.TrimSpace(value.Unit) != "" {
		m.SetUnit(value.Unit)
	}
	m.SetValue(value.Value)
	m.SetAvailable(true)
	return nil
}

type LibreHardwareMonitorSensor struct {
	*BaseMonitorItem
	registry      *MonitorRegistry
	source        string
	specifiedUnit string
}

func NewLibreHardwareMonitorSensor(cfg *CustomMonitorConfig, registry *MonitorRegistry) MonitorItem {
	return &LibreHardwareMonitorSensor{
		BaseMonitorItem: buildCustomMonitorBase(cfg, cfg.Name, strings.TrimSpace(cfg.Unit), 2, 0, 0),
		registry:        registry,
		source:          strings.TrimSpace(cfg.Source),
		specifiedUnit:   strings.TrimSpace(cfg.Unit),
	}
}

func (m *LibreHardwareMonitorSensor) Update() error {
	if m.registry == nil || m.source == "" {
		m.SetAvailable(false)
		return nil
	}
	sourceItem := m.registry.Get(m.source)
	if sourceItem == nil || !sourceItem.IsAvailable() {
		m.SetAvailable(false)
		return nil
	}
	value := sourceItem.GetValue()
	if value == nil {
		m.SetAvailable(false)
		return nil
	}
	if strings.TrimSpace(m.specifiedUnit) == "" && strings.TrimSpace(value.Unit) != "" {
		m.SetUnit(value.Unit)
	}
	m.SetValue(value.Value)
	m.SetAvailable(true)
	return nil
}

func buildCustomMonitorBase(cfg *CustomMonitorConfig, defaultLabel, defaultUnit string, defaultPrecision int, defaultMin, defaultMax float64) *BaseMonitorItem {
	label := cfg.Label
	if label == "" {
		label = defaultLabel
	}

	unit := cfg.Unit
	if unit == "" {
		unit = defaultUnit
	}

	precision := defaultPrecision
	if cfg.Precision != nil {
		precision = *cfg.Precision
	}

	min := defaultMin
	if cfg.Min != nil {
		min = *cfg.Min
	}

	max := defaultMax
	if cfg.Max != nil {
		max = *cfg.Max
	}

	return NewBaseMonitorItem(cfg.Name, label, min, max, unit, precision)
}

func normalizeAggregateMethod(method string) string {
	switch strings.ToLower(strings.TrimSpace(method)) {
	case "min", "lowest", "low", "minimum", "min_value", "最低", "最小":
		return "min"
	case "avg", "average", "mean", "平均":
		return "avg"
	default:
		return "max"
	}
}

func initializeCustomMonitors(registry *MonitorRegistry) {
	config := GetGlobalMonitorConfig()
	if config == nil || len(config.CustomMonitors) == 0 {
		return
	}

	for i := range config.CustomMonitors {
		custom := config.CustomMonitors[i]
		if strings.TrimSpace(custom.Name) == "" {
			logWarnModule("custom", "skip custom monitor #%d: empty name", i+1)
			continue
		}

		if existing := registry.Get(custom.Name); existing != nil {
			logWarnModule("custom", "skip custom monitor '%s': name already exists", custom.Name)
			continue
		}

		var monitor MonitorItem
		switch normalizeCustomMonitorType(custom.Type) {
		case "file":
			if strings.TrimSpace(custom.Path) == "" {
				logWarnModule("custom", "skip custom monitor '%s': missing path", custom.Name)
				continue
			}
			monitor = NewCustomFileMonitor(&custom)
		case "mixed":
			if len(custom.Sources) == 0 {
				logWarnModule("custom", "skip custom monitor '%s': missing sources", custom.Name)
				continue
			}
			monitor = NewCustomMixedMonitor(&custom, registry)
		case "coolercontrol":
			if strings.TrimSpace(custom.Source) == "" {
				logWarnModule("custom", "skip custom monitor '%s': missing source", custom.Name)
				continue
			}
			monitor = NewCoolerControlMonitor(&custom, registry)
		case "librehardwaremonitor":
			if strings.TrimSpace(custom.Source) == "" {
				logWarnModule("custom", "skip custom monitor '%s': missing source", custom.Name)
				continue
			}
			monitor = NewLibreHardwareMonitorSensor(&custom, registry)
		default:
			logWarnModule("custom", "skip custom monitor '%s': unsupported type '%s'", custom.Name, custom.Type)
			continue
		}

		registry.Register(monitor)
		logInfoModule("custom", "registered custom monitor '%s' (%s)", custom.Name, custom.Type)
	}
}
