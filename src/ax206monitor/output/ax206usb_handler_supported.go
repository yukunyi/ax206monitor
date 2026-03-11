//go:build linux || (windows && cgo)

package output

import (
	"image"
	"sync"
	"sync/atomic"
	"time"
)

const (
	defaultAX206ReconnectInterval = 3 * time.Second
	minAX206ReconnectInterval     = 100 * time.Millisecond
	maxAX206ReconnectInterval     = 60 * time.Second
)

var ax206ReconnectIntervalNS atomic.Int64

func init() {
	ax206ReconnectIntervalNS.Store(int64(defaultAX206ReconnectInterval))
}

func normalizeAX206ReconnectInterval(interval time.Duration) time.Duration {
	if interval <= 0 {
		return defaultAX206ReconnectInterval
	}
	if interval < minAX206ReconnectInterval {
		return minAX206ReconnectInterval
	}
	if interval > maxAX206ReconnectInterval {
		return maxAX206ReconnectInterval
	}
	return interval
}

func SetAX206ReconnectInterval(interval time.Duration) {
	ax206ReconnectIntervalNS.Store(int64(normalizeAX206ReconnectInterval(interval)))
}

func getAX206ReconnectInterval() time.Duration {
	value := time.Duration(ax206ReconnectIntervalNS.Load())
	return normalizeAX206ReconnectInterval(value)
}

type AX206USBOutputHandler struct {
	deviceMu sync.RWMutex
	device   *AX206USB
	rgb565   *ImageRGB565

	stopOnce sync.Once
	stopCh   chan struct{}
	loopWg   sync.WaitGroup

	reconnectCh chan struct{}
	frameCh     chan image.Image

	lastConnectErrMu sync.Mutex
	lastConnectErrAt time.Time

	lastTransferErrMu sync.Mutex
	lastTransferErrAt time.Time
}

func NewAX206USBOutputHandler() (*AX206USBOutputHandler, error) {
	handler := &AX206USBOutputHandler{
		stopCh:      make(chan struct{}),
		reconnectCh: make(chan struct{}, 1),
		frameCh:     make(chan image.Image, 1),
	}
	handler.loopWg.Add(2)
	go handler.connectionLoop()
	go handler.outputLoop()
	return handler, nil
}

func (h *AX206USBOutputHandler) GetType() string {
	return TypeAX206USB
}

func (h *AX206USBOutputHandler) Output(img image.Image) error {
	if img == nil {
		return nil
	}
	enqueueLatestAX206Frame(h.frameCh, img)
	return nil
}

func (h *AX206USBOutputHandler) Close() error {
	h.stopOnce.Do(func() {
		close(h.stopCh)
		h.loopWg.Wait()
		h.detachDevice("Disconnected", nil)
	})
	return nil
}

func (h *AX206USBOutputHandler) connectionLoop() {
	defer h.loopWg.Done()

	h.tryConnect()
	for {
		timer := time.NewTimer(getAX206ReconnectInterval())
		select {
		case <-h.stopCh:
			if !timer.Stop() {
				select {
				case <-timer.C:
				default:
				}
			}
			return
		case <-h.reconnectCh:
			if !timer.Stop() {
				select {
				case <-timer.C:
				default:
				}
			}
			h.tryConnect()
		case <-timer.C:
			h.tryConnect()
		}
	}
}

func (h *AX206USBOutputHandler) outputLoop() {
	defer h.loopWg.Done()
	for {
		select {
		case <-h.stopCh:
			return
		case img := <-h.frameCh:
			device := h.getDevice()
			if device == nil || img == nil {
				continue
			}
			startedAt := time.Now()
			h.rgb565 = convertImageToRGB565(h.rgb565, img)
			err := device.Blit(h.rgb565)
			recordAX206DeviceFrameRuntime(time.Since(startedAt), err)
			if err != nil {
				h.handleTransferFailure(device, err)
			}
		}
	}
}

