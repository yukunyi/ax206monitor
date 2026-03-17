package output

import (
	"bufio"
	"context"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"net"
	neturl "net/url"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

const (
	tcpPushProtocolName       = "ESP32MON/3-JSONACK"
	tcpPushRequestMagic       = "EMF3"
	tcpPushResponseMagic      = "EMA3"
	tcpPushProtocolVersion    = 3
	tcpPushRequestHeaderBytes = 36
	tcpPushResponseHeaderSize = 44

	tcpPushOpcodePushFrame         byte = 1
	tcpPushOpcodePingStatus        byte = 2
	tcpPushOpcodeClearImage        byte = 3
	tcpPushOpcodeRepaintImage      byte = 4
	tcpPushOpcodeQueryAvailability byte = 5

	tcpPushMaxBytes  = 2 * 1024 * 1024
	tcpPushMaxWidth  = 800
	tcpPushMaxHeight = 480
	tcpPushMaxPixels = 6000000

	tcpPushAvailabilityProbeInterval = time.Second
)

type TCPPushOutputHandler struct {
	cfg      OutputConfig
	typeName string

	stopOnce sync.Once
	stopCh   chan struct{}
	loopWg   sync.WaitGroup
	frameCh  chan *OutputFrame

	connMu   sync.Mutex
	conn     net.Conn
	reader   *bufio.Reader
	lastUsed time.Time

	reqMu sync.Mutex

	seq atomic.Uint32

	lastErrorMu sync.Mutex
	lastErrorAt time.Time

	ackLogMu          sync.Mutex
	ackLogged         bool
	lastAckStatusCode int
	lastAckStage      string

	availabilityMu               sync.Mutex
	availabilityProbeMode        bool
	availabilityInitialized      bool
	nextAvailabilityProbeAt      time.Time
	lastAvailabilityStateLogged  bool
	lastAvailabilityCanSend      bool
	lastAvailabilityReason       string
	lastAvailabilityPriorityMode string
}

type tcpPushRequest struct {
	Opcode       byte
	Codec        byte
	Seq          uint32
	Width        uint16
	Height       uint16
	PaletteCount uint16
	Token        []byte
	FileName     []byte
	Payload      []byte
}

type tcpPushResponse struct {
	Opcode            byte
	Codec             byte
	StatusCode        int
	Seq               uint32
	Width             int
	Height            int
	FramePayloadBytes int
	BodyLen           int
	ValidateMS        uint32
	RenderMS          uint32
	TotalMS           uint32
	Body              string
	Message           string
	Hint              string
	Stage             string
}

type tcpPushAckBody struct {
	OK      bool            `json:"ok"`
	Code    json.RawMessage `json:"code"`
	Stage   string          `json:"stage"`
	Message string          `json:"message"`
	Hint    string          `json:"hint"`
}

type tcpPushAvailabilityResponse struct {
	Available         bool   `json:"available"`
	ShouldSendFrame   bool   `json:"shouldSendFrame"`
	UserPriority      int    `json:"userPriority"`
	HighestPriority   int    `json:"highestPriority"`
	ActivePriority    int    `json:"activePriority"`
	ActiveUser        string `json:"activeUser"`
	LowerPriorityMode string `json:"lowerPriorityMode"`
	Reason            string `json:"reason"`
	ActiveSessionID   any    `json:"activeSessionId"`
}

func NewTCPPushOutputHandler(cfg OutputConfig, typeName string) *TCPPushOutputHandler {
	handler := &TCPPushOutputHandler{
		cfg:      cfg,
		typeName: typeName,
		stopCh:   make(chan struct{}),
		frameCh:  make(chan *OutputFrame, 1),
	}
	handler.loopWg.Add(1)
	go handler.loop()
	return handler
}

func (h *TCPPushOutputHandler) GetType() string {
	return h.typeName
}

func (h *TCPPushOutputHandler) OutputFrame(frame *OutputFrame) error {
	if frame == nil {
		return nil
	}
	enqueueLatestHTTPPushFrame(h.frameCh, frame)
	return nil
}

func (h *TCPPushOutputHandler) Close() error {
	h.stopOnce.Do(func() {
		close(h.stopCh)
		h.loopWg.Wait()
		h.closeConnWithReason("handler closed")
	})
	return nil
}

func (h *TCPPushOutputHandler) PingStatus() (string, error) {
	response, err := h.doControlRequest(tcpPushOpcodePingStatus)
	if err != nil {
		return "", err
	}
	return response.Body, nil
}

func (h *TCPPushOutputHandler) ClearImage() error {
	_, err := h.doControlRequest(tcpPushOpcodeClearImage)
	return err
}

func (h *TCPPushOutputHandler) RepaintImage() error {
	_, err := h.doControlRequest(tcpPushOpcodeRepaintImage)
	return err
}

func (h *TCPPushOutputHandler) QueryAvailability() (*tcpPushAvailabilityResponse, error) {
	return h.queryAvailability()
}

func (h *TCPPushOutputHandler) loop() {
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

func (h *TCPPushOutputHandler) push(frame *OutputFrame) {
	if frame == nil {
		return
	}

	startedAt := time.Now()
	err := h.doRequestFromFrame(frame)
	recordTCPPushRuntime(h.typeName, time.Since(startedAt), err)
	if err != nil {
		h.logError("push failed: %v", err)
	}
}

func (h *TCPPushOutputHandler) doRequestFromFrame(frame *OutputFrame) error {
	canSend, err := h.ensurePushAllowed()
	if err != nil {
		return err
	}
	if !canSend {
		return nil
	}

	request, err := h.newPushRequest(frame)
	if err != nil {
		return fmt.Errorf("encode failed: %w", err)
	}
	response, err := h.doRequest(request)
	if err != nil {
		return err
	}
	if h.isSuccessStatus(response.StatusCode) {
		h.logAckSuccess(response)
		return nil
	}
	if requiresAvailabilityProbe(response) {
		h.enterAvailabilityProbeMode(response)
		return nil
	}
	return buildTCPPushStatusError(response)
}

func (h *TCPPushOutputHandler) doControlRequest(opcode byte) (*tcpPushResponse, error) {
	request := h.newRequest(opcode, tcpPushCodecNone, 0, 0, 0, nil, "")
	response, err := h.doRequest(request)
	if err != nil {
		return nil, err
	}
	if !h.isSuccessStatus(response.StatusCode) {
		return nil, buildTCPPushStatusError(response)
	}
	h.logAckSuccess(response)
	return response, nil
}

func (h *TCPPushOutputHandler) newPushRequest(frame *OutputFrame) (*tcpPushRequest, error) {
	encoded, err := encodeTCPPushFramePayload(frame, h.cfg)
	if err != nil {
		return nil, err
	}
	return h.newRequest(
		tcpPushOpcodePushFrame,
		encoded.Codec,
		encoded.Width,
		encoded.Height,
		encoded.PaletteCount,
		encoded.Payload,
		encoded.FileName,
	), nil
}

func (h *TCPPushOutputHandler) newRequest(opcode, codec byte, width, height, paletteCount uint16, payload []byte, fileName string) *tcpPushRequest {
	return &tcpPushRequest{
		Opcode:       opcode,
		Codec:        codec,
		Seq:          h.seq.Add(1),
		Width:        width,
		Height:       height,
		PaletteCount: paletteCount,
		Token:        []byte(strings.TrimSpace(h.cfg.UploadToken)),
		FileName:     []byte(fileName),
		Payload:      payload,
	}
}

func (h *TCPPushOutputHandler) doRequest(request *tcpPushRequest) (*tcpPushResponse, error) {
	if request == nil {
		return nil, fmt.Errorf("request is nil")
	}
	h.reqMu.Lock()
	defer h.reqMu.Unlock()

	conn, reader, err := h.ensureConn()
	if err != nil {
		return nil, err
	}

	if err := conn.SetDeadline(time.Now().Add(time.Duration(h.cfg.TimeoutMS) * time.Millisecond)); err != nil {
		h.closeConnWithError(err)
		return nil, err
	}

	if err := writeTCPPushRequest(conn, request); err != nil {
		h.closeConnWithError(err)
		return nil, err
	}
	response, err := readTCPPushResponse(reader, request)
	if err != nil {
		h.closeConnWithError(err)
		return nil, err
	}
	h.touchConn()
	h.updateLastAckState(response)
	return response, nil
}

func (h *TCPPushOutputHandler) ensurePushAllowed() (bool, error) {
	if !h.shouldQueryAvailabilityNow() {
		return true, nil
	}

	availability, err := h.queryAvailability()
	if err != nil {
		return false, err
	}
	canSend := availability != nil && availability.Available && availability.ShouldSendFrame
	h.updateAvailabilityState(availability, canSend)
	return canSend, nil
}

func (h *TCPPushOutputHandler) queryAvailability() (*tcpPushAvailabilityResponse, error) {
	request := h.newRequest(tcpPushOpcodeQueryAvailability, tcpPushCodecNone, 0, 0, 0, nil, "")
	response, err := h.doRequest(request)
	if err != nil {
		return nil, err
	}
	if !h.isSuccessStatus(response.StatusCode) {
		return nil, buildTCPPushStatusError(response)
	}
	h.logAckSuccess(response)

	availability := &tcpPushAvailabilityResponse{}
	if strings.TrimSpace(response.Body) == "" {
		return availability, nil
	}
	if err := json.Unmarshal([]byte(response.Body), availability); err != nil {
		return nil, fmt.Errorf("invalid availability json: %w", err)
	}
	return availability, nil
}

func (h *TCPPushOutputHandler) shouldQueryAvailabilityNow() bool {
	h.availabilityMu.Lock()
	defer h.availabilityMu.Unlock()
	if !h.availabilityProbeMode {
		return false
	}
	if h.nextAvailabilityProbeAt.IsZero() {
		return true
	}
	return !time.Now().Before(h.nextAvailabilityProbeAt)
}

func (h *TCPPushOutputHandler) enterAvailabilityProbeMode(response *tcpPushResponse) {
	h.availabilityMu.Lock()
	h.availabilityProbeMode = true
	h.nextAvailabilityProbeAt = time.Now()
	h.availabilityMu.Unlock()
	h.logAvailabilityProbeMode(response)
	h.publishTCPPushStats(true, nil, false)
}

func (h *TCPPushOutputHandler) updateAvailabilityState(availability *tcpPushAvailabilityResponse, canSend bool) {
	h.availabilityMu.Lock()

	reason := ""
	lowerPriorityMode := ""
	if availability != nil {
		reason = strings.TrimSpace(availability.Reason)
		lowerPriorityMode = strings.TrimSpace(availability.LowerPriorityMode)
	}

	stateChanged := !h.lastAvailabilityStateLogged ||
		h.lastAvailabilityCanSend != canSend ||
		h.lastAvailabilityReason != reason ||
		h.lastAvailabilityPriorityMode != lowerPriorityMode

	h.lastAvailabilityStateLogged = true
	h.availabilityInitialized = true
	h.lastAvailabilityCanSend = canSend
	h.lastAvailabilityReason = reason
	h.lastAvailabilityPriorityMode = lowerPriorityMode

	if canSend {
		h.availabilityProbeMode = false
		h.nextAvailabilityProbeAt = time.Time{}
	} else {
		h.availabilityProbeMode = true
		h.nextAvailabilityProbeAt = time.Now().Add(tcpPushAvailabilityProbeInterval)
	}

	if stateChanged {
		h.logAvailabilityStateLocked(availability, canSend)
	}
	h.availabilityMu.Unlock()
	h.publishTCPPushStats(true, availability, canSend)
}

func writeTCPPushRequest(writer io.Writer, request *tcpPushRequest) error {
	if request == nil {
		return fmt.Errorf("request is nil")
	}
	if len(request.Token) > 0xffff {
		return fmt.Errorf("token is too large: %d", len(request.Token))
	}
	if len(request.FileName) > 0xffff {
		return fmt.Errorf("file name is too large: %d", len(request.FileName))
	}
	if len(request.Payload) > tcpPushMaxBytes {
		return fmt.Errorf("payload exceeds limit: %d", len(request.Payload))
	}

	header := make([]byte, tcpPushRequestHeaderBytes)
	copy(header[0:4], tcpPushRequestMagic)
	header[4] = tcpPushProtocolVersion
	header[5] = tcpPushRequestHeaderBytes
	header[6] = request.Opcode
	header[7] = request.Codec
	binary.LittleEndian.PutUint16(header[8:10], 0)
	binary.LittleEndian.PutUint32(header[10:14], request.Seq)
	binary.LittleEndian.PutUint32(header[14:18], uint32(len(request.Payload)))
	binary.LittleEndian.PutUint16(header[18:20], request.Width)
	binary.LittleEndian.PutUint16(header[20:22], request.Height)
	binary.LittleEndian.PutUint16(header[22:24], request.PaletteCount)
	binary.LittleEndian.PutUint16(header[24:26], uint16(len(request.Token)))
	binary.LittleEndian.PutUint16(header[26:28], uint16(len(request.FileName)))

	if err := writeAll(writer, header); err != nil {
		return err
	}
	if err := writeAll(writer, request.Token); err != nil {
		return err
	}
	if err := writeAll(writer, request.FileName); err != nil {
		return err
	}
	return writeAll(writer, request.Payload)
}

func readTCPPushResponse(reader *bufio.Reader, request *tcpPushRequest) (*tcpPushResponse, error) {
	header := make([]byte, tcpPushResponseHeaderSize)
	if _, err := io.ReadFull(reader, header); err != nil {
		return nil, err
	}
	if string(header[0:4]) != tcpPushResponseMagic {
		return nil, fmt.Errorf("unexpected response magic: %q", string(header[0:4]))
	}
	if int(header[4]) != tcpPushProtocolVersion {
		return nil, fmt.Errorf("unexpected response version: %d", header[4])
	}
	if int(header[5]) != tcpPushResponseHeaderSize {
		return nil, fmt.Errorf("unexpected response header size: %d", header[5])
	}

	response := &tcpPushResponse{
		Opcode:            header[6],
		Codec:             header[7],
		StatusCode:        int(binary.LittleEndian.Uint16(header[8:10])),
		Seq:               binary.LittleEndian.Uint32(header[10:14]),
		Width:             int(binary.LittleEndian.Uint16(header[14:16])),
		Height:            int(binary.LittleEndian.Uint16(header[16:18])),
		FramePayloadBytes: int(binary.LittleEndian.Uint32(header[18:22])),
		BodyLen:           int(binary.LittleEndian.Uint32(header[22:26])),
		ValidateMS:        binary.LittleEndian.Uint32(header[26:30]),
		RenderMS:          binary.LittleEndian.Uint32(header[30:34]),
		TotalMS:           binary.LittleEndian.Uint32(header[34:38]),
	}
	if response.Opcode != request.Opcode {
		return nil, fmt.Errorf("unexpected response opcode: %d", response.Opcode)
	}
	if response.Codec != request.Codec {
		return nil, fmt.Errorf("unexpected response codec: %d", response.Codec)
	}
	if response.Seq != request.Seq {
		return nil, fmt.Errorf("unexpected response seq: %d", response.Seq)
	}
	if response.FramePayloadBytes != len(request.Payload) {
		return nil, fmt.Errorf("unexpected response frame payload bytes: %d", response.FramePayloadBytes)
	}
	if response.BodyLen < 0 || response.BodyLen > tcpPushMaxBytes {
		return nil, fmt.Errorf("unexpected response body length: %d", response.BodyLen)
	}
	if response.BodyLen == 0 {
		return nil, fmt.Errorf("response body is empty")
	}

	body := make([]byte, response.BodyLen)
	if _, err := io.ReadFull(reader, body); err != nil {
		return nil, err
	}
	if !json.Valid(body) {
		return nil, fmt.Errorf("invalid response json body")
	}
	response.Body = string(body)

	if request.Opcode == tcpPushOpcodePushFrame || request.Opcode == tcpPushOpcodeClearImage || request.Opcode == tcpPushOpcodeRepaintImage {
		ackBody := tcpPushAckBody{}
		if err := json.Unmarshal(body, &ackBody); err != nil {
			return nil, fmt.Errorf("invalid ack json body: %w", err)
		}
		response.Message = strings.TrimSpace(ackBody.Message)
		response.Hint = strings.TrimSpace(ackBody.Hint)
		response.Stage = strings.TrimSpace(ackBody.Stage)
	}
	return response, nil
}

func (h *TCPPushOutputHandler) ensureConn() (net.Conn, *bufio.Reader, error) {
	h.connMu.Lock()
	defer h.connMu.Unlock()

	if strings.TrimSpace(h.cfg.UploadToken) == "" {
		return nil, nil, fmt.Errorf("tcp key is required")
	}
	if h.conn != nil && h.cfg.IdleTimeoutSec > 0 && time.Since(h.lastUsed) > time.Duration(h.cfg.IdleTimeoutSec)*time.Second {
		h.closeConnLocked("idle timeout exceeded")
	}
	if h.conn != nil && h.reader != nil {
		return h.conn, h.reader, nil
	}

	address, err := parseTCPPushAddress(h.cfg.URL)
	if err != nil {
		return nil, nil, err
	}
	dialer := &net.Dialer{
		Timeout:   time.Duration(h.cfg.TimeoutMS) * time.Millisecond,
		KeepAlive: 30 * time.Second,
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(h.cfg.TimeoutMS)*time.Millisecond)
	defer cancel()
	conn, err := dialer.DialContext(ctx, "tcp", address)
	if err != nil {
		return nil, nil, err
	}
	h.conn = conn
	h.reader = bufio.NewReader(conn)
	h.lastUsed = time.Now()
	h.logConnected(address)
	h.publishTCPPushStats(true, nil, false)
	return h.conn, h.reader, nil
}

func (h *TCPPushOutputHandler) touchConn() {
	h.connMu.Lock()
	defer h.connMu.Unlock()
	h.lastUsed = time.Now()
}

func (h *TCPPushOutputHandler) closeConnWithError(err error) {
	if err == nil {
		h.closeConnWithReason("connection closed")
		return
	}
	h.closeConnWithReason(fmt.Sprintf("connection error: %v", err))
}

func (h *TCPPushOutputHandler) closeConnWithReason(reason string) {
	h.connMu.Lock()
	defer h.connMu.Unlock()
	h.closeConnLocked(reason)
}

func (h *TCPPushOutputHandler) closeConnLocked(reason string) {
	if h.conn != nil {
		_ = h.conn.Close()
		h.logDisconnected(reason)
	}
	h.conn = nil
	h.reader = nil
	h.lastUsed = time.Time{}
	h.availabilityMu.Lock()
	h.availabilityInitialized = false
	h.availabilityProbeMode = false
	h.nextAvailabilityProbeAt = time.Time{}
	h.lastAvailabilityStateLogged = false
	h.lastAvailabilityCanSend = false
	h.lastAvailabilityReason = ""
	h.lastAvailabilityPriorityMode = ""
	h.availabilityMu.Unlock()
	h.publishTCPPushStats(false, nil, false)
}

func (h *TCPPushOutputHandler) isSuccessStatus(statusCode int) bool {
	if len(h.cfg.SuccessCodes) == 0 {
		return statusCode == 200
	}
	for _, code := range h.cfg.SuccessCodes {
		if code == statusCode {
			return true
		}
	}
	return false
}

func (h *TCPPushOutputHandler) logError(format string, args ...interface{}) {
	h.lastErrorMu.Lock()
	defer h.lastErrorMu.Unlock()
	if time.Since(h.lastErrorAt) < 3*time.Second {
		return
	}
	h.lastErrorAt = time.Now()
	logWarnModule("tcppush", format, args...)
}

func (h *TCPPushOutputHandler) logConnected(address string) {
	logInfoModule(
		"tcppush",
		"Connected addr=%s protocol=%s format=%s quality=%d timeout_ms=%d idle_timeout_sec=%d file_name=%q token=%s lower_priority_mode=discard",
		address,
		tcpPushProtocolName,
		normalizeTCPPushFormat(h.cfg.Format),
		normalizeHTTPPushQuality(h.cfg.Quality),
		normalizeHTTPPushTimeoutMS(h.cfg.TimeoutMS),
		normalizeTCPPushIdleTimeoutSec(h.cfg.IdleTimeoutSec),
		tcpPushFileName(h.cfg, ""),
		tcpPushTokenLogValue(h.cfg.UploadToken),
	)
}

func (h *TCPPushOutputHandler) logDisconnected(reason string) {
	logInfoModule("tcppush", "Disconnected reason=%s", strings.TrimSpace(reason))
}

func (h *TCPPushOutputHandler) logAckSuccess(response *tcpPushResponse) {
	if response == nil {
		return
	}

	stage := strings.TrimSpace(response.Stage)

	h.ackLogMu.Lock()
	defer h.ackLogMu.Unlock()

	shouldLog := !h.ackLogged || h.lastAckStatusCode != response.StatusCode || h.lastAckStage != stage
	h.ackLogged = true
	h.lastAckStatusCode = response.StatusCode
	h.lastAckStage = stage
	if !shouldLog {
		return
	}

	logInfoModule(
		"tcppush",
		"ACK status=%d opcode=%d codec=%d seq=%d frame_payload_bytes=%d body_len=%d stage=%q validate_ms=%d render_ms=%d total_ms=%d message=%q hint=%q",
		response.StatusCode,
		response.Opcode,
		response.Codec,
		response.Seq,
		response.FramePayloadBytes,
		response.BodyLen,
		stage,
		response.ValidateMS,
		response.RenderMS,
		response.TotalMS,
		strings.TrimSpace(response.Message),
		strings.TrimSpace(response.Hint),
	)
}

func (h *TCPPushOutputHandler) updateLastAckState(response *tcpPushResponse) {
	if response == nil {
		return
	}
	h.ackLogMu.Lock()
	defer h.ackLogMu.Unlock()
	h.lastAckStatusCode = response.StatusCode
	h.lastAckStage = strings.TrimSpace(response.Stage)
}

func (h *TCPPushOutputHandler) logAvailabilityProbeMode(response *tcpPushResponse) {
	if response == nil {
		return
	}
	logInfoModule(
		"tcppush",
		"Switching to availability probe mode status=%d stage=%q message=%q hint=%q",
		response.StatusCode,
		strings.TrimSpace(response.Stage),
		strings.TrimSpace(response.Message),
		strings.TrimSpace(response.Hint),
	)
}

func (h *TCPPushOutputHandler) logAvailabilityStateLocked(availability *tcpPushAvailabilityResponse, canSend bool) {
	if availability == nil {
		logInfoModule("tcppush", "Availability available=false should_send_frame=false reason=%q lower_priority_mode=%q", "", "")
		return
	}
	logInfoModule(
		"tcppush",
		"Availability available=%t should_send_frame=%t can_send=%t user_priority=%d highest_priority=%d active_priority=%d active_session_id=%v active_user=%q lower_priority_mode=%q reason=%q",
		availability.Available,
		availability.ShouldSendFrame,
		canSend,
		availability.UserPriority,
		availability.HighestPriority,
		availability.ActivePriority,
		availability.ActiveSessionID,
		strings.TrimSpace(availability.ActiveUser),
		strings.TrimSpace(availability.LowerPriorityMode),
		strings.TrimSpace(availability.Reason),
	)
}

func (h *TCPPushOutputHandler) publishTCPPushStats(connected bool, availability *tcpPushAvailabilityResponse, canSend bool) {
	h.availabilityMu.Lock()
	probeMode := h.availabilityProbeMode
	reason := h.lastAvailabilityReason
	lowerPriorityMode := h.lastAvailabilityPriorityMode
	h.availabilityMu.Unlock()

	h.ackLogMu.Lock()
	lastStatusCode := h.lastAckStatusCode
	lastStage := h.lastAckStage
	h.ackLogMu.Unlock()

	stats := TCPPushAvailabilityStats{
		Type:              h.typeName,
		Connected:         connected,
		ProbeMode:         probeMode,
		CanSend:           connected && canSend,
		LowerPriorityMode: lowerPriorityMode,
		Reason:            reason,
		LastStatusCode:    lastStatusCode,
		LastStage:         lastStage,
	}
	if availability != nil {
		stats.Available = availability.Available
		stats.ShouldSendFrame = availability.ShouldSendFrame
		stats.UserPriority = availability.UserPriority
		stats.HighestPriority = availability.HighestPriority
		stats.ActivePriority = availability.ActivePriority
		stats.ActiveSessionID = strings.TrimSpace(fmt.Sprint(availability.ActiveSessionID))
		stats.ActiveUser = strings.TrimSpace(availability.ActiveUser)
		stats.LowerPriorityMode = strings.TrimSpace(availability.LowerPriorityMode)
		stats.Reason = strings.TrimSpace(availability.Reason)
	}
	RecordTCPPushAvailabilityStats(stats)
}

func parseTCPPushAddress(rawURL string) (string, error) {
	value := strings.TrimSpace(rawURL)
	if value == "" {
		return "", fmt.Errorf("url is empty")
	}
	parsed, err := neturl.Parse(value)
	if err != nil {
		return "", err
	}
	if !strings.EqualFold(parsed.Scheme, "tcp") {
		return "", fmt.Errorf("unsupported scheme: %s", parsed.Scheme)
	}
	if strings.TrimSpace(parsed.Host) == "" {
		return "", fmt.Errorf("tcp host is empty")
	}
	return parsed.Host, nil
}

func tcpPushFileName(cfg OutputConfig, fallback string) string {
	if strings.TrimSpace(cfg.FileName) != "" {
		return filepath.Base(cfg.FileName)
	}
	return fallback
}

func tcpPushTokenLogValue(token string) string {
	value := strings.TrimSpace(token)
	if value == "" {
		return "disabled"
	}
	return fmt.Sprintf("enabled(len=%d)", len(value))
}

func buildTCPPushStatusError(response *tcpPushResponse) error {
	if response == nil {
		return fmt.Errorf("tcp push failed")
	}
	message := strings.TrimSpace(response.Message)
	if message == "" {
		message = "tcp push failed"
	}
	details := make([]string, 0, 3)
	if strings.TrimSpace(response.Stage) != "" {
		details = append(details, "stage="+strings.TrimSpace(response.Stage))
	}
	if strings.TrimSpace(response.Hint) != "" {
		details = append(details, "hint="+strings.TrimSpace(response.Hint))
	}
	if response.TotalMS > 0 {
		details = append(details, fmt.Sprintf("total_ms=%d", response.TotalMS))
	}
	if len(details) == 0 {
		return fmt.Errorf("status %d: %s", response.StatusCode, message)
	}
	return fmt.Errorf("status %d: %s (%s)", response.StatusCode, message, strings.Join(details, ", "))
}

func requiresAvailabilityProbe(response *tcpPushResponse) bool {
	if response == nil {
		return false
	}
	text := strings.ToLower(strings.Join([]string{
		strings.TrimSpace(response.Message),
		strings.TrimSpace(response.Hint),
		strings.TrimSpace(response.Stage),
	}, " "))
	return strings.Contains(text, "busy") || strings.Contains(text, "discard")
}

func writeAll(writer io.Writer, data []byte) error {
	for len(data) > 0 {
		written, err := writer.Write(data)
		if err != nil {
			return err
		}
		if written <= 0 {
			return io.ErrShortWrite
		}
		data = data[written:]
	}
	return nil
}
