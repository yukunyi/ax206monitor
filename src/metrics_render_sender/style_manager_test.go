package main

import "testing"

func TestStyleCodeDefaultFullChartFillColorIsTransparent(t *testing.T) {
	value, ok := styleCodeDefault(itemTypeFullChart, "chart_fill_color")
	if !ok {
		t.Fatalf("expected chart_fill_color default to exist")
	}
	if value != "rgba(0,0,0,0)" {
		t.Fatalf("expected transparent default, got %v", value)
	}
}

func TestWebStyleKeyMetaIncludesChartFillColor(t *testing.T) {
	list := WebStyleKeyMeta()
	for _, meta := range list {
		if meta.Key != "chart_fill_color" {
			continue
		}
		if meta.Label != "折线区域颜色" {
			t.Fatalf("unexpected label: %s", meta.Label)
		}
		if len(meta.Types) != 1 || meta.Types[0] != itemTypeFullChart {
			t.Fatalf("unexpected types: %#v", meta.Types)
		}
		if meta.Default != "rgba(0,0,0,0)" {
			t.Fatalf("unexpected default: %#v", meta.Default)
		}
		return
	}
	t.Fatalf("chart_fill_color meta not found")
}
