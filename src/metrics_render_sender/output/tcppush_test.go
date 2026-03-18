package output

import (
	"bufio"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"image"
	"io"
	"net"
	"strings"
	"sync"
	"testing"
	"time"
)

type tcpPushTestRequest struct {
	Opcode       byte
	Codec        byte
	Seq          uint32
	Width        uint16
	Height       uint16
	PaletteCount uint16
	Token        string
	FileName     string
	PayloadSize  int
}

type tcpPushAvailabilityTestResponse struct {
	Available         bool   `json:"available"`
	ShouldSendFrame   bool   `json:"shouldSendFrame"`
	UserPriority      int    `json:"userPriority"`
	HighestPriority   int    `json:"highestPriority"`
	ActivePriority    int    `json:"activePriority"`
	ActiveSessionID   any    `json:"activeSessionId"`
	ActiveUser        string `json:"activeUser"`
	LowerPriorityMode string `json:"lowerPriorityMode"`
	Reason            string `json:"reason"`
}

func TestTCPPushReusesConnectionAcrossFrames(t *testing.T) {
	var connCount int
	var requests []tcpPushTestRequest
	var mu sync.Mutex

	handler, serverConn := newTCPPushPipeHandler(OutputConfig{
		Type:           TypeTCPPush,
		URL:            "tcp://127.0.0.1:9100",
		Format:         "jpeg",
		Quality:        75,
		UploadToken:    "secret-token",
		TimeoutMS:      2000,
		IdleTimeoutSec: 120,
		FileName:       "frame-test.jpg",
		SuccessCodes:   []int{200},
	}, TypeTCPPush)
	defer serverConn.Close()
	defer handler.Close()

	done := make(chan struct{})
	go func() {
		defer close(done)
		defer serverConn.Close()

		mu.Lock()
		connCount++
		mu.Unlock()

		reader := bufio.NewReader(serverConn)
		for idx := 0; idx < 2; idx++ {
			request, readErr := readTCPPushTestRequest(reader)
			if readErr != nil {
				return
			}
			mu.Lock()
			requests = append(requests, request)
			mu.Unlock()
			if writeErr := writeTCPPushTestResponse(serverConn, request, 200, "ok", "", "render"); writeErr != nil {
				return
			}
		}
	}()

	frameOne := NewOutputFrame(image.NewRGBA(image.Rect(0, 0, 8, 8)))
	frameTwo := NewOutputFrame(image.NewRGBA(image.Rect(0, 0, 8, 8)))

	if err := handler.doRequestFromFrame(frameOne); err != nil {
		t.Fatalf("first push failed: %v", err)
	}
	if err := handler.doRequestFromFrame(frameTwo); err != nil {
		t.Fatalf("second push failed: %v", err)
	}

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("tcp test server did not finish")
	}

	mu.Lock()
	defer mu.Unlock()
	if connCount != 1 {
		t.Fatalf("expected 1 tcp connection, got %d", connCount)
	}
	if len(requests) != 2 {
		t.Fatalf("expected 2 requests, got %#v", requests)
	}
	if requests[0].Opcode != tcpPushOpcodePushFrame || requests[1].Opcode != tcpPushOpcodePushFrame {
		t.Fatalf("unexpected opcodes: %#v", requests)
	}
	if requests[0].Codec != tcpPushCodecJPEG || requests[1].Codec != tcpPushCodecJPEG {
		t.Fatalf("unexpected codecs: %#v", requests)
	}
	if requests[0].Token != "secret-token" || requests[1].Token != "secret-token" {
		t.Fatalf("unexpected token values: %#v", requests)
	}
	if requests[0].FileName != "frame-test.jpg" || requests[1].FileName != "frame-test.jpg" {
		t.Fatalf("unexpected file names: %#v", requests)
	}
	if requests[0].PayloadSize == 0 || requests[1].PayloadSize == 0 {
		t.Fatalf("unexpected payload sizes: %#v", requests)
	}
	if requests[0].Seq == 0 || requests[1].Seq != requests[0].Seq+1 {
		t.Fatalf("unexpected seq values: %#v", requests)
	}
}

