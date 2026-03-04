package output

import (
	"bytes"
	"fmt"
	"image"
	"image/png"
)

type MemImgOutputHandler struct{}

func NewMemImgOutputHandler() *MemImgOutputHandler {
	return &MemImgOutputHandler{}
}

func (m *MemImgOutputHandler) GetType() string {
	return "memimg"
}

func (m *MemImgOutputHandler) Output(img image.Image) error {
	var buffer bytes.Buffer
	if err := png.Encode(&buffer, img); err != nil {
		return fmt.Errorf("failed to encode memimg png: %w", err)
	}
	SetMemImgPNG(buffer.Bytes())
	return nil
}

func (m *MemImgOutputHandler) Close() error {
	return nil
}
