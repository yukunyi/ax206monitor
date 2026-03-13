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
		ensureOutputMetricItems(c, outputTypeMemImg, "Output memimg")
		ensureOutputMetricItems(c, outputTypeAX206USB, "AX206 refresh")
		ensureOutputMetricItems(c, outputTypeHTTPPush, "HTTP push")
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

	cfg := GetGlobalCollectorConfig()
	if cfg != nil {
		for _, outputCfg := range cfg.Outputs {
			typeName := strings.ToLower(strings.TrimSpace(outputCfg.Type))
			if typeName == "" {
				continue
			}
			ensureOutputMetricItems(c, typeName, "Output "+typeName)
		}
	}
	for typeName := range stats.OutputStats {
		ensureOutputMetricItems(c, typeName, "Output "+typeName)
		setOutputTypeMetric(c, typeName, stats.OutputStats)
	}
	setAX206DeviceOutputMetrics(c, outputTypeAX206USB, GetAX206DeviceFrameRuntimeStats())
	for typeName, pushStats := range GetHTTPPushRuntimeStats() {
		ensureOutputMetricItems(c, typeName, "HTTP push "+typeName)
		setOutputMetricValues(c, typeName, pushStats.Calls, pushStats.LastMS, pushStats.MaxMS, pushStats.AvgMS)
	}
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
	lastKey, maxKey, avgKey := outputMetricKeys(typeName)
	lastItem := c.getItem(lastKey)
	maxItem := c.getItem(maxKey)
	avgItem := c.getItem(avgKey)
	if lastItem == nil && maxItem == nil && avgItem == nil {
		return
	}
	entry, ok := stats[typeName]
	if !ok {
		if lastItem != nil {
			lastItem.SetAvailable(false)
		}
		if maxItem != nil {
			maxItem.SetAvailable(false)
		}
		if avgItem != nil {
			avgItem.SetAvailable(false)
		}
		return
	}
	setOutputMetricValues(c, typeName, entry.Calls, entry.LastMS, entry.MaxMS, entry.AvgMS)
}

func setAX206DeviceOutputMetrics(c *GoNativeSystemCollector, typeName string, stats AX206DeviceFrameRuntimeStats) {
	if c == nil {
		return
	}
	setOutputMetricValues(c, typeName, stats.Calls, stats.LastMS, stats.MaxMS, stats.AvgMS)
}

func sanitizeOutputMetricType(typeName string) string {
	normalized := strings.ToLower(strings.TrimSpace(typeName))
	if normalized == "" {
		return "unknown"
	}
	builder := strings.Builder{}
	builder.Grow(len(normalized))
	lastUnderscore := false
	for _, ch := range normalized {
		if (ch >= 'a' && ch <= 'z') || (ch >= '0' && ch <= '9') {
			builder.WriteRune(ch)
			lastUnderscore = false
			continue
		}
		if !lastUnderscore {
			builder.WriteByte('_')
			lastUnderscore = true
		}
	}
	result := strings.Trim(builder.String(), "_")
	if result == "" {
		return "unknown"
	}
	return result
}

func outputMetricKeys(typeName string) (string, string, string) {
	metricType := sanitizeOutputMetricType(typeName)
	prefix := "go_native.system.output." + metricType
	return prefix + ".last_ms", prefix + ".max_ms", prefix + ".avg_ms"
}

func ensureOutputMetricItems(c *GoNativeSystemCollector, typeName string, labelPrefix string) {
	if c == nil {
		return
	}
	lastKey, maxKey, avgKey := outputMetricKeys(typeName)
	typeLabel := strings.ToLower(strings.TrimSpace(typeName))
	if strings.TrimSpace(labelPrefix) == "" {
		labelPrefix = "Output " + typeLabel
	}
	if c.getItem(lastKey) == nil {
		c.setItem(lastKey, NewCollectItem(lastKey, labelPrefix+" last duration", "ms", 0, 0, 0))
	}
	if c.getItem(maxKey) == nil {
		c.setItem(maxKey, NewCollectItem(maxKey, labelPrefix+" max duration", "ms", 0, 0, 0))
	}
	if c.getItem(avgKey) == nil {
		c.setItem(avgKey, NewCollectItem(avgKey, labelPrefix+" avg duration", "ms", 0, 0, 0))
	}
}

func setOutputMetricValues(c *GoNativeSystemCollector, typeName string, calls, lastMS, maxMS, avgMS int64) {
	if c == nil {
		return
	}
	lastKey, maxKey, avgKey := outputMetricKeys(typeName)
	lastItem := c.getItem(lastKey)
	maxItem := c.getItem(maxKey)
	avgItem := c.getItem(avgKey)
	if calls <= 0 {
		if lastItem != nil {
			lastItem.SetAvailable(false)
		}
		if maxItem != nil {
			maxItem.SetAvailable(false)
		}
		if avgItem != nil {
			avgItem.SetAvailable(false)
		}
		return
	}
	if lastItem != nil {
		lastItem.SetValue(lastMS)
		lastItem.SetAvailable(true)
	}
	if maxItem != nil {
		maxItem.SetValue(maxMS)
		maxItem.SetAvailable(true)
	}
	if avgItem != nil {
		avgItem.SetValue(avgMS)
		avgItem.SetAvailable(true)
	}
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
