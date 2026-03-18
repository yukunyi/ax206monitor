package output

import (
	"encoding/binary"
	"fmt"
	"strings"
)

const (
	tcpPushCodecNone byte = iota
	tcpPushCodecJPEG
	tcpPushCodecRGB565LE
	tcpPushCodecRGB565LERLE
	tcpPushCodecIndex8RLE
)

type tcpPushEncodedFrame struct {
	Codec        byte
	Payload      []byte
	Width        uint16
	Height       uint16
	PaletteCount uint16
	FileName     string
}

func encodeTCPPushFramePayload(frame *OutputFrame, cfg OutputConfig) (*tcpPushEncodedFrame, error) {
	if frame == nil || frame.Image == nil {
		return nil, fmt.Errorf("frame image is empty")
	}

	bounds := frame.Image.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()
	if width <= 0 || height <= 0 {
		return nil, fmt.Errorf("frame size is invalid: %dx%d", width, height)
	}
	if width > tcpPushMaxWidth || height > tcpPushMaxHeight {
		return nil, fmt.Errorf("frame size exceeds limit: %dx%d", width, height)
	}
	if width*height > tcpPushMaxPixels {
		return nil, fmt.Errorf("frame pixels exceed limit: %d", width*height)
	}

	switch cfg.Format {
	case "rgb565le":
		payload, payloadWidth, payloadHeight, err := frame.RGB565LE()
		if err != nil {
			return nil, err
		}
		return buildEncodedTCPPushFrame(
			tcpPushCodecRGB565LE,
			payload,
			uint16(payloadWidth),
			uint16(payloadHeight),
			0,
			tcpPushRequestFileName(cfg),
		)
	case "rgb565le_rle":
		payload, payloadWidth, payloadHeight, err := encodeRGB565LERLE(frame)
		if err != nil {
			return nil, err
		}
		return buildEncodedTCPPushFrame(
			tcpPushCodecRGB565LERLE,
			payload,
			uint16(payloadWidth),
			uint16(payloadHeight),
			0,
			tcpPushRequestFileName(cfg),
		)
	case "index8_rle":
		payload, payloadWidth, payloadHeight, paletteCount, err := encodeIndex8RLE(frame)
		if err != nil {
			return nil, err
		}
		return buildEncodedTCPPushFrame(
			tcpPushCodecIndex8RLE,
			payload,
			uint16(payloadWidth),
			uint16(payloadHeight),
			paletteCount,
			tcpPushRequestFileName(cfg),
		)
	default:
		payload, err := frame.JPEGBaseline(cfg.Quality)
		if err != nil {
			return nil, err
		}
		return buildEncodedTCPPushFrame(
			tcpPushCodecJPEG,
			payload,
			0,
			0,
			0,
			tcpPushRequestFileName(cfg),
		)
	}
}

func tcpPushRequestFileName(cfg OutputConfig) string {
	return strings.TrimSpace(cfg.FileName)
}

func buildEncodedTCPPushFrame(codec byte, payload []byte, width, height, paletteCount uint16, fileName string) (*tcpPushEncodedFrame, error) {
	if len(payload) > tcpPushMaxBytes {
		return nil, fmt.Errorf("frame size exceeds limit: %d", len(payload))
	}
	return &tcpPushEncodedFrame{
		Codec:        codec,
		Payload:      payload,
		Width:        width,
		Height:       height,
		PaletteCount: paletteCount,
		FileName:     fileName,
	}, nil
}

func encodeRGB565LERLE(frame *OutputFrame) ([]byte, int, int, error) {
	data, width, height, err := frame.RGB565LE()
	if err != nil {
		return nil, 0, 0, err
	}
	if len(data) == 0 {
		return []byte{}, width, height, nil
	}

	encoded := make([]byte, 0, len(data))
	current := binary.LittleEndian.Uint16(data[:2])
	runLength := uint16(1)
	for offset := 2; offset < len(data); offset += 2 {
		value := binary.LittleEndian.Uint16(data[offset : offset+2])
		if value == current && runLength < 0xffff {
			runLength++
			continue
		}
		encoded = appendUint16LE(encoded, runLength)
		encoded = appendUint16LE(encoded, current)
		current = value
		runLength = 1
	}
	encoded = appendUint16LE(encoded, runLength)
	encoded = appendUint16LE(encoded, current)
	return encoded, width, height, nil
}

