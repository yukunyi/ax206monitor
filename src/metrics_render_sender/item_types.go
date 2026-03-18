package main

import "strings"

const (
	itemTypeSimpleValue    = "simple_value"
	itemTypeSimpleProgress = "simple_progress"
	itemTypeSimpleChart    = "simple_line_chart"
	itemTypeSimpleLine     = "simple_line"
	itemTypeSimpleLabel    = "simple_label"
	itemTypeSimpleRect     = "simple_rect"
	itemTypeSimpleCircle   = "simple_circle"
	itemTypeLabelText      = "label_text"

	itemTypeFullChart     = "full_chart"
	itemTypeFullTable     = "full_table"
	itemTypeFullProgressH = "full_progress_h"
	itemTypeFullProgressV = "full_progress_v"
	itemTypeFullGauge     = "full_gauge"
)

var simpleItemTypes = []string{
	itemTypeSimpleValue,
	itemTypeSimpleProgress,
	itemTypeSimpleChart,
	itemTypeSimpleLine,
	itemTypeSimpleLabel,
	itemTypeSimpleRect,
	itemTypeSimpleCircle,
	itemTypeLabelText,
}

var fullItemTypes = []string{
	itemTypeFullChart,
	itemTypeFullTable,
	itemTypeFullProgressH,
	itemTypeFullProgressV,
	itemTypeFullGauge,
}

var allItemTypes = append(append([]string{}, simpleItemTypes...), fullItemTypes...)
var allItemTypeSet = toItemTypeSet(allItemTypes)
var fullItemTypeSet = toItemTypeSet(fullItemTypes)

var collectorBoundItemTypeSet = toItemTypeSet(append([]string{
	itemTypeSimpleValue,
	itemTypeSimpleProgress,
	itemTypeSimpleChart,
	itemTypeLabelText,
}, fullItemTypes...))

var rangeItemTypeSet = toItemTypeSet([]string{
	itemTypeSimpleProgress,
	itemTypeSimpleChart,
	itemTypeFullChart,
	itemTypeFullProgressH,
	itemTypeFullProgressV,
	itemTypeFullGauge,
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
	_, ok := fullItemTypeSet[itemType]
	return ok
}
