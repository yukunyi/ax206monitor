package main

import (
	"fmt"
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

// DiskNameMonitor displays disk name/model
type DiskNameMonitor struct {
	*BaseMonitorItem
	diskIndex int
}

func NewDiskNameMonitor(diskIndex int) *DiskNameMonitor {
	return &DiskNameMonitor{
		BaseMonitorItem: NewBaseMonitorItem(
			fmt.Sprintf("disk%d_name", diskIndex),
			fmt.Sprintf("Disk %d", diskIndex),
			0, 0,
			"",
			0,
		),
		diskIndex: diskIndex,
	}
}

func (d *DiskNameMonitor) Update() error {
	initializeCache()
	if len(cachedDiskInfo) > d.diskIndex-1 && d.diskIndex > 0 {
		disk := cachedDiskInfo[d.diskIndex-1]
		d.SetValue(fmt.Sprintf("%s (%s)", disk.Name, disk.Model))
		d.SetAvailable(true)
	} else {
		d.SetAvailable(false)
	}
	return nil
}

// DiskSizeMonitor displays disk size
type DiskSizeMonitor struct {
	*BaseMonitorItem
	diskIndex int
}

func NewDiskSizeMonitor(diskIndex int) *DiskSizeMonitor {
	return &DiskSizeMonitor{
		BaseMonitorItem: NewBaseMonitorItem(
			fmt.Sprintf("disk%d_size", diskIndex),
			fmt.Sprintf("Disk %d Size", diskIndex),
			0, 0,
			"GB",
			0,
		),
		diskIndex: diskIndex,
	}
}

func (d *DiskSizeMonitor) Update() error {
	initializeCache()
	if len(cachedDiskInfo) > d.diskIndex-1 && d.diskIndex > 0 {
		disk := cachedDiskInfo[d.diskIndex-1]
		d.SetValue(disk.Size)
		d.SetAvailable(true)
	} else {
		d.SetAvailable(false)
	}
	return nil
}

// DiskTempMonitorByIndex displays disk temperature by index
type DiskTempMonitorByIndex struct {
	*BaseMonitorItem
	diskIndex int
}

func NewDiskTempMonitorByIndex(diskIndex int) *DiskTempMonitorByIndex {
	return &DiskTempMonitorByIndex{
		BaseMonitorItem: NewBaseMonitorItem(
			fmt.Sprintf("disk%d_temp", diskIndex),
			fmt.Sprintf("Disk %d Temp", diskIndex),
			0, 80,
			"°C",
			0,
		),
		diskIndex: diskIndex,
	}
}

func (d *DiskTempMonitorByIndex) Update() error {
	initializeCache()
	if len(cachedDiskInfo) > d.diskIndex-1 && d.diskIndex > 0 {
		disk := cachedDiskInfo[d.diskIndex-1]
		if disk.Temperature > 0 {
			d.SetValue(disk.Temperature)
			d.SetAvailable(true)
		} else {
			d.SetAvailable(false)
		}
	} else {
		d.SetAvailable(false)
	}
	return nil
}

// DiskReadSpeedMonitor displays disk read speed
type DiskReadSpeedMonitor struct {
	*BaseMonitorItem
}

func NewDiskReadSpeedMonitor() *DiskReadSpeedMonitor {
	return &DiskReadSpeedMonitor{
		BaseMonitorItem: NewBaseMonitorItem(
			"disk_read_speed",
			"Disk Read",
			0, 0,
			"MB/s",
			1,
		),
	}
}

func (d *DiskReadSpeedMonitor) Update() error {
	readSpeed := getDiskReadSpeed()
	if readSpeed >= 0 {
		d.SetValue(readSpeed)
		d.SetAvailable(true)
	} else {
		d.SetAvailable(false)
	}
	return nil
}

// DiskWriteSpeedMonitor displays disk write speed
type DiskWriteSpeedMonitor struct {
	*BaseMonitorItem
}

func NewDiskWriteSpeedMonitor() *DiskWriteSpeedMonitor {
	return &DiskWriteSpeedMonitor{
		BaseMonitorItem: NewBaseMonitorItem(
			"disk_write_speed",
			"Disk Write",
			0, 0,
			"MB/s",
			1,
		),
	}
}

func (d *DiskWriteSpeedMonitor) Update() error {
	writeSpeed := getDiskWriteSpeed()
	if writeSpeed >= 0 {
		d.SetValue(writeSpeed)
		d.SetAvailable(true)
	} else {
		d.SetAvailable(false)
	}
	return nil
}
