package main

import (
	"fmt"
	"io/ioutil"
	"net"
	"sort"
	"strings"
	"sync"
	"time"

	gopsutilNet "github.com/shirou/gopsutil/v3/net"
)

const (
	speedWindowDuration = 3 * time.Second
	maxSpeedSamples     = 10
	minSamplesForAvg    = 2
	minNetSampleGap     = 150 * time.Millisecond
)

// 刷新间隔与单例
type singleton struct{}

var (
	interfaceRefreshInterval = 1 * time.Minute
	fastRefreshInterval      = 5 * time.Second
	bootupRefreshInterval    = 2 * time.Second
	bootupDuration           = 2 * time.Minute

	networkInterfaceManagerOnce sync.Once
	networkInterfaceManager     *NetworkInterfaceManager
)

// 网络后台采样缓存
type netSample struct {
	time time.Time
	tx   uint64
	rx   uint64
}

type netSampler struct {
	mutex       sync.RWMutex
	iface       string
	last        netSample
	avgUpload   float64 // MB/s
	avgDownload float64 // MB/s
	running     bool
	stopCh      chan struct{}
}

var globalNetSampler = &netSampler{stopCh: make(chan struct{}, 1)}

func (ns *netSampler) setInterface(name string) {
	ns.mutex.Lock()
	if ns.iface != name {
		ns.iface = name
		ns.last = netSample{}
		ns.avgUpload = 0
		ns.avgDownload = 0
	}
	ns.mutex.Unlock()
}

func (ns *netSampler) start() {
	ns.mutex.Lock()
	if ns.running {
		ns.mutex.Unlock()
		return
	}
	ns.running = true
	ns.mutex.Unlock()
	go ns.loop()
}

func (ns *netSampler) stop() {
	select {
	case ns.stopCh <- struct{}{}:
	default:
	}
}

func (ns *netSampler) loop() {
	ticker := time.NewTicker(200 * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			if !isRenderActive() {
				continue
			}
			ns.sampleOnce()
		case <-ns.stopCh:
			ns.mutex.Lock()
			ns.running = false
			ns.mutex.Unlock()
			return
		}
	}
}

func (ns *netSampler) sampleOnce() {
	ns.mutex.RLock()
	iface := ns.iface
	prev := ns.last
	ns.mutex.RUnlock()
	if iface == "" {
		return
	}
	stats, err := gopsutilNet.IOCounters(true)
	if err != nil {
		return
	}
	for _, s := range stats {
		if s.Name == iface {
			now := time.Now()
			cur := netSample{time: now, tx: s.BytesSent, rx: s.BytesRecv}
			if !prev.time.IsZero() {
				dt := now.Sub(prev.time).Seconds()
				if dt >= 0.15 && dt <= 10.0 {
					u := float64(cur.tx-prev.tx) / dt / 1024 / 1024
					d := float64(cur.rx-prev.rx) / dt / 1024 / 1024
					ns.mutex.Lock()
					ns.avgUpload = (ns.avgUpload*0.7 + u*0.3)
					ns.avgDownload = (ns.avgDownload*0.7 + d*0.3)
					ns.last = cur
					ns.mutex.Unlock()
					return
				}
			}
			ns.mutex.Lock()
			ns.last = cur
			ns.mutex.Unlock()
			return
		}
	}
}

func (ns *netSampler) get(upload bool) (float64, bool) {
	ns.mutex.RLock()
	defer ns.mutex.RUnlock()
	if ns.iface == "" {
		return 0, false
	}
	if upload {
		return ns.avgUpload, true
	}
	return ns.avgDownload, true
}

// NetworkInterfaceMonitor（精简，不再持有采样窗口等状态）
type NetworkInterfaceMonitor struct {
	*BaseMonitorItem
	interfaceName string
	metricType    string
	mutex         sync.RWMutex
}

func (n *NetworkInterfaceMonitor) GetInterfaceName() string { return n.interfaceName }

func NewNetworkInterfaceMonitor(interfaceName, metricType, prefix string) *NetworkInterfaceMonitor {
	if prefix == "" {
		prefix = "net_default"
	}
	var name, label, unit string
	var precision int
	switch metricType {
	case "upload":
		name = fmt.Sprintf("%s_upload", prefix)
		label = "Upload"
		unit = " MiB/s"
		precision = 2
	case "download":
		name = fmt.Sprintf("%s_download", prefix)
		label = "Download"
		unit = " MiB/s"
		precision = 2
	case "ip":
		name = fmt.Sprintf("%s_ip", prefix)
		label = "IP"
		precision = 0
	case "name":
		name = fmt.Sprintf("%s_interface", prefix)
		label = ""
		precision = 0
	default:
		return nil
	}
	mon := &NetworkInterfaceMonitor{BaseMonitorItem: NewBaseMonitorItem(name, label, 0, 0, unit, precision), interfaceName: interfaceName, metricType: metricType}
	if interfaceName != "" {
		globalNetSampler.setInterface(interfaceName)
		globalNetSampler.start()
	}
	return mon
}

// NetworkInterfaceManager
type NetworkInterfaceManager struct {
	orderedInterfaces     []string
	defaultInterface      string
	lastRefresh           time.Time
	unavailable           bool
	lastUnavailableTime   time.Time
	refreshInterval       time.Duration
	fastRefreshInterval   time.Duration
	bootupRefreshInterval time.Duration
	mutex                 sync.RWMutex
	startTime             time.Time
	callbacks             []func(string)
	refreshRunning        bool
}

