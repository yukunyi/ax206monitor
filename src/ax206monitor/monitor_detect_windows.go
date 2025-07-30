//go:build windows

package main

import (
	"runtime"
	"syscall"
	"unsafe"
)

var (
	kernel32            = syscall.NewLazyDLL("kernel32.dll")
	procGetSystemInfo   = kernel32.NewProc("GetSystemInfo")
	procGetComputerName = kernel32.NewProc("GetComputerNameW")
)

type systemInfo struct {
	ProcessorArchitecture     uint16
	Reserved                  uint16
	PageSize                  uint32
	MinimumApplicationAddress uintptr
	MaximumApplicationAddress uintptr
	ActiveProcessorMask       uintptr
	NumberOfProcessors        uint32
	ProcessorType             uint32
	AllocationGranularity     uint32
	ProcessorLevel            uint16
	ProcessorRevision         uint16
}

func detectCPUInfo() *CPUInfo {
	cpuInfo := &CPUInfo{
		Model:        "Unknown CPU",
		Cores:        runtime.NumCPU(),
		Threads:      runtime.NumCPU(),
		Architecture: runtime.GOARCH,
		Vendor:       "unknown",
		MaxFreq:      0,
		MinFreq:      0,
	}

	// Try to get more detailed CPU information via WMI or registry
	// For now, use basic runtime information
	var si systemInfo
	procGetSystemInfo.Call(uintptr(unsafe.Pointer(&si)))

	cpuInfo.Cores = int(si.NumberOfProcessors)
	cpuInfo.Threads = int(si.NumberOfProcessors)

	// Try to get CPU model from environment or registry
	// This is a simplified implementation
	switch runtime.GOARCH {
	case "amd64":
		cpuInfo.Model = "x64 Processor"
		cpuInfo.Architecture = "x64"
	case "386":
		cpuInfo.Model = "x86 Processor"
		cpuInfo.Architecture = "x86"
	case "arm64":
		cpuInfo.Model = "ARM64 Processor"
		cpuInfo.Architecture = "ARM64"
	}

	logInfoModule("cpu", "Detected CPU: %s (%d cores)", cpuInfo.Model, cpuInfo.Cores)
	return cpuInfo
}

func detectGPUInfo() *GPUInfo {
	gpuInfo := &GPUInfo{
		Model:       "Unknown GPU",
		Vendor:      "unknown",
		Memory:      0,
		MemoryUsed:  0,
		FanCount:    0,
		Fans:        []FanInfo{},
		Temperature: 0,
		Usage:       0,
		Frequency:   0,
	}

	// Try to detect GPU via WMI or DirectX
	// For now, return basic information
	// In a real implementation, you would use WMI queries like:
	// SELECT * FROM Win32_VideoController WHERE AdapterCompatibility IS NOT NULL

	logWarnModule("gpu", "GPU detection not fully implemented on Windows, use LibreHardwareMonitor for detailed info")
	return gpuInfo
}

func detectDiskInfo() []*DiskInfo {
	var disks []*DiskInfo

	// Try to detect disks via WMI
	// For now, return empty slice
	// In a real implementation, you would use WMI queries like:
	// SELECT * FROM Win32_DiskDrive
	// SELECT * FROM Win32_LogicalDisk

	logWarnModule("disk", "Disk detection not fully implemented on Windows, use LibreHardwareMonitor for detailed info")
	return disks
}

// getComputerName gets the Windows computer name
func getComputerName() string {
	var size uint32 = 256
	buf := make([]uint16, size)

	ret, _, _ := procGetComputerName.Call(
		uintptr(unsafe.Pointer(&buf[0])),
		uintptr(unsafe.Pointer(&size)),
	)

	if ret != 0 {
		return syscall.UTF16ToString(buf[:size])
	}

	return "Windows-PC"
}
