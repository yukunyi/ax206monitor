package main

type GPUUsageMonitor struct {
	*BaseMonitorItem
}

func NewGPUUsageMonitor() *GPUUsageMonitor {
	return &GPUUsageMonitor{
		BaseMonitorItem: NewBaseMonitorItem(
			"gpu_usage",
			"GPU",
			0, 100,
			"%",
			0,
		),
	}
}

func (g *GPUUsageMonitor) Update() error {
	if cachedValue := GetCachedValue("gpu_usage"); cachedValue != nil {
		if usage, ok := cachedValue.(float64); ok && usage >= 0 {
			g.SetValue(usage)
			g.SetAvailable(true)
		} else {
			g.SetAvailable(false)
		}
	} else {
		g.SetAvailable(false)
	}

	return nil
}

type GPUTempMonitor struct {
	*BaseMonitorItem
}

func NewGPUTempMonitor() *GPUTempMonitor {
	return &GPUTempMonitor{
		BaseMonitorItem: NewBaseMonitorItem(
			"gpu_temp",
			"GPU Temp",
			0, 100,
			"°C",
			0,
		),
	}
}

func (g *GPUTempMonitor) Update() error {
	if cachedValue := GetCachedValue("gpu_temp"); cachedValue != nil {
		if temp, ok := cachedValue.(float64); ok && temp > 0 {
			g.SetValue(temp)
			g.SetAvailable(true)
		} else {
			g.SetAvailable(false)
		}
	} else {
		g.SetAvailable(false)
	}

	return nil
}

type GPUFreqMonitor struct {
	*BaseMonitorItem
}

func NewGPUFreqMonitor() *GPUFreqMonitor {
	return &GPUFreqMonitor{
		BaseMonitorItem: NewBaseMonitorItem(
			"gpu_freq",
			"GPU Freq",
			0, 0,
			"MHz",
			0,
		),
	}
}

func (g *GPUFreqMonitor) Update() error {
	if cachedValue := GetCachedValue("gpu_freq"); cachedValue != nil {
		if freq, ok := cachedValue.(float64); ok && freq > 0 {
			g.SetValue(freq)
			g.SetAvailable(true)
		} else {
			g.SetAvailable(false)
		}
	} else {
		g.SetAvailable(false)
	}

	return nil
}

type GPUFPSMonitor struct {
	*BaseMonitorItem
}

func NewGPUFPSMonitor() *GPUFPSMonitor {
	return &GPUFPSMonitor{
		BaseMonitorItem: NewBaseMonitorItem(
			"gpu_fps",
			"GPU FPS",
			0, 240,
			"FPS",
			0,
		),
	}
}

func (g *GPUFPSMonitor) Update() error {
	fps := getGPUFPS()
	if fps > 0 {
		g.SetValue(fps)
		g.SetAvailable(true)
	} else {
		g.SetAvailable(false)
	}

	return nil
}

// GPUModelMonitor displays GPU model information
type GPUModelMonitor struct {
	*BaseMonitorItem
}

func NewGPUModelMonitor() *GPUModelMonitor {
	return &GPUModelMonitor{
		BaseMonitorItem: NewBaseMonitorItem(
			"gpu_model",
			"GPU Model",
			0, 0,
			"",
			0,
		),
	}
}

func (g *GPUModelMonitor) Update() error {
	initializeCache()
	if cachedGPUInfo != nil {
		g.SetValue(cachedGPUInfo.Model)
		g.SetAvailable(true)
	} else {
		g.SetAvailable(false)
	}
	return nil
}

// GPUMemoryTotalMonitor displays total GPU memory
type GPUMemoryTotalMonitor struct {
	*BaseMonitorItem
}

func NewGPUMemoryTotalMonitor() *GPUMemoryTotalMonitor {
	return &GPUMemoryTotalMonitor{
		BaseMonitorItem: NewBaseMonitorItem(
			"gpu_memory_total",
			"GPU Memory",
			0, 0,
			"MB",
			0,
		),
	}
}

func (g *GPUMemoryTotalMonitor) Update() error {
	initializeCache()
	if cachedGPUInfo != nil && cachedGPUInfo.Memory > 0 {
		g.SetValue(cachedGPUInfo.Memory)
		g.SetAvailable(true)
	} else {
		g.SetAvailable(false)
	}
	return nil
}

// GPUMemoryUsedMonitor displays used GPU memory
type GPUMemoryUsedMonitor struct {
	*BaseMonitorItem
}

func NewGPUMemoryUsedMonitor() *GPUMemoryUsedMonitor {
	return &GPUMemoryUsedMonitor{
		BaseMonitorItem: NewBaseMonitorItem(
			"gpu_memory_used",
			"GPU Mem Used",
			0, 0,
			"MB",
			0,
		),
	}
}

func (g *GPUMemoryUsedMonitor) Update() error {
	initializeCache()
	if cachedGPUInfo != nil && cachedGPUInfo.MemoryUsed > 0 {
		g.SetValue(cachedGPUInfo.MemoryUsed)
		g.SetAvailable(true)
	} else {
		g.SetAvailable(false)
	}
	return nil
}

// GPUMemoryUsageMonitor displays GPU memory usage percentage
type GPUMemoryUsageMonitor struct {
	*BaseMonitorItem
}

func NewGPUMemoryUsageMonitor() *GPUMemoryUsageMonitor {
	return &GPUMemoryUsageMonitor{
		BaseMonitorItem: NewBaseMonitorItem(
			"gpu_memory_usage",
			"GPU Mem Usage",
			0, 100,
			"%",
			1,
		),
	}
}

func (g *GPUMemoryUsageMonitor) Update() error {
	initializeCache()
	if cachedGPUInfo != nil && cachedGPUInfo.Memory > 0 && cachedGPUInfo.MemoryUsed > 0 {
		usage := float64(cachedGPUInfo.MemoryUsed) / float64(cachedGPUInfo.Memory) * 100
		g.SetValue(usage)
		g.SetAvailable(true)
	} else {
		g.SetAvailable(false)
	}
	return nil
}
