package main

import (
	"fmt"
	"net"
	"os"
	"sort"
	"strings"
	"sync"
	"time"

	gopsutilNet "github.com/shirou/gopsutil/v3/net"
)

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
	last        map[string]netSample
	avgUpload   map[string]float64 // MB/s
	avgDownload map[string]float64 // MB/s
	watchSet    map[string]struct{}
	running     bool
}

var globalNetSampler = &netSampler{
	last:        make(map[string]netSample),
	avgUpload:   make(map[string]float64),
	avgDownload: make(map[string]float64),
	watchSet:    make(map[string]struct{}),
}

func (ns *netSampler) watchInterface(name string) {
	name = strings.TrimSpace(name)
	if name == "" {
		return
	}
	ns.mutex.Lock()
	if _, exists := ns.watchSet[name]; !exists {
		ns.watchSet[name] = struct{}{}
		ns.last[name] = netSample{}
		ns.avgUpload[name] = 0
		ns.avgDownload[name] = 0
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

func (ns *netSampler) loop() {
	ticker := time.NewTicker(200 * time.Millisecond)
	defer ticker.Stop()
	for {
		<-ticker.C
		if !isRenderActive() {
			continue
		}
		ns.sampleOnce()
	}
}

func (ns *netSampler) sampleOnce() {
	ns.mutex.RLock()
	watchSet := make(map[string]struct{}, len(ns.watchSet))
	for iface := range ns.watchSet {
		watchSet[iface] = struct{}{}
	}
	lastSnapshot := make(map[string]netSample, len(ns.last))
	for iface, sample := range ns.last {
		lastSnapshot[iface] = sample
	}
	avgUploadSnapshot := make(map[string]float64, len(ns.avgUpload))
	for iface, speed := range ns.avgUpload {
		avgUploadSnapshot[iface] = speed
	}
	avgDownloadSnapshot := make(map[string]float64, len(ns.avgDownload))
	for iface, speed := range ns.avgDownload {
		avgDownloadSnapshot[iface] = speed
	}
	ns.mutex.RUnlock()
	if len(watchSet) == 0 {
		return
	}
	stats, err := gopsutilNet.IOCounters(true)
	if err != nil {
		return
	}
	now := time.Now()
	lastUpdates := make(map[string]netSample, len(watchSet))
	avgUploadUpdates := make(map[string]float64, len(watchSet))
	avgDownloadUpdates := make(map[string]float64, len(watchSet))
	for _, s := range stats {
		if _, watching := watchSet[s.Name]; !watching {
			continue
		}
		prev := lastSnapshot[s.Name]
		cur := netSample{time: now, tx: s.BytesSent, rx: s.BytesRecv}
		lastUpdates[s.Name] = cur
		if !prev.time.IsZero() {
			if cur.tx < prev.tx || cur.rx < prev.rx {
				continue
			}
			dt := now.Sub(prev.time).Seconds()
			if dt >= 0.15 && dt <= 10.0 {
				u := float64(cur.tx-prev.tx) / dt / 1024 / 1024
				d := float64(cur.rx-prev.rx) / dt / 1024 / 1024
				avgUploadUpdates[s.Name] = avgUploadSnapshot[s.Name]*0.7 + u*0.3
				avgDownloadUpdates[s.Name] = avgDownloadSnapshot[s.Name]*0.7 + d*0.3
			}
		}
	}
	if len(lastUpdates) == 0 && len(avgUploadUpdates) == 0 && len(avgDownloadUpdates) == 0 {
		return
	}
	ns.mutex.Lock()
	for iface, sample := range lastUpdates {
		ns.last[iface] = sample
	}
	for iface, value := range avgUploadUpdates {
		ns.avgUpload[iface] = value
	}
	for iface, value := range avgDownloadUpdates {
		ns.avgDownload[iface] = value
	}
	ns.mutex.Unlock()
}

func (ns *netSampler) get(iface string, upload bool) (float64, bool) {
	ns.mutex.RLock()
	defer ns.mutex.RUnlock()
	iface = strings.TrimSpace(iface)
	if iface == "" {
		return 0, false
	}
	if _, ok := ns.watchSet[iface]; !ok {
		return 0, false
	}
	if upload {
		return ns.avgUpload[iface], true
	}
	return ns.avgDownload[iface], true
}

// NetworkInterfaceMonitor（精简，不再持有采样窗口等状态）
type NetworkInterfaceMonitor struct {
	*BaseMonitorItem
	interfaceName string
	configuredIF  string
	interfaceIdx  int
	metricType    string
	mutex         sync.RWMutex
}

func (n *NetworkInterfaceMonitor) GetInterfaceName() string { return n.interfaceName }

func NewNetworkInterfaceMonitor(interfaceName, metricType, prefix string) *NetworkInterfaceMonitor {
	return newNetworkInterfaceMonitor(interfaceName, metricType, prefix, 0)
}

func NewNetworkInterfaceMonitorByIndex(interfaceIndex int, metricType string) *NetworkInterfaceMonitor {
	if interfaceIndex <= 0 {
		return nil
	}
	return newNetworkInterfaceMonitor("", metricType, fmt.Sprintf("net%d", interfaceIndex), interfaceIndex)
}

func newNetworkInterfaceMonitor(interfaceName, metricType, prefix string, interfaceIndex int) *NetworkInterfaceMonitor {
	if prefix == "" {
		prefix = "net"
	}
	var name, label, unit string
	var precision int
	switch metricType {
	case "upload":
		name = fmt.Sprintf("%s_upload", prefix)
		label = "Network upload speed"
		unit = " MiB/s"
		precision = 2
	case "download":
		name = fmt.Sprintf("%s_download", prefix)
		label = "Network download speed"
		unit = " MiB/s"
		precision = 2
	case "ip":
		name = fmt.Sprintf("%s_ip", prefix)
		label = "Network interface IP"
		precision = 0
	case "name":
		name = fmt.Sprintf("%s_interface", prefix)
		label = "Network interface name"
		precision = 0
	default:
		return nil
	}
	mon := &NetworkInterfaceMonitor{
		BaseMonitorItem: NewBaseMonitorItem(name, label, 0, 0, unit, precision),
		interfaceName:   strings.TrimSpace(interfaceName),
		configuredIF:    strings.TrimSpace(interfaceName),
		interfaceIdx:    interfaceIndex,
		metricType:      metricType,
	}
	if interfaceName != "" {
		globalNetSampler.watchInterface(interfaceName)
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
		// Avoid blocking during initialization: refresh in background
		go networkInterfaceManager.refreshInterface()
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

func (nim *NetworkInterfaceManager) GetInterfaceByIndex(index int) string {
	nim.mutex.RLock()
	defer nim.mutex.RUnlock()
	if index < 0 || index >= len(nim.orderedInterfaces) {
		return ""
	}
	return nim.orderedInterfaces[index]
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
	iface := n.resolveInterface(manager)
	if iface != n.interfaceName {
		n.interfaceName = iface
		globalNetSampler.watchInterface(iface)
		globalNetSampler.start()
	}
	switch n.metricType {
	case "upload":
		if v, ok := globalNetSampler.get(n.interfaceName, true); ok {
			n.SetValue(v)
			n.SetAvailable(true)
		} else {
			n.SetAvailable(false)
		}
	case "download":
		if v, ok := globalNetSampler.get(n.interfaceName, false); ok {
			n.SetValue(v)
			n.SetAvailable(true)
		} else {
			n.SetAvailable(false)
		}
	case "ip":
		refreshIPIfNeeded(n.interfaceName)
		ip := getCachedIP(n.interfaceName)
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

func (n *NetworkInterfaceMonitor) resolveInterface(manager *NetworkInterfaceManager) string {
	if n.interfaceIdx > 0 {
		return manager.GetInterfaceByIndex(n.interfaceIdx - 1)
	}
	if n.configuredIF != "" && !strings.EqualFold(n.configuredIF, "auto") {
		return n.configuredIF
	}
	return manager.GetDefaultInterface()
}

func (n *NetworkInterfaceMonitor) GetDisplayValue() float64 {
	n.mutex.RLock()
	defer n.mutex.RUnlock()
	if val, ok := n.value.Value.(float64); ok {
		return val
	}
	return 0.0
}

func getActiveNetworkInterfaces() []string {
	interfaces, err := gopsutilNet.Interfaces()
	if err != nil {
		return []string{}
	}
	var activeInterfaces []string
	seen := make(map[string]struct{})
	for _, iface := range interfaces {
		ifaceName := strings.TrimSpace(iface.Name)
		if ifaceName == "" {
			continue
		}
		if isVirtualInterface(ifaceName) {
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
				if _, exists := seen[ifaceName]; exists {
					continue
				}
				seen[ifaceName] = struct{}{}
				activeInterfaces = append(activeInterfaces, ifaceName)
			}
		}
	}
	sort.Strings(activeInterfaces)
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
	data, err := os.ReadFile("/proc/net/route")
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

func GetConfiguredNetworkInterface(configInterface string) string {
	if configInterface == "" || configInterface == "auto" {
		manager := GetNetworkInterfaceManager()
		manager.TryRefreshAsync()
		return manager.GetDefaultInterface()
	}
	return configInterface
}

var (
	ipCacheMutex sync.Mutex
	ipCacheTTL   = 2 * time.Second
	ipCache      = make(map[string]*ipCacheState)
)

type ipCacheState struct {
	value     string
	lastCheck time.Time
	fetching  bool
}

func getIPCacheState(iface string) *ipCacheState {
	state, ok := ipCache[iface]
	if ok {
		return state
	}
	state = &ipCacheState{value: "-"}
	ipCache[iface] = state
	return state
}

func refreshIPIfNeeded(iface string) {
	iface = strings.TrimSpace(iface)
	if iface == "" {
		return
	}
	ipCacheMutex.Lock()
	now := time.Now()
	state := getIPCacheState(iface)
	if now.Sub(state.lastCheck) < ipCacheTTL || state.fetching {
		ipCacheMutex.Unlock()
		return
	}
	state.fetching = true
	ipCacheMutex.Unlock()
	go func(ifn string) {
		ip := "-"
		ifaces, err := gopsutilNet.Interfaces()
		if err == nil {
		found:
			for _, it := range ifaces {
				if it.Name == ifn {
					for _, a := range it.Addrs {
						if a.Addr != "" {
							p := net.ParseIP(strings.Split(a.Addr, "/")[0])
							if p != nil && !p.IsLoopback() {
								ip = p.String()
								break found
							}
						}
					}
				}
			}
		}
		ipCacheMutex.Lock()
		state := getIPCacheState(ifn)
		state.value = ip
		state.lastCheck = time.Now()
		state.fetching = false
		ipCacheMutex.Unlock()
	}(iface)
}

func getCachedIP(iface string) string {
	iface = strings.TrimSpace(iface)
	if iface == "" {
		return "-"
	}
	ipCacheMutex.Lock()
	defer ipCacheMutex.Unlock()
	return getIPCacheState(iface).value
}
