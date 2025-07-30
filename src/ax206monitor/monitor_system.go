package main

import (
	"time"

	"github.com/shirou/gopsutil/v3/disk"
	"github.com/shirou/gopsutil/v3/load"
)

type DiskTempMonitor struct {
	*BaseMonitorItem
}

func NewDiskTempMonitor() *DiskTempMonitor {
	return &DiskTempMonitor{
		BaseMonitorItem: NewBaseMonitorItem(
			"disk_temp",
			"Disk Temp",
			0, 80,
			"°C",
			0,
		),
	}
}

func (d *DiskTempMonitor) Update() error {
	temp := getDiskTemperature()
	if temp > 0 {
		d.SetValue(temp)
		d.SetAvailable(true)
	} else {
		d.SetAvailable(false)
	}

	return nil
}

type LoadAvgMonitor struct {
	*BaseMonitorItem
}

func NewLoadAvgMonitor() *LoadAvgMonitor {
	return &LoadAvgMonitor{
		BaseMonitorItem: NewBaseMonitorItem(
			"load_avg",
			"Load Avg",
			0, 0,
			"",
			1,
		),
	}
}

func (l *LoadAvgMonitor) Update() error {
	loadInfo, err := load.Avg()
	if err != nil {
		l.SetAvailable(false)
		return err
	}

	l.SetValue(loadInfo.Load1)
	l.SetAvailable(true)
	return nil
}

type CurrentTimeMonitor struct {
	*BaseMonitorItem
}

func NewCurrentTimeMonitor() *CurrentTimeMonitor {
	return &CurrentTimeMonitor{
		BaseMonitorItem: NewBaseMonitorItem(
			"current_time",
			"Time",
			0, 0,
			"",
			0,
		),
	}
}

func (c *CurrentTimeMonitor) Update() error {
	now := time.Now()
	c.SetValue(now.Format("2006-01-02 15:04:05"))
	c.SetAvailable(true)
	return nil
}

type DiskUsageMonitor struct {
	*BaseMonitorItem
}

func NewDiskUsageMonitor() *DiskUsageMonitor {
	return &DiskUsageMonitor{
		BaseMonitorItem: NewBaseMonitorItem(
			"disk_usage",
			"Disk Usage",
			0, 100,
			"%",
			1,
		),
	}
}

func (d *DiskUsageMonitor) Update() error {
	usage, err := disk.Usage("/")
	if err != nil {
		d.SetAvailable(false)
		return err
	}

	d.SetValue(usage.UsedPercent)
	d.SetAvailable(true)
	return nil
}
