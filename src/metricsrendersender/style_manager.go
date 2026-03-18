package main

import (
	"fmt"
	"reflect"
	"sort"
	"strings"
)

const (
	styleScopeBase = "base"
	styleScopeType = "type"
	styleScopeItem = "item"
)

type StyleOption struct {
	Label string `json:"label"`
	Value string `json:"value"`
}

type StyleKeyMeta struct {
	Key      string                 `json:"key"`
	Label    string                 `json:"label"`
	Kind     string                 `json:"kind"`
	Scopes   []string               `json:"scopes"`
	Types    []string               `json:"types,omitempty"`
	Options  []StyleOption          `json:"options,omitempty"`
	Default  interface{}            `json:"default,omitempty"`
	Defaults map[string]interface{} `json:"defaults,omitempty"`
}

type styleMetaEntry struct {
	scopeSet  map[string]struct{}
	typeSet   map[string]struct{}
	allowType bool
}

var styleMetaList = []StyleKeyMeta{
	{Key: "font_family", Label: "字体", Kind: "select", Scopes: []string{styleScopeBase}},
	{Key: "text_font_size", Label: "文本字号", Kind: "int", Scopes: []string{styleScopeBase, styleScopeType, styleScopeItem}},
	{Key: "unit_font_size", Label: "单位字号", Kind: "int", Scopes: []string{styleScopeBase, styleScopeType, styleScopeItem}},
	{Key: "value_font_size", Label: "值字号", Kind: "int", Scopes: []string{styleScopeBase, styleScopeType, styleScopeItem}},
	{Key: "color", Label: "文字色", Kind: "color", Scopes: []string{styleScopeBase, styleScopeType, styleScopeItem}},
	{Key: "bg", Label: "背景色", Kind: "color", Scopes: []string{styleScopeBase, styleScopeType, styleScopeItem}},
	{Key: "unit_color", Label: "单位色", Kind: "color", Scopes: []string{styleScopeBase, styleScopeType, styleScopeItem}},
	{Key: "border_width", Label: "边框宽度", Kind: "float", Scopes: []string{styleScopeBase, styleScopeType, styleScopeItem}},
	{Key: "border_color", Label: "边框颜色", Kind: "color", Scopes: []string{styleScopeBase, styleScopeType, styleScopeItem}},
	{Key: "radius", Label: "圆角", Kind: "int", Scopes: []string{styleScopeBase, styleScopeType, styleScopeItem}},
	{Key: "history_points", Label: "历史点数", Kind: "int", Scopes: []string{styleScopeBase, styleScopeType, styleScopeItem}, Types: []string{itemTypeSimpleChart, itemTypeFullChart}},
	{Key: "content_padding_x", Label: "左右边距", Kind: "int", Scopes: []string{styleScopeType, styleScopeItem}, Types: []string{itemTypeLabelText, itemTypeFullChart, itemTypeFullTable, itemTypeFullProgressH, itemTypeFullProgressV, itemTypeFullGauge}},
	{Key: "content_padding_y", Label: "上下边距", Kind: "int", Scopes: []string{styleScopeType, styleScopeItem}, Types: []string{itemTypeLabelText, itemTypeFullChart, itemTypeFullTable, itemTypeFullProgressH, itemTypeFullProgressV, itemTypeFullGauge}},
	{Key: "body_gap", Label: "标题间距", Kind: "int", Scopes: []string{styleScopeType, styleScopeItem}, Types: []string{itemTypeFullChart, itemTypeFullTable, itemTypeFullProgressH}},
	{Key: "header_height", Label: "标题栏高度", Kind: "int", Scopes: []string{styleScopeType, styleScopeItem}, Types: []string{itemTypeFullChart, itemTypeFullTable, itemTypeFullProgressH}},
	{Key: "header_divider", Label: "标题分隔线", Kind: "bool", Scopes: []string{styleScopeType, styleScopeItem}, Types: []string{itemTypeFullChart, itemTypeFullTable, itemTypeFullProgressH}},
	{Key: "header_divider_width", Label: "分隔线宽", Kind: "float", Scopes: []string{styleScopeType, styleScopeItem}, Types: []string{itemTypeFullChart, itemTypeFullTable, itemTypeFullProgressH}},
	{Key: "header_divider_offset", Label: "分隔线偏移", Kind: "float", Scopes: []string{styleScopeType, styleScopeItem}, Types: []string{itemTypeFullChart, itemTypeFullTable, itemTypeFullProgressH}},
	{Key: "header_divider_color", Label: "分隔线颜色", Kind: "color", Scopes: []string{styleScopeType, styleScopeItem}, Types: []string{itemTypeFullChart, itemTypeFullTable, itemTypeFullProgressH}},
	{Key: "show_segment_lines", Label: "分段线", Kind: "bool", Scopes: []string{styleScopeType, styleScopeItem}, Types: []string{itemTypeFullChart}},
	{Key: "show_grid_lines", Label: "网格线", Kind: "bool", Scopes: []string{styleScopeType, styleScopeItem}, Types: []string{itemTypeFullChart}},
	{Key: "grid_lines", Label: "网格线数量", Kind: "int", Scopes: []string{styleScopeType, styleScopeItem}, Types: []string{itemTypeFullChart}},
	{Key: "enable_threshold_colors", Label: "阈值分段颜色", Kind: "bool", Scopes: []string{styleScopeType, styleScopeItem}, Types: []string{itemTypeSimpleChart, itemTypeFullChart}},
	{Key: "line_width", Label: "线宽", Kind: "float", Scopes: []string{styleScopeType, styleScopeItem}, Types: []string{itemTypeSimpleChart, itemTypeSimpleLine, itemTypeFullChart}},
	{Key: "line_orientation", Label: "线方向", Kind: "select", Scopes: []string{styleScopeType, styleScopeItem}, Types: []string{itemTypeSimpleLine}, Options: []StyleOption{{Label: "横向", Value: "horizontal"}, {Label: "竖向", Value: "vertical"}}},
	{Key: "show_avg_line", Label: "均线", Kind: "bool", Scopes: []string{styleScopeType, styleScopeItem}, Types: []string{itemTypeFullChart}},
	{Key: "chart_color", Label: "折线颜色", Kind: "color", Scopes: []string{styleScopeType, styleScopeItem}, Types: []string{itemTypeFullChart}},
	{Key: "chart_area_bg", Label: "图表区背景", Kind: "color", Scopes: []string{styleScopeType, styleScopeItem}, Types: []string{itemTypeFullChart}},
	{Key: "chart_area_border_color", Label: "图表区边框", Kind: "color", Scopes: []string{styleScopeType, styleScopeItem}, Types: []string{itemTypeFullChart}},
	{Key: "progress_style", Label: "进度样式", Kind: "select", Scopes: []string{styleScopeType, styleScopeItem}, Types: []string{itemTypeFullProgressH, itemTypeFullProgressV}, Options: []StyleOption{{Label: "gradient", Value: "gradient"}, {Label: "solid", Value: "solid"}, {Label: "segmented", Value: "segmented"}, {Label: "stripes", Value: "stripes"}}},
	{Key: "bar_height", Label: "条高度", Kind: "float", Scopes: []string{styleScopeType, styleScopeItem}, Types: []string{itemTypeFullProgressH, itemTypeFullProgressV}},
	{Key: "bar_radius", Label: "条圆角", Kind: "float", Scopes: []string{styleScopeType, styleScopeItem}, Types: []string{itemTypeFullProgressH, itemTypeFullProgressV}},
	{Key: "track_color", Label: "轨道颜色", Kind: "color", Scopes: []string{styleScopeType, styleScopeItem}, Types: []string{itemTypeFullProgressH, itemTypeFullProgressV, itemTypeFullGauge}},
	{Key: "segments", Label: "分段数量", Kind: "int", Scopes: []string{styleScopeType, styleScopeItem}, Types: []string{itemTypeFullProgressH, itemTypeFullProgressV}},
	{Key: "segment_gap", Label: "分段间隔", Kind: "float", Scopes: []string{styleScopeType, styleScopeItem}, Types: []string{itemTypeFullProgressH, itemTypeFullProgressV}},
	{Key: "card_radius", Label: "外框圆角", Kind: "float", Scopes: []string{styleScopeType, styleScopeItem}, Types: []string{itemTypeFullChart, itemTypeFullTable, itemTypeFullProgressH, itemTypeFullProgressV, itemTypeFullGauge}},
	{Key: "table_row_gap", Label: "行间距", Kind: "float", Scopes: []string{styleScopeType, styleScopeItem}, Types: []string{itemTypeFullTable}},
	{Key: "table_row_radius", Label: "行圆角", Kind: "float", Scopes: []string{styleScopeType, styleScopeItem}, Types: []string{itemTypeFullTable}},
	{Key: "table_row_bg", Label: "行背景", Kind: "color", Scopes: []string{styleScopeType, styleScopeItem}, Types: []string{itemTypeFullTable}},
	{Key: "table_row_alt_bg", Label: "交替行背景", Kind: "color", Scopes: []string{styleScopeType, styleScopeItem}, Types: []string{itemTypeFullTable}},
	{Key: "table_column_gap", Label: "列间距", Kind: "float", Scopes: []string{styleScopeType, styleScopeItem}, Types: []string{itemTypeFullTable}},
	{Key: "table_label_width_ratio", Label: "标签列比例", Kind: "float", Scopes: []string{styleScopeType, styleScopeItem}, Types: []string{itemTypeFullTable}},
	{Key: "table_show_units", Label: "显示单位", Kind: "bool", Scopes: []string{styleScopeType, styleScopeItem}, Types: []string{itemTypeFullTable}},
	{Key: "gauge_thickness", Label: "仪表盘厚度", Kind: "float", Scopes: []string{styleScopeType, styleScopeItem}, Types: []string{itemTypeFullGauge}},
	{Key: "gauge_gap_degrees", Label: "底部缺口角度", Kind: "float", Scopes: []string{styleScopeType, styleScopeItem}, Types: []string{itemTypeFullGauge}},
	{Key: "gauge_text_gap", Label: "文字间距", Kind: "float", Scopes: []string{styleScopeType, styleScopeItem}, Types: []string{itemTypeFullGauge}},
}

