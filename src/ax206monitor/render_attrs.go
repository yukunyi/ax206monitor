package main

import (
	"fmt"
	"strconv"
	"strings"
)

func itemRenderAttrs(item *ItemConfig) map[string]interface{} {
	if item == nil || item.RenderAttrsMap == nil {
		return nil
	}
	return item.RenderAttrsMap
}

func getItemAttr(item *ItemConfig, key string) (interface{}, bool) {
	attrs := itemRenderAttrs(item)
	if len(attrs) == 0 {
		return nil, false
	}
	value, exists := attrs[key]
	return value, exists
}

func getTypeDefaultAttr(config *MonitorConfig, itemType, key string) (interface{}, bool) {
	if config == nil {
		return nil, false
	}
	defaults := config.GetTypeDefaults(itemType)
	if len(defaults.RenderAttrsMap) == 0 {
		return nil, false
	}
	value, exists := defaults.RenderAttrsMap[key]
	return value, exists
}

func getItemAttrWithDefaults(item *ItemConfig, config *MonitorConfig, key string) (interface{}, bool) {
	if isStyleRenderAttrKey(key) {
		if value, exists := resolveStyleRaw(item, config, key); exists {
			return value, true
		}
		if item != nil {
			return nil, false
		}
	}
	if value, exists := getItemAttr(item, key); exists {
		return value, true
	}
	if item == nil {
		return nil, false
	}
	return getTypeDefaultAttr(config, item.Type, key)
}

func getItemAttrString(item *ItemConfig, key, fallback string) string {
	raw, exists := getItemAttr(item, key)
	if !exists || raw == nil {
		return fallback
	}
	switch value := raw.(type) {
	case string:
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			return fallback
		}
		return trimmed
	case fmt.Stringer:
		trimmed := strings.TrimSpace(value.String())
		if trimmed == "" {
			return fallback
		}
		return trimmed
	case float64:
		return strconv.FormatFloat(value, 'f', -1, 64)
	case float32:
		return strconv.FormatFloat(float64(value), 'f', -1, 64)
	case int:
		return strconv.Itoa(value)
	case int64:
		return strconv.FormatInt(value, 10)
	case bool:
		if value {
			return "true"
		}
		return "false"
	default:
		return fallback
	}
}

func getItemAttrFloat(item *ItemConfig, key string, fallback float64) float64 {
	raw, exists := getItemAttr(item, key)
	if !exists || raw == nil {
		return fallback
	}
	switch value := raw.(type) {
	case float64:
		return value
	case float32:
		return float64(value)
	case int:
		return float64(value)
	case int64:
		return float64(value)
	case uint64:
		return float64(value)
	case string:
		if parsed, err := strconv.ParseFloat(strings.TrimSpace(value), 64); err == nil {
			return parsed
		}
	}
	return fallback
}

func getItemAttrInt(item *ItemConfig, key string, fallback int) int {
	return int(getItemAttrFloat(item, key, float64(fallback)))
}

func getItemAttrBool(item *ItemConfig, key string, fallback bool) bool {
	raw, exists := getItemAttr(item, key)
	if !exists || raw == nil {
		return fallback
	}
	switch value := raw.(type) {
	case bool:
		return value
	case string:
		parsed, err := strconv.ParseBool(strings.TrimSpace(value))
		if err != nil {
			return fallback
		}
		return parsed
	case float64:
		return value != 0
	case int:
		return value != 0
	}
	return fallback
}

func getItemAttrColor(item *ItemConfig, key, fallback string) string {
	value := strings.TrimSpace(getItemAttrString(item, key, fallback))
	if value == "" {
		return fallback
	}
	return value
}

func getItemAttrStringCfg(item *ItemConfig, config *MonitorConfig, key, fallback string) string {
	raw, exists := getItemAttrWithDefaults(item, config, key)
	if !exists || raw == nil {
		return fallback
	}
	switch value := raw.(type) {
	case string:
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			return fallback
		}
		return trimmed
	case fmt.Stringer:
		trimmed := strings.TrimSpace(value.String())
		if trimmed == "" {
			return fallback
		}
		return trimmed
	case float64:
		return strconv.FormatFloat(value, 'f', -1, 64)
	case float32:
		return strconv.FormatFloat(float64(value), 'f', -1, 64)
	case int:
		return strconv.Itoa(value)
	case int64:
		return strconv.FormatInt(value, 10)
	case bool:
		if value {
			return "true"
		}
		return "false"
	default:
		return fallback
	}
}

func getItemAttrFloatCfg(item *ItemConfig, config *MonitorConfig, key string, fallback float64) float64 {
	raw, exists := getItemAttrWithDefaults(item, config, key)
	if !exists || raw == nil {
		return fallback
	}
	switch value := raw.(type) {
	case float64:
		return value
	case float32:
		return float64(value)
	case int:
		return float64(value)
	case int64:
		return float64(value)
	case uint64:
		return float64(value)
	case string:
		if parsed, err := strconv.ParseFloat(strings.TrimSpace(value), 64); err == nil {
			return parsed
		}
	}
	return fallback
}

func getItemAttrIntCfg(item *ItemConfig, config *MonitorConfig, key string, fallback int) int {
	return int(getItemAttrFloatCfg(item, config, key, float64(fallback)))
}

func getItemAttrBoolCfg(item *ItemConfig, config *MonitorConfig, key string, fallback bool) bool {
	raw, exists := getItemAttrWithDefaults(item, config, key)
	if !exists || raw == nil {
		return fallback
	}
	switch value := raw.(type) {
	case bool:
		return value
	case string:
		parsed, err := strconv.ParseBool(strings.TrimSpace(value))
		if err != nil {
			return fallback
		}
		return parsed
	case float64:
		return value != 0
	case int:
		return value != 0
	}
	return fallback
}

func getItemAttrColorCfg(item *ItemConfig, config *MonitorConfig, key, fallback string) string {
	value := strings.TrimSpace(getItemAttrStringCfg(item, config, key, fallback))
	if value == "" {
		return fallback
	}
	return value
}
