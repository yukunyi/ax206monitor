package main

import (
	"github.com/shirou/gopsutil/v3/mem"
)

type MemoryUsageMonitor struct {
	*BaseMonitorItem
}

func NewMemoryUsageMonitor() *MemoryUsageMonitor {
	return &MemoryUsageMonitor{
		BaseMonitorItem: NewBaseMonitorItem(
			"memory_usage",
			"Memory",
			0, 100,
			"%",
			0,
		),
	}
}

func (m *MemoryUsageMonitor) Update() error {
	memInfo, err := mem.VirtualMemory()
	if err != nil {
		m.SetAvailable(false)
		return err
	}

	m.SetValue(memInfo.UsedPercent)
	m.SetAvailable(true)
	return nil
}

type MemoryUsedMonitor struct {
	*BaseMonitorItem
}

func NewMemoryUsedMonitor() *MemoryUsedMonitor {
	return &MemoryUsedMonitor{
		BaseMonitorItem: NewBaseMonitorItem(
			"memory_used",
			"Mem Used",
			0, 0,
			"GB",
			1,
		),
	}
}

func (m *MemoryUsedMonitor) Update() error {
	memInfo, err := mem.VirtualMemory()
	if err != nil {
		m.SetAvailable(false)
		return err
	}

	usedGB := float64(memInfo.Used) / (1024 * 1024 * 1024)
	m.SetValue(usedGB)
	m.SetAvailable(true)
	return nil
}

type MemoryTotalMonitor struct {
	*BaseMonitorItem
}

func NewMemoryTotalMonitor() *MemoryTotalMonitor {
	return &MemoryTotalMonitor{
		BaseMonitorItem: NewBaseMonitorItem(
			"memory_total",
			"Mem Total",
			0, 0,
			"GB",
			1,
		),
	}
}

func (m *MemoryTotalMonitor) Update() error {
	memInfo, err := mem.VirtualMemory()
	if err != nil {
		m.SetAvailable(false)
		return err
	}

	totalGB := float64(memInfo.Total) / (1024 * 1024 * 1024)
	m.SetValue(totalGB)
	m.SetAvailable(true)
	return nil
}