var styleMetaByKey = buildStyleMetaByKey()

func buildStyleMetaByKey() map[string]styleMetaEntry {
	set := make(map[string]styleMetaEntry, len(styleMetaList))
	for _, meta := range styleMetaList {
		key := strings.TrimSpace(meta.Key)
		if key == "" {
			continue
		}
		scopeSet := make(map[string]struct{}, len(meta.Scopes))
		for _, scope := range meta.Scopes {
			name := strings.TrimSpace(scope)
			if name == "" {
				continue
			}
			scopeSet[name] = struct{}{}
		}
		typeSet := make(map[string]struct{}, len(meta.Types))
		for _, itemType := range meta.Types {
			name := normalizeItemTypeName(itemType)
			if name == "" {
				continue
			}
			typeSet[name] = struct{}{}
		}
		set[key] = styleMetaEntry{
			scopeSet:  scopeSet,
			typeSet:   typeSet,
			allowType: len(typeSet) == 0,
		}
	}
	return set
}

func WebStyleKeyMeta() []StyleKeyMeta {
	out := make([]StyleKeyMeta, len(styleMetaList))
	copy(out, styleMetaList)
	for idx := range out {
		meta := &out[idx]
		baseDefault, hasBase := styleCodeDefault("", meta.Key)
		if len(meta.Types) == 1 {
			if typeDefault, ok := styleCodeDefault(meta.Types[0], meta.Key); ok {
				meta.Default = typeDefault
			} else if hasBase {
				meta.Default = baseDefault
			}
		} else if hasBase {
			meta.Default = baseDefault
		}

		defaults := make(map[string]interface{})
		for _, itemType := range allItemTypes {
			typeDefault, ok := styleCodeDefault(itemType, meta.Key)
			if !ok {
				continue
			}
			if hasBase && reflect.DeepEqual(typeDefault, baseDefault) {
				continue
			}
			defaults[itemType] = typeDefault
		}
		if len(defaults) > 0 {
			meta.Defaults = defaults
		}
	}
	return out
}