func TestTCPPushPingStatusUsesBinaryControlFrame(t *testing.T) {
	handler, serverConn := newTCPPushPipeHandler(OutputConfig{
		Type:           TypeTCPPush,
		URL:            "tcp://127.0.0.1:9100",
		UploadToken:    "token",
		TimeoutMS:      2000,
		IdleTimeoutSec: 120,
	}, TypeTCPPush)
	defer serverConn.Close()
	defer handler.Close()

	done := make(chan struct{})
	go func() {
		defer close(done)
		defer serverConn.Close()
		reader := bufio.NewReader(serverConn)

		request, readErr := readTCPPushTestRequest(reader)
		if readErr != nil {
			return
		}
		_ = writeTCPPushTestResponseWithOptions(serverConn, request, 200, "", "", "", tcpPushTestResponseOptions{
			BodyOverride: `{"status":"ok"}`,
		})
	}()

	status, err := handler.PingStatus()
	if err != nil {
		t.Fatalf("ping failed: %v", err)
	}
	if status != `{"status":"ok"}` {
		t.Fatalf("unexpected ping status: %q", status)
	}

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("tcp ping test server did not finish")
	}
}

func TestReadTCPPushResponseAcceptsStringBytesWhenPayloadBytesIsZero(t *testing.T) {
	request := &tcpPushRequest{
		Opcode: tcpPushOpcodePushFrame,
		Codec:  tcpPushCodecJPEG,
		Seq:    7,
	}
	testRequest := tcpPushTestRequest{
		Opcode: request.Opcode,
		Codec:  request.Codec,
		Seq:    request.Seq,
	}

	serverReader, clientWriter := net.Pipe()
	defer serverReader.Close()
	defer clientWriter.Close()

	go func() {
		_ = writeTCPPushTestResponseWithOptions(clientWriter, testRequest, 200, "rendered", "ok", "done", tcpPushTestResponseOptions{
			FramePayloadBytesOverride: 0,
			HasFramePayloadOverride:   true,
		})
	}()

	response, err := readTCPPushResponse(bufio.NewReader(serverReader), request)
	if err != nil {
		t.Fatalf("read response failed: %v", err)
	}
	if response.Message != "rendered" || response.Hint != "ok" || response.Stage != "done" {
		t.Fatalf("unexpected response: %#v", response)
	}
}

func TestReadTCPPushResponseUsesBodyLenInsteadOfFramePayloadBytes(t *testing.T) {
	request := &tcpPushRequest{
		Opcode: tcpPushOpcodePushFrame,
		Codec:  tcpPushCodecIndex8RLE,
		Seq:    9,
	}
	testRequest := tcpPushTestRequest{
		Opcode: request.Opcode,
		Codec:  request.Codec,
		Seq:    request.Seq,
	}

	serverReader, clientWriter := net.Pipe()
	defer serverReader.Close()
	defer clientWriter.Close()

	go func() {
		_ = writeTCPPushTestResponseWithOptions(clientWriter, testRequest, 200, "rendered", "ok", "done", tcpPushTestResponseOptions{
			FramePayloadBytesOverride: 1234,
			HasFramePayloadOverride:   true,
		})
	}()

	response, err := readTCPPushResponse(bufio.NewReader(serverReader), request)
	if err == nil || !strings.Contains(err.Error(), "unexpected response frame payload bytes") {
		t.Fatalf("expected frame payload bytes error, got %v", err)
	}
	if response != nil {
		t.Fatalf("expected nil response, got %#v", response)
	}
}