func NewNetworkInterfaceManager() *NetworkInterfaceManager {
	return &NetworkInterfaceManager{
		refreshInterval:       interfaceRefreshInterval,
		fastRefreshInterval:   fastRefreshInterval,
		bootupRefreshInterval: bootupRefreshInterval,
		startTime:             time.Now(),
		callbacks:             make([]func(string), 0),
	}
}

func GetNetworkInterfaceManager() *NetworkInterfaceManager {
	networkInterfaceManagerOnce.Do(func() {
		networkInterfaceManager = NewNetworkInterfaceManager()
		networkInterfaceManager.refreshInterface()
	})
	return networkInterfaceManager
}

func (nim *NetworkInterfaceManager) needsRefresh(now time.Time) bool {
	nim.mutex.RLock()
	defer nim.mutex.RUnlock()
	interval := nim.refreshInterval
	if now.Sub(nim.startTime) < bootupDuration {
		interval = nim.bootupRefreshInterval
	} else if nim.unavailable {
		interval = nim.fastRefreshInterval
	}
	return now.Sub(nim.lastRefresh) >= interval
}

func (nim *NetworkInterfaceManager) TryRefreshAsync() {
	if !nim.needsRefresh(time.Now()) {
		return
	}
	nim.mutex.Lock()
	if nim.refreshRunning {
		nim.mutex.Unlock()
		return
	}
	nim.refreshRunning = true
	nim.mutex.Unlock()
	go func() { nim.refreshInterface(); nim.mutex.Lock(); nim.refreshRunning = false; nim.mutex.Unlock() }()
}

func (nim *NetworkInterfaceManager) GetDefaultInterface() string {
	nim.mutex.RLock()
	defer nim.mutex.RUnlock()
	if nim.defaultInterface != "" {
		return nim.defaultInterface
	}
	if len(nim.orderedInterfaces) > 0 {
		return nim.orderedInterfaces[0]
	}
	return ""
}

func (nim *NetworkInterfaceManager) refreshInterface() {
	interfaces := getActiveNetworkInterfaces()
	now := time.Now()
	nim.mutex.Lock()
	defer nim.mutex.Unlock()
	if len(interfaces) > 0 {
		nim.orderedInterfaces = interfaces
		prevDefault := nim.defaultInterface
		nim.defaultInterface = ""
		for _, ifn := range interfaces {
			if hasDefaultGateway(ifn) {
				nim.defaultInterface = ifn
				break
			}
		}
		if nim.defaultInterface == "" {
			nim.defaultInterface = interfaces[0]
		}
		if prevDefault != nim.defaultInterface {
			logInfoModule("network", "Default network interface: %s", nim.defaultInterface)
			for _, cb := range nim.callbacks {
				go cb(nim.defaultInterface)
			}
		}
		nim.lastRefresh = now
		if nim.unavailable {
			nim.unavailable = false
			logInfoModule("network", "Network interface recovered: %s", nim.defaultInterface)
		}
	} else {
		if !nim.unavailable {
			nim.unavailable = true
			nim.lastUnavailableTime = now
			if now.Sub(nim.startTime) < bootupDuration {
				logInfoModule("network", "No active network interface found during bootup, will retry")
			} else {
				logWarnModule("network", "No active network interface found, will retry")
			}
		}
		nim.orderedInterfaces = nil
		nim.defaultInterface = ""
		nim.lastRefresh = now
	}
}

func (n *NetworkInterfaceMonitor) Update() error {
	n.mutex.Lock()
	defer n.mutex.Unlock()
	noteRenderAccess()
	manager := GetNetworkInterfaceManager()
	manager.TryRefreshAsync()
	iface := manager.GetDefaultInterface()
	if iface != n.interfaceName {
		n.interfaceName = iface
		globalNetSampler.setInterface(iface)
		globalNetSampler.start()
	}
	switch n.metricType {
	case "upload":
		if v, ok := globalNetSampler.get(true); ok {
			n.SetValue(v)
			n.SetAvailable(true)
		} else {
			n.SetAvailable(false)
		}
	case "download":
		if v, ok := globalNetSampler.get(false); ok {
			n.SetValue(v)
			n.SetAvailable(true)
		} else {
			n.SetAvailable(false)
		}
	case "ip":
		ip := "-"
		ifaces, err := gopsutilNet.Interfaces()
		if err == nil {
			for _, it := range ifaces {
				if it.Name == n.interfaceName {
					for _, a := range it.Addrs {
						if a.Addr != "" {
							p := net.ParseIP(strings.Split(a.Addr, "/")[0])
							if p != nil && !p.IsLoopback() {
								ip = p.String()
								break
							}
						}
					}
				}
			}
		}
		n.SetValue(ip)
		n.SetAvailable(ip != "-")
	case "name":
		if n.interfaceName == "" {
			n.SetValue("-")
			n.SetAvailable(false)
		} else {
			n.SetValue(n.interfaceName)
			n.SetAvailable(true)
		}
	}
	return nil
}

func (n *NetworkInterfaceMonitor) GetDisplayValue() float64 {
	n.mutex.RLock()
	defer n.mutex.RUnlock()
	if val, ok := n.value.Value.(float64); ok {
		return val
	}
	return 0.0
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
	sort.Strings(activeInterfaces)
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
	data, err := ioutil.ReadFile("/proc/net/route")
	if err != nil {
		return false
	}
	lines := strings.Split(string(data), "\n")
	for _, line := range lines[1:] {
		if line == "" {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 4 {
			continue
		}
		iface := fields[0]
		destination := fields[1]
		mask := fields[7]
		if iface == interfaceName && destination == "00000000" && mask == "00000000" {
			return true
		}
	}
	return false
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
		manager := GetNetworkInterfaceManager()
		manager.TryRefreshAsync()
		return manager.GetDefaultInterface()
	}
	return configInterface
}