func isStyleRenderAttrKey(key string) bool {
	return isKnownStyleKey(key)
}

func isKnownStyleKey(key string) bool {
	_, exists := styleMetaByKey[strings.TrimSpace(key)]
	return exists
}

func styleMetaForKey(key string) (styleMetaEntry, bool) {
	meta, ok := styleMetaByKey[strings.TrimSpace(key)]
	return meta, ok
}

func keySupportsScope(key string, scope string) bool {
	meta, ok := styleMetaForKey(key)
	if !ok {
		return false
	}
	_, ok = meta.scopeSet[strings.TrimSpace(scope)]
	return ok
}

func keySupportsType(key string, itemType string) bool {
	meta, ok := styleMetaForKey(key)
	if !ok {
		return false
	}
	if meta.allowType {
		return true
	}
	_, ok = meta.typeSet[normalizeItemTypeName(itemType)]
	return ok
}

func normalizeStyleMap(input map[string]interface{}, scope string, itemType string) map[string]interface{} {
	if len(input) == 0 {
		return map[string]interface{}{}
	}
	out := make(map[string]interface{}, len(input))
	allowAllKeys := strings.TrimSpace(scope) == styleScopeBase
	for rawKey, rawValue := range input {
		key := strings.TrimSpace(rawKey)
		if !isKnownStyleKey(key) {
			continue
		}
		if !allowAllKeys {
			if !keySupportsScope(key, scope) {
				continue
			}
			if !keySupportsType(key, itemType) {
				continue
			}
		}
		out[key] = normalizeStyleValueByKey(key, rawValue)
	}
	return out
}

