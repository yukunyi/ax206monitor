package main

import "strings"

const (
	itemTypeSimpleValue    = "simple_value"
	itemTypeSimpleProgress = "simple_progress"
	itemTypeSimpleChart    = "simple_line_chart"
	itemTypeSimpleLabel    = "simple_label"
	itemTypeSimpleRect     = "simple_rect"
	itemTypeSimpleCircle   = "simple_circle"
	itemTypeLabelText1     = "label_text1"
	itemTypeLabelText2     = "label_text2"

	itemTypeFullChart     = "full_chart"
	itemTypeFullProgress  = "full_progress"
	itemTypeFullGauge     = "full_gauge"
	itemTypeFullRing      = "full_ring"
	itemTypeFullMinMax    = "full_minmax"
	itemTypeFullDelta     = "full_delta"
	itemTypeFullStatus    = "full_status"
	itemTypeFullMeterH    = "full_meter_h"
	itemTypeFullMeterV    = "full_meter_v"
	itemTypeFullHeatStrip = "full_heat_strip"
)

var simpleItemTypes = []string{
	itemTypeSimpleValue,
	itemTypeSimpleProgress,
	itemTypeSimpleChart,
	itemTypeSimpleLabel,
	itemTypeSimpleRect,
	itemTypeSimpleCircle,
	itemTypeLabelText1,
	itemTypeLabelText2,
}

var fullItemTypes = []string{
	itemTypeFullChart,
	itemTypeFullProgress,
}

var allItemTypes = append(append([]string{}, simpleItemTypes...), fullItemTypes...)
var allItemTypeSet = toItemTypeSet(allItemTypes)

var legacyItemTypeAliases = map[string]string{
	"value":        itemTypeSimpleValue,
	"progress":     itemTypeSimpleProgress,
	"line_chart":   itemTypeSimpleChart,
	"label":        itemTypeSimpleLabel,
	"rect":         itemTypeSimpleRect,
	"circle":       itemTypeSimpleCircle,
	"linechart":    itemTypeSimpleChart,
	"simple_value": itemTypeSimpleValue,
	"simple_label": itemTypeSimpleLabel,
	"labeltext1":   itemTypeLabelText1,
	"labeltext2":   itemTypeLabelText2,
}

var collectorBoundItemTypeSet = toItemTypeSet(append([]string{
	itemTypeSimpleValue,
	itemTypeSimpleProgress,
	itemTypeSimpleChart,
	itemTypeLabelText1,
	itemTypeLabelText2,
}, fullItemTypes...))

var rangeItemTypeSet = toItemTypeSet([]string{
	itemTypeSimpleProgress,
	itemTypeSimpleChart,
	itemTypeFullChart,
	itemTypeFullProgress,
})

var historyItemTypeSet = toItemTypeSet([]string{
	itemTypeSimpleChart,
	itemTypeFullChart,
})

var shapeItemTypeSet = toItemTypeSet([]string{
	itemTypeSimpleRect,
	itemTypeSimpleCircle,
})

func toItemTypeSet(types []string) map[string]struct{} {
	set := make(map[string]struct{}, len(types))
	for _, itemType := range types {
		set[itemType] = struct{}{}
	}
	return set
}

func webItemTypes() []string {
	return append([]string{}, allItemTypes...)
}

func normalizeItemTypeName(itemType string) string {
	trimmed := strings.ToLower(strings.TrimSpace(itemType))
	if trimmed == "" {
		return itemTypeSimpleValue
	}
	if mapped, ok := legacyItemTypeAliases[trimmed]; ok {
		return mapped
	}
	if _, ok := allItemTypeSet[trimmed]; ok {
		return trimmed
	}
	return itemTypeSimpleValue
}

func isCollectorItemType(itemType string) bool {
	_, ok := collectorBoundItemTypeSet[itemType]
	return ok
}

func isRangeItemType(itemType string) bool {
	_, ok := rangeItemTypeSet[itemType]
	return ok
}

func isHistoryItemType(itemType string) bool {
	_, ok := historyItemTypeSet[itemType]
	return ok
}

func isShapeItemType(itemType string) bool {
	_, ok := shapeItemTypeSet[itemType]
	return ok
}

func isFullItemType(itemType string) bool {
	return strings.HasPrefix(itemType, "full_")
}
