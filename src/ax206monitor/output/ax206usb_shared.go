package output

import (
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

func NewSharedAX206USBOutputHandler(cfg OutputConfig) (*AX206USBSharedOutputHandler, error) {
	globalAX206Shared.mu.Lock()
	defer globalAX206Shared.mu.Unlock()

	if globalAX206Shared.handler == nil {
		handler, err := NewAX206USBOutputHandler(cfg)
		if err != nil {
			return nil, err
		}
		globalAX206Shared.handler = handler
	} else {
		globalAX206Shared.handler.UpdateConfig(cfg)
	}
	globalAX206Shared.refs++
	return &AX206USBSharedOutputHandler{}, nil
}

func (h *AX206USBSharedOutputHandler) GetType() string {
	return TypeAX206USB
}

func (h *AX206USBSharedOutputHandler) OutputFrame(frame *OutputFrame) error {
	globalAX206Shared.mu.Lock()
	handler := globalAX206Shared.handler
	globalAX206Shared.mu.Unlock()
	if handler == nil || frame == nil {
		return nil
	}
	return handler.OutputFrame(frame)
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
