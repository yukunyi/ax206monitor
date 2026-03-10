package main

import (
	"github.com/shirou/gopsutil/v3/load"
	"os"
	"strings"
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
		c.setItem("go_native.system.hostname", NewCollectItem("go_native.system.hostname", "Host name", "", 0, 0, 0))
		c.setItem("go_native.system.resolution", NewCollectItem("go_native.system.resolution", "Display resolution", "", 0, 0, 0))
		c.setItem("go_native.system.refresh_rate", NewCollectItem("go_native.system.refresh_rate", "Display refresh rate", "", 0, 0, 0))
		c.setItem("go_native.system.display", NewCollectItem("go_native.system.display", "Display mode", "", 0, 0, 0))
		c.setItem("go_native.system.collect.max_ms", NewCollectItem("go_native.system.collect.max_ms", "Collect max duration", "ms", 0, 0, 0))
		c.setItem("go_native.system.collect.avg_ms", NewCollectItem("go_native.system.collect.avg_ms", "Collect avg duration", "ms", 0, 0, 0))
		c.setItem("go_native.system.render.max_ms", NewCollectItem("go_native.system.render.max_ms", "Render max duration", "ms", 0, 0, 0))
		c.setItem("go_native.system.render.avg_ms", NewCollectItem("go_native.system.render.avg_ms", "Render avg duration", "ms", 0, 0, 0))
		c.setItem("go_native.system.output.max_ms", NewCollectItem("go_native.system.output.max_ms", "Output max duration", "ms", 0, 0, 0))
		c.setItem("go_native.system.output.avg_ms", NewCollectItem("go_native.system.output.avg_ms", "Output avg duration", "ms", 0, 0, 0))
		c.setItem("go_native.system.output.memimg.max_ms", NewCollectItem("go_native.system.output.memimg.max_ms", "Output memimg max duration", "ms", 0, 0, 0))
		c.setItem("go_native.system.output.memimg.avg_ms", NewCollectItem("go_native.system.output.memimg.avg_ms", "Output memimg avg duration", "ms", 0, 0, 0))
		c.setItem("go_native.system.output.ax206usb.max_ms", NewCollectItem("go_native.system.output.ax206usb.max_ms", "Output ax206usb max duration", "ms", 0, 0, 0))
		c.setItem("go_native.system.output.ax206usb.avg_ms", NewCollectItem("go_native.system.output.ax206usb.avg_ms", "Output ax206usb avg duration", "ms", 0, 0, 0))
	}
	if item := c.getItem("go_native.system.current_time"); item != nil {
		item.SetValue(time.Now().Format("2006-01-02 15:04:05"))
		item.SetAvailable(true)
	}
	if item := c.getItem("go_native.system.hostname"); item != nil {
		if hostName, err := os.Hostname(); err == nil && strings.TrimSpace(hostName) != "" {
			item.SetValue(hostName)
			item.SetAvailable(true)
		} else {
			item.SetAvailable(false)
		}
	}
	updateSystemDisplayItems(c)
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
	if item := c.getItem("go_native.system.hostname"); item != nil {
		if hostName, err := os.Hostname(); err == nil && strings.TrimSpace(hostName) != "" {
			item.SetValue(hostName)
			item.SetAvailable(true)
		} else {
			item.SetAvailable(false)
		}
	}
	updateSystemDisplayItems(c)

	stats := GetCollectorManager().Stats()
	setSystemMetricItem(c.getItem("go_native.system.collect.max_ms"), stats.CollectMaxMS)
	setSystemMetricItem(c.getItem("go_native.system.collect.avg_ms"), stats.CollectAvgMS)
	setSystemMetricItem(c.getItem("go_native.system.render.max_ms"), stats.RenderMaxMS)
	setSystemMetricItem(c.getItem("go_native.system.render.avg_ms"), stats.RenderAvgMS)
	setSystemMetricItem(c.getItem("go_native.system.output.max_ms"), stats.OutputMaxMS)
	setSystemMetricItem(c.getItem("go_native.system.output.avg_ms"), stats.OutputAvgMS)

	setOutputTypeMetric(c, "memimg", stats.OutputStats)
	setOutputTypeMetric(c, "ax206usb", stats.OutputStats)
	return err
}

func setSystemMetricItem(item *CollectItem, value int64) {
	if item == nil {
		return
	}
	item.SetValue(value)
	item.SetAvailable(true)
}

func setOutputTypeMetric(c *GoNativeSystemCollector, typeName string, stats map[string]OutputHandlerRuntimeStats) {
	if c == nil {
		return
	}
	maxKey := "go_native.system.output." + strings.ToLower(typeName) + ".max_ms"
	avgKey := "go_native.system.output." + strings.ToLower(typeName) + ".avg_ms"
	maxItem := c.getItem(maxKey)
	avgItem := c.getItem(avgKey)
	if maxItem == nil || avgItem == nil {
		return
	}
	entry, ok := stats[typeName]
	if !ok {
		maxItem.SetAvailable(false)
		avgItem.SetAvailable(false)
		return
	}
	maxItem.SetValue(entry.MaxMS)
	maxItem.SetAvailable(true)
	avgItem.SetValue(entry.AvgMS)
	avgItem.SetAvailable(true)
}

func updateSystemDisplayItems(c *GoNativeSystemCollector) {
	if c == nil {
		return
	}
	resolution, refreshRate, ok := getDisplayInfoSnapshot(2 * time.Minute)
	resolutionItem := c.getItem("go_native.system.resolution")
	refreshItem := c.getItem("go_native.system.refresh_rate")
	displayItem := c.getItem("go_native.system.display")
	if resolutionItem != nil {
		if ok && strings.TrimSpace(resolution) != "" {
			resolutionItem.SetValue(resolution)
			resolutionItem.SetAvailable(true)
		} else {
			resolutionItem.SetAvailable(false)
		}
	}
	if refreshItem != nil {
		if ok && strings.TrimSpace(refreshRate) != "" {
			refreshItem.SetValue(refreshRate)
			refreshItem.SetAvailable(true)
		} else {
			refreshItem.SetAvailable(false)
		}
	}
	if displayItem != nil {
		if ok {
			displayItem.SetValue(composeDisplayModeValue(resolution, refreshRate))
			displayItem.SetAvailable(true)
		} else {
			displayItem.SetAvailable(false)
		}
	}
}

func composeDisplayModeValue(resolution, refreshRate string) string {
	resolution = strings.TrimSpace(resolution)
	refreshRate = strings.TrimSpace(refreshRate)
	if resolution == "" {
		resolution = "-"
	}
	if refreshRate == "" {
		refreshRate = "-"
	}
	return resolution + "@" + refreshRate
}
