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
		initializeMonitorItems(nil, "")
		performInitialUpdate()
	}
	return globalMonitorRegistry
}

func GetMonitorRegistryWithConfig(requiredMonitors []string, networkInterface string) *MonitorRegistry {
	if globalMonitorRegistry == nil {
		globalMonitorRegistry = NewMonitorRegistry()
		initializeMonitorItems(requiredMonitors, networkInterface)
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

func initializeMonitorItems(requiredMonitors []string, networkInterface string) {
	registry := globalMonitorRegistry

	// Create a map for quick lookup of required monitors
	required := make(map[string]bool)
	if requiredMonitors != nil {
		for _, monitor := range requiredMonitors {
			required[monitor] = true
		}
	}

	// Helper function to check if a monitor is required
	isRequired := func(name string) bool {
		if requiredMonitors == nil {
			return true // Initialize all if no filter provided
		}
		return required[name]
	}

	// Initialize basic monitors only if required
	if isRequired("cpu_usage") {
		registry.Register(NewCPUUsageMonitor())
	}
	if isRequired("cpu_temp") {
		registry.Register(NewCPUTempMonitor())
	}
	if isRequired("cpu_freq") {
		registry.Register(NewCPUFreqMonitor())
	}
	if isRequired("memory_usage") {
		registry.Register(NewMemoryUsageMonitor())
	}
	if isRequired("memory_used") {
		registry.Register(NewMemoryUsedMonitor())
	}
	if isRequired("memory_total") {
		registry.Register(NewMemoryTotalMonitor())
	}
	if isRequired("gpu_usage") {
		registry.Register(NewGPUUsageMonitor())
	}
	if isRequired("gpu_temp") {
		registry.Register(NewGPUTempMonitor())
	}
	if isRequired("gpu_freq") {
		registry.Register(NewGPUFreqMonitor())
	}
	if isRequired("gpu_fps") {
		registry.Register(NewGPUFPSMonitor())
	}
	if isRequired("disk_temp") {
		registry.Register(NewDiskTempMonitor())
	}
	if isRequired("disk_usage") {
		registry.Register(NewDiskUsageMonitor())
	}
	if isRequired("load_avg") {
		registry.Register(NewLoadAvgMonitor())
	}
	if isRequired("current_time") {
		registry.Register(NewCurrentTimeMonitor())
	}

	initializeNetworkMonitors(registry, requiredMonitors, networkInterface)
	initializeFanMonitors(registry, requiredMonitors)
}

func initializeNetworkMonitors(registry *MonitorRegistry, requiredMonitors []string, networkInterface string) {
	// Helper function to check if a monitor is required
	isRequired := func(name string) bool {
		if requiredMonitors == nil {
			return true
		}
		for _, monitor := range requiredMonitors {
			if monitor == name {
				return true
			}
		}
		return false
	}

	// Get the configured network interface
	configuredInterface := GetConfiguredNetworkInterface(networkInterface)
	if configuredInterface == "" {
		return // No valid interface found
	}

	// Initialize network monitors only if required
	if isRequired("net_upload") {
		registry.Register(NewNetworkInterfaceMonitor(configuredInterface, "upload", ""))
	}
	if isRequired("net_download") {
		registry.Register(NewNetworkInterfaceMonitor(configuredInterface, "download", ""))
	}
	if isRequired("net_ip") {
		registry.Register(NewNetworkInterfaceMonitor(configuredInterface, "ip", ""))
	}
	if isRequired("net_interface") {
		registry.Register(NewNetworkInterfaceMonitor(configuredInterface, "name", ""))
	}
}

func initializeFanMonitors(registry *MonitorRegistry, requiredMonitors []string) {
	fans := GetAvailableFans()
	for i, fan := range fans {
		fanMonitorName := fmt.Sprintf("fan_%d", i)
		if requiredMonitors != nil {
			required := false
			for _, monitor := range requiredMonitors {
				if monitor == fanMonitorName {
					required = true
					break
				}
			}
			if !required {
				continue
			}
		}
		registry.Register(NewFanMonitor(i, fan.Name))
	}
}
