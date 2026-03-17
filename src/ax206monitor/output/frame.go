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

	mu          sync.Mutex
	pngData     []byte
	pngReady    bool
	rgb565LE    []byte
	rgb565Ready bool
	jpegByQ     map[int][]byte
	jpegErrors  map[int]error
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

func (f *OutputFrame) JPEGBaseline(quality int) ([]byte, error) {
	// Go stdlib image/jpeg encoder emits baseline JPEG.
	return f.JPEG(quality)
}

func (f *OutputFrame) RGB565LE() ([]byte, int, int, error) {
	if f == nil || f.Image == nil {
		return nil, 0, 0, nil
	}

	bounds := f.Image.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()
	if width <= 0 || height <= 0 {
		return nil, width, height, fmt.Errorf("invalid image size: %dx%d", width, height)
	}

	f.mu.Lock()
	defer f.mu.Unlock()
	if f.rgb565Ready {
		return f.rgb565LE, width, height, nil
	}

	data := make([]byte, width*height*2)
	minX := bounds.Min.X
	minY := bounds.Min.Y
	offset := 0
	for y := 0; y < height; y++ {
		srcY := minY + y
		for x := 0; x < width; x++ {
			r, g, b, _ := f.Image.At(minX+x, srcY).RGBA()
			value := uint16((r & 0xF800) | ((g & 0xFC00) >> 5) | ((b & 0xF800) >> 11))
			data[offset] = uint8(value)
			data[offset+1] = uint8(value >> 8)
			offset += 2
		}
	}

	f.rgb565LE = data
	f.rgb565Ready = true
	return f.rgb565LE, width, height, nil
}
