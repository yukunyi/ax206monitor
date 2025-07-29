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
	usage := getRealGPUUsage()
	if usage >= 0 {
		g.SetValue(usage)
		g.SetAvailable(true)
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
	temp := getRealGPUTemperature()
	if temp > 0 {
		g.SetValue(temp)
		g.SetAvailable(true)
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
	freq := getRealGPUFrequency()
	if freq > 0 {
		g.SetValue(freq)
		g.SetAvailable(true)
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