func normalizeStyleValueByKey(key string, value interface{}) interface{} {
	switch key {
	case "text_font_size", "unit_font_size", "value_font_size", "header_height", "history_points", "grid_lines", "segments", "content_padding_x", "content_padding_y", "body_gap", "radius":
		n, ok := toStyleNumber(value)
		if !ok {
			return 0
		}
		if n < 0 {
			n = 0
		}
		if key == "history_points" && n > 0 && n < 10 {
			n = 10
		}
		if key == "segments" && n > 0 && n < 4 {
			n = 4
		}
		return int(n)
	case "border_width", "line_width", "bar_height", "bar_radius", "segment_gap", "card_radius", "gauge_thickness", "gauge_gap_degrees", "gauge_text_gap", "header_divider_width", "header_divider_offset":
		n, ok := toStyleNumber(value)
		if !ok {
			return 0.0
		}
		if n < 0 {
			n = 0
		}
		return n
	case "header_divider", "show_segment_lines", "show_grid_lines", "enable_threshold_colors", "show_avg_line":
		return toStyleBool(value)
	case "line_orientation":
		text := strings.ToLower(strings.TrimSpace(fmt.Sprintf("%v", value)))
		if text != "vertical" {
			return "horizontal"
		}
		return text
	case "progress_style":
		text := strings.ToLower(strings.TrimSpace(fmt.Sprintf("%v", value)))
		switch text {
		case "solid", "segmented", "stripes", "vertical":
			return text
		default:
			return "gradient"
		}
	default:
		return strings.TrimSpace(fmt.Sprintf("%v", value))
	}
}

func toStyleNumber(value interface{}) (float64, bool) {
	if value == nil {
		return 0, false
	}
	if number, ok := tryGetFloat64(value); ok {
		return number, true
	}
	return 0, false
}

func toStyleBool(value interface{}) bool {
	switch typed := value.(type) {
	case bool:
		return typed
	case int:
		return typed != 0
	case int64:
		return typed != 0
	case float64:
		return typed != 0
	case string:
		text := strings.ToLower(strings.TrimSpace(typed))
		return text == "1" || text == "true" || text == "yes" || text == "on"
	default:
		return false
	}
}

func normalizeThresholdPercentageArray(value interface{}) []float64 {
	out := []float64{25, 50, 75, 100}
	switch typed := value.(type) {
	case []float64:
		for i := 0; i < 4 && i < len(typed); i++ {
			out[i] = clampFloat64(typed[i], 0, 100)
		}
	case []interface{}:
		for i := 0; i < 4 && i < len(typed); i++ {
			if n, ok := toStyleNumber(typed[i]); ok {
				out[i] = clampFloat64(n, 0, 100)
			}
		}
	}
	sort.Float64s(out)
	return out
}

func normalizeLevelColorArrayValue(value interface{}) []string {
	out := []string{"#22c55e", "#eab308", "#f97316", "#ef4444"}
	switch typed := value.(type) {
	case []string:
		for i := 0; i < 4 && i < len(typed); i++ {
			text := strings.TrimSpace(typed[i])
			if text != "" {
				out[i] = text
			}
		}
	case []interface{}:
		for i := 0; i < 4 && i < len(typed); i++ {
			text := strings.TrimSpace(fmt.Sprintf("%v", typed[i]))
			if text != "" {
				out[i] = text
			}
		}
	}
	return out
}

