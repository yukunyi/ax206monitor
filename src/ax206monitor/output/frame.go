package output

import (
	"bytes"
	"fmt"
	"image"
	"image/jpeg"
	"image/png"
	"sync"
)

type OutputFrame struct {
	Image image.Image

	mu         sync.Mutex
	pngData    []byte
	pngReady   bool
	jpegByQ    map[int][]byte
	jpegErrors map[int]error
}

func NewOutputFrame(img image.Image) *OutputFrame {
	if img == nil {
		return nil
	}
	return &OutputFrame{
		Image:      img,
		jpegByQ:    make(map[int][]byte),
		jpegErrors: make(map[int]error),
	}
}

func (f *OutputFrame) PNG() ([]byte, error) {
	if f == nil || f.Image == nil {
		return nil, nil
	}

	f.mu.Lock()
	defer f.mu.Unlock()
	if f.pngReady {
		return f.pngData, nil
	}

	var buffer bytes.Buffer
	if err := png.Encode(&buffer, f.Image); err != nil {
		return nil, fmt.Errorf("encode png: %w", err)
	}
	f.pngData = buffer.Bytes()
	f.pngReady = true
	return f.pngData, nil
}

func (f *OutputFrame) JPEG(quality int) ([]byte, error) {
	if f == nil || f.Image == nil {
		return nil, nil
	}

	normalizedQuality := normalizeHTTPPushQuality(quality)

	f.mu.Lock()
	defer f.mu.Unlock()
	if data, ok := f.jpegByQ[normalizedQuality]; ok {
		return data, nil
	}
	if err, ok := f.jpegErrors[normalizedQuality]; ok {
		return nil, err
	}

	var buffer bytes.Buffer
	opts := &jpeg.Options{Quality: normalizedQuality}
	if err := jpeg.Encode(&buffer, f.Image, opts); err != nil {
		wrapped := fmt.Errorf("encode jpeg quality=%d: %w", normalizedQuality, err)
		f.jpegErrors[normalizedQuality] = wrapped
		return nil, wrapped
	}
	data := buffer.Bytes()
	f.jpegByQ[normalizedQuality] = data
	return data, nil
}
