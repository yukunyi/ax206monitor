package main

import (
	"runtime"
	"strings"
	"sync"
)

type CoolerControlCollector struct {
	*BaseCollector
	mu      sync.RWMutex
	client  *CoolerControlClient
	sources map[string]string
	url     string
}

func isCoolerControlEnabled(cfg *MonitorConfig) bool {
	if runtime.GOOS != "linux" {
		return false
	}
	return cfg != nil && cfg.IsCollectorEnabled(collectorCoolerControl, false)
}

func getConfiguredCoolerControlClient(cfg *MonitorConfig) *CoolerControlClient {
	if !isCoolerControlEnabled(cfg) {
		return nil
	}
	url := cfg.GetCoolerControlURL()
	if url == "" {
		return nil
	}
	return GetCoolerControlClient(url, cfg.GetCoolerControlPassword())
}

func listConfiguredCoolerControlOptions(cfg *MonitorConfig) ([]CoolerControlMonitorOption, error) {
	client := getConfiguredCoolerControlClient(cfg)
	if client == nil {
		return []CoolerControlMonitorOption{}, nil
	}
	items, err := client.ListMonitorOptions()
	if err != nil {
		return nil, err
	}
	if items == nil {
		return []CoolerControlMonitorOption{}, nil
	}
	return items, nil
}

func NewCoolerControlCollector(cfg *MonitorConfig) *CoolerControlCollector {
	if runtime.GOOS != "linux" {
		return nil
	}
	collector := &CoolerControlCollector{
		BaseCollector: NewBaseCollector(collectorCoolerControl),
		sources:       make(map[string]string),
	}
	collector.ApplyConfig(cfg)
	return collector
}

func (c *CoolerControlCollector) ApplyConfig(cfg *MonitorConfig) {
	if runtime.GOOS != "linux" {
		return
	}
	enabled := cfg != nil && cfg.IsCollectorEnabled(collectorCoolerControl, false)
	client := getConfiguredCoolerControlClient(cfg)
	nextURL := ""
	if cfg != nil {
		nextURL = strings.TrimSpace(cfg.GetCoolerControlURL())
	}

	c.mu.Lock()
	defer c.mu.Unlock()
	c.SetEnabled(enabled)
	if !enabled {
		c.client = nil
		c.sources = map[string]string{}
		c.clearItems()
		c.url = ""
		return
	}
	if c.url != nextURL {
		c.sources = map[string]string{}
		c.clearItems()
	}
	c.client = client
	c.url = nextURL
}

func (c *CoolerControlCollector) GetAllItems() map[string]*CollectItem {
	c.mu.RLock()
	client := c.client
	c.mu.RUnlock()
	if client == nil {
		return c.ItemsSnapshot()
	}
	options, err := client.ListMonitorOptions()
	if err != nil {
		return c.ItemsSnapshot()
	}
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
	_, err := client.FetchSnapshot()
	if err != nil {
		return err
	}
	for key, sourceName := range sources {
		item := c.getItem(key)
		if item == nil || !item.IsEnabled() {
			continue
		}
		value, unit, ok, getErr := client.GetMonitorValueByNameCached(sourceName)
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
