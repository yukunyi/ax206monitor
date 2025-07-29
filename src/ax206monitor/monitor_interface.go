package main

import (
	"fmt"
	"sync"
)

type MonitorItemType int

const (
	TypeUsage MonitorItemType = iota
	TypeTemperature
	TypeFrequency
	TypeMemory
	TypeNetwork
	TypeString
	TypeInt
)

type MonitorValue struct {
	Value     interface{}
	Unit      string
	Min       float64
	Max       float64
	Precision int
}

type MonitorItem interface {
	GetName() string
	GetLabel() string
	Update() error
	GetValue() *MonitorValue
	IsAvailable() bool
}

type BaseMonitorItem struct {
	name      string
	label     string
	value     *MonitorValue
	available bool
	mutex     sync.RWMutex
}

func NewBaseMonitorItem(name, label string, min, max float64, unit string, precision int) *BaseMonitorItem {
	return &BaseMonitorItem{
		name:      name,
		label:     label,
		available: true,
		value: &MonitorValue{
			Value:     0.0,
			Unit:      unit,
			Min:       min,
			Max:       max,
			Precision: precision,
		},
	}
}

func (b *BaseMonitorItem) GetName() string {
	return b.name
}

func (b *BaseMonitorItem) GetLabel() string {
	return b.label
}

func (b *BaseMonitorItem) GetValue() *MonitorValue {
	b.mutex.RLock()
	defer b.mutex.RUnlock()
	return b.value
}

func (b *BaseMonitorItem) IsAvailable() bool {
	b.mutex.RLock()
	defer b.mutex.RUnlock()
	return b.available
}

func (b *BaseMonitorItem) SetValue(value interface{}) {
	b.mutex.Lock()
	defer b.mutex.Unlock()
	b.value.Value = value
}

func (b *BaseMonitorItem) SetAvailable(available bool) {
	b.mutex.Lock()
	defer b.mutex.Unlock()
	b.available = available
}

func FormatMonitorValue(value *MonitorValue, showUnit bool, unitOverride string) string {
	if value == nil {
		return "N/A"
	}

	unit := value.Unit
	if unitOverride != "" {
		unit = unitOverride
	}

	switch v := value.Value.(type) {
	case string:
		return v
	case float64, float32, int, int64, uint64:
		val := getFloat64Value(v)
		format := fmt.Sprintf("%%.%df", value.Precision)
		text := fmt.Sprintf(format, val)
		if showUnit && unit != "" {
			text += unit
		}
		return text
	default:
		return fmt.Sprintf("%v", value.Value)
	}
}

func getFloat64Value(value interface{}) float64 {
	switch v := value.(type) {
	case float64:
		return v
	case float32:
		return float64(v)
	case int:
		return float64(v)
	case int64:
		return float64(v)
	case uint64:
		return float64(v)
	default:
		return 0.0
	}
}

type MonitorRegistry struct {
	items map[string]MonitorItem
	mutex sync.RWMutex
}

func NewMonitorRegistry() *MonitorRegistry {
	return &MonitorRegistry{
		items: make(map[string]MonitorItem),
	}
}

func (r *MonitorRegistry) Register(item MonitorItem) {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	r.items[item.GetName()] = item
}

func (r *MonitorRegistry) Get(name string) MonitorItem {
	r.mutex.RLock()
	defer r.mutex.RUnlock()
	return r.items[name]
}

func (r *MonitorRegistry) GetAll() map[string]MonitorItem {
	r.mutex.RLock()
	defer r.mutex.RUnlock()
	result := make(map[string]MonitorItem)
	for k, v := range r.items {
		result[k] = v
	}
	return result
}

func (r *MonitorRegistry) Update(names []string) error {
	r.mutex.RLock()
	items := make([]MonitorItem, 0, len(names))
	for _, name := range names {
		if item, exists := r.items[name]; exists {
			items = append(items, item)
		}
	}
	r.mutex.RUnlock()

	for _, item := range items {
		if err := item.Update(); err != nil {
			continue
		}
	}
	return nil
}

var globalMonitorRegistry *MonitorRegistry

func GetMonitorRegistry() *MonitorRegistry {
	if globalMonitorRegistry == nil {
		globalMonitorRegistry = NewMonitorRegistry()
		initializeMonitorItems()
		performInitialUpdate()
	}
	return globalMonitorRegistry
}

func performInitialUpdate() {
	registry := globalMonitorRegistry
	registry.mutex.RLock()
	items := make([]MonitorItem, 0, len(registry.items))
	for _, item := range registry.items {
		items = append(items, item)
	}
	registry.mutex.RUnlock()

	for _, item := range items {
		go func(item MonitorItem) {
			item.Update()
		}(item)
	}
}

func initializeMonitorItems() {
	registry := globalMonitorRegistry

	registry.Register(NewCPUUsageMonitor())
	registry.Register(NewCPUTempMonitor())
	registry.Register(NewCPUFreqMonitor())
	registry.Register(NewMemoryUsageMonitor())
	registry.Register(NewMemoryUsedMonitor())
	registry.Register(NewMemoryTotalMonitor())
	registry.Register(NewGPUUsageMonitor())
	registry.Register(NewGPUTempMonitor())
	registry.Register(NewGPUFreqMonitor())
	registry.Register(NewGPUFPSMonitor())
	registry.Register(NewDiskTempMonitor())
	registry.Register(NewLoadAvgMonitor())
	registry.Register(NewCurrentTimeMonitor())

	initializeNetworkMonitors(registry)
	initializeFanMonitors(registry)
}

func initializeNetworkMonitors(registry *MonitorRegistry) {
	interfaces := getActiveNetworkInterfaces()
	for i, iface := range interfaces {
		if i == 0 {
			registry.Register(NewNetworkInterfaceMonitor(iface, "upload", "default"))
			registry.Register(NewNetworkInterfaceMonitor(iface, "download", "default"))
			registry.Register(NewNetworkInterfaceMonitor(iface, "ip", "default"))
			registry.Register(NewNetworkInterfaceMonitor(iface, "name", "default"))
		}
		registry.Register(NewNetworkInterfaceMonitor(iface, "upload", ""))
		registry.Register(NewNetworkInterfaceMonitor(iface, "download", ""))
		registry.Register(NewNetworkInterfaceMonitor(iface, "ip", ""))
		registry.Register(NewNetworkInterfaceMonitor(iface, "name", ""))
	}
}

func initializeFanMonitors(registry *MonitorRegistry) {
	fans := GetAvailableFans()
	for i, fan := range fans {
		registry.Register(NewFanMonitor(i, fan.Name))
	}
}
