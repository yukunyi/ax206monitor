//go:build windows

package main

import "runtime"

func detectCPUInfo() *CPUInfo {
	return &CPUInfo{
		Model:        "Windows CPU",
		Cores:        runtime.NumCPU(),
		Threads:      runtime.NumCPU(),
		Architecture: runtime.GOARCH,
		Vendor:       "unknown",
	}
}

func detectGPUInfo() *GPUInfo {
	return &GPUInfo{
		Model:  "Windows GPU",
		Vendor: "unknown",
	}
}

func detectDiskInfo() []*DiskInfo {
	return []*DiskInfo{}
}