func styleCodeDefault(itemType, key string) (interface{}, bool) {
	switch key {
	case "font_family":
		return "", true
	case "text_font_size":
		return 16, true
	case "unit_font_size":
		return 14, true
	case "value_font_size":
		return 18, true
	case "color":
		return "#f8fafc", true
	case "bg":
		return "", true
	case "unit_color":
		return "#f8fafc", true
	case "border_width":
		if itemType == itemTypeSimpleChart || itemType == itemTypeLabelText || isFullItemType(itemType) {
			return 1.0, true
		}
		return 0.0, true
	case "border_color":
		if itemType == itemTypeSimpleChart || itemType == itemTypeLabelText || isFullItemType(itemType) {
			return "#cbd5e1", true
		}
		return "#475569", true
	case "radius":
		return 0, true
	case "history_points":
		return 150, true
	case "content_padding_x", "content_padding_y":
		if itemType == itemTypeLabelText || strings.HasPrefix(itemType, "full_") {
			return 1, true
		}
		return 3, true
	case "body_gap":
		if itemType == itemTypeFullChart {
			return 4, true
		}
		return 0, true
	case "header_height":
		return 0, true
	case "header_divider":
		return true, true
	case "header_divider_width":
		return 1.0, true
	case "header_divider_offset":
		return 3.0, true
	case "header_divider_color":
		return "#cbd5e166", true
	case "show_segment_lines":
		return false, true
	case "show_grid_lines":
		return false, true
	case "grid_lines":
		return 4, true
	case "enable_threshold_colors":
		return false, true
	case "line_width":
		return 1.0, true
	case "line_orientation":
		return "horizontal", true
	case "show_avg_line":
		return false, true
	case "chart_color":
		return "#38bdf8", true
	case "chart_area_bg":
		if itemType == itemTypeFullChart {
			return "#000000", true
		}
		return "", true
	case "chart_area_border_color":
		return "", true
	case "table_row_gap":
		return 0.0, true
	case "table_row_radius":
		return 0.0, true
	case "table_row_bg":
		return "", true
	case "table_row_alt_bg":
		return "", true
	case "table_column_gap":
		return 0.0, true
	case "table_label_width_ratio":
		return 0.46, true
	case "table_show_units":
		return true, true
	case "progress_style":
		return "gradient", true
	case "bar_height":
		return 0.0, true
	case "bar_radius":
		return 0.0, true
	case "track_color":
		return "#1f2937", true
	case "segments":
		return 12, true
	case "segment_gap":
		return 2.0, true
	case "card_radius":
		return 0.0, true
	case "gauge_thickness":
		return 10.0, true
	case "gauge_gap_degrees":
		return 76.0, true
	case "gauge_text_gap":
		return 1.0, true
	}
	return nil, false
}

func resolveStyleRaw(item *ItemConfig, config *MonitorConfig, key string) (interface{}, bool) {
	normalizedKey := strings.TrimSpace(key)
	if normalizedKey == "" || !isKnownStyleKey(normalizedKey) {
		return nil, false
	}
	itemType := ""
	hasItemType := false
	if item != nil {
		itemType = normalizeItemTypeName(item.Type)
		hasItemType = true
	}

	if canUseItemCustomStyle(item, config) {
		if value, ok := readStyleMapValue(item.Style, normalizedKey); ok {
			return value, true
		}
	}

	if config != nil {
		if hasItemType {
			defaults := config.GetTypeDefaults(itemType)
			if value, ok := readStyleMapValue(defaults.Style, normalizedKey); ok {
				return value, true
			}
		}
		if value, ok := readStyleMapValue(config.StyleBase, normalizedKey); ok {
			return value, true
		}
	}

	if value, ok := styleCodeDefault(itemType, normalizedKey); ok {
		return value, true
	}
	return nil, false
}

func readStyleMapValue(style map[string]interface{}, key string) (interface{}, bool) {
	if len(style) == 0 {
		return nil, false
	}
	value, exists := style[key]
	if !exists || value == nil {
		return nil, false
	}
	return normalizeStyleValueByKey(key, value), true
}

func resolveStyleInt(item *ItemConfig, config *MonitorConfig, key string, fallback int) int {
	raw, exists := resolveStyleRaw(item, config, key)
	if !exists {
		return fallback
	}
	number, ok := toStyleNumber(raw)
	if !ok {
		return fallback
	}
	return int(number)
}

func resolveStyleFloat(item *ItemConfig, config *MonitorConfig, key string, fallback float64) float64 {
	raw, exists := resolveStyleRaw(item, config, key)
	if !exists {
		return fallback
	}
	number, ok := toStyleNumber(raw)
	if !ok {
		return fallback
	}
	return number
}

func resolveStyleBool(item *ItemConfig, config *MonitorConfig, key string, fallback bool) bool {
	raw, exists := resolveStyleRaw(item, config, key)
	if !exists {
		return fallback
	}
	return toStyleBool(raw)
}

