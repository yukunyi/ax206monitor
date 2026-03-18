package main

import (
	"metricsrendersender/monitorutil"
	"time"
)

type CachedSensorPath = monitorutil.CachedSensorPath
type MonitorDataCache = monitorutil.MonitorDataCache
type NetworkInfoData = monitorutil.NetworkInfoData

var (
	CPUSensorPatterns  = monitorutil.CPUSensorPatterns
	GPUSensorPatterns  = monitorutil.GPUSensorPatterns
	DiskSensorPatterns = monitorutil.DiskSensorPatterns
)

const (
	CPUTempMin  = monitorutil.CPUTempMin
	CPUTempMax  = monitorutil.CPUTempMax
	GPUTempMin  = monitorutil.GPUTempMin
	GPUTempMax  = monitorutil.GPUTempMax
	DiskTempMin = monitorutil.DiskTempMin
	DiskTempMax = monitorutil.DiskTempMax
	CPUFreqMin  = monitorutil.CPUFreqMin
	CPUFreqMax  = monitorutil.CPUFreqMax
	GPUFreqMin  = monitorutil.GPUFreqMin
	GPUFreqMax  = monitorutil.GPUFreqMax
)

func NewCachedSensorPath(checkPeriod time.Duration) *CachedSensorPath {
	return monitorutil.NewCachedSensorPath(checkPeriod)
}

func NewMonitorDataCache(ttl time.Duration) *MonitorDataCache {
	return monitorutil.NewMonitorDataCache(ttl)
}

func readSysFile(path string) (string, error) {
	return monitorutil.ReadSysFile(path)
}

func readSysFileInt(path string) (int, error) {
	return monitorutil.ReadSysFileInt(path)
}

func findHwmonSensor(namePatterns []string, tempFile string) (string, float64, error) {
	return monitorutil.FindHwmonSensor(namePatterns, tempFile)
}
