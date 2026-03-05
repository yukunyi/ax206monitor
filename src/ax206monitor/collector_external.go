package main

import (
	"ax206monitor/rtsssource"
	"fmt"
	"runtime"
	"strings"
	"time"
)

type CoolerControlCollector struct {
	*BaseCollector
	client  *CoolerControlClient
	sources map[string]string
}

func NewCoolerControlCollector(cfg *MonitorConfig) *CoolerControlCollector {
	if cfg == nil {
		return nil
	}
	if !cfg.IsCollectorEnabled("external.coolercontrol", false) {
		return nil
	}
	url := cfg.GetCoolerControlURL()
	if url == "" {
		return nil
	}
	collector := &CoolerControlCollector{
		BaseCollector: NewBaseCollector("external.coolercontrol"),
		client:        GetCoolerControlClient(url, cfg.GetCoolerControlUsername(), cfg.GetCoolerControlPassword()),
		sources:       make(map[string]string),
	}
	collector.SetEnabled(cfg.IsCollectorEnabled("external.coolercontrol", true))
	return collector
}

func (c *CoolerControlCollector) GetAllItems() map[string]*CollectItem {
	if c.client == nil {
		return c.ItemsSnapshot()
	}
	options, err := c.client.ListMonitorOptions()
	if err != nil {
		return c.ItemsSnapshot()
	}
	for _, option := range options {
		name := strings.TrimSpace(option.Name)
		if name == "" {
			continue
		}
		if c.getItem(name) != nil {
			continue
		}
		unit := strings.TrimSpace(option.Unit)
		precision := 2
		maxValue := 0.0
		switch unit {
		case "°C":
			precision = 1
			maxValue = 120
		case "%":
			precision = 0
			maxValue = 100
		case "RPM", "MHz":
			precision = 0
		case "W":
			precision = 1
		}
		item := NewCollectItem(name, option.Label, unit, 0, maxValue, precision)
		c.setItem(name, item)
		c.sources[name] = name
	}
	return c.ItemsSnapshot()
}

func (c *CoolerControlCollector) UpdateItems() error {
	if !c.IsEnabled() || c.client == nil {
		return nil
	}
	_, err := c.client.FetchSnapshot()
	if err != nil {
		return err
	}
	for key, sourceName := range c.sources {
		item := c.getItem(key)
		if item == nil || !item.IsEnabled() {
			continue
		}
		value, unit, ok, getErr := c.client.GetMonitorValueByNameCached(sourceName)
		if getErr != nil || !ok {
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

type LibreHardwareMonitorCollector struct {
	*BaseCollector
	client  *LibreHardwareMonitorClient
	sources map[string]string
}

func NewLibreHardwareMonitorCollector(cfg *MonitorConfig) *LibreHardwareMonitorCollector {
	if cfg == nil {
		return nil
	}
	if !cfg.IsCollectorEnabled("external.librehardwaremonitor", false) {
		return nil
	}
	url := cfg.GetLibreHardwareMonitorURL()
	if url == "" {
		return nil
	}
	collector := &LibreHardwareMonitorCollector{
		BaseCollector: NewBaseCollector("external.librehardwaremonitor"),
		client:        GetLibreHardwareMonitorClient(url),
		sources:       make(map[string]string),
	}
	collector.SetEnabled(cfg.IsCollectorEnabled("external.librehardwaremonitor", true))
	return collector
}

func (c *LibreHardwareMonitorCollector) GetAllItems() map[string]*CollectItem {
	if c.client == nil {
		return c.ItemsSnapshot()
	}
	options, err := c.client.ListMonitorOptions()
	if err != nil {
		return c.ItemsSnapshot()
	}
	for _, option := range options {
		name := strings.TrimSpace(option.Name)
		if name == "" {
			continue
		}
		if c.getItem(name) != nil {
			continue
		}
		unit := strings.TrimSpace(option.Unit)
		precision := 2
		maxValue := 0.0
		switch strings.ToUpper(unit) {
		case "°C":
			precision = 1
			maxValue = 120
		case "%":
			precision = 0
			maxValue = 100
		case "RPM", "MHZ", "GHZ", "HZ":
			precision = 0
		}
		item := NewCollectItem(name, option.Label, unit, 0, maxValue, precision)
		c.setItem(name, item)
		c.sources[name] = name
	}
	return c.ItemsSnapshot()
}

func (c *LibreHardwareMonitorCollector) UpdateItems() error {
	if !c.IsEnabled() || c.client == nil {
		return nil
	}
	if err := c.client.FetchData(); err != nil {
		return err
	}
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