func resolveStyleString(item *ItemConfig, config *MonitorConfig, key string, fallback string) string {
	raw, exists := resolveStyleRaw(item, config, key)
	if !exists {
		return fallback
	}
	text := strings.TrimSpace(fmt.Sprintf("%v", raw))
	if text == "" {
		return fallback
	}
	return text
}

func resolveStyleColor(item *ItemConfig, config *MonitorConfig, key string, fallback string) string {
	color := resolveStyleString(item, config, key, fallback)
	if strings.TrimSpace(color) == "" {
		return fallback
	}
	return color
}

func resolveStyleOverrideRaw(item *ItemConfig, config *MonitorConfig, key string) (interface{}, bool) {
	normalizedKey := strings.TrimSpace(key)
	if normalizedKey == "" || !isKnownStyleKey(normalizedKey) {
		return nil, false
	}
	itemType := ""
	if item != nil {
		itemType = normalizeItemTypeName(item.Type)
	}
	if canUseItemCustomStyle(item, config) {
		if value, ok := readStyleMapValue(item.Style, normalizedKey); ok {
			return value, true
		}
	}
	if config != nil && itemType != "" {
		defaults := config.GetTypeDefaults(itemType)
		if value, ok := readStyleMapValue(defaults.Style, normalizedKey); ok {
			return value, true
		}
	}
	return nil, false
}

func resolveStyleOverrideString(item *ItemConfig, config *MonitorConfig, key string) string {
	raw, exists := resolveStyleOverrideRaw(item, config, key)
	if !exists {
		return ""
	}
	return strings.TrimSpace(fmt.Sprintf("%v", raw))
}

func resolveStyleOverrideColor(item *ItemConfig, config *MonitorConfig, key string) string {
	return resolveStyleOverrideString(item, config, key)
}

func resolveStyleThresholdsPercent(item *ItemConfig, config *MonitorConfig) []float64 {
	raw, exists := resolveStyleRaw(item, config, "thresholds")
	if !exists {
		return []float64{25, 50, 75, 100}
	}
	return normalizeThresholdPercentageArray(raw)
}

func resolveStyleLevelColors(item *ItemConfig, config *MonitorConfig) []string {
	raw, exists := resolveStyleRaw(item, config, "level_colors")
	if !exists {
		return []string{"#22c55e", "#eab308", "#f97316", "#ef4444"}
	}
	return normalizeLevelColorArrayValue(raw)
}

func normalizeStyleConfiguration(cfg *MonitorConfig) {
	if cfg == nil {
		return
	}
	cfg.StyleBase = normalizeStyleMap(cfg.StyleBase, styleScopeBase, "")
	if cfg.TypeDefaults == nil {
		cfg.TypeDefaults = map[string]ItemTypeDefaults{}
	}
	for _, itemType := range webItemTypes() {
		entry := cfg.TypeDefaults[itemType]
		entry.Style = normalizeStyleMap(entry.Style, styleScopeType, itemType)
		entry.RenderAttrsMap = stripStyleKeysFromRenderAttrs(entry.RenderAttrsMap)
		cfg.TypeDefaults[itemType] = entry
	}
	for itemType, entry := range cfg.TypeDefaults {
		normalizedType := normalizeItemTypeName(itemType)
		entry.Style = normalizeStyleMap(entry.Style, styleScopeType, normalizedType)
		entry.RenderAttrsMap = stripStyleKeysFromRenderAttrs(entry.RenderAttrsMap)
		cfg.TypeDefaults[normalizedType] = entry
		if normalizedType != itemType {
			delete(cfg.TypeDefaults, itemType)
		}
	}
}

func normalizeItemStyleConfiguration(cfg *MonitorConfig, item *ItemConfig) {
	if cfg == nil || item == nil {
		return
	}
	item.Style = normalizeStyleMap(item.Style, styleScopeItem, item.Type)
	item.RenderAttrsMap = stripStyleKeysFromRenderAttrs(item.RenderAttrsMap)
}

func stripStyleKeysFromRenderAttrs(attrs map[string]interface{}) map[string]interface{} {
	if len(attrs) == 0 {
		return map[string]interface{}{}
	}
	filtered := make(map[string]interface{}, len(attrs))
	for key, value := range attrs {
		if isStyleRenderAttrKey(strings.TrimSpace(key)) {
			continue
		}
		filtered[key] = value
	}
	return filtered
}
