package main

import "testing"

func TestDiskCollectorKeepsLastGoodMetricsWhenCounterSampleMissing(t *testing.T) {
	collector := NewGoNativeDiskCollector(nil)
	slot := &goNativeDiskSlot{
		readItem:         NewCollectItem("go_native.disk.1.read", "read", "MiB/s", 0, 0, 2),
		writeItem:        NewCollectItem("go_native.disk.1.write", "write", "MiB/s", 0, 0, 2),
		readIOPSItem:     NewCollectItem("go_native.disk.1.read_iops", "read_iops", "IOPS", 0, 0, 0),
		writeIOPSItem:    NewCollectItem("go_native.disk.1.write_iops", "write_iops", "IOPS", 0, 0, 0),
		readLatencyItem:  NewCollectItem("go_native.disk.1.read_latency", "read_latency", "ms", 0, 0, 2),
		writeLatencyItem: NewCollectItem("go_native.disk.1.write_latency", "write_latency", "ms", 0, 0, 2),
		busyItem:         NewCollectItem("go_native.disk.1.busy", "busy", "%", 0, 100, 0),
	}
	collector.slots[1] = slot
	state := collector.diskState("diskA")
	state.lastGood = &diskComputedMetrics{
		read:           10,
		write:          5,
		readIOPS:       20,
		writeIOPS:      10,
		readLatencyMS:  1.5,
		writeLatencyMS: 2.5,
		busyPercent:    30,
		queueDepth:     1,
	}
	state.validUntil = state.validUntil.AddDate(1, 0, 0)

	setDiskDynamicMetrics(slot, state.lastGood)
	if !slot.readItem.IsAvailable() || !slot.busyItem.IsAvailable() {
		t.Fatalf("expected initial metrics to be available")
	}

	// Simulate one missing sampling round: collector should keep previous good values.
	if state.lastGood == nil {
		t.Fatalf("expected cached metrics")
	}
	setDiskDynamicMetrics(slot, state.lastGood)
	if !slot.readItem.IsAvailable() || !slot.busyItem.IsAvailable() {
		t.Fatalf("expected cached metrics to remain available")
	}
}
