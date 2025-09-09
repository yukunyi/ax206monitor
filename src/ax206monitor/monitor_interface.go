package main

import (
	"fmt"
	"sync"
	"sync/atomic"
	"time"
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

func (b *BaseMonitorItem) GetName() string  { return b.name }
func (b *BaseMonitorItem) GetLabel() string { return b.label }
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

type monitorRunState struct {
	running   int32
	lastStart int64 // unix nano
}

type MonitorRegistry struct {
	items   map[string]MonitorItem
	mutex   sync.RWMutex
	states  map[string]*monitorRunState
	stateMu sync.RWMutex
}

func NewMonitorRegistry() *MonitorRegistry {
	return &MonitorRegistry{items: make(map[string]MonitorItem), states: make(map[string]*monitorRunState)}
}

func (r *MonitorRegistry) Register(item MonitorItem) {
	r.mutex.Lock()
	r.items[item.GetName()] = item
	r.mutex.Unlock()
	r.stateMu.Lock()
	if _, ok := r.states[item.GetName()]; !ok {
		r.states[item.GetName()] = &monitorRunState{}
	}
	r.stateMu.Unlock()
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

func (r *MonitorRegistry) scheduleUpdate(item MonitorItem) {
	name := item.GetName()
	r.stateMu.RLock()
	st := r.states[name]
	r.stateMu.RUnlock()
	if st == nil {
		r.stateMu.Lock()
		if r.states[name] == nil {
			r.states[name] = &monitorRunState{}
		}
		st = r.states[name]
		r.stateMu.Unlock()
	}
	// 看门狗：如果运行超过10秒，强制清理
	if atomic.LoadInt32(&st.running) == 1 {
		last := atomic.LoadInt64(&st.lastStart)
		if last > 0 && time.Since(time.Unix(0, last)) > 10*time.Second {
			if atomic.CompareAndSwapInt32(&st.running, 1, 0) {
				logWarn("Monitor '%s' previous update stuck >10s, clearing state", name)
			}
		}
	}
	if !atomic.CompareAndSwapInt32(&st.running, 0, 1) {
		return
	}
	atomic.StoreInt64(&st.lastStart, time.Now().UnixNano())
	go func(m MonitorItem, state *monitorRunState) {
		defer func() {
			if rec := recover(); rec != nil {
				logWarn("Monitor '%s' update panic: %v", m.GetName(), rec)
			}
			atomic.StoreInt32(&state.running, 0)
		}()
		start := time.Now()
		_ = m.Update()
		elapsed := time.Since(start)
		if elapsed > 500*time.Millisecond {
			logWarn("Monitor '%s' slow update: %v", m.GetName(), elapsed)
		}
	}(item, st)
}

func (r *MonitorRegistry) Update(names []string) error {
	r.mutex.RLock()
	for _, name := range names {
		if item, ok := r.items[name]; ok {
			r.scheduleUpdate(item)
		}
	}
	r.mutex.RUnlock()
	return nil
}

func (r *MonitorRegistry) UpdateAll() error {
	r.mutex.RLock()
	for _, item := range r.items {
		r.scheduleUpdate(item)
	}
	r.mutex.RUnlock()
	return nil
}

var (
	globalMonitorRegistry *MonitorRegistry
	globalMonitorConfig   *MonitorConfig
)

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

// Set/Get global config
func SetGlobalMonitorConfig(config *MonitorConfig) { globalMonitorConfig = config }
func GetGlobalMonitorConfig() *MonitorConfig       { return globalMonitorConfig }

func performInitialUpdate() {
	registry := globalMonitorRegistry
	// 改为异步调度，避免初始化阶段阻塞
	_ = registry.UpdateAll()
}

type MonitorItemConfig struct {
	Name     string
	Creator  func() MonitorItem
	Required bool
}

type MonitorRegistryConfig struct{ Monitors []MonitorItemConfig }

func getMonitorRegistryConfig() *MonitorRegistryConfig {
	return &MonitorRegistryConfig{Monitors: []MonitorItemConfig{
		{"cpu_usage", func() MonitorItem { return NewCPUUsageMonitor() }, true},
		{"cpu_temp", func() MonitorItem { return NewCPUTempMonitor() }, true},
		{"cpu_freq", func() MonitorItem { return NewCPUFreqMonitor() }, true},
		{"cpu_model", func() MonitorItem { return NewCPUModelMonitor() }, true},
		{"cpu_cores", func() MonitorItem { return NewCPUCoresMonitor() }, true},
		{"memory_usage", func() MonitorItem { return NewMemoryUsageMonitor() }, true},
		{"memory_used", func() MonitorItem { return NewMemoryUsedMonitor() }, true},
		{"memory_total", func() MonitorItem { return NewMemoryTotalMonitor() }, true},
		{"memory_usage_text", func() MonitorItem { return NewMemoryUsageTextMonitor() }, true},
		{"memory_usage_progress", func() MonitorItem { return NewMemoryUsageProgressMonitor() }, true},
		{"swap_usage", func() MonitorItem { return NewSwapUsageMonitor() }, true},
		{"gpu_usage", NewGPUUsageMonitor, true},
		{"gpu_temp", NewGPUTempMonitor, true},
		{"gpu_freq", NewGPUFreqMonitor, true},
		{"gpu_fps", NewGPUFPSMonitor, true},
		{"gpu_model", NewGPUModelMonitor, true},
		{"gpu_memory_total", NewGPUMemoryTotalMonitor, true},
		{"gpu_memory_used", NewGPUMemoryUsedMonitor, true},
		{"gpu_memory_usage", NewGPUMemoryUsageMonitor, true},
		{"disk_default_temp", NewDiskDefaultTempMonitor, true},
		{"disk_default_read_speed", NewDiskDefaultReadSpeedMonitor, true},
		{"disk_default_write_speed", NewDiskDefaultWriteSpeedMonitor, true},
		{"disk_default_usage", NewDiskDefaultUsageMonitor, true},
		{"disk_default_model", NewDiskDefaultModelMonitor, true},
		{"disk_default_name", NewDiskDefaultNameMonitor, true},
		{"net_default_upload", func() MonitorItem {
			var ni string
			if cfg := GetGlobalMonitorConfig(); cfg != nil {
				ni = cfg.GetNetworkInterface()
			}
			return NewNetworkInterfaceMonitor(GetConfiguredNetworkInterface(ni), "upload", "net_default")
		}, true},
		{"net_default_download", func() MonitorItem {
			var ni string
			if cfg := GetGlobalMonitorConfig(); cfg != nil {
				ni = cfg.GetNetworkInterface()
			}
			return NewNetworkInterfaceMonitor(GetConfiguredNetworkInterface(ni), "download", "net_default")
		}, true},
		{"net_default_ip", func() MonitorItem {
			var ni string
			if cfg := GetGlobalMonitorConfig(); cfg != nil {
				ni = cfg.GetNetworkInterface()
			}
			return NewNetworkInterfaceMonitor(GetConfiguredNetworkInterface(ni), "ip", "net_default")
		}, true},
		{"net_default_interface", func() MonitorItem {
			var ni string
			if cfg := GetGlobalMonitorConfig(); cfg != nil {
				ni = cfg.GetNetworkInterface()
			}
			return NewNetworkInterfaceMonitor(GetConfiguredNetworkInterface(ni), "name", "net_default")
		}, true},
		{"current_time", func() MonitorItem { return NewCurrentTimeMonitor() }, true},
	}}
}

func initializeMonitorItems(requiredMonitors []string, networkInterface string) {
	registry := globalMonitorRegistry
	config := getMonitorRegistryConfig()
	for _, monitorConfig := range config.Monitors {
		registry.Register(monitorConfig.Creator())
	}
	for fanIndex := 1; fanIndex <= 10; fanIndex++ {
		registry.Register(NewSystemFanMonitor(fanIndex))
	}
	for diskIndex := 1; diskIndex <= 5; diskIndex++ {
		registry.Register(NewDiskNameMonitor(diskIndex))
		registry.Register(NewDiskSizeMonitor(diskIndex))
		registry.Register(NewDiskTempMonitorByIndex(diskIndex))
	}
	initializeFanMonitors(registry, requiredMonitors)
}

func initializeFanMonitors(registry *MonitorRegistry, requiredMonitors []string) {
	for fanIndex := 1; fanIndex <= 10; fanIndex++ {
		fanMonitorName := fmt.Sprintf("fan%d", fanIndex)
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
		registry.Register(NewFanMonitor(fanIndex, ""))
	}
}
