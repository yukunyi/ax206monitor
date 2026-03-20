package main

import (
	"runtime"
	"strings"
	"sync"
)

type LibreHardwareMonitorCollector struct {
	*BaseCollector
	mu          sync.RWMutex
	client      *LibreHardwareMonitorClient
	sources     map[string]string
	endpointKey string
}

func isLibreHardwareMonitorEnabled(cfg *MonitorConfig) bool {
	if runtime.GOOS != "windows" {
		return false
	}
	return cfg != nil && cfg.IsCollectorEnabled(collectorLibreHardwareMonitor, false)
}

func getConfiguredLibreHardwareMonitorClient(cfg *MonitorConfig) *LibreHardwareMonitorClient {
	if !isLibreHardwareMonitorEnabled(cfg) {
		return nil
	}
	url := cfg.GetLibreHardwareMonitorURL()
	if url == "" {
		return nil
	}
	return GetLibreHardwareMonitorClient(
		url,
		cfg.GetLibreHardwareMonitorUsername(),
		cfg.GetLibreHardwareMonitorPassword(),
	)
}

func listConfiguredLibreHardwareMonitorOptions(cfg *MonitorConfig) ([]LibreHardwareMonitorMonitorOption, error) {
	client := getConfiguredLibreHardwareMonitorClient(cfg)
	if client == nil {
		return []LibreHardwareMonitorMonitorOption{}, nil
	}
	items, err := client.ListMonitorOptions()
	if err != nil {
		return nil, err
	}
	if items == nil {
		return []LibreHardwareMonitorMonitorOption{}, nil
	}
	return items, nil
}

func NewLibreHardwareMonitorCollector(cfg *MonitorConfig) *LibreHardwareMonitorCollector {
	if runtime.GOOS != "windows" {
		return nil
	}
	collector := &LibreHardwareMonitorCollector{
		BaseCollector: NewBaseCollector(collectorLibreHardwareMonitor),
		sources:       make(map[string]string),
	}
	return collector
}

func (c *LibreHardwareMonitorCollector) ApplyConfig(cfg *MonitorConfig) {
	if runtime.GOOS != "windows" {
		return
	}
	enabled := cfg != nil && cfg.IsCollectorEnabled(collectorLibreHardwareMonitor, false)
	client := getConfiguredLibreHardwareMonitorClient(cfg)
	nextEndpointKey := ""
	if cfg != nil {
		nextEndpointKey = strings.TrimSpace(cfg.GetLibreHardwareMonitorURL())
		nextEndpointKey += "|" + cfg.GetLibreHardwareMonitorUsername()
		nextEndpointKey += "|" + cfg.GetLibreHardwareMonitorPassword()
	}

	c.mu.Lock()
	defer c.mu.Unlock()
	c.SetEnabled(enabled)
	if !enabled {
		c.client = nil
		c.sources = map[string]string{}
		c.clearItems()
		c.endpointKey = ""
		return
	}
	if c.endpointKey != nextEndpointKey {
		c.sources = map[string]string{}
		c.clearItems()
	}
	c.client = client
	c.endpointKey = nextEndpointKey
}

func (c *LibreHardwareMonitorCollector) GetAllItems() map[string]*CollectItem {
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
		unit := strings.TrimSpace(option.Unit)
		if item := c.getItem(name); item != nil {
			if unit != "" {
				item.SetUnit(unit)
			}
			continue
		}
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
	if err := client.FetchData(); err != nil {
		return err
	}
	for key, sourceName := range sources {
		item := c.getItem(key)
		if item == nil || !item.IsEnabled() {
			continue
		}
		value, _, ok, err := client.GetMonitorValueByNameCached(sourceName)
		if err != nil || !ok {
			item.SetAvailable(false)
			continue
		}
		item.SetValue(value)
		item.SetAvailable(true)
	}
	return nil
}
