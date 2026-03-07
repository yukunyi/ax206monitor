package main

import (
	"github.com/shirou/gopsutil/v3/cpu"
	"sync"
	"sync/atomic"
	"time"
)

type GoNativeCPUCollector struct {
	*BaseCollector

	usageLastTotal float64
	usageLastIdle  float64
	usageLastValue float64
	usageReady     bool

	tempMu       sync.RWMutex
	tempValue    float64
	tempOK       bool
	tempAt       time.Time
	tempUpdating int32

	freqMu       sync.RWMutex
	freqValue    float64
	freqOK       bool
	freqAt       time.Time
	freqUpdating int32
}

func NewGoNativeCPUCollector() *GoNativeCPUCollector {
	collector := &GoNativeCPUCollector{BaseCollector: NewBaseCollector("go_native.cpu")}
	collector.triggerTempRefresh()
	collector.triggerFreqRefresh()
	return collector
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

	now := time.Now()
	c.maybeRefreshTemp(now)
	c.maybeRefreshFreq(now)

	usageValue, usageOK, usageErr := c.sampleCPUUsage()
	if usage := c.getItem("go_native.cpu.usage"); usage != nil {
		if usageOK {
			usage.SetValue(usageValue)
			usage.SetAvailable(true)
		} else {
			usage.SetAvailable(false)
		}
	}

	if temp := c.getItem("go_native.cpu.temp"); temp != nil {
		if value, ok := c.getCachedTemp(); ok {
			temp.SetValue(value)
			temp.SetAvailable(true)
		} else {
			temp.SetAvailable(false)
		}
	}

	if freq := c.getItem("go_native.cpu.freq"); freq != nil {
		if value, ok := c.getCachedFreq(); ok {
			freq.SetValue(value)
			freq.SetAvailable(true)
		} else {
			freq.SetAvailable(false)
		}
	}

	return usageErr
}

func (c *GoNativeCPUCollector) sampleCPUUsage() (float64, bool, error) {
	stats, err := cpu.Times(false)
	if err != nil || len(stats) == 0 {
		return 0, false, err
	}
	sample := stats[0]
	total := sample.User + sample.System + sample.Idle + sample.Nice + sample.Iowait + sample.Irq + sample.Softirq + sample.Steal + sample.Guest + sample.GuestNice
	idle := sample.Idle + sample.Iowait

	if !c.usageReady {
		c.usageLastTotal = total
		c.usageLastIdle = idle
		c.usageReady = true

		percent, percentErr := cpu.Percent(0, false)
		if percentErr == nil && len(percent) > 0 {
			value := clampPercentage(percent[0])
			c.usageLastValue = value
			return value, true, nil
		}
		return 0, false, percentErr
	}

	totalDelta := total - c.usageLastTotal
	idleDelta := idle - c.usageLastIdle
	c.usageLastTotal = total
	c.usageLastIdle = idle
	if totalDelta <= 0 {
		if c.usageLastValue >= 0 {
			return c.usageLastValue, true, nil
		}
		return 0, false, nil
	}

	usage := ((totalDelta - idleDelta) / totalDelta) * 100.0
	usage = clampPercentage(usage)
	c.usageLastValue = usage
	return usage, true, nil
}

func clampPercentage(value float64) float64 {
	if value < 0 {
		return 0
	}
	if value > 100 {
		return 100
	}
	return value
}

func (c *GoNativeCPUCollector) maybeRefreshTemp(now time.Time) {
	c.tempMu.RLock()
	stale := c.tempAt.IsZero() || now.Sub(c.tempAt) >= 2*time.Second
	c.tempMu.RUnlock()
	if stale {
		c.triggerTempRefresh()
	}
}

func (c *GoNativeCPUCollector) triggerTempRefresh() {
	if !atomic.CompareAndSwapInt32(&c.tempUpdating, 0, 1) {
		return
	}
	go func() {
		defer atomic.StoreInt32(&c.tempUpdating, 0)
		value := getRealCPUTemperature()
		now := time.Now()
		c.tempMu.Lock()
		c.tempAt = now
		if value > 0 {
			c.tempValue = value
			c.tempOK = true
		} else {
			c.tempOK = false
		}
		c.tempMu.Unlock()
	}()
}

func (c *GoNativeCPUCollector) getCachedTemp() (float64, bool) {
	c.tempMu.RLock()
	defer c.tempMu.RUnlock()
	if !c.tempOK {
		return 0, false
	}
	return c.tempValue, true
}

func (c *GoNativeCPUCollector) maybeRefreshFreq(now time.Time) {
	c.freqMu.RLock()
	stale := c.freqAt.IsZero() || now.Sub(c.freqAt) >= 3*time.Second
	c.freqMu.RUnlock()
	if stale {
		c.triggerFreqRefresh()
	}
}

func (c *GoNativeCPUCollector) triggerFreqRefresh() {
	if !atomic.CompareAndSwapInt32(&c.freqUpdating, 0, 1) {
		return
	}
	go func() {
		defer atomic.StoreInt32(&c.freqUpdating, 0)
		current, _ := getRealCPUFrequency()
		now := time.Now()
		c.freqMu.Lock()
		c.freqAt = now
		if current > 0 {
			c.freqValue = current
			c.freqOK = true
		} else {
			c.freqOK = false
		}
		c.freqMu.Unlock()
	}()
}

func (c *GoNativeCPUCollector) getCachedFreq() (float64, bool) {
	c.freqMu.RLock()
	defer c.freqMu.RUnlock()
	if !c.freqOK {
		return 0, false
	}
	return c.freqValue, true
}
