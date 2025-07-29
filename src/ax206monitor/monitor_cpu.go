package main

import (
	"time"

	"github.com/shirou/gopsutil/v3/cpu"
)

type CPUUsageMonitor struct {
	*BaseMonitorItem
}

func NewCPUUsageMonitor() *CPUUsageMonitor {
	return &CPUUsageMonitor{
		BaseMonitorItem: NewBaseMonitorItem(
			"cpu_usage",
			"CPU",
			0, 100,
			"%",
			0,
		),
	}
}

func (c *CPUUsageMonitor) Update() error {
	cpuPercent, err := cpu.Percent(100*time.Millisecond, false)
	if err != nil {
		c.SetAvailable(false)
		return err
	}

	if len(cpuPercent) > 0 {
		c.SetValue(cpuPercent[0])
		c.SetAvailable(true)
	} else {
		c.SetAvailable(false)
	}

	return nil
}

type CPUTempMonitor struct {
	*BaseMonitorItem
}

func NewCPUTempMonitor() *CPUTempMonitor {
	return &CPUTempMonitor{
		BaseMonitorItem: NewBaseMonitorItem(
			"cpu_temp",
			"CPU Temp",
			0, 100,
			"°C",
			0,
		),
	}
}

func (c *CPUTempMonitor) Update() error {
	if cachedValue := GetCachedValue("cpu_temp"); cachedValue != nil {
		if temp, ok := cachedValue.(float64); ok && temp > 0 {
			c.SetValue(temp)
			c.SetAvailable(true)
		} else {
			c.SetAvailable(false)
		}
	} else {
		c.SetAvailable(false)
	}

	return nil
}

type CPUFreqMonitor struct {
	*BaseMonitorItem
}

func NewCPUFreqMonitor() *CPUFreqMonitor {
	return &CPUFreqMonitor{
		BaseMonitorItem: NewBaseMonitorItem(
			"cpu_freq",
			"CPU Freq",
			0, 0,
			"MHz",
			0,
		),
	}
}

func (c *CPUFreqMonitor) Update() error {
	if cachedValue := GetCachedValue("cpu_freq"); cachedValue != nil {
		if freq, ok := cachedValue.(float64); ok && freq > 0 {
			c.SetValue(freq)
			c.SetAvailable(true)
		} else {
			c.SetAvailable(false)
		}
	} else {
		c.SetAvailable(false)
	}

	return nil
}
