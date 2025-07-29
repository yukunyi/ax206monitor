package main

import (
	"sync"
	"time"
)

type CacheEntry struct {
	Value     interface{}
	Timestamp time.Time
}

type MonitorCache struct {
	cache     map[string]*CacheEntry
	mutex     sync.RWMutex
	renderID  string
	lastClear time.Time
}

var globalCache *MonitorCache
var cacheMutex sync.Mutex

func GetMonitorCache() *MonitorCache {
	cacheMutex.Lock()
	defer cacheMutex.Unlock()

	if globalCache == nil {
		globalCache = &MonitorCache{
			cache:     make(map[string]*CacheEntry),
			lastClear: time.Now(),
		}
	}
	return globalCache
}

func (mc *MonitorCache) StartRender() string {
	mc.mutex.Lock()
	defer mc.mutex.Unlock()

	renderID := time.Now().Format("20060102150405.000000")
	mc.renderID = renderID

	if time.Since(mc.lastClear) > 100*time.Millisecond {
		mc.cache = make(map[string]*CacheEntry)
		mc.lastClear = time.Now()
	}

	return renderID
}

func (mc *MonitorCache) Get(key string) (interface{}, bool) {
	mc.mutex.RLock()
	defer mc.mutex.RUnlock()

	if entry, exists := mc.cache[key]; exists {
		return entry.Value, true
	}
	return nil, false
}

func (mc *MonitorCache) Set(key string, value interface{}) {
	mc.mutex.Lock()
	defer mc.mutex.Unlock()

	mc.cache[key] = &CacheEntry{
		Value:     value,
		Timestamp: time.Now(),
	}
}

func (mc *MonitorCache) SetMultiple(values map[string]interface{}) {
	mc.mutex.Lock()
	defer mc.mutex.Unlock()

	timestamp := time.Now()
	for key, value := range values {
		mc.cache[key] = &CacheEntry{
			Value:     value,
			Timestamp: timestamp,
		}
	}
}

type CachedDataProvider interface {
	GetCachedData(cache *MonitorCache, requiredKeys []string) map[string]interface{}
}

type CPUDataProvider struct{}

func (p *CPUDataProvider) GetCachedData(cache *MonitorCache, requiredKeys []string) map[string]interface{} {
	result := make(map[string]interface{})
	needsUpdate := false

	for _, key := range requiredKeys {
		if _, exists := cache.Get(key); !exists {
			needsUpdate = true
			break
		}
	}

	if !needsUpdate {
		for _, key := range requiredKeys {
			if value, exists := cache.Get(key); exists {
				result[key] = value
			}
		}
		return result
	}

	initializeCache()

	cpuData := make(map[string]interface{})

	// Batch fetch CPU data to minimize hardware queries
	var temp float64
	var curFreq, maxFreq float64

	tempNeeded := false
	freqNeeded := false

	for _, key := range requiredKeys {
		switch key {
		case "cpu_temp":
			tempNeeded = true
		case "cpu_freq":
			freqNeeded = true
		}
	}

	if tempNeeded {
		temp = getRealCPUTemperature()
		cpuData["cpu_temp"] = temp
	}

	if freqNeeded {
		curFreq, maxFreq = getRealCPUFrequency()
		cpuData["cpu_freq"] = curFreq
		cpuData["cpu_freq_max"] = maxFreq
	}

	cache.SetMultiple(cpuData)

	for _, key := range requiredKeys {
		if value, exists := cpuData[key]; exists {
			result[key] = value
		}
	}

	return result
}

type GPUDataProvider struct{}

func (p *GPUDataProvider) GetCachedData(cache *MonitorCache, requiredKeys []string) map[string]interface{} {
	result := make(map[string]interface{})
	needsUpdate := false

	for _, key := range requiredKeys {
		if _, exists := cache.Get(key); !exists {
			needsUpdate = true
			break
		}
	}

	if !needsUpdate {
		for _, key := range requiredKeys {
			if value, exists := cache.Get(key); exists {
				result[key] = value
			}
		}
		return result
	}

	initializeCache()

	gpuData := make(map[string]interface{})

	for _, key := range requiredKeys {
		switch key {
		case "gpu_temp":
			gpuData[key] = getRealGPUTemperature()
		case "gpu_usage":
			gpuData[key] = getRealGPUUsage()
		case "gpu_freq":
			gpuData[key] = getRealGPUFrequency()
		}
	}

	cache.SetMultiple(gpuData)

	for _, key := range requiredKeys {
		if value, exists := gpuData[key]; exists {
			result[key] = value
		}
	}

	return result
}

type NetworkDataProvider struct{}

func (p *NetworkDataProvider) GetCachedData(cache *MonitorCache, requiredKeys []string) map[string]interface{} {
	result := make(map[string]interface{})
	needsUpdate := false

	for _, key := range requiredKeys {
		if _, exists := cache.Get(key); !exists {
			needsUpdate = true
			break
		}
	}

	if !needsUpdate {
		for _, key := range requiredKeys {
			if value, exists := cache.Get(key); exists {
				result[key] = value
			}
		}
		return result
	}

	networkInfo := getNetworkInfo()
	networkData := map[string]interface{}{
		"network_ip":       networkInfo.IP,
		"network_upload":   networkInfo.UploadSpeed,
		"network_download": networkInfo.DownloadSpeed,
	}

	cache.SetMultiple(networkData)

	for _, key := range requiredKeys {
		if value, exists := networkData[key]; exists {
			result[key] = value
		}
	}

	return result
}

var (
	cpuProvider     = &CPUDataProvider{}
	gpuProvider     = &GPUDataProvider{}
	networkProvider = &NetworkDataProvider{}
)

func GetCachedValue(monitorName string) interface{} {
	cache := GetMonitorCache()

	switch {
	case monitorName == "cpu_temp" || monitorName == "cpu_freq":
		data := cpuProvider.GetCachedData(cache, []string{monitorName})
		return data[monitorName]
	case monitorName == "gpu_temp" || monitorName == "gpu_usage" || monitorName == "gpu_freq":
		data := gpuProvider.GetCachedData(cache, []string{monitorName})
		return data[monitorName]
	case monitorName == "network_ip" || monitorName == "network_upload" || monitorName == "network_download":
		data := networkProvider.GetCachedData(cache, []string{monitorName})
		return data[monitorName]
	default:
		return nil
	}
}
