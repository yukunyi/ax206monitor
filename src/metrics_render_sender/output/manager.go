package output

import "time"

type OutputHandler interface {
	OutputFrame(frame *OutputFrame) error
	Close() error
	GetType() string
}

type OutputManager struct {
	handlers []OutputHandler
}

func NewOutputManager() *OutputManager {
	return &OutputManager{
		handlers: make([]OutputHandler, 0),
	}
}

func (om *OutputManager) AddHandler(handler OutputHandler) {
	om.handlers = append(om.handlers, handler)
}

func (om *OutputManager) OutputFrame(frame *OutputFrame) error {
	var hasSuccess bool
	var lastErr error
	for _, handler := range om.handlers {
		startedAt := time.Now()
		err := handler.OutputFrame(frame)
		duration := time.Since(startedAt)
		recordOutputRuntime(handler.GetType(), duration, err)
		if err != nil {
			logWarnModule("output", "%s failed: %v", handler.GetType(), err)
			lastErr = err
			continue
		}
		hasSuccess = true
	}
	if !hasSuccess && lastErr != nil {
		return lastErr
	}
	return nil
}

func (om *OutputManager) Close() {
	for _, handler := range om.handlers {
		handler.Close()
	}
}