func TestTCPPushLogsConnectAndDisconnect(t *testing.T) {
	var logs []string
	var logMu sync.Mutex
	SetLoggerHooks(LoggerHooks{
		InfoModule: func(module, format string, args ...interface{}) {
			logMu.Lock()
			defer logMu.Unlock()
			logs = append(logs, module+": "+fmt.Sprintf(format, args...))
		},
	})
	defer SetLoggerHooks(LoggerHooks{})

	handler, serverConn := newTCPPushPipeHandler(OutputConfig{
		Type:           TypeTCPPush,
		URL:            "tcp://127.0.0.1:9100",
		Format:         "jpeg",
		Quality:        75,
		UploadToken:    "super-secret-token",
		TimeoutMS:      2000,
		IdleTimeoutSec: 120,
		FileName:       "frame-test.jpg",
	}, TypeTCPPush)
	defer serverConn.Close()

	done := make(chan struct{})
	go func() {
		defer close(done)
		defer serverConn.Close()
		reader := bufio.NewReader(serverConn)

		request, readErr := readTCPPushTestRequest(reader)
		if readErr != nil {
			return
		}
		_ = writeTCPPushTestResponse(serverConn, request, 200, "ok", "", "render")
	}()

	if err := handler.doRequestFromFrame(NewOutputFrame(image.NewRGBA(image.Rect(0, 0, 4, 4)))); err != nil {
		t.Fatalf("push failed: %v", err)
	}
	_ = handler.Close()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("tcp log test server did not finish")
	}

	logMu.Lock()
	defer logMu.Unlock()
	joined := strings.Join(logs, "\n")
	if !strings.Contains(joined, "tcppush: Connected addr=") {
		t.Fatalf("expected connect log, got %q", joined)
	}
	if !strings.Contains(joined, "tcppush: ACK status=200") {
		t.Fatalf("expected ack log, got %q", joined)
	}
	if !strings.Contains(joined, "stage=\"render\"") {
		t.Fatalf("expected ack stage log, got %q", joined)
	}
	if !strings.Contains(joined, "protocol=ESP32MON/3-JSONACK") {
		t.Fatalf("expected protocol log, got %q", joined)
	}
	if !strings.Contains(joined, "token=enabled(len=18)") {
		t.Fatalf("expected masked token log, got %q", joined)
	}
	if strings.Contains(joined, "super-secret-token") {
		t.Fatalf("expected token masked, got %q", joined)
	}
	if !strings.Contains(joined, "tcppush: Disconnected reason=handler closed") {
		t.Fatalf("expected disconnect log, got %q", joined)
	}
}

