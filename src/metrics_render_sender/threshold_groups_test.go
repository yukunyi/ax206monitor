package main

import "testing"

func TestResolveThresholdRangesColorClampsToLastRange(t *testing.T) {
	min50 := 50.0
	min75 := 75.0
	max50 := 50.0
	max75 := 75.0
	max100 := 100.0

	ranges := []ThresholdRangeConfig{
		{Max: &max50, Color: "#low"},
		{Min: &min50, Max: &max75, Color: "#mid"},
		{Min: &min75, Max: &max100, Color: "#high"},
	}

	if color := resolveThresholdRangesColor(ranges, 120); color != "#high" {
		t.Fatalf("expected last range color for overflow value, got %q", color)
	}
}
