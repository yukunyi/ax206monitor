package main

import (
	"ax206monitor/output"
	"time"
)

type OutputHandler = output.OutputHandler
type OutputManager = output.OutputManager
type MemImgOutputHandler = output.MemImgOutputHandler
type AX206USBOutputHandler = output.AX206USBOutputHandler
type OutputRuntimeStats = output.OutputRuntimeStats
type OutputHandlerRuntimeStats = output.OutputHandlerRuntimeStats
type AX206DeviceFrameRuntimeStats = output.AX206DeviceFrameRuntimeStats

func NewOutputManager() *OutputManager {
	return output.NewOutputManager()
}

func NewMemImgOutputHandler() *MemImgOutputHandler {
	return output.NewMemImgOutputHandler()
}

func NewAX206USBOutputHandler() (*AX206USBOutputHandler, error) {
	return output.NewAX206USBOutputHandler()
}

func SetMemImgPNG(data []byte) {
	output.SetMemImgPNG(data)
}

func GetMemImgPNG() ([]byte, bool) {
	return output.GetMemImgPNG()
}

func GetOutputRuntimeStats() OutputRuntimeStats {
	return output.GetRuntimeStats()
}

func GetAX206DeviceFrameRuntimeStats() AX206DeviceFrameRuntimeStats {
	return output.GetAX206DeviceFrameRuntimeStats()
}

func SetAX206ReconnectInterval(interval time.Duration) {
	output.SetAX206ReconnectInterval(interval)
}