func TestTCPPushSwitchesToAvailabilityProbeOnDiscarded(t *testing.T) {
	var requests []tcpPushTestRequest
	var mu sync.Mutex

	handler, serverConn := newTCPPushPipeHandler(OutputConfig{
		Type:           TypeTCPPush,
		URL:            "tcp://127.0.0.1:9100",
		Format:         "jpeg",
		Quality:        75,
		UploadToken:    "secret-token",
		TimeoutMS:      3000,
		IdleTimeoutSec: 120,
		FileName:       "frame-test.jpg",
	}, TypeTCPPush)
	defer serverConn.Close()
	defer handler.Close()

	done := make(chan struct{})
	go func() {
		defer close(done)
		defer serverConn.Close()
		reader := bufio.NewReader(serverConn)

		request, err := readTCPPushTestRequest(reader)
		if err != nil {
			return
		}
		mu.Lock()
		requests = append(requests, request)
		mu.Unlock()
		_ = writeTCPPushTestResponse(serverConn, request, 503, "discarded", "low priority session", "busy")

		request, err = readTCPPushTestRequest(reader)
		if err != nil {
			return
		}
		mu.Lock()
		requests = append(requests, request)
		mu.Unlock()
		availabilityBlocked, _ := json.Marshal(tcpPushAvailabilityTestResponse{
			Available:         false,
			ShouldSendFrame:   false,
			UserPriority:      1,
			HighestPriority:   9,
			ActivePriority:    9,
			ActiveSessionID:   "session-a",
			ActiveUser:        "user-high",
			LowerPriorityMode: "discard",
			Reason:            "higher priority session active",
		})
		_ = writeTCPPushTestResponseWithOptions(serverConn, request, 200, "", "", "", tcpPushTestResponseOptions{
			BodyOverride: string(availabilityBlocked),
		})

		time.Sleep(tcpPushAvailabilityProbeInterval + 200*time.Millisecond)

		request, err = readTCPPushTestRequest(reader)
		if err != nil {
			return
		}
		mu.Lock()
		requests = append(requests, request)
		mu.Unlock()
		availabilityReady, _ := json.Marshal(tcpPushAvailabilityTestResponse{
			Available:         true,
			ShouldSendFrame:   true,
			UserPriority:      1,
			HighestPriority:   1,
			ActivePriority:    1,
			ActiveSessionID:   "session-a",
			ActiveUser:        "user-low",
			LowerPriorityMode: "discard",
			Reason:            "",
		})
		_ = writeTCPPushTestResponseWithOptions(serverConn, request, 200, "", "", "", tcpPushTestResponseOptions{
			BodyOverride: string(availabilityReady),
		})

		request, err = readTCPPushTestRequest(reader)
		if err != nil {
			return
		}
		mu.Lock()
		requests = append(requests, request)
		mu.Unlock()
		_ = writeTCPPushTestResponse(serverConn, request, 200, "ok", "", "render")
	}()

	if err := handler.doRequestFromFrame(NewOutputFrame(image.NewRGBA(image.Rect(0, 0, 8, 8)))); err != nil {
		t.Fatalf("first push failed: %v", err)
	}
	if err := handler.doRequestFromFrame(NewOutputFrame(image.NewRGBA(image.Rect(0, 0, 8, 8)))); err != nil {
		t.Fatalf("probe push failed: %v", err)
	}
	time.Sleep(tcpPushAvailabilityProbeInterval + 250*time.Millisecond)
	if err := handler.doRequestFromFrame(NewOutputFrame(image.NewRGBA(image.Rect(0, 0, 8, 8)))); err != nil {
		t.Fatalf("resume push failed: %v", err)
	}

	stats := GetTCPPushAvailabilityStats()[TypeTCPPush]
	if !stats.Connected || !stats.CanSend || stats.ProbeMode {
		t.Fatalf("unexpected tcp push stats: %#v", stats)
	}

	select {
	case <-done:
	case <-time.After(4 * time.Second):
		t.Fatal("tcp availability test server did not finish")
	}

	mu.Lock()
	defer mu.Unlock()
	if len(requests) != 4 {
		t.Fatalf("expected 4 requests, got %#v", requests)
	}
	if requests[0].Opcode != tcpPushOpcodePushFrame {
		t.Fatalf("expected first request push_frame, got %#v", requests)
	}
	if requests[1].Opcode != tcpPushOpcodeQueryAvailability {
		t.Fatalf("expected second request query_availability, got %#v", requests)
	}
	if requests[2].Opcode != tcpPushOpcodeQueryAvailability {
		t.Fatalf("expected third request query_availability, got %#v", requests)
	}
	if requests[3].Opcode != tcpPushOpcodePushFrame {
		t.Fatalf("expected fourth request push_frame, got %#v", requests)
	}
}

func newTCPPushPipeHandler(cfg OutputConfig, typeName string) (*TCPPushOutputHandler, net.Conn) {
	clientConn, serverConn := net.Pipe()
	handler := NewTCPPushOutputHandler(cfg, typeName)
	handler.connMu.Lock()
	handler.conn = clientConn
	handler.reader = bufio.NewReader(clientConn)
	handler.lastUsed = time.Now()
	handler.connMu.Unlock()
	handler.logConnected("pipe")
	return handler, serverConn
}

