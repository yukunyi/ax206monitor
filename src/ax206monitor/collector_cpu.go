package main

import (
	"github.com/shirou/gopsutil/v3/cpu"
	"time"
)

type GoNativeCPUCollector struct {
	*BaseCollector
}

func NewGoNativeCPUCollector() *GoNativeCPUCollector {
	return &GoNativeCPUCollector{BaseCollector: NewBaseCollector("go_native.cpu")}
}

func (c *GoNativeCPUCollector) GetAllItems() map[string]*CollectItem {
	if c.getItem("go_native.cpu.usage") == nil {
		c.setItem("go_native.cpu.usage", NewCollectItem("go_native.cpu.usage", "CPU usage", "%", 0, 100, 0))
		c.setItem("go_native.cpu.temp", NewCollectItem("go_native.cpu.temp", "CPU temperature", "°C", 0, 120, 0))
		c.setItem("go_native.cpu.freq", NewCollectItem("go_native.cpu.freq", "CPU frequency", "MHz", 0, 0, 0))
		c.setItem("go_native.cpu.model", NewCollectItem("go_native.cpu.model", "CPU model", "", 0, 0, 0))
		c.setItem("go_native.cpu.cores", NewCollectItem("go_native.cpu.cores", "CPU cores", "", 0, 0, 0))
	}

	initializeCache()
	if cachedCPUInfo != nil {
		if item := c.getItem("go_native.cpu.model"); item != nil {
			item.SetValue(cachedCPUInfo.Model)
			item.SetAvailable(true)
		}
		if item := c.getItem("go_native.cpu.cores"); item != nil {
			item.SetValue(cachedCPUInfo.Cores)
			item.SetAvailable(true)
		}
	} else {
		if item := c.getItem("go_native.cpu.model"); item != nil {
			item.SetAvailable(false)
		}
		if item := c.getItem("go_native.cpu.cores"); item != nil {
			item.SetAvailable(false)
		}
	}
	return c.ItemsSnapshot()
}

func (c *GoNativeCPUCollector) UpdateItems() error {
	if !c.IsEnabled() {
		return nil
	}
	_ = c.GetAllItems()

	cpuPercent, err := cpu.Percent(100*time.Millisecond, false)
	if usage := c.getItem("go_native.cpu.usage"); usage != nil {
		if err == nil && len(cpuPercent) > 0 {
			usage.SetValue(cpuPercent[0])
			usage.SetAvailable(true)
		} else {
			usage.SetAvailable(false)
		}
	}

	if temp := c.getItem("go_native.cpu.temp"); temp != nil {
		value := getRealCPUTemperature()
		if value > 0 {
			temp.SetValue(value)
			temp.SetAvailable(true)
		} else {
			temp.SetAvailable(false)
		}
	}

	if freq := c.getItem("go_native.cpu.freq"); freq != nil {
		current, _ := getRealCPUFrequency()
		if current > 0 {
			freq.SetValue(current)
			freq.SetAvailable(true)
		} else {
			freq.SetAvailable(false)
		}
	}

	return err
}
