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
			"Memory usage",
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
			"Memory used",
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
			"Memory total",
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
			"Memory usage detail",
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
			"Memory usage progress",
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

type SwapUsageMonitor struct {
	*BaseMonitorItem
}

func NewSwapUsageMonitor() *SwapUsageMonitor {
	return &SwapUsageMonitor{
		BaseMonitorItem: NewBaseMonitorItem(
			"swap_usage",
			"Swap usage",
			0, 100,
			"%",
			0,
		),
	}
}

func (s *SwapUsageMonitor) Update() error {
	swapInfo, err := mem.SwapMemory()
	if err != nil {
		s.SetAvailable(false)
		return err
	}

	s.SetValue(swapInfo.UsedPercent)
	s.SetAvailable(true)
	return nil
}
