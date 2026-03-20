package main

import (
	"fmt"
	"net"
	"sort"
	"strconv"
	"strings"

	gopsutilNet "github.com/shirou/gopsutil/v3/net"
)

type goNativeNetworkSlot struct {
	uploadItem    *CollectItem
	downloadItem  *CollectItem
	ipItem        *CollectItem
	nameItem      *CollectItem
	interfaceName string
	ipv4          string
}

type GoNativeNetworkCollector struct {
	*BaseCollector
	requiredProvider func() []string
	slots            map[int]*goNativeNetworkSlot
}

func NewGoNativeNetworkCollector(requiredProvider func() []string) *GoNativeNetworkCollector {
	return &GoNativeNetworkCollector{
		BaseCollector:    NewBaseCollector("go_native.network"),
		requiredProvider: requiredProvider,
		slots:            make(map[int]*goNativeNetworkSlot),
	}
}

func (c *GoNativeNetworkCollector) requiredMaxIndex() int {
	required := []string{}
	if c.requiredProvider != nil {
		required = c.requiredProvider()
	}
	maxIndex := 0
	for _, key := range required {
		trimmed := strings.TrimSpace(key)
		if !strings.HasPrefix(trimmed, "go_native.net.") {
			continue
		}
		rest := strings.TrimPrefix(trimmed, "go_native.net.")
		parts := strings.Split(rest, ".")
		if len(parts) != 2 {
			continue
		}
		idx, err := strconv.Atoi(parts[0])
		if err != nil || idx <= 0 {
			continue
		}
		switch parts[1] {
		case "upload", "download", "ip", "interface":
			if idx > maxIndex {
				maxIndex = idx
			}
		}
	}
	return maxIndex
}

func (c *GoNativeNetworkCollector) ensureSlots() {
	names, _ := getActiveNetworkInterfacesAndIPv4()
	c.ensureSlotsForCount(len(names))
}

func (c *GoNativeNetworkCollector) ensureSlotsForCount(detected int) {
	requiredMax := c.requiredMaxIndex()
	slotCount := max(detected, requiredMax)
	if slotCount > 16 {
		slotCount = 16
	}
	for index := 1; index <= slotCount; index++ {
		if _, exists := c.slots[index]; exists {
			continue
		}
		slot := &goNativeNetworkSlot{
			uploadItem:   NewCollectItem(fmt.Sprintf("go_native.net.%d.upload", index), fmt.Sprintf("Net %d upload", index), " MiB/s", 0, 0, 2),
			downloadItem: NewCollectItem(fmt.Sprintf("go_native.net.%d.download", index), fmt.Sprintf("Net %d download", index), " MiB/s", 0, 0, 2),
			ipItem:       NewCollectItem(fmt.Sprintf("go_native.net.%d.ip", index), fmt.Sprintf("Net %d ip", index), "", 0, 0, 0),
			nameItem:     NewCollectItem(fmt.Sprintf("go_native.net.%d.interface", index), fmt.Sprintf("Net %d interface", index), "", 0, 0, 0),
		}
		c.slots[index] = slot
		c.setItem(slot.uploadItem.GetName(), slot.uploadItem)
		c.setItem(slot.downloadItem.GetName(), slot.downloadItem)
		c.setItem(slot.ipItem.GetName(), slot.ipItem)
		c.setItem(slot.nameItem.GetName(), slot.nameItem)
	}
}

func (c *GoNativeNetworkCollector) GetAllItems() map[string]*CollectItem {
	interfaces, ipv4ByName := getActiveNetworkInterfacesAndIPv4()
	c.ensureSlotsForCount(len(interfaces))
	for index, slot := range c.slots {
		iface := resolveInterfaceByIndex(interfaces, index)
		slot.interfaceName = iface
		slot.ipv4 = strings.TrimSpace(ipv4ByName[iface])
		if strings.TrimSpace(iface) == "" {
			slot.nameItem.SetValue("-")
			slot.nameItem.SetAvailable(false)
			slot.ipItem.SetValue("-")
			slot.ipItem.SetAvailable(false)
			continue
		}
		slot.nameItem.SetValue(iface)
		slot.nameItem.SetAvailable(true)
		ip := slot.ipv4
		if ip == "" {
			slot.ipItem.SetValue("-")
			slot.ipItem.SetAvailable(false)
		} else {
			slot.ipItem.SetValue(ip)
			slot.ipItem.SetAvailable(true)
		}
	}
	return c.ItemsSnapshot()
}

