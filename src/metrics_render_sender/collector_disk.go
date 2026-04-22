package main

import (
	"fmt"
	"strconv"
	"strings"
)

func detectNamedDiskCount() int {
	count := 0
	for _, disk := range detectDiskInfo() {
		if disk == nil {
			continue
		}
		if strings.TrimSpace(disk.Name) == "" {
			continue
		}
		count++
	}
	return count
}

type goNativeDiskSlot struct {
	nameItem         *CollectItem
	sizeItem         *CollectItem
	usedItem         *CollectItem
	availableItem    *CollectItem
	usageItem        *CollectItem
	busyItem         *CollectItem
	readItem         *CollectItem
	writeItem        *CollectItem
	readIOPSItem     *CollectItem
	writeIOPSItem    *CollectItem
	readLatencyItem  *CollectItem
	writeLatencyItem *CollectItem
}

type GoNativeDiskCollector struct {
	*BaseCollector
	requiredProvider func() []string
	slots            map[int]*goNativeDiskSlot
}

func NewGoNativeDiskCollector(requiredProvider func() []string) *GoNativeDiskCollector {
	return &GoNativeDiskCollector{
		BaseCollector:    NewBaseCollector("go_native.disk"),
		requiredProvider: requiredProvider,
		slots:            make(map[int]*goNativeDiskSlot),
	}
}

func (c *GoNativeDiskCollector) requiredMaxIndex() int {
	required := []string{}
	if c.requiredProvider != nil {
		required = c.requiredProvider()
	}
	maxIndex := 0
	for _, key := range required {
		trimmed := strings.TrimSpace(key)
		if !strings.HasPrefix(trimmed, "go_native.disk.") {
			continue
		}
		rest := strings.TrimPrefix(trimmed, "go_native.disk.")
		parts := strings.Split(rest, ".")
		if len(parts) != 2 {
			continue
		}
		idx, err := strconv.Atoi(parts[0])
		if err != nil || idx <= 0 {
			continue
		}
		switch parts[1] {
		case "name", "size", "used", "available", "usage", "busy", "read", "write", "read_iops", "write_iops", "read_latency", "write_latency":
			if idx > maxIndex {
				maxIndex = idx
			}
		}
	}
	return maxIndex
}

func (c *GoNativeDiskCollector) ensureSlots() {
	c.ensureSlotsForCount(detectNamedDiskCount())
}

func (c *GoNativeDiskCollector) ensureSlotsForCount(detected int) {
	requiredMax := c.requiredMaxIndex()
	slotCount := max(detected, requiredMax)
	if slotCount > 16 {
		slotCount = 16
	}
	for index := 1; index <= slotCount; index++ {
		if _, exists := c.slots[index]; exists {
			continue
		}
		slot := &goNativeDiskSlot{
			nameItem:         NewCollectItem(fmt.Sprintf("go_native.disk.%d.name", index), fmt.Sprintf("Disk %d name", index), "", 0, 0, 0),
			sizeItem:         NewCollectItem(fmt.Sprintf("go_native.disk.%d.size", index), fmt.Sprintf("Disk %d size", index), "GB", 0, 0, 0),
			usedItem:         NewCollectItem(fmt.Sprintf("go_native.disk.%d.used", index), fmt.Sprintf("Disk %d used", index), "GB", 0, 0, 0),
			availableItem:    NewCollectItem(fmt.Sprintf("go_native.disk.%d.available", index), fmt.Sprintf("Disk %d available", index), "GB", 0, 0, 0),
			usageItem:        NewCollectItem(fmt.Sprintf("go_native.disk.%d.usage", index), fmt.Sprintf("Disk %d usage", index), "%", 0, 100, 0),
			busyItem:         NewCollectItem(fmt.Sprintf("go_native.disk.%d.busy", index), fmt.Sprintf("Disk %d busy", index), "%", 0, 100, 0),
			readItem:         NewCollectItem(fmt.Sprintf("go_native.disk.%d.read", index), fmt.Sprintf("Disk %d read speed", index), "MiB/s", 0, 0, 2),
			writeItem:        NewCollectItem(fmt.Sprintf("go_native.disk.%d.write", index), fmt.Sprintf("Disk %d write speed", index), "MiB/s", 0, 0, 2),
			readIOPSItem:     NewCollectItem(fmt.Sprintf("go_native.disk.%d.read_iops", index), fmt.Sprintf("Disk %d read IOPS", index), "IOPS", 0, 0, 0),
			writeIOPSItem:    NewCollectItem(fmt.Sprintf("go_native.disk.%d.write_iops", index), fmt.Sprintf("Disk %d write IOPS", index), "IOPS", 0, 0, 0),
			readLatencyItem:  NewCollectItem(fmt.Sprintf("go_native.disk.%d.read_latency", index), fmt.Sprintf("Disk %d read latency", index), "ms", 0, 0, 2),
			writeLatencyItem: NewCollectItem(fmt.Sprintf("go_native.disk.%d.write_latency", index), fmt.Sprintf("Disk %d write latency", index), "ms", 0, 0, 2),
		}
		c.slots[index] = slot
		c.setItem(slot.nameItem.GetName(), slot.nameItem)
		c.setItem(slot.sizeItem.GetName(), slot.sizeItem)
		c.setItem(slot.usedItem.GetName(), slot.usedItem)
		c.setItem(slot.availableItem.GetName(), slot.availableItem)
		c.setItem(slot.usageItem.GetName(), slot.usageItem)
		c.setItem(slot.busyItem.GetName(), slot.busyItem)
		c.setItem(slot.readItem.GetName(), slot.readItem)
		c.setItem(slot.writeItem.GetName(), slot.writeItem)
		c.setItem(slot.readIOPSItem.GetName(), slot.readIOPSItem)
		c.setItem(slot.writeIOPSItem.GetName(), slot.writeIOPSItem)
		c.setItem(slot.readLatencyItem.GetName(), slot.readLatencyItem)
		c.setItem(slot.writeLatencyItem.GetName(), slot.writeLatencyItem)
	}
}

