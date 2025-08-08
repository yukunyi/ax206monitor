package main

import (
	"fmt"

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
	// Get total memory for max value
	var maxMemory float64 = 32.0 // Default fallback
	if memInfo, err := mem.VirtualMemory(); err == nil {
		maxMemory = float64(memInfo.Total) / (1024 * 1024 * 1024)
	}

	return &MemoryUsedMonitor{
		BaseMonitorItem: NewBaseMonitorItem(
			"memory_used",
			"Mem Used",
			0, maxMemory,
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
	// Get total memory for max value
	var maxMemory float64 = 32.0 // Default fallback
	if memInfo, err := mem.VirtualMemory(); err == nil {
		maxMemory = float64(memInfo.Total) / (1024 * 1024 * 1024)
	}

	return &MemoryTotalMonitor{
		BaseMonitorItem: NewBaseMonitorItem(
			"memory_total",
			"Mem Total",
			0, maxMemory,
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

type MemoryUsageTextMonitor struct {
	*BaseMonitorItem
}

func NewMemoryUsageTextMonitor() *MemoryUsageTextMonitor {
	return &MemoryUsageTextMonitor{
		BaseMonitorItem: NewBaseMonitorItem(
			"memory_usage_text",
			"Memory",
			0, 0,
			"",
			0,
		),
	}
}

func (m *MemoryUsageTextMonitor) Update() error {
	memInfo, err := mem.VirtualMemory()
	if err != nil {
		m.SetAvailable(false)
		return err
	}

	usedGB := float64(memInfo.Used) / (1024 * 1024 * 1024)
	totalGB := float64(memInfo.Total) / (1024 * 1024 * 1024)
	usagePercent := memInfo.UsedPercent

	// Format as "Used/Total (Percent%)"
	text := fmt.Sprintf("%.1f/%.1f GB (%.0f%%)", usedGB, totalGB, usagePercent)
	m.SetValue(text)
	m.SetAvailable(true)
	return nil
}

type MemoryUsageProgressMonitor struct {
	*BaseMonitorItem
}

func NewMemoryUsageProgressMonitor() *MemoryUsageProgressMonitor {
	return &MemoryUsageProgressMonitor{
		BaseMonitorItem: NewBaseMonitorItem(
			"memory_usage_progress",
			"Memory",
			0, 100,
			"%",
			0,
		),
	}
}

func (m *MemoryUsageProgressMonitor) Update() error {
	memInfo, err := mem.VirtualMemory()
	if err != nil {
		m.SetAvailable(false)
		return err
	}

	m.SetValue(memInfo.UsedPercent)
	m.SetAvailable(true)
	return nil
}
