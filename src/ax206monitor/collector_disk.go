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
	nameItem  *CollectItem
	sizeItem  *CollectItem
	readItem  *CollectItem
	writeItem *CollectItem
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
		case "name", "size", "read", "write":
			if idx > maxIndex {
				maxIndex = idx
			}
		}
	}
	return maxIndex
}

func (c *GoNativeDiskCollector) ensureSlots() {
	detected := detectNamedDiskCount()
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
			nameItem:  NewCollectItem(fmt.Sprintf("go_native.disk.%d.name", index), fmt.Sprintf("Disk %d name", index), "", 0, 0, 0),
			sizeItem:  NewCollectItem(fmt.Sprintf("go_native.disk.%d.size", index), fmt.Sprintf("Disk %d size", index), "GB", 0, 0, 0),
			readItem:  NewCollectItem(fmt.Sprintf("go_native.disk.%d.read", index), fmt.Sprintf("Disk %d read speed", index), "MiB/s", 0, 0, 2),
			writeItem: NewCollectItem(fmt.Sprintf("go_native.disk.%d.write", index), fmt.Sprintf("Disk %d write speed", index), "MiB/s", 0, 0, 2),
		}
		c.slots[index] = slot
		c.setItem(slot.nameItem.GetName(), slot.nameItem)
		c.setItem(slot.sizeItem.GetName(), slot.sizeItem)
		c.setItem(slot.readItem.GetName(), slot.readItem)
		c.setItem(slot.writeItem.GetName(), slot.writeItem)
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
}

func updateDiskRateItems(slot *goNativeDiskSlot, disk *DiskInfo) {
	if slot == nil {
		return
	}
	if disk != nil && disk.ReadSpeed >= 0 {
		slot.readItem.SetValue(disk.ReadSpeed)
		slot.readItem.SetAvailable(true)
	} else {
		slot.readItem.SetAvailable(false)
	}
	if disk != nil && disk.WriteSpeed >= 0 {
		slot.writeItem.SetValue(disk.WriteSpeed)
		slot.writeItem.SetAvailable(true)
	} else {
		slot.writeItem.SetAvailable(false)
	}
}

func (c *GoNativeDiskCollector) GetAllItems() map[string]*CollectItem {
	c.ensureSlots()
	disks := c.snapshotDisks()
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
	c.ensureSlots()
	disks := c.snapshotDisks()
	for index, slot := range c.slots {
		if slot == nil {
			continue
		}
		var disk *DiskInfo
		if index > 0 && index <= len(disks) {
			disk = disks[index-1]
		}
		if slot.readItem.IsEnabled() || slot.writeItem.IsEnabled() {
			updateDiskRateItems(slot, disk)
		}
	}
	return nil
}
