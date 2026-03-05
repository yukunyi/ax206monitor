package main

import (
	"net"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/host"
	gopsutilNet "github.com/shirou/gopsutil/v3/net"
)

type netRateSnapshot struct {
	counter gopsutilNet.IOCountersStat
	at      time.Time
}

var (
	networkRateMu    sync.Mutex
	networkRateCache = make(map[string]netRateSnapshot)
)

func getRealCPUTemperature() float64 {
	if temp := getTemperatureByKeywords([]string{"cpu", "package", "core", "tctl", "ccd"}); temp > 0 {
		return temp
	}
	if data, ok := getLibreHardwareMonitorData(); ok && data.CPUTemp > 0 {
		return data.CPUTemp
	}
	return 0
}

func getRealCPUFrequency() (float64, float64) {
	if current, maxFreq, ok := getCPUFrequencyByGopsutil(); ok {
		return current, maxFreq
	}
	if data, ok := getLibreHardwareMonitorData(); ok && data.CPUFreq > 0 {
		return data.CPUFreq, data.CPUFreq
	}
	return 0, 0
}

func getCPUFrequencyByGopsutil() (float64, float64, bool) {
	infos, err := cpu.Info()
	if err != nil || len(infos) == 0 {
		return 0, 0, false
	}

	var (
		total float64
		maxV  float64
		count int
	)
	for _, item := range infos {
		if item.Mhz <= 0 {
			continue
		}
		total += item.Mhz
		count++
		if item.Mhz > maxV {
			maxV = item.Mhz
		}
	}
	if count == 0 || maxV <= 0 {
		return 0, 0, false
	}
	return total / float64(count), maxV, true
}

func getDiskTemperature() float64 {
	if temp := getTemperatureByKeywords([]string{"nvme", "disk", "drive", "storage", "ssd", "hdd"}); temp > 0 {
		return temp
	}
	return 0
}

func getNetworkInfo() NetworkInfoData {
	info := NetworkInfoData{}
	ifaceName, ip := getPrimaryIPv4Interface()
	info.IP = ip
	if ifaceName != "" {
		upload, download, ok := getNetworkSpeedByInterface(ifaceName)
		if ok {
			info.UploadSpeed = upload
			info.DownloadSpeed = download
			return info
		}
	}

	if data, ok := getLibreHardwareMonitorData(); ok {
		if info.UploadSpeed == 0 {
			info.UploadSpeed = data.NetworkUpload
		}
		if info.DownloadSpeed == 0 {
			info.DownloadSpeed = data.NetworkDownload
		}
	}

	return info
}

func getPrimaryIPv4Interface() (string, string) {
	interfaces, err := net.Interfaces()
	if err != nil {
		return "", ""
	}

	sort.Slice(interfaces, func(i, j int) bool {
		return interfaces[i].Index < interfaces[j].Index
	})

	for _, iface := range interfaces {
		name := strings.TrimSpace(iface.Name)
		if name == "" {
			continue
		}
		if iface.Flags&net.FlagUp == 0 || iface.Flags&net.FlagLoopback != 0 {
			continue
		}
		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}
		for _, addr := range addrs {
			ipnet, ok := addr.(*net.IPNet)
			if !ok || ipnet == nil || ipnet.IP == nil || ipnet.IP.IsLoopback() {
				continue
			}
			if ip4 := ipnet.IP.To4(); ip4 != nil {
				return name, ip4.String()
			}
		}
	}
	return "", ""
}

func getNetworkSpeedByInterface(interfaceName string) (float64, float64, bool) {
	stats, err := gopsutilNet.IOCounters(true)
	if err != nil {
		return 0, 0, false
	}
	current, ok := findNetworkCounter(stats, interfaceName)
	if !ok {
		return 0, 0, false
	}

	now := time.Now()
	networkRateMu.Lock()
	previous, hasPrevious := networkRateCache[interfaceName]
	networkRateCache[interfaceName] = netRateSnapshot{counter: current, at: now}
	networkRateMu.Unlock()

	if !hasPrevious || previous.at.IsZero() {
		return 0, 0, false
	}
	seconds := now.Sub(previous.at).Seconds()
	if seconds <= 0 {
		return 0, 0, false
	}
	if current.BytesSent < previous.counter.BytesSent || current.BytesRecv < previous.counter.BytesRecv {
		return 0, 0, false
	}

	upload := float64(current.BytesSent-previous.counter.BytesSent) / seconds / 1024 / 1024
	download := float64(current.BytesRecv-previous.counter.BytesRecv) / seconds / 1024 / 1024
	return upload, download, true
}

func findNetworkCounter(stats []gopsutilNet.IOCountersStat, interfaceName string) (gopsutilNet.IOCountersStat, bool) {
	for _, item := range stats {
		if item.Name == interfaceName {
			return item, true
		}
	}
	return gopsutilNet.IOCountersStat{}, false
}

func getTemperatureByKeywords(keywords []string) float64 {
	temps, err := host.SensorsTemperatures()
	if err != nil {
		return 0
	}
	maxTemp := 0.0
	for _, stat := range temps {
		key := strings.ToLower(strings.TrimSpace(stat.SensorKey))
		if key == "" {
			continue
		}
		if stat.Temperature <= 0 || stat.Temperature > 130 {
			continue
		}
		for _, keyword := range keywords {
			if strings.Contains(key, keyword) {
				if stat.Temperature > maxTemp {
					maxTemp = stat.Temperature
				}
				break
			}
		}
	}
	return maxTemp
}
