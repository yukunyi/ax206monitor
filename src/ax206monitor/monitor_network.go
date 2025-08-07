package main

import (
	"fmt"
	"net"
	"strings"
	"sync"
	"time"

	gopsutilNet "github.com/shirou/gopsutil/v3/net"
)

const (
	// Network speed calculation constants
	speedWindowDuration = 3 * time.Second // 3-second sliding window
	maxSpeedSamples     = 10              // Maximum samples to keep
	minSamplesForAvg    = 2               // Minimum samples needed for average calculation
)

var (
	currentNetworkInterface     string
	lastInterfaceRefresh        time.Time
	interfaceRefreshInterval    = 1 * time.Minute
	interfaceRefreshMutex       sync.RWMutex
	networkInterfaceUnavailable bool
	lastUnavailableTime         time.Time
	fastRefreshInterval         = 10 * time.Second // Fast refresh when interface unavailable
)

// NetworkSpeedSample represents a single speed measurement
type NetworkSpeedSample struct {
	timestamp time.Time
	bytes     uint64
	speed     float64 // MB/s
}

type NetworkInterfaceMonitor struct {
	*BaseMonitorItem
	interfaceName string
	metricType    string
	lastBytes     uint64
	lastTime      time.Time
	speedSamples  []NetworkSpeedSample // Sliding window for speed calculation
	mutex         sync.RWMutex
}

func (n *NetworkInterfaceMonitor) GetInterfaceName() string {
	return n.interfaceName
}

func NewNetworkInterfaceMonitor(interfaceName, metricType, prefix string) *NetworkInterfaceMonitor {
	var name, label, unit string
	var precision int
	var maxValue float64

	switch metricType {
	case "upload":
		name = "net1_upload"
		label = "Upload"
		unit = "" // Dynamic unit will be set in SetValue
		precision = 2
		maxValue = getNetworkInterfaceMaxSpeed(interfaceName)
	case "download":
		name = "net1_download"
		label = "Download"
		unit = "" // Dynamic unit will be set in SetValue
		precision = 2
		maxValue = getNetworkInterfaceMaxSpeed(interfaceName)
	case "ip":
		name = "net1_ip"
		label = "IP"
		unit = ""
		precision = 0
		maxValue = 0
	case "name":
		name = "net1_interface"
		label = ""
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
		speedSamples:    make([]NetworkSpeedSample, 0, maxSpeedSamples),
	}

	// For name type, set the value immediately if interface is available
	if metricType == "name" {
		if interfaceName != "" {
			monitor.SetValue(interfaceName)
			monitor.SetAvailable(true)
		} else {
			monitor.SetValue("-")
			monitor.SetAvailable(false)
		}
	}

	return monitor
}

// formatNetworkSpeed formats network speed with appropriate unit and spacing
func formatNetworkSpeed(speedMBps float64) (float64, string) {
	if speedMBps >= 1.0 {
		return speedMBps, " MiB/s"
	} else if speedMBps >= 0.001 {
		return speedMBps * 1024, " KiB/s"
	} else {
		return speedMBps * 1024 * 1024, " B/s"
	}
}

// addSpeedSample adds a new speed sample to the sliding window
func (n *NetworkInterfaceMonitor) addSpeedSample(timestamp time.Time, bytes uint64, speed float64) {
	// Remove old samples outside the window
	cutoff := timestamp.Add(-speedWindowDuration)
	validSamples := make([]NetworkSpeedSample, 0, len(n.speedSamples))
	for _, sample := range n.speedSamples {
		if sample.timestamp.After(cutoff) {
			validSamples = append(validSamples, sample)
		}
	}

	// Add new sample
	newSample := NetworkSpeedSample{
		timestamp: timestamp,
		bytes:     bytes,
		speed:     speed,
	}
	validSamples = append(validSamples, newSample)

	// Keep only the most recent samples if we exceed the limit
	if len(validSamples) > maxSpeedSamples {
		validSamples = validSamples[len(validSamples)-maxSpeedSamples:]
	}

	n.speedSamples = validSamples
}

// calculateAverageSpeed calculates the average speed from samples in the sliding window
func (n *NetworkInterfaceMonitor) calculateAverageSpeed() float64 {
	if len(n.speedSamples) < minSamplesForAvg {
		return 0.0
	}

	now := time.Now()
	cutoff := now.Add(-speedWindowDuration)

	var totalSpeed float64
	var validSamples int
	var totalWeight float64

	// Use weighted average based on sample age (newer samples have more weight)
	for _, sample := range n.speedSamples {
		if sample.timestamp.After(cutoff) {
			// Calculate weight based on age (newer = higher weight)
			age := now.Sub(sample.timestamp).Seconds()
			weight := 1.0 - (age / speedWindowDuration.Seconds())
			if weight < 0.1 {
				weight = 0.1 // Minimum weight
			}

			totalSpeed += sample.speed * weight
			totalWeight += weight
			validSamples++
		}
	}

	if validSamples == 0 || totalWeight == 0 {
		return 0.0
	}

	return totalSpeed / totalWeight
}

// SetNetworkSpeedValue sets the network speed value with dynamic unit
func (n *NetworkInterfaceMonitor) SetNetworkSpeedValue(speedMBps float64) {
	n.mutex.Lock()
	defer n.mutex.Unlock()

	value, unit := formatNetworkSpeed(speedMBps)
	n.value.Value = value
	n.value.Unit = unit
}

