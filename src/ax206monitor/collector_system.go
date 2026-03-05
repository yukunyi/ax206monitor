package main

import (
	"github.com/shirou/gopsutil/v3/load"
	"time"
)

type GoNativeSystemCollector struct {
	*BaseCollector
}

func NewGoNativeSystemCollector() *GoNativeSystemCollector {
	return &GoNativeSystemCollector{BaseCollector: NewBaseCollector("go_native.system")}
}

func (c *GoNativeSystemCollector) GetAllItems() map[string]*CollectItem {
	if c.getItem("go_native.system.load_avg") == nil {
		c.setItem("go_native.system.load_avg", NewCollectItem("go_native.system.load_avg", "System load average", "", 0, 0, 1))
		c.setItem("go_native.system.current_time", NewCollectItem("go_native.system.current_time", "Current time", "", 0, 0, 0))
	}
	if item := c.getItem("go_native.system.current_time"); item != nil {
		item.SetValue(time.Now().Format("2006-01-02 15:04:05"))
		item.SetAvailable(true)
	}
	return c.ItemsSnapshot()
}

func (c *GoNativeSystemCollector) UpdateItems() error {
	if !c.IsEnabled() {
		return nil
	}
	_ = c.GetAllItems()

	var err error
	if item := c.getItem("go_native.system.load_avg"); item != nil {
		loadInfo, loadErr := load.Avg()
		if loadErr != nil {
			item.SetAvailable(false)
			err = loadErr
		} else {
			item.SetValue(loadInfo.Load1)
			item.SetAvailable(true)
		}
	}
	if item := c.getItem("go_native.system.current_time"); item != nil {
		item.SetValue(time.Now().Format("2006-01-02 15:04:05"))
		item.SetAvailable(true)
	}
	return err
}
