package main

import (
	"strings"
	"time"
)

const (
	renderSpecialFormatNone       = ""
	renderSpecialFormatTime       = "time"
	renderSpecialFormatDisplay    = "display"
	renderSpecialFormatResolution = "resolution"
	renderSpecialFormatRefresh    = "refresh"
)

func resolveItemDisplayValueParts(item *ItemConfig, monitor *RenderMonitorSnapshot, value *CollectValue, config *MonitorConfig) (string, string) {
	fallbackValue, fallbackUnit := FormatCollectValueParts(value, resolveUnitOverride(item))
	format := resolveRenderSpecialFormat(item, monitor)

	switch format.kind {
	case renderSpecialFormatTime:
		return time.Now().Format(format.timeLayout), ""
	case renderSpecialFormatDisplay, renderSpecialFormatResolution, renderSpecialFormatRefresh:
		resolution, refresh, ok := getDisplayInfoSnapshot(2 * time.Minute)
		if !ok {
			return fallbackValue, fallbackUnit
		}
		return formatDisplayTemplate(format.displayTemplate, resolution, refresh), ""
	default:
		_ = config
		return fallbackValue, fallbackUnit
	}
}

func prepareRenderSpecialFormatRuntime(item *ItemConfig) renderSpecialFormatRuntime {
	runtime := renderSpecialFormatRuntime{
		monitorKey: normalizeRenderMonitorKey(itemMonitorName(item)),
	}
	runtime.kind = detectRenderSpecialFormatKind(runtime.monitorKey)

	rawFormat := strings.TrimSpace(getItemAttrString(item, "format", ""))
	switch runtime.kind {
	case renderSpecialFormatTime:
		runtime.timeLayout = normalizeTimeLayout(rawFormat)
	case renderSpecialFormatDisplay, renderSpecialFormatResolution, renderSpecialFormatRefresh:
		runtime.displayTemplate = resolveDisplayTemplate(rawFormat, runtime.kind)
	}
	return runtime
}

func resolveRenderSpecialFormat(item *ItemConfig, monitor *RenderMonitorSnapshot) renderSpecialFormatRuntime {
	if item != nil && item.runtime.prepared {
		runtime := item.runtime.specialFormat
		if runtime.kind != renderSpecialFormatNone {
			return runtime
		}
		if monitor == nil {
			return runtime
		}
		monitorKey := normalizeRenderMonitorKey(monitor.name)
		if monitorKey == "" || monitorKey == runtime.monitorKey {
			return runtime
		}
		runtime.monitorKey = monitorKey
		runtime.kind = detectRenderSpecialFormatKind(monitorKey)
		switch runtime.kind {
		case renderSpecialFormatTime:
			if runtime.timeLayout == "" {
				runtime.timeLayout = normalizeTimeLayout("")
			}
		case renderSpecialFormatDisplay, renderSpecialFormatResolution, renderSpecialFormatRefresh:
			runtime.displayTemplate = resolveDisplayTemplate("", runtime.kind)
		}
		return runtime
	}

	runtime := prepareRenderSpecialFormatRuntime(item)
	if runtime.kind != renderSpecialFormatNone || monitor == nil {
		return runtime
	}
	monitorKey := normalizeRenderMonitorKey(monitor.name)
	if monitorKey == "" || monitorKey == runtime.monitorKey {
		return runtime
	}
	runtime.monitorKey = monitorKey
	runtime.kind = detectRenderSpecialFormatKind(monitorKey)
	switch runtime.kind {
	case renderSpecialFormatTime:
		runtime.timeLayout = normalizeTimeLayout(strings.TrimSpace(getItemAttrString(item, "format", "")))
	case renderSpecialFormatDisplay, renderSpecialFormatResolution, renderSpecialFormatRefresh:
		runtime.displayTemplate = resolveDisplayTemplate(strings.TrimSpace(getItemAttrString(item, "format", "")), runtime.kind)
	}
	return runtime
}

func itemMonitorName(item *ItemConfig) string {
	if item == nil {
		return ""
	}
	return item.Monitor
}

func normalizeRenderMonitorKey(name string) string {
	return strings.ToLower(strings.TrimSpace(name))
}

func detectRenderSpecialFormatKind(monitorKey string) string {
	switch monitorKey {
	case "alias.system.time", "go_native.system.current_time":
		return renderSpecialFormatTime
	case "alias.system.display", "go_native.system.display":
		return renderSpecialFormatDisplay
	case "alias.system.resolution", "go_native.system.resolution":
		return renderSpecialFormatResolution
	case "alias.system.refresh_rate", "go_native.system.refresh_rate":
		return renderSpecialFormatRefresh
	default:
		return renderSpecialFormatNone
	}
}

func resolveDisplayTemplate(template, kind string) string {
	template = strings.TrimSpace(template)
	if template != "" {
		return template
	}
	switch kind {
	case renderSpecialFormatResolution:
		return "{resolution}"
	case renderSpecialFormatRefresh:
		return "{refresh_rate}"
	default:
		return "{resolution}@{refresh_rate}"
	}
}

func formatDisplayTemplate(template, resolution, refresh string) string {
	resolution = strings.TrimSpace(resolution)
	refresh = strings.TrimSpace(refresh)
	if resolution == "" {
		resolution = "-"
	}
	if refresh == "" {
		refresh = "-"
	}
	width := ""
	height := ""
	parts := strings.SplitN(resolution, "x", 2)
	if len(parts) == 2 {
		width = strings.TrimSpace(parts[0])
		height = strings.TrimSpace(parts[1])
	}

	out := strings.NewReplacer(
		"{resolution}", resolution,
		"{refresh_rate}", refresh,
		"{refresh}", refresh,
		"{width}", width,
		"{height}", height,
	).Replace(template)
	out = strings.TrimSpace(out)
	if out == "" {
		return resolution + "@" + refresh
	}
	return out
}

func normalizeTimeLayout(layout string) string {
	text := strings.TrimSpace(layout)
	if text == "" {
		return "2006-01-02 15:04:05"
	}
	if !strings.Contains(text, "%") {
		return text
	}
	text = strings.ReplaceAll(text, "%%", "__PERCENT__")
	text = strings.NewReplacer(
		"%Y", "2006",
		"%y", "06",
		"%m", "01",
		"%d", "02",
		"%H", "15",
		"%M", "04",
		"%S", "05",
		"%b", "Jan",
		"%B", "January",
		"%a", "Mon",
		"%A", "Monday",
	).Replace(text)
	text = strings.ReplaceAll(text, "__PERCENT__", "%")
	return text
}
