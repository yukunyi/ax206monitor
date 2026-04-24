package main

import "testing"

func buildAvailableFloatItem(name string, value float64) *CollectItem {
	item := NewCollectItem(name, name, "", 0, 0, 2)
	item.SetValue(value)
	item.SetAvailable(true)
	return item
}

func TestAggregateCPUMinFreqUsesLibreCoreClocks(t *testing.T) {
	items := map[string]*CollectItem{
		"libre_intelcpu_0_clock_0": buildAvailableFloatItem("libre_intelcpu_0_clock_0", 100),
		"libre_intelcpu_0_clock_1": buildAvailableFloatItem("libre_intelcpu_0_clock_1", 4200),
		"libre_intelcpu_0_clock_2": buildAvailableFloatItem("libre_intelcpu_0_clock_2", 3100),
		"libre_intelcpu_0_clock_3": buildAvailableFloatItem("libre_intelcpu_0_clock_3", 3600),
		"go_native.cpu.freq":       buildAvailableFloatItem("go_native.cpu.freq", 9999),
	}

	got := aggregateCPUMinFreq(items)
	if !got.ok || got.value != 3100 {
		t.Fatalf("expected min core clock 3100, got %#v", got)
	}
}

func TestAggregateCPUMinFreqFallsBackToAverageFreq(t *testing.T) {
	items := map[string]*CollectItem{
		"go_native.cpu.freq": buildAvailableFloatItem("go_native.cpu.freq", 2800),
	}

	got := aggregateCPUMinFreq(items)
	if !got.ok || got.value != 2800 {
		t.Fatalf("expected fallback average clock 2800, got %#v", got)
	}
}

func TestAggregateDiskRuntimeMetrics(t *testing.T) {
	items := map[string]*CollectItem{
		"go_native.disk.1.read":  buildAvailableFloatItem("go_native.disk.1.read", 12.5),
		"go_native.disk.2.read":  buildAvailableFloatItem("go_native.disk.2.read", 7.5),
		"go_native.disk.1.write": buildAvailableFloatItem("go_native.disk.1.write", 4),
		"go_native.disk.2.write": buildAvailableFloatItem("go_native.disk.2.write", 6),
		"go_native.disk.1.busy":  buildAvailableFloatItem("go_native.disk.1.busy", 15),
		"go_native.disk.2.busy":  buildAvailableFloatItem("go_native.disk.2.busy", 45),
	}

	totalRead, totalWrite, maxBusy := aggregateDiskRuntimeMetrics(items)
	if !totalRead.ok || totalRead.value != 20 {
		t.Fatalf("expected total read 20, got %#v", totalRead)
	}
	if !totalWrite.ok || totalWrite.value != 10 {
		t.Fatalf("expected total write 10, got %#v", totalWrite)
	}
	if !maxBusy.ok || maxBusy.value != 45 {
		t.Fatalf("expected max busy 45, got %#v", maxBusy)
	}
}
