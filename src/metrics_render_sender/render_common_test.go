package main

import "testing"

func float64Ptr(value float64) *float64 {
	v := value
	return &v
}

func TestResolveMonitorValueColorPrecedence(t *testing.T) {
	config := &MonitorConfig{
		AllowCustomStyle: true,
		StyleBase: map[string]interface{}{
			"color": "#base",
		},
		TypeDefaults: map[string]ItemTypeDefaults{
			itemTypeSimpleValue: {
				Style: map[string]interface{}{
					"color": "#type",
				},
			},
		},
		ThresholdGroups: []ThresholdGroupConfig{
			{
				Name:     "cpu_temp",
				Monitors: []string{"cpu.temp"},
				Ranges: []ThresholdRangeConfig{
					{Min: float64Ptr(70), Max: float64Ptr(100), Color: "#threshold"},
				},
			},
		},
	}
	item := &ItemConfig{
		Type:    itemTypeSimpleValue,
		Monitor: "cpu.temp",
	}
	value := &CollectValue{Value: 80.0}

	color := resolveMonitorValueColor(item, item.Monitor, value, 80, config)
	if color != "#type" {
		t.Fatalf("expected type override color, got %q", color)
	}

	item.CustomStyle = true
	item.Style = map[string]interface{}{
		"color": "#item",
	}
	color = resolveMonitorValueColor(item, item.Monitor, value, 80, config)
	if color != "#item" {
		t.Fatalf("expected item override color, got %q", color)
	}
}

func TestResolveMonitorValueColorUsesThresholdBeforeBaseDefault(t *testing.T) {
	config := &MonitorConfig{
		StyleBase: map[string]interface{}{
			"color": "#base",
		},
		ThresholdGroups: []ThresholdGroupConfig{
			{
				Name:     "cpu_temp",
				Monitors: []string{"cpu.temp"},
				Ranges: []ThresholdRangeConfig{
					{Min: float64Ptr(70), Max: float64Ptr(100), Color: "#threshold"},
				},
			},
		},
	}
	item := &ItemConfig{
		Type:    itemTypeSimpleValue,
		Monitor: "cpu.temp",
	}
	value := &CollectValue{Value: 80.0}

	color := resolveMonitorValueColor(item, item.Monitor, value, 80, config)
	if color != "#threshold" {
		t.Fatalf("expected threshold color, got %q", color)
	}
}

func TestResolveMonitorValueColorFallsBackToSystemDefault(t *testing.T) {
	config := &MonitorConfig{
		StyleBase: map[string]interface{}{
			"color": "#base",
		},
	}
	item := &ItemConfig{
		Type:    itemTypeSimpleValue,
		Monitor: "cpu.temp",
	}
	value := &CollectValue{Value: 80.0}

	color := resolveMonitorValueColor(item, item.Monitor, value, 80, config)
	if color != "#base" {
		t.Fatalf("expected system default color, got %q", color)
	}
}
