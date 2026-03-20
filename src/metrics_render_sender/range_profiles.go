package main

import (
	"math"
	"strings"
)

const (
	rangeDynamicPaddingRatio = 0.05
	rangeTemperatureMin      = 30.0
	rangeTemperatureMax      = 110.0
	rangePercentMin          = 0.0
	rangePercentMax          = 100.0
)

type rangeProfile struct {
	Name string
	Min  float64
	Max  float64
}

var (
	temperatureRangeProfile = rangeProfile{
		Name: "temperature_30_110",
		Min:  rangeTemperatureMin,
		Max:  rangeTemperatureMax,
	}
	percentRangeProfile = rangeProfile{
		Name: "percent_0_100",
		Min:  rangePercentMin,
		Max:  rangePercentMax,
	}
)

func normalizeRangeUnitToken(unit string) string {
	return strings.ToLower(strings.Join(strings.Fields(strings.TrimSpace(unit)), ""))
}

func inferRangeProfileForUnit(unit string) (rangeProfile, bool) {
	switch normalizeRangeUnitToken(unit) {
	case "°c", "℃", "celsius":
		return temperatureRangeProfile, true
	case "%", "percent", "percentage", "pct":
		return percentRangeProfile, true
	default:
		return rangeProfile{}, false
	}
}

func resolveEffectiveRangeUnit(item *ItemConfig, value *CollectValue) string {
	if item != nil {
		unit := strings.TrimSpace(item.Unit)
		if unit != "" && !strings.EqualFold(unit, "auto") {
			return unit
		}
	}
	if value == nil {
		return ""
	}
	return strings.TrimSpace(value.Unit)
}

func resolveConfiguredRange(item *ItemConfig) (float64, float64, bool, bool) {
	if item == nil {
		return 0, 0, false, false
	}

	var minValue float64
	hasMin := item.MinValue != nil && !math.IsNaN(*item.MinValue) && !math.IsInf(*item.MinValue, 0)
	if hasMin {
		minValue = *item.MinValue
	}

	var maxValue float64
	hasMax := item.MaxValue != nil && !math.IsNaN(*item.MaxValue) && !math.IsInf(*item.MaxValue, 0)
	if hasMax {
		maxValue = *item.MaxValue
	}

	return minValue, maxValue, hasMin, hasMax
}

func historyMaxValue(values []float64) (float64, bool) {
	if len(values) == 0 {
		return 0, false
	}
	result := 0.0
	valid := false
	for _, value := range values {
		if !isFiniteHistoryValue(value) {
			continue
		}
		if !valid || value > result {
			result = value
			valid = true
		}
	}
	return result, valid
}

func resolveAutoRangeBounds(item *ItemConfig, value *CollectValue, history []float64, currentValue float64) (float64, float64) {
	if profile, ok := inferRangeProfileForUnit(resolveEffectiveRangeUnit(item, value)); ok {
		return profile.Min, profile.Max
	}

	baseMax, ok := historyMaxValue(history)
	if !ok && isFiniteHistoryValue(currentValue) {
		baseMax = currentValue
		ok = true
	}
	if !ok {
		return 0, 1
	}

	maxValue := baseMax * (1 + rangeDynamicPaddingRatio)
	if !isFiniteHistoryValue(maxValue) || maxValue <= 0 {
		maxValue = math.Abs(baseMax) * (1 + rangeDynamicPaddingRatio)
	}
	if !isFiniteHistoryValue(maxValue) || maxValue <= 0 {
		maxValue = 1
	}
	return 0, maxValue
}

func resolveEffectiveMinMax(item *ItemConfig, value *CollectValue, history []float64, currentValue float64) (float64, float64) {
	minValue, maxValue := resolveAutoRangeBounds(item, value, history, currentValue)

	configuredMin, configuredMax, hasMin, hasMax := resolveConfiguredRange(item)
	if hasMin {
		minValue = configuredMin
	}
	if hasMax {
		maxValue = configuredMax
	}
	if maxValue <= minValue {
		maxValue = minValue + 1
	}
	return minValue, maxValue
}
