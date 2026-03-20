package main

import "testing"

func TestResolveEffectiveMinMaxUsesExplicitItemRange(t *testing.T) {
	item := &ItemConfig{
		Type:     itemTypeFullGauge,
		MinValue: float64Ptr(10),
		MaxValue: float64Ptr(80),
	}
	value := &CollectValue{
		Unit: "%",
		Min:  5,
		Max:  15,
	}

	minValue, maxValue := resolveEffectiveMinMax(item, value, []float64{20, 40, 60}, 60)
	if minValue != 10 || maxValue != 80 {
		t.Fatalf("expected explicit item range 10-80, got %.2f-%.2f", minValue, maxValue)
	}
}

func TestResolveEffectiveMinMaxUsesUnitProfiles(t *testing.T) {
	tests := []struct {
		name string
		unit string
		min  float64
		max  float64
	}{
		{name: "temperature", unit: "°C", min: 30, max: 110},
		{name: "percent", unit: "%", min: 0, max: 100},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			item := &ItemConfig{Type: itemTypeSimpleProgress}
			value := &CollectValue{
				Unit: tc.unit,
				Min:  -1,
				Max:  -1,
			}

			minValue, maxValue := resolveEffectiveMinMax(item, value, []float64{42}, 42)
			if minValue != tc.min || maxValue != tc.max {
				t.Fatalf("expected %.2f-%.2f, got %.2f-%.2f", tc.min, tc.max, minValue, maxValue)
			}
		})
	}
}

func TestResolveEffectiveMinMaxSupportsPartialExplicitOverrides(t *testing.T) {
	item := &ItemConfig{
		Type:     itemTypeFullProgressH,
		MinValue: float64Ptr(5),
	}
	value := &CollectValue{Unit: "%"}

	minValue, maxValue := resolveEffectiveMinMax(item, value, []float64{10, 20, 30}, 30)
	if minValue != 5 || maxValue != 100 {
		t.Fatalf("expected partial explicit override 5-100, got %.2f-%.2f", minValue, maxValue)
	}

	item.MinValue = nil
	item.MaxValue = float64Ptr(80)
	minValue, maxValue = resolveEffectiveMinMax(item, value, []float64{10, 20, 30}, 30)
	if minValue != 0 || maxValue != 80 {
		t.Fatalf("expected partial explicit override 0-80, got %.2f-%.2f", minValue, maxValue)
	}
}

func TestResolveEffectiveMinMaxUsesDynamicHistoryForUnknownUnits(t *testing.T) {
	item := &ItemConfig{Type: itemTypeFullChart}
	value := &CollectValue{Unit: "RPM"}

	minValue, maxValue := resolveEffectiveMinMax(item, value, []float64{12, 40}, 30)
	if minValue != 0 || maxValue != 42 {
		t.Fatalf("expected dynamic range 0-42, got %.2f-%.2f", minValue, maxValue)
	}
}

func TestResolveEffectiveMinMaxUsesCurrentValueWhenHistoryMissing(t *testing.T) {
	item := &ItemConfig{Type: itemTypeSimpleProgress}
	value := &CollectValue{Unit: "MiB/s"}

	minValue, maxValue := resolveEffectiveMinMax(item, value, nil, 20)
	if minValue != 0 || maxValue != 21 {
		t.Fatalf("expected current-value fallback range 0-21, got %.2f-%.2f", minValue, maxValue)
	}
}
