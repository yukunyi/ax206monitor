package main

import (
	"strings"
	"time"
)

func resolveItemDisplayValueParts(item *ItemConfig, monitor *CollectItem, value *CollectValue, config *MonitorConfig) (string, string) {
	fallbackValue, fallbackUnit := FormatCollectValueParts(value, resolveUnitOverride(item))
	keys := collectMonitorNameKeys(item, monitor)

	if matchMonitorName(keys, "alias.system.time", "go_native.system.current_time") {
		layout := strings.TrimSpace(getItemAttrString(item, "format", ""))
		if layout == "" {
			layout = "2006-01-02 15:04:05"
		}
		return time.Now().Format(normalizeTimeLayout(layout)), ""
	}

	displayKind := detectDisplayFormatKind(keys)
	if displayKind != "" {
		resolution, refresh, ok := getDisplayInfoSnapshot(2 * time.Minute)
		if !ok {
			return fallbackValue, fallbackUnit
		}
		template := strings.TrimSpace(getItemAttrString(item, "format", ""))
		if template == "" {
			switch displayKind {
			case "resolution":
				template = "{resolution}"
			case "refresh":
				template = "{refresh_rate}"
			default:
				template = "{resolution}@{refresh_rate}"
			}
		}
		return formatDisplayTemplate(template, resolution, refresh), ""
	}

	_ = config
	return fallbackValue, fallbackUnit
}

func collectMonitorNameKeys(item *ItemConfig, monitor *CollectItem) []string {
	seen := map[string]struct{}{}
	keys := make([]string, 0, 2)
	appendKey := func(raw string) {
		name := strings.ToLower(strings.TrimSpace(raw))
		if name == "" {
			return
		}
		if _, exists := seen[name]; exists {
			return
		}
		seen[name] = struct{}{}
		keys = append(keys, name)
	}
	if item != nil {
		appendKey(item.Monitor)
	}
	if monitor != nil {
		appendKey(monitor.GetName())
	}
	return keys
}

func matchMonitorName(keys []string, candidates ...string) bool {
	if len(keys) == 0 || len(candidates) == 0 {
		return false
	}
	set := make(map[string]struct{}, len(candidates))
	for _, candidate := range candidates {
		name := strings.ToLower(strings.TrimSpace(candidate))
		if name == "" {
			continue
		}
		set[name] = struct{}{}
	}
	for _, key := range keys {
		if _, ok := set[key]; ok {
			return true
		}
	}
	return false
}

func detectDisplayFormatKind(keys []string) string {
	if matchMonitorName(keys, "alias.system.display", "go_native.system.display") {
		return "display"
	}
	if matchMonitorName(keys, "alias.system.resolution", "go_native.system.resolution") {
		return "resolution"
	}
	if matchMonitorName(keys, "alias.system.refresh_rate", "go_native.system.refresh_rate") {
		return "refresh"
	}
	return ""
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
