package main

import (
	"image"
)

type OutputHandler interface {
	Output(img image.Image) error
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

func (om *OutputManager) Output(img image.Image) error {
	var lastErr error
	hasSuccess := false

	for _, handler := range om.handlers {
		if err := handler.Output(img); err != nil {
			logWarnModule("output", "%s failed: %v", handler.GetType(), err)
			lastErr = err
		} else {
			hasSuccess = true
		}
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
