//go:build linux || (windows && cgo)

package output

import "image"

func (f *OutputFrame) RGB565(dst *ImageRGB565) *ImageRGB565 {
	if f == nil || f.Image == nil {
		return dst
	}
	return convertImageToRGB565(dst, f.Image)
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