func convertImageToRGB565(dst *ImageRGB565, src image.Image) *ImageRGB565 {
	bounds := src.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()
	if width <= 0 || height <= 0 {
		return dst
	}
	required := width * height * 2
	if dst == nil || dst.Stride != width*2 || dst.Rect.Dx() != width || dst.Rect.Dy() != height || cap(dst.Pix) < required {
		dst = &ImageRGB565{
			Pix:    make([]uint8, required),
			Stride: width * 2,
			Rect:   image.Rect(0, 0, width, height),
		}
	} else {
		dst.Rect = image.Rect(0, 0, width, height)
		dst.Pix = dst.Pix[:required]
	}

	minX := bounds.Min.X
	minY := bounds.Min.Y
	for y := 0; y < height; y++ {
		dstOff := y * dst.Stride
		srcY := minY + y
		for x := 0; x < width; x++ {
			r, g, b, _ := src.At(minX+x, srcY).RGBA()
			c := uint16((r & 0xF800) | ((g & 0xFC00) >> 5) | ((b & 0xFC00) >> 11))
			dst.Pix[dstOff] = uint8(c >> 8)
			dst.Pix[dstOff+1] = uint8(c)
			dstOff += 2
		}
	}
	return dst
}

func enqueueLatestAX206Frame(ch chan image.Image, frame image.Image) {
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

func (h *AX206USBOutputHandler) triggerReconnect() {
	select {
	case h.reconnectCh <- struct{}{}:
	default:
	}
}

func (h *AX206USBOutputHandler) getDevice() *AX206USB {
	h.deviceMu.RLock()
	defer h.deviceMu.RUnlock()
	return h.device
}

func (h *AX206USBOutputHandler) tryConnect() {
	if h.getDevice() != nil {
		return
	}

	device, err := NewAX206USB()
	if err != nil {
		h.logConnectFailure(err)
		return
	}

	if err := device.Brightness(7); err != nil {
		device.Close()
		h.logConnectFailure(err)
		return
	}

	h.deviceMu.Lock()
	if h.device != nil {
		h.deviceMu.Unlock()
		device.Close()
		return
	}
	h.device = device
	h.deviceMu.Unlock()
	logInfoModule("ax206usb", "Connected")
}

func (h *AX206USBOutputHandler) logConnectFailure(err error) {
	if err == nil {
		return
	}
	h.lastConnectErrMu.Lock()
	defer h.lastConnectErrMu.Unlock()
	if time.Since(h.lastConnectErrAt) < 10*time.Second {
		return
	}
	h.lastConnectErrAt = time.Now()
	logWarnModule("ax206usb", "Connect failed, will retry: %v", err)
}

func (h *AX206USBOutputHandler) handleTransferFailure(failedDevice *AX206USB, err error) {
	h.lastTransferErrMu.Lock()
	shouldLog := time.Since(h.lastTransferErrAt) >= 3*time.Second
	if shouldLog {
		h.lastTransferErrAt = time.Now()
	}
	h.lastTransferErrMu.Unlock()
	if shouldLog {
		logWarnModule("ax206usb", "Transfer failed, reconnect scheduled: %v", err)
	}
	h.detachSpecificDevice(failedDevice, "Disconnected", err)
	h.triggerReconnect()
}

func (h *AX206USBOutputHandler) detachSpecificDevice(target *AX206USB, reason string, err error) {
	h.deviceMu.Lock()
	if h.device == nil {
		h.deviceMu.Unlock()
		return
	}
	if target != nil && h.device != target {
		h.deviceMu.Unlock()
		return
	}
	device := h.device
	h.device = nil
	h.deviceMu.Unlock()
	device.Close()
	if err != nil {
		logInfoModule("ax206usb", "%s: %v", reason, err)
		return
	}
	logInfoModule("ax206usb", "%s", reason)
}

func (h *AX206USBOutputHandler) detachDevice(reason string, err error) {
	h.detachSpecificDevice(nil, reason, err)
}
