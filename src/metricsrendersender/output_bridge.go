package main

import (
	"metricsrendersender/output"
	"image"
)

type OutputHandler = output.OutputHandler
type OutputManager = output.OutputManager
type OutputConfig = output.OutputConfig
type OutputConfigSummary = output.ConfigSummary
type OutputFrame = output.OutputFrame
type MemImgOutputHandler = output.MemImgOutputHandler
type AX206USBOutputHandler = output.AX206USBOutputHandler
type OutputRuntimeStats = output.OutputRuntimeStats
type OutputHandlerRuntimeStats = output.OutputHandlerRuntimeStats
type AX206DeviceFrameRuntimeStats = output.AX206DeviceFrameRuntimeStats
type TCPPushAvailabilityStats = output.TCPPushAvailabilityStats

func NewOutputManager() *OutputManager {
	return output.NewOutputManager()
}

func NewOutputFrame(img image.Image) *OutputFrame {
	return output.NewOutputFrame(img)
}

func NewMemImgOutputHandler() *MemImgOutputHandler {
	return output.NewMemImgOutputHandler()
}

func SetMemImgPNG(data []byte) {
	output.SetMemImgPNG(data)
}

func GetMemImgPNG() ([]byte, bool) {
	return output.GetMemImgPNG()
}

func GetMemImgPNGSize() int {
	return output.GetMemImgPNGSize()
}

func GetOutputRuntimeStats() OutputRuntimeStats {
	return output.GetRuntimeStats()
}

func GetAX206DeviceFrameRuntimeStats() AX206DeviceFrameRuntimeStats {
	return output.GetAX206DeviceFrameRuntimeStats()
}

func GetHTTPPushRuntimeStats() map[string]OutputHandlerRuntimeStats {
	return output.GetHTTPPushRuntimeStats()
}

func GetTCPPushRuntimeStats() map[string]OutputHandlerRuntimeStats {
	return output.GetTCPPushRuntimeStats()
}

func GetTCPPushAvailabilityStats() map[string]TCPPushAvailabilityStats {
	return output.GetTCPPushAvailabilityStats()
}
