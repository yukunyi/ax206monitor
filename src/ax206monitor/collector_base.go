package main

import "sync"

type CollectItem struct {
	*BaseCollectItem
}

func NewCollectItem(name, label, unit string, min, max float64, precision int) *CollectItem {
	return &CollectItem{
		BaseCollectItem: NewBaseCollectItem(name, label, min, max, unit, precision),
	}
}

func (c *CollectItem) Update() error {
	return nil
}

type Collector interface {
	Name() string
	GetAllItems() map[string]*CollectItem
	UpdateItems() error
}

type CollectorConfigApplier interface {
	ApplyConfig(cfg *MonitorConfig)
}

type BaseCollector struct {
	name    string
	enabled bool
	mutex   sync.RWMutex
	items   map[string]*CollectItem
}

func NewBaseCollector(name string) *BaseCollector {
	return &BaseCollector{
		name:    name,
		enabled: true,
		items:   make(map[string]*CollectItem),
	}
}

func (c *BaseCollector) Name() string {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	return c.name
}

func (c *BaseCollector) SetEnabled(enabled bool) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.enabled = enabled
}

func (c *BaseCollector) IsEnabled() bool {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	return c.enabled
}

func (c *BaseCollector) ItemsSnapshot() map[string]*CollectItem {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	result := make(map[string]*CollectItem, len(c.items))
	for key, item := range c.items {
		result[key] = item
	}
	return result
}

func (c *BaseCollector) setItem(key string, item *CollectItem) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.items[key] = item
}

func (c *BaseCollector) getItem(key string) *CollectItem {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	return c.items[key]
}

func (c *BaseCollector) clearItems() {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.items = make(map[string]*CollectItem)
}
