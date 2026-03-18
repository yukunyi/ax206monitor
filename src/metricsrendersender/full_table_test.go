package main

import (
	"reflect"
	"sort"
	"testing"
)

func TestResolveFullTableRowConfigs(t *testing.T) {
	item := &ItemConfig{
		Type: itemTypeFullTable,
		RenderAttrsMap: map[string]interface{}{
			"rows": []interface{}{
				map[string]interface{}{"monitor": " go_native.cpu.temp ", "label": "CPU"},
				map[string]interface{}{"monitor": ""},
				map[string]interface{}{"monitor": "go_native.gpu.temp"},
			},
		},
	}

	rows := resolveFullTableRowConfigs(item)
	if len(rows) != 3 {
		t.Fatalf("expected 3 rows, got %d", len(rows))
	}
	if rows[0].Monitor != "go_native.cpu.temp" || rows[0].Label != "CPU" {
		t.Fatalf("unexpected first row: %#v", rows[0])
	}
	if rows[1].Monitor != "" || rows[1].Label != "" {
		t.Fatalf("unexpected second row: %#v", rows[1])
	}
	if rows[2].Monitor != "go_native.gpu.temp" || rows[2].Label != "" {
		t.Fatalf("unexpected third row: %#v", rows[2])
	}
}

func TestGetRequiredMonitorsIncludesFullTableRows(t *testing.T) {
	config := &MonitorConfig{
		Items: []ItemConfig{
			{
				Type: itemTypeFullTable,
				RenderAttrsMap: map[string]interface{}{
					"rows": []interface{}{
						map[string]interface{}{"monitor": "custom.combo"},
						map[string]interface{}{"monitor": "go_native.gpu.temp"},
						map[string]interface{}{"monitor": "custom.combo"},
					},
				},
			},
		},
		CustomMonitors: []CustomMonitorConfig{
			{
				Name:    "custom.combo",
				Type:    "mixed",
				Sources: []string{"go_native.cpu.temp", "go_native.memory.used"},
			},
		},
	}

	required := getRequiredMonitors(config)
	sort.Strings(required)

	expected := []string{"custom.combo", "go_native.cpu.temp", "go_native.gpu.temp", "go_native.memory.used"}
	if !reflect.DeepEqual(required, expected) {
		t.Fatalf("unexpected required monitors: got=%v want=%v", required, expected)
	}
}

func TestNormalizeFullTableItemAttrs(t *testing.T) {
	item := &ItemConfig{
		Type: itemTypeFullTable,
		RenderAttrsMap: map[string]interface{}{
			"col_count": 3,
			"row_count": 2,
			"rows": []interface{}{
				map[string]interface{}{"monitor": " go_native.cpu.temp ", "label": " CPU "},
				map[string]interface{}{"monitor": ""},
			},
		},
	}

	normalizeFullTableItemAttrs(item)

	rows, ok := item.RenderAttrsMap["rows"].([]map[string]interface{})
	if !ok {
		t.Fatalf("expected normalized rows slice, got %#v", item.RenderAttrsMap["rows"])
	}
	if len(rows) != 2 {
		t.Fatalf("expected 2 normalized rows, got %d", len(rows))
	}
	if rows[0]["monitor"] != "go_native.cpu.temp" || rows[0]["label"] != "CPU" {
		t.Fatalf("unexpected normalized row: %#v", rows[0])
	}
	if _, ok := rows[1]["monitor"]; ok {
		t.Fatalf("expected second slot monitor to be empty, got %#v", rows[1])
	}
	if item.RenderAttrsMap["col_count"] != 3 {
		t.Fatalf("expected normalized col_count=3, got %#v", item.RenderAttrsMap["col_count"])
	}
	if item.RenderAttrsMap["row_count"] != 2 {
		t.Fatalf("expected normalized row_count=2, got %#v", item.RenderAttrsMap["row_count"])
	}
}
