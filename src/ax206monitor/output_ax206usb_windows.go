//go:build windows

package main

import (
	"fmt"
	"image"
	"sync"
	"time"
)

type AX206USBOutputHandler struct {
	mutex     sync.Mutex
	connected bool
	lastError time.Time
}

func NewAX206USBOutputHandler() (*AX206USBOutputHandler, error) {
	return &AX206USBOutputHandler{
		connected: false,
	}, nil
}

func (h *AX206USBOutputHandler) GetType() string {
	return "ax206usb"
}

func (h *AX206USBOutputHandler) Output(img image.Image) error {
	h.mutex.Lock()
	defer h.mutex.Unlock()

	if time.Since(h.lastError) > 10*time.Second {
		logWarnModule("ax206usb", "AX206 USB output not supported on Windows, use file output instead")
		h.lastError = time.Now()
	}

	return fmt.Errorf("AX206 USB output not supported on Windows")
}

func (h *AX206USBOutputHandler) Close() error {
	return nil
}
