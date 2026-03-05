package main

import (
	"math"
	"strconv"
	"strings"
)

type customEntry struct {
	cfg  CustomMonitorConfig
	item *CollectItem
}

type CustomCollector struct {
	*BaseCollector
	cfg    *MonitorConfig
	lookup func(string) *CollectItem
	items  map[string]customEntry
}

func NewCustomCollector(cfg *MonitorConfig, lookup func(string) *CollectItem) *CustomCollector {
	collector := &CustomCollector{
		BaseCollector: NewBaseCollector("custom.all"),
		cfg:           cfg,
		lookup:        lookup,
		items:         make(map[string]customEntry),
	}
	if cfg != nil {
		collector.SetEnabled(cfg.IsCollectorEnabled("custom.all", true))
	}
	return collector
}

func (c *CustomCollector) GetAllItems() map[string]*CollectItem {
	if c.cfg == nil {
		return c.ItemsSnapshot()
	}
	for _, custom := range c.cfg.CustomMonitors {
		name := strings.TrimSpace(custom.Name)
		if name == "" {
			continue
		}
		if _, exists := c.items[name]; exists {
			continue
		}
		item := buildCustomCollectItem(&custom, custom.Name, "", 2, 0, 0)
		c.items[name] = customEntry{cfg: custom, item: item}
		c.setItem(name, item)
	}
	return c.ItemsSnapshot()
}

func (c *CustomCollector) UpdateItems() error {
	if !c.IsEnabled() {
		return nil
	}
	_ = c.GetAllItems()

	for _, entry := range c.items {
		item := entry.item
		if item == nil || !item.IsEnabled() {
			continue
		}
		custom := entry.cfg
		switch normalizeCustomMonitorType(custom.Type) {
		case "file":
			content, err := readSysFile(custom.Path)
			if err != nil {
				item.SetAvailable(false)
				continue
			}
			value, err := strconv.ParseFloat(content, 64)
			if err != nil {
				item.SetAvailable(false)
				continue
			}
			if custom.Scale != nil {
				value *= *custom.Scale
			} else if math.Abs(value) > 500 {
				value /= 1000.0
			}
			value += custom.Offset
			item.SetValue(value)
			item.SetAvailable(true)
		case "mixed":
			values := make([]float64, 0, len(custom.Sources))
			for _, sourceName := range custom.Sources {
				sourceName = strings.TrimSpace(sourceName)
				if sourceName == "" || c.lookup == nil {
					continue
				}
				source := c.lookup(sourceName)
				if source == nil || !source.IsAvailable() {
					continue
				}
				value := source.GetValue()
				if value == nil {
					continue
				}
				switch v := value.Value.(type) {
				case float64, float32, int, int64, uint64:
					values = append(values, getFloat64Value(v))
				}
			}
			if len(values) == 0 {
				item.SetAvailable(false)
				continue
			}
			result := values[0]
			switch normalizeAggregateMethod(custom.Aggregate) {
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
			item.SetValue(result)
			item.SetAvailable(true)
		case "coolercontrol", "librehardwaremonitor":
			sourceKey := strings.TrimSpace(custom.Source)
			if sourceKey == "" || c.lookup == nil {
				item.SetAvailable(false)
				continue
			}
			source := c.lookup(sourceKey)
			if source == nil || !source.IsAvailable() {
				item.SetAvailable(false)
				continue
			}
			value := source.GetValue()
			if value == nil {
				item.SetAvailable(false)
				continue
			}
			if strings.TrimSpace(custom.Unit) == "" && strings.TrimSpace(value.Unit) != "" {
				item.SetUnit(value.Unit)
			}
			item.SetValue(value.Value)
			item.SetAvailable(true)
		default:
			item.SetAvailable(false)
		}
	}

	return nil
}

func buildCustomCollectItem(
	cfg *CustomMonitorConfig,
	defaultLabel string,
	defaultUnit string,
	defaultPrecision int,
	defaultMin float64,
	defaultMax float64,
) *CollectItem {
	if cfg == nil {
		return NewCollectItem("", strings.TrimSpace(defaultLabel), strings.TrimSpace(defaultUnit), defaultMin, defaultMax, max(0, defaultPrecision))
	}
	name := strings.TrimSpace(cfg.Name)
	label := strings.TrimSpace(cfg.Label)
	unit := strings.TrimSpace(cfg.Unit)
	if label == "" {
		label = strings.TrimSpace(defaultLabel)
	}
	if unit == "" {
		unit = strings.TrimSpace(defaultUnit)
	}
	precision := defaultPrecision
	if cfg.Precision != nil {
		precision = *cfg.Precision
	}
	precision = max(0, precision)

	minValue := defaultMin
	if cfg.Min != nil {
		minValue = *cfg.Min
	}
	maxValue := defaultMax
	if cfg.Max != nil {
		maxValue = *cfg.Max
	}

	return NewCollectItem(name, label, unit, minValue, maxValue, precision)
}

func normalizeAggregateMethod(method string) string {
	switch strings.ToLower(strings.TrimSpace(method)) {
	case "min":
		return "min"
	case "avg", "mean":
		return "avg"
	default:
		return "max"
	}
}
