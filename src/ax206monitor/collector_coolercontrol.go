package main

import "strings"

type CoolerControlCollector struct {
	*BaseCollector
	client  *CoolerControlClient
	sources map[string]string
}

func isCoolerControlEnabled(cfg *MonitorConfig) bool {
	return cfg != nil && cfg.IsCollectorEnabled("external.coolercontrol", false)
}

func getConfiguredCoolerControlClient(cfg *MonitorConfig) *CoolerControlClient {
	if !isCoolerControlEnabled(cfg) {
		return nil
	}
	url := cfg.GetCoolerControlURL()
	if url == "" {
		return nil
	}
	return GetCoolerControlClient(url, cfg.GetCoolerControlUsername(), cfg.GetCoolerControlPassword())
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
	client := getConfiguredCoolerControlClient(cfg)
	if client == nil {
		return nil
	}
	collector := &CoolerControlCollector{
		BaseCollector: NewBaseCollector("external.coolercontrol"),
		client:        client,
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