func encodeIndex8RLE(frame *OutputFrame) ([]byte, int, int, uint16, error) {
	data, width, height, err := frame.RGB565LE()
	if err != nil {
		return nil, 0, 0, 0, err
	}
	if len(data) == 0 {
		return []byte{}, width, height, 0, nil
	}

	palette, exactIndexMap, exact := buildExactIndexPalette(data)
	if exact {
		payload := encodeIndex8RLEPayload(data, palette, func(color uint16) uint8 {
			return exactIndexMap[color]
		})
		return payload, width, height, uint16(len(palette)), nil
	}

	palette, bucketToIndex := buildQuantizedIndexPalette(data)
	payload := encodeIndex8RLEPayload(data, palette, func(color uint16) uint8 {
		return bucketToIndex[quantizeRGB565Bucket(color)]
	})
	return payload, width, height, uint16(len(palette)), nil
}

func buildExactIndexPalette(data []byte) ([]uint16, map[uint16]uint8, bool) {
	palette := make([]uint16, 0, 256)
	indexByColor := make(map[uint16]uint8, 256)
	for offset := 0; offset < len(data); offset += 2 {
		color := binary.LittleEndian.Uint16(data[offset : offset+2])
		if _, exists := indexByColor[color]; exists {
			continue
		}
		if len(palette) >= 256 {
			return nil, nil, false
		}
		indexByColor[color] = uint8(len(palette))
		palette = append(palette, color)
	}
	return palette, indexByColor, true
}

type tcpPushBucketStats struct {
	count uint32
	sumR  uint32
	sumG  uint32
	sumB  uint32
}

func buildQuantizedIndexPalette(data []byte) ([]uint16, [256]uint8) {
	var stats [256]tcpPushBucketStats
	for offset := 0; offset < len(data); offset += 2 {
		color := binary.LittleEndian.Uint16(data[offset : offset+2])
		bucket := quantizeRGB565Bucket(color)
		r5, g6, b5 := rgb565Components(color)
		stats[bucket].count++
		stats[bucket].sumR += uint32(r5)
		stats[bucket].sumG += uint32(g6)
		stats[bucket].sumB += uint32(b5)
	}

	palette := make([]uint16, 0, 256)
	var bucketToIndex [256]uint8
	for bucket := 0; bucket < len(stats); bucket++ {
		if stats[bucket].count == 0 {
			continue
		}
		index := uint8(len(palette))
		bucketToIndex[bucket] = index
		avgR := uint16(stats[bucket].sumR / stats[bucket].count)
		avgG := uint16(stats[bucket].sumG / stats[bucket].count)
		avgB := uint16(stats[bucket].sumB / stats[bucket].count)
		palette = append(palette, (avgR<<11)|(avgG<<5)|avgB)
	}
	return palette, bucketToIndex
}

func encodeIndex8RLEPayload(data []byte, palette []uint16, resolveIndex func(color uint16) uint8) []byte {
	encoded := make([]byte, 0, len(palette)*2+len(data))
	for _, color := range palette {
		encoded = appendUint16LE(encoded, color)
	}

	current := resolveIndex(binary.LittleEndian.Uint16(data[:2]))
	runLength := uint16(1)
	for offset := 2; offset < len(data); offset += 2 {
		index := resolveIndex(binary.LittleEndian.Uint16(data[offset : offset+2]))
		if index == current && runLength < 0xffff {
			runLength++
			continue
		}
		encoded = appendUint16LE(encoded, runLength)
		encoded = append(encoded, current)
		current = index
		runLength = 1
	}
	encoded = appendUint16LE(encoded, runLength)
	encoded = append(encoded, current)
	return encoded
}

func quantizeRGB565Bucket(color uint16) uint8 {
	r5, g6, b5 := rgb565Components(color)
	r3 := uint8((uint32(r5)*7 + 15) / 31)
	g3 := uint8((uint32(g6)*7 + 31) / 63)
	b2 := uint8((uint32(b5)*3 + 15) / 31)
	return (r3 << 5) | (g3 << 2) | b2
}

func rgb565Components(color uint16) (uint8, uint8, uint8) {
	return uint8((color >> 11) & 0x1f), uint8((color >> 5) & 0x3f), uint8(color & 0x1f)
}

func appendUint16LE(dst []byte, value uint16) []byte {
	return append(dst, uint8(value), uint8(value>>8))
}
