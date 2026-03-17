package main

import (
	"testing"
	"time"
)

func TestBuiltinRateWindowUsesDefaultSlidingWindowForThroughputMetrics(t *testing.T) {
	testCases := []struct {
		name string
		unit string
	}{
		{name: "go_native.disk.1.read", unit: "MiB/s"},
		{name: "go_native.disk.1.write", unit: "MiB/s"},
		{name: "go_native.net.1.upload", unit: " MiB/s"},
		{name: "go_native.net.1.download", unit: " MiB/s"},
	}

	for _, tc := range testCases {
		if got := builtinRateWindow(tc.name, tc.unit); got != defaultThroughputRateWindow {
			t.Fatalf("builtinRateWindow(%q, %q) = %v, want %v", tc.name, tc.unit, got, defaultThroughputRateWindow)
		}
	}
}

func TestBuiltinRateWindowDisablesSlidingWindowForNonThroughputMetrics(t *testing.T) {
	if got := builtinRateWindow("go_native.cpu.temp", "°C"); got != 0 {
		t.Fatalf("expected no sliding window for non-throughput metric, got %v", got)
	}
	if got := builtinRateWindow("custom.disk.read", "MiB/s"); got != 0 {
		t.Fatalf("expected no built-in sliding window for non-go_native metric, got %v", got)
	}
}

func TestBaseCollectItemAveragesValuesWithinSlidingWindow(t *testing.T) {
	base := NewBaseCollectItem("go_native.disk.1.read", "Disk read", 0, 0, "MiB/s", 2)
	now := time.Now()
	base.rateWindow = 3 * time.Second
	base.rateSamples = []rateSample{
		{at: now.Add(-2500 * time.Millisecond), value: 3},
		{at: now.Add(-1500 * time.Millisecond), value: 6},
	}

	base.SetValue(9.0)
	value := base.GetValue()
	if value == nil {
		t.Fatal("expected value")
	}
	got, ok := value.Value.(float64)
	if !ok {
		t.Fatalf("expected float64 value, got %T", value.Value)
	}
	if got != 6 {
		t.Fatalf("expected 3s sliding average 6, got %v", got)
	}
}
