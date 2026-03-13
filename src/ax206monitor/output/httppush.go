package output

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"
)

type HTTPPushOutputHandler struct {
	cfg      OutputConfig
	typeName string
	client   *http.Client

	stopOnce sync.Once
	stopCh   chan struct{}
	loopWg   sync.WaitGroup
	frameCh  chan *OutputFrame

	lastErrorMu sync.Mutex
	lastErrorAt time.Time
}

func NewHTTPPushOutputHandler(cfg OutputConfig, typeName string) *HTTPPushOutputHandler {
	handler := &HTTPPushOutputHandler{
		cfg:      cfg,
		typeName: typeName,
		client: &http.Client{
			Timeout: 5 * time.Second,
		},
		stopCh:  make(chan struct{}),
		frameCh: make(chan *OutputFrame, 1),
	}
	handler.loopWg.Add(1)
	go handler.loop()
	return handler
}

func (h *HTTPPushOutputHandler) GetType() string {
	return h.typeName
}

func (h *HTTPPushOutputHandler) OutputFrame(frame *OutputFrame) error {
	if frame == nil {
		return nil
	}
	enqueueLatestHTTPPushFrame(h.frameCh, frame)
	return nil
}

func (h *HTTPPushOutputHandler) Close() error {
	h.stopOnce.Do(func() {
		close(h.stopCh)
		h.loopWg.Wait()
	})
	return nil
}

func (h *HTTPPushOutputHandler) loop() {
	defer h.loopWg.Done()
	for {
		select {
		case <-h.stopCh:
			return
		case frame := <-h.frameCh:
			h.push(frame)
		}
	}
}

func (h *HTTPPushOutputHandler) push(frame *OutputFrame) {
	if frame == nil {
		return
	}
	body, contentType, encodeErr := h.encodeFrame(frame)
	if encodeErr != nil {
		h.logError("encode failed: %v", encodeErr)
		recordHTTPPushRuntime(h.typeName, 0, encodeErr)
		return
	}

	startedAt := time.Now()
	err := h.doRequest(body, contentType)
	recordHTTPPushRuntime(h.typeName, time.Since(startedAt), err)
	if err != nil {
		h.logError("push failed: %v", err)
	}
}

func (h *HTTPPushOutputHandler) doRequest(body []byte, contentType string) error {
	if strings.TrimSpace(h.cfg.URL) == "" {
		return fmt.Errorf("url is empty")
	}
	ctx, cancel := context.WithTimeout(context.Background(), h.client.Timeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, h.cfg.URL, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", contentType)

	resp, err := h.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, resp.Body)
	if resp.StatusCode >= http.StatusBadRequest {
		return fmt.Errorf("unexpected status: %s", resp.Status)
	}
	return nil
}

func (h *HTTPPushOutputHandler) encodeFrame(frame *OutputFrame) ([]byte, string, error) {
	switch h.cfg.Format {
	case "png":
		data, err := frame.PNG()
		if err != nil {
			return nil, "", err
		}
		return data, "image/png", nil
	default:
		data, err := frame.JPEG(h.cfg.Quality)
		if err != nil {
			return nil, "", err
		}
		return data, "image/jpeg", nil
	}
}

func enqueueLatestHTTPPushFrame(ch chan *OutputFrame, frame *OutputFrame) {
	select {
	case ch <- frame:
		return
	default:
	}
	select {
	case <-ch:
	default:
	}
	select {
	case ch <- frame:
	default:
	}
}

func (h *HTTPPushOutputHandler) logError(format string, args ...interface{}) {
	h.lastErrorMu.Lock()
	defer h.lastErrorMu.Unlock()
	if time.Since(h.lastErrorAt) < 3*time.Second {
		return
	}
	h.lastErrorAt = time.Now()
	logWarnModule("httppush", format, args...)
}
