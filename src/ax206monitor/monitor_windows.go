//go:build windows

package main

import (
	"net"
	"runtime"
)

func tryGetLibreHardwareMonitorClient() *LibreHardwareMonitorClient {
	config := GetGlobalMonitorConfig()
	if config != nil && config.LibreHardwareMonitorURL != "" {
		return GetLibreHardwareMonitorClient(config.LibreHardwareMonitorURL)
	}
	return nil
}

func getCPUUsage() float64 {
	if client := tryGetLibreHardwareMonitorClient(); client != nil {
		if err := client.FetchData(); err == nil {
			data := client.GetData()
			if data.CPUUsage > 0 {
				return data.CPUUsage
			}
		}
	}
	return 0.0
}

func getRealCPUTemperature() float64 {
	if client := tryGetLibreHardwareMonitorClient(); client != nil {
		if err := client.FetchData(); err == nil {
			data := client.GetData()
			if data.CPUTemp > 0 {
				return data.CPUTemp
			}
		}
	}
	return 0.0
}

func getRealCPUFrequency() (float64, float64) {
	if client := tryGetLibreHardwareMonitorClient(); client != nil {
		if err := client.FetchData(); err == nil {
			data := client.GetData()
			if data.CPUFreq > 0 {
				return data.CPUFreq, data.CPUFreq * 1.2
			}
		}
	}
	return 0.0, 0.0
}

func getRealGPUTemperature() float64 {
	if client := tryGetLibreHardwareMonitorClient(); client != nil {
		if err := client.FetchData(); err == nil {
			data := client.GetData()
			if data.GPUTemp > 0 {
				return data.GPUTemp
			}
		}
	}
	return 0.0
}

func getRealGPUUsage() float64 {
	if client := tryGetLibreHardwareMonitorClient(); client != nil {
		if err := client.FetchData(); err == nil {
			data := client.GetData()
			if data.GPUUsage > 0 {
				return data.GPUUsage
			}
		}
	}
	return 0.0
}

func getRealGPUFrequency() float64 {
	if client := tryGetLibreHardwareMonitorClient(); client != nil {
		if err := client.FetchData(); err == nil {
			data := client.GetData()
			if data.GPUFreq > 0 {
				return data.GPUFreq
			}
		}
	}
	return 0.0
}

func getMemoryInfo() (total float64, used float64, usagePercent float64) {
	if client := tryGetLibreHardwareMonitorClient(); client != nil {
		if err := client.FetchData(); err == nil {
			data := client.GetData()
			if data.MemoryTotal > 0 {
				return data.MemoryTotal, data.MemoryUsed, data.MemoryUsage
			}
		}
	}
	return 16.0, 8.0, 50.0
}

func getFanInfo() []FanInfo {
	if client := tryGetLibreHardwareMonitorClient(); client != nil {
		if err := client.FetchData(); err == nil {
			data := client.GetData()
			if len(data.Fans) > 0 {
				return data.Fans
			}
		}
	}
	return []FanInfo{
		{Name: "CPU Fan", Speed: 1200, Index: 1},
		{Name: "Case Fan", Speed: 800, Index: 2},
	}
}

type NetworkInfoData struct {
	IP            string
	UploadSpeed   float64
	DownloadSpeed float64
}

type SystemInfo struct {
	OS           string
	Architecture string
	CPUCores     int
	Hostname     string
}

func getNetworkInfo() NetworkInfoData {
	info := NetworkInfoData{}

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

	if client := tryGetLibreHardwareMonitorClient(); client != nil {
		if err := client.FetchData(); err == nil {
			data := client.GetData()
			if data.NetworkUpload > 0 || data.NetworkDownload > 0 {
				info.UploadSpeed = data.NetworkUpload
				info.DownloadSpeed = data.NetworkDownload
				return info
			}
		}
	}

	info.UploadSpeed = 0.5
	info.DownloadSpeed = 2.1
	return info
}

func getDiskInfo() []DiskInfo {
	return []DiskInfo{
		{Name: "C:", Usage: 65.0, Size: 500, Model: "Windows Disk"},
	}
}

func getSystemInfo() SystemInfo {
	return SystemInfo{
		OS:           "Windows",
		Architecture: runtime.GOARCH,
		CPUCores:     runtime.NumCPU(),
		Hostname:     "Windows-PC",
	}
}

func initializeSystemSpecific() {
}

func cleanupSystemSpecific() {
}

func getGPUFPS() float64 {
	return 0.0
}

func getDiskTemperature() float64 {
	return 0.0
}
