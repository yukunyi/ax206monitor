package output

import (
	"encoding/hex"
	"image"
	"image/color"
	"net"
	"os"
	"testing"
	"time"
)

func TestTCPPushIntegrationRealServer(t *testing.T) {
	addr := os.Getenv("AX206_TCPPUSH_ADDR")
	if addr == "" {
		t.Skip("AX206_TCPPUSH_ADDR is not set")
	}
	token := os.Getenv("AX206_TCPPUSH_TOKEN")
	if token == "" {
		t.Skip("AX206_TCPPUSH_TOKEN is not set")
	}

	t.Run("ping_status", func(t *testing.T) {
		handler := NewTCPPushOutputHandler(OutputConfig{
			Type:           TypeTCPPush,
			URL:            addr,
			UploadToken:    token,
			TimeoutMS:      8000,
			IdleTimeoutSec: 120,
		}, TypeTCPPush)
		defer handler.Close()

		status, err := handler.PingStatus()
		if err != nil {
			t.Fatalf("ping status failed: %v", err)
		}
		t.Logf("ping status response: %s", status)
	})

	t.Run("query_availability", func(t *testing.T) {
		handler := NewTCPPushOutputHandler(OutputConfig{
			Type:           TypeTCPPush,
			URL:            addr,
			UploadToken:    token,
			TimeoutMS:      8000,
			IdleTimeoutSec: 120,
		}, TypeTCPPush)
		defer handler.Close()

		availability, err := handler.QueryAvailability()
		if err != nil {
			t.Fatalf("query availability failed: %v", err)
		}
		t.Logf("availability: %+v", availability)
	})

	t.Run("push_jpeg", func(t *testing.T) {
		cfg := OutputConfig{
			Type:           TypeTCPPush,
			URL:            addr,
			Format:         "jpeg",
			Quality:        90,
			UploadToken:    token,
			TimeoutMS:      8000,
			IdleTimeoutSec: 120,
		}
		handler := NewTCPPushOutputHandler(cfg, TypeTCPPush)
		defer handler.Close()

		frame := newIntegrationTestFrame()
		request, err := handler.newPushRequest(frame)
		if err != nil {
			t.Fatalf("build jpeg push request failed: %v", err)
		}
		t.Logf(
			"jpeg request: opcode=%d codec=%d seq=%d payload_len=%d width=%d height=%d palette_count=%d token_len=%d file_name_len=%d file_name=%q",
			request.Opcode,
			request.Codec,
			request.Seq,
			len(request.Payload),
			request.Width,
			request.Height,
			request.PaletteCount,
			len(request.Token),
			len(request.FileName),
			string(request.FileName),
		)
		if err := handler.doRequestFromFrame(frame); err != nil {
			t.Logf("jpeg push failed: %v", err)
			statusHandler := NewTCPPushOutputHandler(cfg, TypeTCPPush)
			defer statusHandler.Close()
			status, statusErr := statusHandler.PingStatus()
			if statusErr != nil {
				t.Fatalf("jpeg push failed: %v; ping after failure also failed: %v", err, statusErr)
			}
			t.Fatalf("jpeg push failed: %v; status after failure: %s", err, status)
		}
	})

	t.Run("push_index8_rle", func(t *testing.T) {
		cfg := OutputConfig{
			Type:           TypeTCPPush,
			URL:            addr,
			Format:         "index8_rle",
			UploadToken:    token,
			TimeoutMS:      8000,
			IdleTimeoutSec: 120,
		}
		handler := NewTCPPushOutputHandler(cfg, TypeTCPPush)
		defer handler.Close()

		frame := newIntegrationTestFrame()
		request, err := handler.newPushRequest(frame)
		if err != nil {
			t.Fatalf("build index8_rle push request failed: %v", err)
		}
		t.Logf(
			"index8_rle request: opcode=%d codec=%d seq=%d payload_len=%d width=%d height=%d palette_count=%d token_len=%d file_name_len=%d file_name=%q",
			request.Opcode,
			request.Codec,
			request.Seq,
			len(request.Payload),
			request.Width,
			request.Height,
			request.PaletteCount,
			len(request.Token),
			len(request.FileName),
			string(request.FileName),
		)
		if err := handler.doRequestFromFrame(frame); err != nil {
			t.Logf("index8_rle push failed: %v", err)
			statusHandler := NewTCPPushOutputHandler(cfg, TypeTCPPush)
			defer statusHandler.Close()
			status, statusErr := statusHandler.PingStatus()
			if statusErr != nil {
				t.Fatalf("index8_rle push failed: %v; ping after failure also failed: %v", err, statusErr)
			}
			t.Fatalf("index8_rle push failed: %v; status after failure: %s", err, status)
		}
	})

	if os.Getenv("AX206_TCPPUSH_DEBUG_RAW_ACK") != "" {
		t.Run("capture_raw_push_ack_jpeg", func(t *testing.T) {
			captureRawPushAck(t, OutputConfig{
				Type:        TypeTCPPush,
				URL:         addr,
				Format:      "jpeg",
				Quality:     90,
				UploadToken: token,
			})
		})

		t.Run("capture_raw_push_ack_index8_rle", func(t *testing.T) {
			captureRawPushAck(t, OutputConfig{
				Type:        TypeTCPPush,
				URL:         addr,
				Format:      "index8_rle",
				UploadToken: token,
			})
		})

		t.Run("capture_raw_ping_ack", func(t *testing.T) {
			captureRawControlAck(t, OutputConfig{
				Type:        TypeTCPPush,
				URL:         addr,
				UploadToken: token,
			}, tcpPushOpcodePingStatus)
		})

		t.Run("capture_raw_query_ack", func(t *testing.T) {
			captureRawControlAck(t, OutputConfig{
				Type:        TypeTCPPush,
				URL:         addr,
				UploadToken: token,
			}, tcpPushOpcodeQueryAvailability)
		})
	}
}

