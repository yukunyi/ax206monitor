package main

// NewGPUUsageMonitor creates a GPU usage monitor using the factory
func NewGPUUsageMonitor() MonitorItem {
	return CreateCachedValueMonitor("gpu_usage", "GPU", "%", 0, 100, 0, "gpu_usage")
}

// NewGPUTempMonitor creates a GPU temperature monitor using the factory
func NewGPUTempMonitor() MonitorItem {
	return CreateCachedValueMonitor("gpu_temp", "GPU Temp", "°C", 0, 100, 0, "gpu_temp")
}

// NewGPUFreqMonitor creates a GPU frequency monitor using the factory
func NewGPUFreqMonitor() MonitorItem {
	return CreateCachedValueMonitor("gpu_freq", "GPU Freq", "MHz", 0, 0, 0, "gpu_freq")
}

// NewGPUFPSMonitor creates a GPU FPS monitor using the factory
func NewGPUFPSMonitor() MonitorItem {
	factory := GetMonitorFactory()
	return factory.CreateFrequencyMonitor("gpu_fps", "GPU FPS", func() (float64, bool) {
		fps := getGPUFPS()
		return fps, fps > 0
	})
}

// NewGPUModelMonitor creates a GPU model monitor using the factory
func NewGPUModelMonitor() MonitorItem {
	factory := GetMonitorFactory()
	return factory.CreateStringMonitor("gpu_model", "GPU Model", func() (string, bool) {
		initializeCache()
		if cachedGPUInfo != nil && cachedGPUInfo.Model != "Unknown GPU" {
			return cachedGPUInfo.Model, true
		}
		return "", false
	})
}

// NewGPUMemoryTotalMonitor creates a GPU total memory monitor using the factory
func NewGPUMemoryTotalMonitor() MonitorItem {
	factory := GetMonitorFactory()
	return factory.CreateMemoryMonitor("gpu_memory_total", "GPU Memory", func() (float64, bool) {
		initializeCache()
		if cachedGPUInfo != nil && cachedGPUInfo.Memory > 0 {
			return float64(cachedGPUInfo.Memory), true
		}
		return 0, false
	})
}

// NewGPUMemoryUsedMonitor creates a GPU used memory monitor using the factory
func NewGPUMemoryUsedMonitor() MonitorItem {
	factory := GetMonitorFactory()
	return factory.CreateMemoryMonitor("gpu_memory_used", "GPU Mem Used", func() (float64, bool) {
		initializeCache()
		if cachedGPUInfo != nil && cachedGPUInfo.MemoryUsed > 0 {
			return float64(cachedGPUInfo.MemoryUsed), true
		}
		return 0, false
	})
}

// NewGPUMemoryUsageMonitor creates a GPU memory usage monitor using the factory
func NewGPUMemoryUsageMonitor() MonitorItem {
	factory := GetMonitorFactory()
	return factory.CreateUsageMonitor("gpu_memory_usage", "GPU Mem Usage", func() (float64, bool) {
		initializeCache()
		if cachedGPUInfo != nil && cachedGPUInfo.Memory > 0 && cachedGPUInfo.MemoryUsed > 0 {
			usage := float64(cachedGPUInfo.MemoryUsed) / float64(cachedGPUInfo.Memory) * 100
			return usage, true
		}
		return 0, false
	})
}