func (c *GoNativeNetworkCollector) UpdateItems() error {
	if !c.IsEnabled() {
		return nil
	}
	interfaceNames := make([]string, 0, len(c.slots))
	for _, slot := range c.slots {
		if slot == nil {
			continue
		}
		if name := strings.TrimSpace(slot.interfaceName); name != "" {
			interfaceNames = append(interfaceNames, name)
		}
	}
	speedByName := getNetworkSpeedSnapshots(interfaceNames)

	for _, slot := range c.slots {
		if slot == nil {
			continue
		}
		if strings.TrimSpace(slot.interfaceName) == "" {
			if slot.uploadItem.IsEnabled() {
				slot.uploadItem.SetAvailable(false)
			}
			if slot.downloadItem.IsEnabled() {
				slot.downloadItem.SetAvailable(false)
			}
			continue
		}

		if slot.uploadItem.IsEnabled() {
			if speed, ok := speedByName[slot.interfaceName]; ok && speed.OK {
				slot.uploadItem.SetValue(speed.Upload)
				slot.uploadItem.SetAvailable(true)
			} else {
				slot.uploadItem.SetAvailable(false)
			}
		}
		if slot.downloadItem.IsEnabled() {
			if speed, ok := speedByName[slot.interfaceName]; ok && speed.OK {
				slot.downloadItem.SetValue(speed.Download)
				slot.downloadItem.SetAvailable(true)
			} else {
				slot.downloadItem.SetAvailable(false)
			}
		}
	}
	return nil
}

func resolveInterfaceByIndex(names []string, index int) string {
	if index <= 0 || index > len(names) {
		return ""
	}
	return names[index-1]
}

func getActiveNetworkInterfaces() []string {
	names, _ := getActiveNetworkInterfacesAndIPv4()
	return names
}

func getActiveNetworkInterfacesAndIPv4() ([]string, map[string]string) {
	interfaces, err := gopsutilNet.Interfaces()
	if err != nil {
		return []string{}, map[string]string{}
	}
	active := make([]string, 0, len(interfaces))
	ipv4ByName := make(map[string]string, len(interfaces))
	seen := make(map[string]struct{}, len(interfaces))
	for _, iface := range interfaces {
		name := strings.TrimSpace(iface.Name)
		if name == "" {
			continue
		}
		if isVirtualInterface(name) {
			continue
		}
		if len(iface.Flags) == 0 {
			continue
		}
		hasUp := false
		hasLoopback := false
		for _, flag := range iface.Flags {
			if flag == "up" {
				hasUp = true
			}
			if flag == "loopback" {
				hasLoopback = true
			}
		}
		if !hasUp || hasLoopback || !hasValidIP(iface) {
			continue
		}
		if _, exists := seen[name]; exists {
			continue
		}
		seen[name] = struct{}{}
		active = append(active, name)
		ipv4ByName[name] = extractInterfaceIPv4(iface)
	}
	sort.Strings(active)
	return active, ipv4ByName
}

func isVirtualInterface(name string) bool {
	virtualPrefixes := []string{
		"docker", "br-", "veth", "virbr", "vmnet", "vboxnet",
		"tap", "tun", "lo", "dummy", "bond", "team", "vlan",
	}
	for _, prefix := range virtualPrefixes {
		if strings.HasPrefix(name, prefix) {
			return true
		}
	}
	return false
}

func hasValidIP(iface gopsutilNet.InterfaceStat) bool {
	for _, addr := range iface.Addrs {
		if addr.Addr == "" {
			continue
		}
		ip := net.ParseIP(strings.Split(addr.Addr, "/")[0])
		if ip == nil {
			continue
		}
		if !ip.IsLoopback() && !ip.IsLinkLocalUnicast() {
			return true
		}
	}
	return false
}

func getInterfaceIPv4(interfaceName string) string {
	interfaceName = strings.TrimSpace(interfaceName)
	if interfaceName == "" {
		return ""
	}
	ifaces, err := gopsutilNet.Interfaces()
	if err != nil {
		return ""
	}
	for _, iface := range ifaces {
		if iface.Name != interfaceName {
			continue
		}
		return extractInterfaceIPv4(iface)
	}
	return ""
}

func extractInterfaceIPv4(iface gopsutilNet.InterfaceStat) string {
	for _, addr := range iface.Addrs {
		if strings.TrimSpace(addr.Addr) == "" {
			continue
		}
		ip := net.ParseIP(strings.Split(addr.Addr, "/")[0])
		if ip == nil || ip.IsLoopback() {
			continue
		}
		if ipv4 := ip.To4(); ipv4 != nil {
			return ipv4.String()
		}
	}
	return ""
}