// setNetworkSpeedValueUnsafe sets the network speed value without locking (internal use)
func (n *NetworkInterfaceMonitor) setNetworkSpeedValueUnsafe(speedMBps float64) {
	value, unit := formatNetworkSpeed(speedMBps)
	n.value.Value = value
	n.value.Unit = unit
}

// GetDisplayValue returns the value that will be displayed (for color calculation)
func (n *NetworkInterfaceMonitor) GetDisplayValue() float64 {
	n.mutex.RLock()
	defer n.mutex.RUnlock()

	if val, ok := n.value.Value.(float64); ok {
		return val
	}
	return 0.0
}

func (n *NetworkInterfaceMonitor) Update() error {
	n.mutex.Lock()
	defer n.mutex.Unlock()

	// Update interface name if needed
	n.refreshInterfaceIfNeeded()

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

func (n *NetworkInterfaceMonitor) refreshInterfaceIfNeeded() {
	// First check if refresh is needed (with read lock)
	interfaceRefreshMutex.RLock()
	now := time.Now()
	refreshInterval := interfaceRefreshInterval
	if networkInterfaceUnavailable {
		refreshInterval = fastRefreshInterval
	}
	needsRefresh := now.Sub(lastInterfaceRefresh) >= refreshInterval
	currentInterface := currentNetworkInterface
	interfaceRefreshMutex.RUnlock()

	if needsRefresh {
		// Only acquire write lock if refresh is actually needed
		interfaceRefreshMutex.Lock()

		// Double-check after acquiring write lock
		if now.Sub(lastInterfaceRefresh) >= refreshInterval {
			interfaces := getActiveNetworkInterfaces()
			if len(interfaces) > 0 {
				newInterface := interfaces[0]
				if currentNetworkInterface != newInterface {
					logInfoModule("network", "Interface changed from '%s' to '%s'", currentNetworkInterface, newInterface)
					currentNetworkInterface = newInterface
				}
				lastInterfaceRefresh = now

				// Mark interface as available if it was previously unavailable
				if networkInterfaceUnavailable {
					networkInterfaceUnavailable = false
					logInfoModule("network", "Network interface recovered: %s", currentNetworkInterface)
				}
				currentInterface = currentNetworkInterface
			} else {
				// No active interfaces found
				if !networkInterfaceUnavailable {
					networkInterfaceUnavailable = true
					lastUnavailableTime = now
					logWarnModule("network", "No active network interface found, will retry every %v", fastRefreshInterval)
				}
				lastInterfaceRefresh = now
			}
		} else {
			// Another goroutine already refreshed
			currentInterface = currentNetworkInterface
		}

		interfaceRefreshMutex.Unlock()
	}

	// Update local interface name
	n.interfaceName = currentInterface
}

func (n *NetworkInterfaceMonitor) updateSpeed() error {
	// If no interface name, try to refresh
	if n.interfaceName == "" {
		n.refreshInterfaceIfNeeded()
		if n.interfaceName == "" {
			n.SetAvailable(false)
			return fmt.Errorf("no network interface available")
		}
	}

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

			// Calculate instantaneous speed if we have previous data
			if !n.lastTime.IsZero() {
				duration := now.Sub(n.lastTime).Seconds()

				// Only calculate if duration is reasonable (between 0.1s and 10s)
				if duration >= 0.1 && duration <= 10.0 {
					if currentBytes >= n.lastBytes {
						// Normal case: bytes increased
						instantSpeed := float64(currentBytes-n.lastBytes) / duration / 1024 / 1024

						// Filter out unrealistic speeds (> 10 GB/s)
						if instantSpeed <= 10000.0 {
							// Add sample to sliding window
							n.addSpeedSample(now, currentBytes, instantSpeed)

							// Calculate and set average speed
							avgSpeed := n.calculateAverageSpeed()
							n.setNetworkSpeedValueUnsafe(avgSpeed)
							n.SetAvailable(true)

							logDebugModule("network", "Interface %s: instant=%.2f MB/s, avg=%.2f MB/s, samples=%d",
								n.interfaceName, instantSpeed, avgSpeed, len(n.speedSamples))
						} else {
							logWarnModule("network", "Interface %s: unrealistic speed %.2f MB/s, ignoring",
								n.interfaceName, instantSpeed)
						}
					} else {
						// Handle counter reset (currentBytes < lastBytes)
						logDebugModule("network", "Interface %s: counter reset detected, resetting samples", n.interfaceName)
						n.speedSamples = n.speedSamples[:0] // Clear samples
						n.setNetworkSpeedValueUnsafe(0.0)
					}
				} else {
					logDebugModule("network", "Interface %s: unusual duration %.2fs, skipping calculation",
						n.interfaceName, duration)
				}
			}

			n.lastBytes = currentBytes
			n.lastTime = now
			return nil
		}
	}

	// Interface not found in stats, try to refresh interface list
	logDebugModule("network", "Interface %s not found in stats, refreshing interface list", n.interfaceName)
	n.refreshInterfaceIfNeeded()

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

func GetConfiguredNetworkInterface(configInterface string) string {
	if configInterface == "" || configInterface == "auto" {
		interfaces := getActiveNetworkInterfaces()
		if len(interfaces) > 0 {
			interfaceRefreshMutex.Lock()
			currentNetworkInterface = interfaces[0]
			lastInterfaceRefresh = time.Now()
			interfaceRefreshMutex.Unlock()
			logInfoModule("network", "Auto-detected network interface: %s", interfaces[0])
			return interfaces[0]
		}
		logWarnModule("network", "No active network interface found")
		return ""
	}
	logInfoModule("network", "Using configured network interface: %s", configInterface)
	return configInterface
}
