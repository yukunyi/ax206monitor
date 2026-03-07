package main

import "ax206monitor/output"

type OutputHandler = output.OutputHandler
type OutputManager = output.OutputManager
type MemImgOutputHandler = output.MemImgOutputHandler
type AX206USBOutputHandler = output.AX206USBOutputHandler
type OutputRuntimeStats = output.OutputRuntimeStats
type OutputHandlerRuntimeStats = output.OutputHandlerRuntimeStats

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