func readTCPPushTestRequest(reader *bufio.Reader) (tcpPushTestRequest, error) {
	header := make([]byte, tcpPushRequestHeaderBytes)
	if _, err := io.ReadFull(reader, header); err != nil {
		return tcpPushTestRequest{}, err
	}
	if string(header[0:4]) != tcpPushRequestMagic {
		return tcpPushTestRequest{}, fmt.Errorf("unexpected magic: %q", string(header[0:4]))
	}
	if int(header[4]) != tcpPushProtocolVersion {
		return tcpPushTestRequest{}, fmt.Errorf("unexpected version: %d", header[4])
	}
	if int(header[5]) != tcpPushRequestHeaderBytes {
		return tcpPushTestRequest{}, fmt.Errorf("unexpected header size: %d", header[5])
	}
	tokenLen := int(binary.LittleEndian.Uint16(header[24:26]))
	fileNameLen := int(binary.LittleEndian.Uint16(header[26:28]))
	payloadLen := int(binary.LittleEndian.Uint32(header[14:18]))

	token := make([]byte, tokenLen)
	if _, err := io.ReadFull(reader, token); err != nil {
		return tcpPushTestRequest{}, err
	}
	fileName := make([]byte, fileNameLen)
	if _, err := io.ReadFull(reader, fileName); err != nil {
		return tcpPushTestRequest{}, err
	}
	payload := make([]byte, payloadLen)
	if _, err := io.ReadFull(reader, payload); err != nil {
		return tcpPushTestRequest{}, err
	}

	return tcpPushTestRequest{
		Opcode:       header[6],
		Codec:        header[7],
		Seq:          binary.LittleEndian.Uint32(header[10:14]),
		Width:        binary.LittleEndian.Uint16(header[18:20]),
		Height:       binary.LittleEndian.Uint16(header[20:22]),
		PaletteCount: binary.LittleEndian.Uint16(header[22:24]),
		Token:        string(token),
		FileName:     string(fileName),
		PayloadSize:  len(payload),
	}, nil
}

func writeTCPPushTestResponse(conn net.Conn, request tcpPushTestRequest, statusCode uint16, message, hint, stage string) error {
	return writeTCPPushTestResponseWithOptions(conn, request, statusCode, message, hint, stage, tcpPushTestResponseOptions{})
}

type tcpPushTestResponseOptions struct {
	BodyOverride              string
	FramePayloadBytesOverride int
	HasFramePayloadOverride   bool
	ValidateMS                uint32
	RenderMS                  uint32
	TotalMS                   uint32
}

func writeTCPPushTestResponseWithOptions(conn net.Conn, request tcpPushTestRequest, statusCode uint16, message, hint, stage string, options tcpPushTestResponseOptions) error {
	body := strings.TrimSpace(options.BodyOverride)
	if body == "" {
		switch request.Opcode {
		case tcpPushOpcodePingStatus, tcpPushOpcodeQueryAvailability:
			body = "{}"
		default:
			bodyBytes, err := json.Marshal(map[string]interface{}{
				"ok":      statusCode >= 200 && statusCode < 300,
				"code":    int(statusCode),
				"stage":   stage,
				"message": message,
				"hint":    hint,
			})
			if err != nil {
				return err
			}
			body = string(bodyBytes)
		}
	}
	bodyBytes := []byte(body)

	header := make([]byte, tcpPushResponseHeaderSize)
	copy(header[0:4], tcpPushResponseMagic)
	header[4] = tcpPushProtocolVersion
	header[5] = tcpPushResponseHeaderSize
	header[6] = request.Opcode
	header[7] = request.Codec
	binary.LittleEndian.PutUint16(header[8:10], statusCode)
	binary.LittleEndian.PutUint32(header[10:14], request.Seq)
	binary.LittleEndian.PutUint16(header[14:16], request.Width)
	binary.LittleEndian.PutUint16(header[16:18], request.Height)
	framePayloadBytes := request.PayloadSize
	if options.HasFramePayloadOverride {
		framePayloadBytes = options.FramePayloadBytesOverride
	}
	binary.LittleEndian.PutUint32(header[18:22], uint32(framePayloadBytes))
	binary.LittleEndian.PutUint32(header[22:26], uint32(len(bodyBytes)))
	validateMS := options.ValidateMS
	renderMS := options.RenderMS
	totalMS := options.TotalMS
	if validateMS == 0 {
		validateMS = 1
	}
	if renderMS == 0 {
		renderMS = 2
	}
	if totalMS == 0 {
		totalMS = 3
	}
	binary.LittleEndian.PutUint32(header[26:30], validateMS)
	binary.LittleEndian.PutUint32(header[30:34], renderMS)
	binary.LittleEndian.PutUint32(header[34:38], totalMS)

	if _, err := conn.Write(header); err != nil {
		return err
	}
	if len(bodyBytes) == 0 {
		return nil
	}
	_, err := conn.Write(bodyBytes)
	return err
}
