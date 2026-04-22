package main

import "testing"

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
		readBytes:   100 * 1024 * 1024,
		writeBytes:  80 * 1024 * 1024,
		readCount:   150,
		writeCount:  110,
		readTimeMS:  500,
		writeTimeMS: 220,
		busyTimeMS:  600,
		queueDepth:  1,
	}

	got := computeDiskMetricsSnapshot(current, previous, 2, 2000)
	if !almostEqualFloat64(got.Read, 100) {
		t.Fatalf("expected read speed 100 MiB/s, got %v", got.Read)
	}
	if !almostEqualFloat64(got.Write, 50) {
		t.Fatalf("expected write speed 50 MiB/s, got %v", got.Write)
	}
	if !almostEqualFloat64(got.ReadIOPS, 100) {
		t.Fatalf("expected read IOPS 100, got %v", got.ReadIOPS)
	}
	if !almostEqualFloat64(got.WriteIOPS, 50) {
		t.Fatalf("expected write IOPS 50, got %v", got.WriteIOPS)
	}
	if !almostEqualFloat64(got.ReadLatencyMS, 2) {
		t.Fatalf("expected read latency 2ms, got %v", got.ReadLatencyMS)
	}
	if !almostEqualFloat64(got.WriteLatencyMS, 2) {
		t.Fatalf("expected write latency 2ms, got %v", got.WriteLatencyMS)
	}
	if !almostEqualFloat64(got.BusyPercent, 30) {
		t.Fatalf("expected busy percent 30, got %v", got.BusyPercent)
	}
	if !almostEqualFloat64(got.QueueDepth, 2) {
		t.Fatalf("expected queue depth 2, got %v", got.QueueDepth)
	}
}
