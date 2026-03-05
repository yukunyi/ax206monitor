package main

import (
	"fmt"
	"github.com/shirou/gopsutil/v3/mem"
	"time"
)

type GoNativeMemoryCollector struct {
	*BaseCollector
}

func NewGoNativeMemoryCollector() *GoNativeMemoryCollector {
	return &GoNativeMemoryCollector{BaseCollector: NewBaseCollector("go_native.memory")}
}

func (c *GoNativeMemoryCollector) GetAllItems() map[string]*CollectItem {
	if c.getItem("go_native.memory.usage") == nil {
		c.setItem("go_native.memory.usage", NewCollectItem("go_native.memory.usage", "Memory usage", "%", 0, 100, 0))
		c.setItem("go_native.memory.used", NewCollectItem("go_native.memory.used", "Memory used", "GB", 0, 0, 1))
		c.setItem("go_native.memory.total", NewCollectItem("go_native.memory.total", "Memory total", "GB", 0, 0, 1))
		c.setItem("go_native.memory.usage_text", NewCollectItem("go_native.memory.usage_text", "Memory usage detail", "", 0, 0, 0))
		c.setItem("go_native.memory.usage_progress", NewCollectItem("go_native.memory.usage_progress", "Memory usage progress", "%", 0, 100, 0))
		c.setItem("go_native.memory.swap_usage", NewCollectItem("go_native.memory.swap_usage", "Swap usage", "%", 0, 100, 0))
	}

	if info, err := mem.VirtualMemory(); err == nil && info != nil {
		totalGB := float64(info.Total) / (1024 * 1024 * 1024)
		if item := c.getItem("go_native.memory.total"); item != nil {
			item.SetValue(totalGB)
			item.SetAvailable(true)
		}
	}
	return c.ItemsSnapshot()
}

func (c *GoNativeMemoryCollector) UpdateItems() error {
	if !c.IsEnabled() {
		return nil
	}
	_ = c.GetAllItems()

	err := fetchMemorySnapshot(250 * time.Millisecond)
	virtualInfo, virtualOK := getVirtualMemorySnapshot()
	swapInfo, swapOK := getSwapMemorySnapshot()

	if item := c.getItem("go_native.memory.usage"); item != nil {
		if virtualOK && virtualInfo != nil {
			item.SetValue(virtualInfo.UsedPercent)
			item.SetAvailable(true)
		} else {
			item.SetAvailable(false)
		}
	}

	if item := c.getItem("go_native.memory.used"); item != nil {
		if virtualOK && virtualInfo != nil {
			item.SetValue(float64(virtualInfo.Used) / (1024 * 1024 * 1024))
			item.SetAvailable(true)
		} else {
			item.SetAvailable(false)
		}
	}

	if item := c.getItem("go_native.memory.usage_text"); item != nil {
		if virtualOK && virtualInfo != nil {
			usedGB := float64(virtualInfo.Used) / (1024 * 1024 * 1024)
			totalGB := float64(virtualInfo.Total) / (1024 * 1024 * 1024)
			item.SetValue(fmt.Sprintf("%.1f/%.1f GB (%.0f%%)", usedGB, totalGB, virtualInfo.UsedPercent))
			item.SetAvailable(true)
		} else {
			item.SetAvailable(false)
		}
	}

	if item := c.getItem("go_native.memory.usage_progress"); item != nil {
		if virtualOK && virtualInfo != nil {
			item.SetValue(virtualInfo.UsedPercent)
			item.SetAvailable(true)
		} else {
			item.SetAvailable(false)
		}
	}

	if item := c.getItem("go_native.memory.swap_usage"); item != nil {
		if swapOK && swapInfo != nil {
			item.SetValue(swapInfo.UsedPercent)
			item.SetAvailable(true)
		} else {
			item.SetAvailable(false)
		}
	}

	if item := c.getItem("go_native.memory.total"); item != nil {
		if virtualOK && virtualInfo != nil {
			item.SetValue(float64(virtualInfo.Total) / (1024 * 1024 * 1024))
			item.SetAvailable(true)
		}
	}

	return err
}