func newIntegrationTestFrame() *OutputFrame {
	img := image.NewRGBA(image.Rect(0, 0, 64, 64))
	for y := 0; y < 64; y++ {
		for x := 0; x < 64; x++ {
			switch {
			case x < 16:
				img.Set(x, y, color.RGBA{R: 255, G: 64, B: 64, A: 255})
			case x < 32:
				img.Set(x, y, color.RGBA{R: 64, G: 255, B: 64, A: 255})
			case x < 48:
				img.Set(x, y, color.RGBA{R: 64, G: 64, B: 255, A: 255})
			default:
				img.Set(x, y, color.RGBA{R: uint8((x * 4) & 0xff), G: uint8((y * 4) & 0xff), B: 160, A: 255})
			}
		}
	}
	return NewOutputFrame(img)
}

func captureRawPushAck(t *testing.T, cfg OutputConfig) {
	t.Helper()

	handler := NewTCPPushOutputHandler(cfg, TypeTCPPush)
	defer handler.Close()

	request, err := handler.newPushRequest(newIntegrationTestFrame())
	if err != nil {
		t.Fatalf("build request failed: %v", err)
	}

	address, err := parseTCPPushAddress(cfg.URL)
	if err != nil {
		t.Fatalf("parse address failed: %v", err)
	}
	conn, err := net.DialTimeout("tcp", address, 5*time.Second)
	if err != nil {
		t.Fatalf("dial failed: %v", err)
	}
	defer conn.Close()

	if err := conn.SetDeadline(time.Now().Add(12 * time.Second)); err != nil {
		t.Fatalf("set deadline failed: %v", err)
	}
	if err := writeTCPPushRequest(conn, request); err != nil {
		t.Fatalf("write request failed: %v", err)
	}

	buffer := make([]byte, 128)
	n, err := conn.Read(buffer)
	if err != nil {
		t.Fatalf("read raw ack failed: %v", err)
	}
	t.Logf(
		"raw ack bytes: n=%d hex=%s",
		n,
		hex.EncodeToString(buffer[:n]),
	)
}

func captureRawControlAck(t *testing.T, cfg OutputConfig, opcode byte) {
	t.Helper()

	handler := NewTCPPushOutputHandler(cfg, TypeTCPPush)
	defer handler.Close()

	request := handler.newRequest(opcode, tcpPushCodecNone, 0, 0, 0, nil, "")
	address, err := parseTCPPushAddress(cfg.URL)
	if err != nil {
		t.Fatalf("parse address failed: %v", err)
	}
	conn, err := net.DialTimeout("tcp", address, 5*time.Second)
	if err != nil {
		t.Fatalf("dial failed: %v", err)
	}
	defer conn.Close()

	if err := conn.SetDeadline(time.Now().Add(12 * time.Second)); err != nil {
		t.Fatalf("set deadline failed: %v", err)
	}
	if err := writeTCPPushRequest(conn, request); err != nil {
		t.Fatalf("write request failed: %v", err)
	}

	buffer := make([]byte, 4096)
	n, err := conn.Read(buffer)
	if err != nil {
		t.Fatalf("read raw ack failed: %v", err)
	}
	t.Logf(
		"raw control ack opcode=%d n=%d hex=%s",
		opcode,
		n,
		hex.EncodeToString(buffer[:n]),
	)
}
