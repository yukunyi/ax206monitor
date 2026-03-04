package main

import "ax206monitor/output"

type OutputHandler = output.OutputHandler
type OutputManager = output.OutputManager
type MemImgOutputHandler = output.MemImgOutputHandler
type AX206USBOutputHandler = output.AX206USBOutputHandler

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
