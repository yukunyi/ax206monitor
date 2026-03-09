package main

import "runtime"

const (
	collectorGoNativeCPU     = "go_native.cpu"
	collectorGoNativeMemory  = "go_native.memory"
	collectorGoNativeSystem  = "go_native.system"
	collectorGoNativeDisk    = "go_native.disk"
	collectorGoNativeNetwork = "go_native.network"
	collectorCustomAll       = "custom.all"

	collectorCoolerControl        = "coolercontrol"
	collectorLibreHardwareMonitor = "librehardwaremonitor"
	collectorRTSS                 = "rtss"

	legacyCollectorCoolerControl        = "external.coolercontrol"
	legacyCollectorLibreHardwareMonitor = "external.librehardwaremonitor"
	legacyCollectorRTSS                 = "external.rtss"
)

func isCollectorSupportedOnCurrentPlatform(name string) bool {
	switch name {
	case collectorCoolerControl:
		return runtime.GOOS == "linux"
	case collectorLibreHardwareMonitor:
		return runtime.GOOS == "windows"
	case collectorRTSS:
		return runtime.GOOS == "windows"
	default:
		return true
	}
}
