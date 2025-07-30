package main

import (
	"fmt"
	"strings"
)

// MonitorFactory provides a centralized way to create monitor items
type MonitorFactory struct{}

// NewMonitorFactory creates a new monitor factory
func NewMonitorFactory() *MonitorFactory {
	return &MonitorFactory{}
}

// CreateUsageMonitor creates a usage monitor with standard 0-100% range
func (f *MonitorFactory) CreateUsageMonitor(name, label string, updateFunc func() (float64, bool)) MonitorItem {
	return &GenericMonitor{
		BaseMonitorItem: NewBaseMonitorItem(name, label, 0, 100, "%", 0),
		updateFunc:      updateFunc,
	}
}

// CreateTemperatureMonitor creates a temperature monitor with 0-100°C range
func (f *MonitorFactory) CreateTemperatureMonitor(name, label string, updateFunc func() (float64, bool)) MonitorItem {
	return &GenericMonitor{
		BaseMonitorItem: NewBaseMonitorItem(name, label, 0, 100, "°C", 0),
		updateFunc:      updateFunc,
	}
}

// CreateFrequencyMonitor creates a frequency monitor with MHz unit
func (f *MonitorFactory) CreateFrequencyMonitor(name, label string, updateFunc func() (float64, bool)) MonitorItem {
	return &GenericMonitor{
		BaseMonitorItem: NewBaseMonitorItem(name, label, 0, 0, "MHz", 0),
		updateFunc:      updateFunc,
	}
}

// CreateMemoryMonitor creates a memory monitor with MB unit
func (f *MonitorFactory) CreateMemoryMonitor(name, label string, updateFunc func() (float64, bool)) MonitorItem {
	return &GenericMonitor{
		BaseMonitorItem: NewBaseMonitorItem(name, label, 0, 0, "MB", 0),
		updateFunc:      updateFunc,
	}
}

// CreateSpeedMonitor creates a speed monitor with MB/s unit
func (f *MonitorFactory) CreateSpeedMonitor(name, label string, updateFunc func() (float64, bool)) MonitorItem {
	return &GenericMonitor{
		BaseMonitorItem: NewBaseMonitorItem(name, label, 0, 0, "MB/s", 1),
		updateFunc:      updateFunc,
	}
}

// CreateStringMonitor creates a string monitor
func (f *MonitorFactory) CreateStringMonitor(name, label string, updateFunc func() (string, bool)) MonitorItem {
	return &GenericStringMonitor{
		BaseMonitorItem: NewBaseMonitorItem(name, label, 0, 0, "", 0),
		updateFunc:      updateFunc,
	}
}

// CreateIntMonitor creates an integer monitor
func (f *MonitorFactory) CreateIntMonitor(name, label, unit string, updateFunc func() (int, bool)) MonitorItem {
	return &GenericIntMonitor{
		BaseMonitorItem: NewBaseMonitorItem(name, label, 0, 0, unit, 0),
		updateFunc:      updateFunc,
	}
}

// GenericMonitor is a generic float64 monitor implementation
type GenericMonitor struct {
	*BaseMonitorItem
	updateFunc func() (float64, bool)
}

func (g *GenericMonitor) Update() error {
	if g.updateFunc != nil {
		if value, available := g.updateFunc(); available {
			g.SetValue(value)
			g.SetAvailable(true)
		} else {
			g.SetAvailable(false)
		}
	} else {
		g.SetAvailable(false)
	}
	return nil
}

// GenericStringMonitor is a generic string monitor implementation
type GenericStringMonitor struct {
	*BaseMonitorItem
	updateFunc func() (string, bool)
}

func (g *GenericStringMonitor) Update() error {
	if g.updateFunc != nil {
		if value, available := g.updateFunc(); available {
			g.SetValue(value)
			g.SetAvailable(true)
		} else {
			g.SetAvailable(false)
		}
	} else {
		g.SetAvailable(false)
	}
	return nil
}

// GenericIntMonitor is a generic integer monitor implementation
type GenericIntMonitor struct {
	*BaseMonitorItem
	updateFunc func() (int, bool)
}

func (g *GenericIntMonitor) Update() error {
	if g.updateFunc != nil {
		if value, available := g.updateFunc(); available {
			g.SetValue(value)
			g.SetAvailable(true)
		} else {
			g.SetAvailable(false)
		}
	} else {
		g.SetAvailable(false)
	}
	return nil
}

// Global factory instance
var globalMonitorFactory = NewMonitorFactory()

// GetMonitorFactory returns the global monitor factory
func GetMonitorFactory() *MonitorFactory {
	return globalMonitorFactory
}

// Helper functions for common monitor creation patterns

// CreateCachedValueMonitor creates a monitor that gets values from cache
func CreateCachedValueMonitor(name, label, unit string, min, max float64, precision int, cacheKey string) MonitorItem {
	return &GenericMonitor{
		BaseMonitorItem: NewBaseMonitorItem(name, label, min, max, unit, precision),
		updateFunc: func() (float64, bool) {
			if cachedValue := GetCachedValue(cacheKey); cachedValue != nil {
				if value, ok := cachedValue.(float64); ok && value >= 0 {
					return value, true
				}
			}
			return 0, false
		},
	}
}

// CreateDiskMonitorByIndex creates a disk monitor for a specific disk index
func CreateDiskMonitorByIndex(diskIndex int, monitorType, unit string, getValue func(*DiskInfo) interface{}) MonitorItem {
	name := fmt.Sprintf("disk%d_%s", diskIndex, monitorType)
	label := fmt.Sprintf("Disk %d %s", diskIndex, strings.Title(monitorType))

	return &GenericMonitor{
		BaseMonitorItem: NewBaseMonitorItem(name, label, 0, 0, unit, 0),
		updateFunc: func() (float64, bool) {
			initializeCache()
			if len(cachedDiskInfo) > diskIndex-1 && diskIndex > 0 {
				disk := cachedDiskInfo[diskIndex-1]
				value := getValue(disk)
				if floatValue, ok := value.(float64); ok {
					return floatValue, true
				}
				if intValue, ok := value.(int64); ok {
					return float64(intValue), true
				}
			}
			return 0, false
		},
	}
}

// CreateDiskStringMonitorByIndex creates a string disk monitor for a specific disk index
func CreateDiskStringMonitorByIndex(diskIndex int, monitorType string, getValue func(*DiskInfo) string) MonitorItem {
	name := fmt.Sprintf("disk%d_%s", diskIndex, monitorType)
	label := fmt.Sprintf("Disk %d %s", diskIndex, monitorType)

	return &GenericStringMonitor{
		BaseMonitorItem: NewBaseMonitorItem(name, label, 0, 0, "", 0),
		updateFunc: func() (string, bool) {
			initializeCache()
			if len(cachedDiskInfo) > diskIndex-1 && diskIndex > 0 {
				disk := cachedDiskInfo[diskIndex-1]
				value := getValue(disk)
				return value, value != ""
			}
			return "", false
		},
	}
}
