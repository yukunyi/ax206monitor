package main

import "testing"

func TestNormalizeAggregateMethodSupportsSum(t *testing.T) {
	if got := normalizeAggregateMethod("sum"); got != "sum" {
		t.Fatalf("expected sum, got %q", got)
	}
}

func TestCustomCollectorMixedSum(t *testing.T) {
	collector := NewCustomCollector(nil, func(name string) *CollectItem {
		switch name {
		case "a":
			item := NewCollectItem("a", "A", "MB/s", 0, 0, 2)
			item.SetValue(1.5)
			item.SetAvailable(true)
			return item
		case "b":
			item := NewCollectItem("b", "B", "MB/s", 0, 0, 2)
			item.SetValue(2.5)
			item.SetAvailable(true)
			return item
		default:
			return nil
		}
	})
	collector.cfg = &MonitorConfig{
		CustomMonitors: []CustomMonitorConfig{
			{
				Name:      "sum.disk.read",
				Label:     "sum",
				Type:      "mixed",
				Sources:   []string{"a", "b"},
				Aggregate: "sum",
				Unit:      "MB/s",
			},
		},
	}
	collector.rebuildItemsLocked()

	if err := collector.UpdateItems(); err != nil {
		t.Fatalf("UpdateItems failed: %v", err)
	}
	item := collector.getItem("sum.disk.read")
	if item == nil {
		t.Fatalf("expected custom item")
	}
	value := item.GetValue()
	if value == nil {
		t.Fatalf("expected custom value")
	}
	got, ok := value.Value.(float64)
	if !ok {
		t.Fatalf("expected float64 value, got %T", value.Value)
	}
	if got != 4.0 {
		t.Fatalf("expected sum 4.0, got %v", got)
	}
}
