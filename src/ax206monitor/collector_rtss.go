package main

import (
	"ax206monitor/rtsssource"
	"fmt"
	"runtime"
	"strings"
	"sync"
	"time"
)

type RTSSCollector struct {
	*BaseCollector
	mu      sync.RWMutex
	client  *rtsssource.RTSSClient
	sources map[string]string
}

func NewRTSSCollector(cfg *MonitorConfig) *RTSSCollector {
	if runtime.GOOS != "windows" {
		return nil
	}
	collector := &RTSSCollector{
		BaseCollector: NewBaseCollector(collectorRTSS),
		client:        rtsssource.GetRTSSClient(),
		sources:       make(map[string]string),
	}
	collector.ApplyConfig(cfg)
	return collector
}

func (c *RTSSCollector) ApplyConfig(cfg *MonitorConfig) {
	enabled := cfg != nil && cfg.IsCollectorEnabled(collectorRTSS, false)
	c.SetEnabled(enabled)
	if !enabled {
		c.mu.Lock()
		c.sources = map[string]string{}
		c.clearItems()
		c.mu.Unlock()
	}
}

func (c *RTSSCollector) GetAllItems() map[string]*CollectItem {
	c.mu.RLock()
	client := c.client
	c.mu.RUnlock()
	if client == nil {
		return c.ItemsSnapshot()
	}
	options := client.ListMonitorOptions()
	c.mu.Lock()
	defer c.mu.Unlock()
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
	c.mu.RLock()
	client := c.client
	sources := make(map[string]string, len(c.sources))
	for key, value := range c.sources {
		sources[key] = value
	}
	c.mu.RUnlock()
	if !c.IsEnabled() || client == nil {
		return nil
	}
	client.RefreshMetrics(250 * time.Millisecond)
	for key, sourceName := range sources {
		item := c.getItem(key)
		if item == nil || !item.IsEnabled() {
			continue
		}
		value, unit, ok, err := client.GetMonitorValueByNameCached(sourceName)
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
