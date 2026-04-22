package main

import (
	"math"
	"testing"

	"github.com/shirou/gopsutil/v3/cpu"
)

func almostEqualFloat64(a, b float64) bool {
	return math.Abs(a-b) < 0.001
}

func TestComputeCPUUsageBreakdown(t *testing.T) {
	sample := cpu.TimesStat{
		User:    25,
		System:  18,
		Idle:    140,
		Iowait:  9,
		Irq:     4,
		Softirq: 2,
	}

	got := computeCPUUsageBreakdown(sample, 68, 110, 10, 8, 4, 1, 1)
	if !almostEqualFloat64(got.Usage, 42.6470588235) {
		t.Fatalf("expected usage ~= 42.647, got %v", got.Usage)
	}
	if !almostEqualFloat64(got.User, 22.0588235294) {
		t.Fatalf("expected user ~= 22.059, got %v", got.User)
	}
	if !almostEqualFloat64(got.System, 14.7058823529) {
		t.Fatalf("expected system ~= 14.706, got %v", got.System)
	}
	if !almostEqualFloat64(got.Idle, 50) {
		t.Fatalf("expected idle 50, got %v", got.Idle)
	}
	if !almostEqualFloat64(got.Iowait, 7.3529411765) {
		t.Fatalf("expected iowait ~= 7.353, got %v", got.Iowait)
	}
	if !almostEqualFloat64(got.Irq, 4.4117647059) {
		t.Fatalf("expected irq ~= 4.412, got %v", got.Irq)
	}
	if !almostEqualFloat64(got.Softirq, 1.4705882353) {
		t.Fatalf("expected softirq ~= 1.471, got %v", got.Softirq)
	}
}
