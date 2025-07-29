//go:build windows

package main

import (
	"fmt"
	"net"
	"sync"
	"syscall"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"
)

// Windows API constants
const (
	PDH_FMT_DOUBLE = 0x00000200
	ERROR_SUCCESS  = 0
)

// PDH structures
type PDH_FMT_COUNTERVALUE struct {
	CStatus     uint32
	DoubleValue float64
}

// Windows DLLs and functions
var (
	pdh                         = windows.NewLazyDLL("pdh.dll")
	pdhOpenQuery                = pdh.NewProc("PdhOpenQueryW")
	pdhAddCounter               = pdh.NewProc("PdhAddCounterW")
	pdhCollectQueryData         = pdh.NewProc("PdhCollectQueryData")
	pdhGetFormattedCounterValue = pdh.NewProc("PdhGetFormattedCounterValueW")
	pdhCloseQuery               = pdh.NewProc("PdhCloseQuery")
)

// Performance counter manager
type PerfCounterManager struct {
	query       uintptr
	counters    map[string]uintptr
	mutex       sync.RWMutex
	initialized bool
}

var perfManager = &PerfCounterManager{
	counters: make(map[string]uintptr),
}

func (pm *PerfCounterManager) Initialize() error {
	pm.mutex.Lock()
	defer pm.mutex.Unlock()

	if pm.initialized {
		return nil
	}

	ret, _, _ := pdhOpenQuery.Call(0, 0, uintptr(unsafe.Pointer(&pm.query)))
	if ret != ERROR_SUCCESS {
		return fmt.Errorf("failed to open PDH query: %d", ret)
	}

	// Add common performance counters
	counters := map[string]string{
		"cpu_usage":    `\Processor(_Total)\% Processor Time`,
		"memory_total": `\Memory\Available Bytes`,
		"disk_time":    `\PhysicalDisk(_Total)\% Disk Time`,
	}

	for name, path := range counters {
		var counter uintptr
		pathPtr, _ := syscall.UTF16PtrFromString(path)
		ret, _, _ := pdhAddCounter.Call(pm.query, uintptr(unsafe.Pointer(pathPtr)), 0, uintptr(unsafe.Pointer(&counter)))
		if ret == ERROR_SUCCESS {
			pm.counters[name] = counter
		}
	}

	pm.initialized = true
	return nil
}

func (pm *PerfCounterManager) GetValue(counterName string) (float64, error) {
	pm.mutex.RLock()
	counter, exists := pm.counters[counterName]
	pm.mutex.RUnlock()

	if !exists {
		return 0, fmt.Errorf("counter %s not found", counterName)
	}

	// Collect data
	ret, _, _ := pdhCollectQueryData.Call(pm.query)
	if ret != ERROR_SUCCESS {
		return 0, fmt.Errorf("failed to collect query data: %d", ret)
	}

	// Get formatted value
	var value PDH_FMT_COUNTERVALUE
	ret, _, _ = pdhGetFormattedCounterValue.Call(counter, PDH_FMT_DOUBLE, 0, uintptr(unsafe.Pointer(&value)))
	if ret != ERROR_SUCCESS {
		return 0, fmt.Errorf("failed to get formatted counter value: %d", ret)
	}

	return value.DoubleValue, nil
}

func getRealCPUTemperature() float64 {
	return hwMonitor.GetCPUTemperature()
}

func getRealCPUFrequency() (float64, float64) {
	return hwMonitor.GetCPUFrequency()
}

func getRealGPUTemperature() float64 {
	_, temp, _ := hwMonitor.GetGPUInfo()
	return temp
}

func getRealGPUUsage() float64 {
	usage, _, _ := hwMonitor.GetGPUInfo()
	return usage
}

func getRealGPUFrequency() float64 {
	_, _, freq := hwMonitor.GetGPUInfo()
	return freq
}

func getDiskTemperature() float64 {
	return hwMonitor.GetDiskTemperature()
}

func getFanInfo() []FanInfo {
	// Fan information requires specialized hardware monitoring
	return []FanInfo{
		{Name: "CPU Fan", Speed: 1200},
		{Name: "Case Fan", Speed: 800},
	}
}

type NetworkInfoData struct {
	IP            string
	UploadSpeed   float64
	DownloadSpeed float64
}

var (
	lastNetTime  time.Time
	lastNetStats map[string]uint64
)

func getNetworkInfo() NetworkInfoData {
	info := NetworkInfoData{}

	// Get IP address
	interfaces, err := net.Interfaces()
	if err == nil {
		for _, iface := range interfaces {
			if iface.Flags&net.FlagUp != 0 && iface.Flags&net.FlagLoopback == 0 {
				addrs, err := iface.Addrs()
				if err == nil {
					for _, addr := range addrs {
						if ipnet, ok := addr.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
							if ipnet.IP.To4() != nil {
								info.IP = ipnet.IP.String()
								break
							}
						}
					}
				}
				if info.IP != "" {
					break
				}
			}
		}
	}

	// Network speed monitoring on Windows requires performance counters
	// For now, return simulated values
	info.UploadSpeed = 0.5
	info.DownloadSpeed = 2.1

	return info
}

func getGPUFPS() float64 {
	return hwMonitor.GetGPUFPS()
}
