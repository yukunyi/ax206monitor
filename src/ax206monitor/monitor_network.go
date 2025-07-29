package main

import (
	"fmt"
	"net"
	"strings"
	"sync"
	"time"

	gopsutilNet "github.com/shirou/gopsutil/v3/net"
)

type NetworkInterfaceMonitor struct {
	*BaseMonitorItem
	interfaceName string
	metricType    string
	lastBytes     uint64
	lastTime      time.Time
	mutex         sync.Mutex
}

func (n *NetworkInterfaceMonitor) GetInterfaceName() string {
	return n.interfaceName
}

func NewNetworkInterfaceMonitor(interfaceName, metricType, prefix string) *NetworkInterfaceMonitor {
	var name, label, unit string
	var precision int
	var maxValue float64

	namePrefix := "net_"
	if prefix != "" {
		namePrefix = fmt.Sprintf("net_%s_", prefix)
	} else {
		namePrefix = fmt.Sprintf("net_%s_", interfaceName)
	}

	switch metricType {
	case "upload":
		name = namePrefix + "upload"
		if prefix != "" {
			label = fmt.Sprintf("Upload (%s)", interfaceName)
		} else {
			label = fmt.Sprintf("%s Upload", interfaceName)
		}
		unit = "MB/s"
		precision = 2
		maxValue = getNetworkInterfaceMaxSpeed(interfaceName)
	case "download":
		name = namePrefix + "download"
		if prefix != "" {
			label = fmt.Sprintf("Download (%s)", interfaceName)
		} else {
			label = fmt.Sprintf("%s Download", interfaceName)
		}
		unit = "MB/s"
		precision = 2
		maxValue = getNetworkInterfaceMaxSpeed(interfaceName)
	case "ip":
		name = namePrefix + "ip"
		if prefix != "" {
			label = fmt.Sprintf("IP (%s)", interfaceName)
		} else {
			label = fmt.Sprintf("%s IP", interfaceName)
		}
		unit = ""
		precision = 0
		maxValue = 0
	case "name":
		name = namePrefix + "name"
		if prefix != "" {
			label = "Interface"
		} else {
			label = fmt.Sprintf("%s Name", interfaceName)
		}
		unit = ""
		precision = 0
		maxValue = 0
	default:
		return nil
	}

	monitor := &NetworkInterfaceMonitor{
		BaseMonitorItem: NewBaseMonitorItem(name, label, 0, maxValue, unit, precision),
		interfaceName:   interfaceName,
		metricType:      metricType,
	}

	// For name type, set the value immediately
	if metricType == "name" {
		monitor.SetValue(interfaceName)
		monitor.SetAvailable(true)
	}

	return monitor
}

func (n *NetworkInterfaceMonitor) Update() error {
	n.mutex.Lock()
	defer n.mutex.Unlock()

	switch n.metricType {
	case "upload", "download":
		return n.updateSpeed()
	case "ip":
		return n.updateIP()
	case "name":
		return n.updateName()
	}
	return nil
}

func (n *NetworkInterfaceMonitor) updateSpeed() error {
	stats, err := gopsutilNet.IOCounters(true)
	if err != nil {
		n.SetAvailable(false)
		return err
	}

	for _, stat := range stats {
		if stat.Name == n.interfaceName {
			now := time.Now()
			var currentBytes uint64

			if n.metricType == "upload" {
				currentBytes = stat.BytesSent
			} else {
				currentBytes = stat.BytesRecv
			}

			if !n.lastTime.IsZero() && currentBytes > n.lastBytes {
				duration := now.Sub(n.lastTime).Seconds()
				if duration > 0 {
					speed := float64(currentBytes-n.lastBytes) / duration / 1024 / 1024
					n.SetValue(speed)
					n.SetAvailable(true)
				}
			}

			n.lastBytes = currentBytes
			n.lastTime = now
			return nil
		}
	}

	n.SetAvailable(false)
	return fmt.Errorf("interface %s not found", n.interfaceName)
}

func (n *NetworkInterfaceMonitor) updateIP() error {
	interfaces, err := gopsutilNet.Interfaces()
	if err != nil {
		n.SetAvailable(false)
		return err
	}

	var ipv4Addr, ipv6Addr string

	for _, iface := range interfaces {
		if iface.Name == n.interfaceName {
			for _, addr := range iface.Addrs {
				if addr.Addr == "" {
					continue
				}

				ip := net.ParseIP(strings.Split(addr.Addr, "/")[0])
				if ip == nil {
					continue
				}

				if isLocalIP(ip) {
					continue
				}

				if ip.To4() != nil {
					ipv4Addr = ip.String()
				} else if ip.To16() != nil {
					ipv6Addr = ip.String()
				}
			}
		}
	}

	if ipv4Addr != "" {
		n.SetValue(ipv4Addr)
		n.SetAvailable(true)
		return nil
	}

	if ipv6Addr != "" {
		n.SetValue(ipv6Addr)
		n.SetAvailable(true)
		return nil
	}

	n.SetAvailable(false)
	return fmt.Errorf("no valid IP found for interface %s", n.interfaceName)
}

func (n *NetworkInterfaceMonitor) updateName() error {
	n.SetValue(n.interfaceName)
	n.SetAvailable(true)
	return nil
}

func isLocalIP(ip net.IP) bool {
	if ip.IsLoopback() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() {
		return true
	}

	return false
}

func getActiveNetworkInterfaces() []string {
	interfaces, err := gopsutilNet.Interfaces()
	if err != nil {
		return []string{}
	}

	var activeInterfaces []string
	var defaultInterface string

	for _, iface := range interfaces {
		if isVirtualInterface(iface.Name) {
			continue
		}

		if len(iface.Flags) > 0 {
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

			if hasUp && !hasLoopback && hasValidIP(iface) {
				activeInterfaces = append(activeInterfaces, iface.Name)

				if hasDefaultGateway(iface.Name) {
					defaultInterface = iface.Name
				}
			}
		}
	}

	if defaultInterface != "" {
		result := []string{defaultInterface}
		for _, iface := range activeInterfaces {
			if iface != defaultInterface {
				result = append(result, iface)
			}
		}
		return result
	}

	return activeInterfaces
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

func hasDefaultGateway(interfaceName string) bool {
	return true
}

func getNetworkInterfaceMaxSpeed(interfaceName string) float64 {
	interfaces, err := gopsutilNet.Interfaces()
	if err != nil {
		return 0
	}

	for _, iface := range interfaces {
		if iface.Name == interfaceName {
			if iface.MTU > 0 {
				switch {
				case iface.MTU >= 9000:
					return 10000
				case iface.MTU >= 1500:
					return 1000
				default:
					return 100
				}
			}
		}
	}

	return 0
}
