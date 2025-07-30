//go:build windows

package main

import (
	"fmt"
	"image"
)

type WindowsUSBDevice struct {
	connected bool
}

func NewUSBDevice() (*WindowsUSBDevice, error) {
	return &WindowsUSBDevice{connected: false}, nil
}

func (d *WindowsUSBDevice) Connect() error {
	d.connected = true
	return nil
}

func (d *WindowsUSBDevice) Disconnect() error {
	d.connected = false
	return nil
}

func (d *WindowsUSBDevice) IsConnected() bool {
	return d.connected
}

func (d *WindowsUSBDevice) SendImage(img image.Image) error {
	if !d.connected {
		return fmt.Errorf("device not connected")
	}
	return nil
}

func (d *WindowsUSBDevice) GetDeviceInfo() (string, error) {
	return "AX206 USB Device (Windows)", nil
}

func FindUSBDevices() ([]*WindowsUSBDevice, error) {
	return []*WindowsUSBDevice{}, nil
}

func InitializeUSB() error {
	return nil
}

func CleanupUSB() {
}