func (c *GoNativeDiskCollector) snapshotDisks() []*DiskInfo {
	initializeCache()
	updateDiskInfo()
	return getCachedDiskInfo()
}

func updateDiskStaticItems(slot *goNativeDiskSlot, disk *DiskInfo) {
	if slot == nil {
		return
	}
	if disk == nil || strings.TrimSpace(disk.Name) == "" {
		slot.nameItem.SetAvailable(false)
		slot.sizeItem.SetAvailable(false)
		slot.usedItem.SetAvailable(false)
		slot.availableItem.SetAvailable(false)
		slot.usageItem.SetAvailable(false)
		return
	}
	name := strings.TrimSpace(disk.Name)
	model := strings.TrimSpace(disk.Model)
	if model != "" {
		name = fmt.Sprintf("%s (%s)", name, model)
	}
	slot.nameItem.SetValue(name)
	slot.nameItem.SetAvailable(true)
	slot.sizeItem.SetValue(disk.Size)
	slot.sizeItem.SetAvailable(true)
	slot.usedItem.SetValue(disk.Used)
	slot.usedItem.SetAvailable(true)
	slot.availableItem.SetValue(disk.Available)
	slot.availableItem.SetAvailable(true)
	slot.usageItem.SetValue(disk.Usage)
	slot.usageItem.SetAvailable(true)
}

func updateDiskRateItems(slot *goNativeDiskSlot, disk *DiskInfo) {
	if slot == nil {
		return
	}
	if disk != nil && disk.DynamicAvailable {
		slot.readItem.SetValue(disk.ReadSpeed)
		slot.readItem.SetAvailable(true)
		slot.writeItem.SetValue(disk.WriteSpeed)
		slot.writeItem.SetAvailable(true)
		slot.readIOPSItem.SetValue(disk.ReadIOPS)
		slot.readIOPSItem.SetAvailable(true)
		slot.writeIOPSItem.SetValue(disk.WriteIOPS)
		slot.writeIOPSItem.SetAvailable(true)
		slot.readLatencyItem.SetValue(disk.ReadLatencyMS)
		slot.readLatencyItem.SetAvailable(true)
		slot.writeLatencyItem.SetValue(disk.WriteLatencyMS)
		slot.writeLatencyItem.SetAvailable(true)
		slot.busyItem.SetValue(disk.BusyPercent)
		slot.busyItem.SetAvailable(true)
		return
	}
	slot.readItem.SetAvailable(false)
	slot.writeItem.SetAvailable(false)
	slot.readIOPSItem.SetAvailable(false)
	slot.writeIOPSItem.SetAvailable(false)
	slot.readLatencyItem.SetAvailable(false)
	slot.writeLatencyItem.SetAvailable(false)
	slot.busyItem.SetAvailable(false)
}

func (c *GoNativeDiskCollector) GetAllItems() map[string]*CollectItem {
	disks := c.snapshotDisks()
	c.ensureSlotsForCount(len(disks))
	for index, slot := range c.slots {
		var disk *DiskInfo
		if index > 0 && index <= len(disks) {
			disk = disks[index-1]
		}
		updateDiskStaticItems(slot, disk)
		updateDiskRateItems(slot, disk)
	}
	return c.ItemsSnapshot()
}

func (c *GoNativeDiskCollector) UpdateItems() error {
	if !c.IsEnabled() {
		return nil
	}
	disks := c.snapshotDisks()
	for index, slot := range c.slots {
		if slot == nil {
			continue
		}
		var disk *DiskInfo
		if index > 0 && index <= len(disks) {
			disk = disks[index-1]
		}
		if slot.nameItem.IsEnabled() || slot.sizeItem.IsEnabled() ||
			slot.usedItem.IsEnabled() || slot.availableItem.IsEnabled() || slot.usageItem.IsEnabled() {
			updateDiskStaticItems(slot, disk)
		}
		if slot.readItem.IsEnabled() || slot.writeItem.IsEnabled() ||
			slot.readIOPSItem.IsEnabled() || slot.writeIOPSItem.IsEnabled() ||
			slot.readLatencyItem.IsEnabled() || slot.writeLatencyItem.IsEnabled() ||
			slot.busyItem.IsEnabled() {
			updateDiskRateItems(slot, disk)
		}
	}
	return nil
}
