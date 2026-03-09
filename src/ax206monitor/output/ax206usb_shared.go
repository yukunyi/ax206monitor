package output

import (
	"image"
	"sync"
)

type sharedAX206State struct {
	mu      sync.Mutex
	handler *AX206USBOutputHandler
	refs    int
}

var globalAX206Shared = &sharedAX206State{}

type AX206USBSharedOutputHandler struct {
	mu     sync.Mutex
	closed bool
}

func NewSharedAX206USBOutputHandler() (*AX206USBSharedOutputHandler, error) {
	globalAX206Shared.mu.Lock()
	defer globalAX206Shared.mu.Unlock()

	if globalAX206Shared.handler == nil {
		handler, err := NewAX206USBOutputHandler()
		if err != nil {
			return nil, err
		}
		globalAX206Shared.handler = handler
		logInfoModule("ax206usb", "Handler ready")
	}
	globalAX206Shared.refs++
	return &AX206USBSharedOutputHandler{}, nil
}

func (h *AX206USBSharedOutputHandler) GetType() string {
	return TypeAX206USB
}

func (h *AX206USBSharedOutputHandler) Output(img image.Image) error {
	globalAX206Shared.mu.Lock()
	handler := globalAX206Shared.handler
	globalAX206Shared.mu.Unlock()
	if handler == nil {
		return nil
	}
	return handler.Output(img)
}

func (h *AX206USBSharedOutputHandler) Close() error {
	h.mu.Lock()
	if h.closed {
		h.mu.Unlock()
		return nil
	}
	h.closed = true
	h.mu.Unlock()

	var toClose *AX206USBOutputHandler
	globalAX206Shared.mu.Lock()
	if globalAX206Shared.refs > 0 {
		globalAX206Shared.refs--
	}
	if globalAX206Shared.refs == 0 && globalAX206Shared.handler != nil {
		toClose = globalAX206Shared.handler
		globalAX206Shared.handler = nil
	}
	globalAX206Shared.mu.Unlock()

	if toClose != nil {
		return toClose.Close()
	}
	return nil
}
