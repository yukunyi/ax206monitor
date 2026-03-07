package main

import (
	"ax206monitor/rtsssource"
	"fmt"
	"runtime"
	"strings"
	"time"
)

type RTSSCollector struct {
	*BaseCollector
	client  *rtsssource.RTSSClient
	sources map[string]string
}

func NewRTSSCollector(cfg *MonitorConfig) *RTSSCollector {
	if runtime.GOOS != "windows" || cfg == nil {
		return nil
	}
	if !cfg.IsCollectorEnabled("external.rtss", false) {
		return nil
	}
	collector := &RTSSCollector{
		BaseCollector: NewBaseCollector("external.rtss"),
		client:        rtsssource.GetRTSSClient(),
		sources:       make(map[string]string),
	}
	collector.SetEnabled(cfg.IsCollectorEnabled("external.rtss", true))
	return collector
}

func (c *RTSSCollector) GetAllItems() map[string]*CollectItem {
	if c.client == nil {
		return c.ItemsSnapshot()
	}
	options := c.client.ListMonitorOptions()
	for _, option := range options {
		name := strings.TrimSpace(option.Name)
		if name == "" {
			continue
		}
		if c.getItem(name) != nil {
			continue
		}
		item := NewCollectItem(name, option.Label, option.Unit, 0, 0, 1)
		c.setItem(name, item)
		c.sources[name] = name
	}
	return c.ItemsSnapshot()
}

func (c *RTSSCollector) UpdateItems() error {
	if !c.IsEnabled() || c.client == nil {
		return nil
	}
	c.client.RefreshMetrics(250 * time.Millisecond)
	for key, sourceName := range c.sources {
		item := c.getItem(key)
		if item == nil || !item.IsEnabled() {
			continue
		}
		value, unit, ok, err := c.client.GetMonitorValueByNameCached(sourceName)
		if err != nil || !ok {
			item.SetAvailable(false)
			continue
		}
		if strings.TrimSpace(unit) != "" {
			item.SetUnit(unit)
		}
		item.SetValue(value)
		item.SetAvailable(true)
	}
	return nil
}

func collectorLabelFallback(name string) string {
	if strings.TrimSpace(name) == "" {
		return "collector item"
	}
	return fmt.Sprintf("%s item", name)
}
