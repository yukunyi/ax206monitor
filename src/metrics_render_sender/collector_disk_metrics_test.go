package main

import (
	"testing"
	"time"
)

func TestComputeDiskMetricsSnapshot(t *testing.T) {
	current := diskCounterSample{
		ReadBytes:   300 * 1024 * 1024,
		WriteBytes:  180 * 1024 * 1024,
		ReadCount:   350,
		WriteCount:  210,
		ReadTimeMS:  900,
		WriteTimeMS: 420,
		BusyTimeMS:  1200,
		QueueDepth:  2,
	}
	previous := diskRateSnapshot{
		ReadBytes:   100 * 1024 * 1024,
		WriteBytes:  80 * 1024 * 1024,
		ReadCount:   150,
		WriteCount:  110,
		ReadTimeMS:  500,
		WriteTimeMS: 220,
		BusyTimeMS:  600,
		QueueDepth:  1,
	}
	previous.at = time.Unix(10, 0)
	current.at = previous.at.Add(2 * time.Second)

	got, ok := computeDiskMetrics(current, previous)
	if !ok || got == nil {
		t.Fatalf("expected computed metrics, got %#v ok=%v", got, ok)
	}
	if !almostEqualFloat64(got.read, 100) {
		t.Fatalf("expected read speed 100 MiB/s, got %v", got.read)
	}
	if !almostEqualFloat64(got.write, 50) {
		t.Fatalf("expected write speed 50 MiB/s, got %v", got.write)
	}
	if !almostEqualFloat64(got.readIOPS, 100) {
		t.Fatalf("expected read IOPS 100, got %v", got.readIOPS)
	}
	if !almostEqualFloat64(got.writeIOPS, 50) {
		t.Fatalf("expected write IOPS 50, got %v", got.writeIOPS)
	}
	if !almostEqualFloat64(got.readLatencyMS, 2) {
		t.Fatalf("expected read latency 2ms, got %v", got.readLatencyMS)
	}
	if !almostEqualFloat64(got.writeLatencyMS, 2) {
		t.Fatalf("expected write latency 2ms, got %v", got.writeLatencyMS)
	}
	if !almostEqualFloat64(got.busyPercent, 30) {
		t.Fatalf("expected busy percent 30, got %v", got.busyPercent)
	}
	if !almostEqualFloat64(got.queueDepth, 2) {
		t.Fatalf("expected queue depth 2, got %v", got.queueDepth)
	}
}
