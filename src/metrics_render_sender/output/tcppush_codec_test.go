package output

import (
	"bytes"
	"image"
	"image/color"
	"testing"
)

func TestEncodeTCPPushFramePayloadSupportsRawAndRLEFormats(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 3, 1))
	img.Set(0, 0, color.RGBA{R: 255, A: 255})
	img.Set(1, 0, color.RGBA{R: 255, A: 255})
	img.Set(2, 0, color.RGBA{B: 255, A: 255})
	frame := NewOutputFrame(img)

	rawFrame, err := encodeTCPPushFramePayload(frame, OutputConfig{Format: "rgb565le"})
	if err != nil {
		t.Fatalf("encode raw failed: %v", err)
	}
	if rawFrame.Codec != tcpPushCodecRGB565LE || rawFrame.Width != 3 || rawFrame.Height != 1 {
		t.Fatalf("unexpected raw frame header: %#v", rawFrame)
	}
	expectedRaw := []byte{0x00, 0xf8, 0x00, 0xf8, 0x1f, 0x00}
	if !bytes.Equal(rawFrame.Payload, expectedRaw) {
		t.Fatalf("unexpected raw payload: %#v", rawFrame.Payload)
	}

	rleFrame, err := encodeTCPPushFramePayload(frame, OutputConfig{Format: "rgb565le_rle"})
	if err != nil {
		t.Fatalf("encode rgb565 rle failed: %v", err)
	}
	expectedRLE := []byte{
		0x02, 0x00, 0x00, 0xf8,
		0x01, 0x00, 0x1f, 0x00,
	}
	if rleFrame.Codec != tcpPushCodecRGB565LERLE || !bytes.Equal(rleFrame.Payload, expectedRLE) {
		t.Fatalf("unexpected rgb565 rle payload: %#v", rleFrame.Payload)
	}

	indexFrame, err := encodeTCPPushFramePayload(frame, OutputConfig{Format: "index8_rle"})
	if err != nil {
		t.Fatalf("encode index8 rle failed: %v", err)
	}
	expectedIndex := []byte{
		0x00, 0xf8, 0x1f, 0x00,
		0x02, 0x00, 0x00,
		0x01, 0x00, 0x01,
	}
	if indexFrame.Codec != tcpPushCodecIndex8RLE || indexFrame.PaletteCount != 2 {
		t.Fatalf("unexpected index8 frame header: %#v", indexFrame)
	}
	if !bytes.Equal(indexFrame.Payload, expectedIndex) {
		t.Fatalf("unexpected index8 payload: %#v", indexFrame.Payload)
	}
}

func TestNormalizeTCPPushFormatAcceptsBinaryAliases(t *testing.T) {
	cases := map[string]string{
		"jpeg_baseline": "jpeg",
		"jpg":           "jpeg",
		"rgb565":        "rgb565le",
		"rgb565_rle":    "rgb565le_rle",
		"palette8_rle":  "index8_rle",
		"unknown":       "jpeg",
	}
	for input, expected := range cases {
		if got := normalizeTCPPushFormat(input); got != expected {
			t.Fatalf("normalizeTCPPushFormat(%q) = %q, want %q", input, got, expected)
		}
	}
}

func TestEncodeTCPPushFramePayloadKeepsEmptyFileNameWhenUnset(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 2, 1))
	frame := NewOutputFrame(img)

	encoded, err := encodeTCPPushFramePayload(frame, OutputConfig{Format: "index8_rle"})
	if err != nil {
		t.Fatalf("encode index8 rle failed: %v", err)
	}
	if encoded.FileName != "" {
		t.Fatalf("expected empty file name, got %q", encoded.FileName)
	}
}
