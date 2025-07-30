//go:build linux

package main

func detectCPUInfo() *CPUInfo {
	return detectLinuxCPUInfo()
}

func detectGPUInfo() *GPUInfo {
	return detectLinuxGPUInfo()
}

func detectDiskInfo() []*DiskInfo {
	return detectLinuxDiskInfo()
}
