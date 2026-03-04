package output

import (
	"image"
	"sync"
	"sync/atomic"
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
	if len(om.handlers) <= 1 {
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

	type handlerResult struct {
		handlerType string
		err         error
	}

	results := make(chan handlerResult, len(om.handlers))
	var hasSuccess int32
	var wg sync.WaitGroup

	for _, handler := range om.handlers {
		h := handler
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := h.Output(img); err != nil {
				results <- handlerResult{handlerType: h.GetType(), err: err}
				return
			}
			atomic.StoreInt32(&hasSuccess, 1)
		}()
	}
	wg.Wait()
	close(results)

	var lastErr error
	for result := range results {
		logWarnModule("output", "%s failed: %v", result.handlerType, result.err)
		lastErr = result.err
	}

	if atomic.LoadInt32(&hasSuccess) == 0 && lastErr != nil {
		return lastErr
	}

	return nil
}

func (om *OutputManager) Close() {
	for _, handler := range om.handlers {
		handler.Close()
	}
}
