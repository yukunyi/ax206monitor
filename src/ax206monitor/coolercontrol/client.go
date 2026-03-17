package coolercontrol

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"
)

type coolerControlTempStatus struct {
	Name string  `json:"name"`
	Temp float64 `json:"temp"`
}

type coolerControlChannelStatus struct {
	Name    string   `json:"name"`
	Duty    *float64 `json:"duty"`
	Freq    *int     `json:"freq"`
	PWMMode *int     `json:"pwm_mode"`
	RPM     *int     `json:"rpm"`
	Watts   *float64 `json:"watts"`
}

type coolerControlStatusHistory struct {
	Temps    []coolerControlTempStatus    `json:"temps"`
	Channels []coolerControlChannelStatus `json:"channels"`
}

type coolerControlDeviceStatus struct {
	Type          string                       `json:"type"`
	TypeIndex     int                          `json:"type_index"`
	UID           string                       `json:"uid"`
	StatusHistory []coolerControlStatusHistory `json:"status_history"`
}

type coolerControlStatusResponse struct {
	Devices []coolerControlDeviceStatus `json:"devices"`
}

type coolerControlDeviceInfoEntry struct {
	Label string `json:"label"`
}

type coolerControlDeviceInfo struct {
	Temps    map[string]coolerControlDeviceInfoEntry `json:"temps"`
	Channels map[string]coolerControlDeviceInfoEntry `json:"channels"`
}

type coolerControlDevicesResponse struct {
	Devices []coolerControlDeviceMeta `json:"devices"`
}

type coolerControlDeviceMeta struct {
	Type      string                  `json:"type"`
	TypeIndex int                     `json:"type_index"`
	UID       string                  `json:"uid"`
	Name      string                  `json:"name"`
	Info      coolerControlDeviceInfo `json:"info"`
}

type coolerControlDeviceSnapshot struct {
	Type      string
	TypeIndex int
	UID       string
	Temps     map[string]coolerControlTempStatus
	Channels  map[string]coolerControlChannelStatus
}

type coolerControlSnapshot struct {
	Devices map[string]coolerControlDeviceSnapshot
}

type coolerControlDeviceNameMap struct {
	Type          string
	TypeIndex     int
	Name          string
	TempLabels    map[string]string
	ChannelLabels map[string]string
}

type CoolerControlMonitorOption struct {
	Name  string `json:"name"`
	Label string `json:"label"`
	Unit  string `json:"unit,omitempty"`
}

type coolerControlMonitorEntry struct {
	CoolerControlMonitorOption
	Value float64
}

type CoolerControlClient struct {
	baseURL  string
	password string

	apiClient    *http.Client
	streamClient *http.Client

	startOnce sync.Once
	readyOnce sync.Once
	readyCh   chan struct{}

	mutex      sync.RWMutex
	snapshot   *coolerControlSnapshot
	snapshotAt time.Time
	lastErr    error
	options    []CoolerControlMonitorOption
	valueMap   map[string]coolerControlMonitorEntry
	deviceMeta map[string]coolerControlDeviceNameMap
}

var (
	coolerControlClients   = make(map[string]*CoolerControlClient)
	coolerControlClientsMu sync.Mutex

	coolerControlCPUModelRegex = regexp.MustCompile(`(?i)\b(i[3579]-\d{4,5}[a-z]{0,3}|\d{4,5}[a-z]{0,4})\b`)
	coolerControlGPUModelRegex = regexp.MustCompile(`(?i)\b((?:RTX|GTX)\s*\d{3,4}(?:\s*(?:ti|super))?|RX\s*\d{3,4}(?:\s*(?:xtx|xt|gre))?)\b`)

	coolerControlSessionUsername = "CCAdmin"
	errCoolerControlUnauthorized = errors.New("coolercontrol unauthorized")
)

func SessionUsername() string {
	return coolerControlSessionUsername
}

func GetCoolerControlClient(baseURL, password string) *CoolerControlClient {
	url := strings.TrimRight(baseURL, "/")
	key := fmt.Sprintf("%s|%s", url, password)

	coolerControlClientsMu.Lock()
	defer coolerControlClientsMu.Unlock()

	if client, ok := coolerControlClients[key]; ok {
		return client
	}

	jar, _ := cookiejar.New(nil)
	client := &CoolerControlClient{
		baseURL:  url,
		password: password,
		apiClient: &http.Client{
			Timeout: 5 * time.Second,
			Jar:     jar,
		},
		streamClient: &http.Client{
			Jar: jar,
		},
		readyCh:    make(chan struct{}),
		deviceMeta: make(map[string]coolerControlDeviceNameMap),
	}
	coolerControlClients[key] = client
	return client
}

func (c *CoolerControlClient) FetchSnapshot() (*coolerControlSnapshot, error) {
	c.startOnce.Do(func() {
		go c.runSSELoop()
	})

	if snapshot := c.getFreshSnapshot(15 * time.Second); snapshot != nil {
		return snapshot, nil
	}

	select {
	case <-c.readyCh:
	case <-time.After(1500 * time.Millisecond):
	}

	if snapshot := c.getFreshSnapshot(15 * time.Second); snapshot != nil {
		return snapshot, nil
	}

	// Fallback: if SSE is temporarily unavailable, perform one normal status request.
	status, err := c.getStatus()
	if err == nil {
		snapshot := buildCoolerControlSnapshot(status)
		c.setSnapshot(snapshot)
		return snapshot, nil
	}

	c.mutex.RLock()
	lastErr := c.lastErr
	c.mutex.RUnlock()

	if lastErr != nil {
		return nil, lastErr
	}
	return nil, err
}

func (c *CoolerControlClient) GetTemperature(deviceUID, tempName string) (float64, bool, error) {
	if tempName == "" {
		return 0, false, fmt.Errorf("temperature name is required")
	}

	snapshot, err := c.FetchSnapshot()
	if err != nil {
		return 0, false, err
	}

	key := strings.ToLower(tempName)
	if deviceUID != "" {
		device, ok := snapshot.Devices[deviceUID]
		if !ok {
			return 0, false, nil
		}
		temp, ok := device.Temps[key]
		if !ok {
			return 0, false, nil
		}
		return temp.Temp, true, nil
	}

	for _, device := range snapshot.Devices {
		if temp, ok := device.Temps[key]; ok {
			return temp.Temp, true, nil
		}
	}
	return 0, false, nil
}

func (c *CoolerControlClient) GetChannelMetric(deviceUID, channelName, metric string) (float64, bool, error) {
	if channelName == "" {
		return 0, false, fmt.Errorf("channel name is required")
	}

	snapshot, err := c.FetchSnapshot()
	if err != nil {
		return 0, false, err
	}

	channelKey := strings.ToLower(channelName)
	metricKey := strings.ToLower(strings.TrimSpace(metric))
	if metricKey == "" {
		metricKey = "rpm"
	}
	if !isSupportedCoolerControlMetric(metricKey) {
		return 0, false, fmt.Errorf("unsupported coolercontrol metric: %s", metricKey)
	}

	if deviceUID != "" {
		device, ok := snapshot.Devices[deviceUID]
		if !ok {
			return 0, false, nil
		}
		return extractCoolerControlChannelMetric(device.Channels[channelKey], metricKey)
	}

	for _, device := range snapshot.Devices {
		if value, ok, _ := extractCoolerControlChannelMetric(device.Channels[channelKey], metricKey); ok {
			return value, true, nil
		}
	}
	return 0, false, nil
}

func (c *CoolerControlClient) ListMonitorOptions() ([]CoolerControlMonitorOption, error) {
	if _, err := c.FetchSnapshot(); err != nil {
		return nil, err
	}
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	if len(c.options) == 0 {
		return []CoolerControlMonitorOption{}, nil
	}
	options := make([]CoolerControlMonitorOption, len(c.options))
	copy(options, c.options)
	return options, nil
}

func (c *CoolerControlClient) GetMonitorValueByName(name string) (float64, string, bool, error) {
	monitorName := strings.TrimSpace(name)
	if monitorName == "" {
		return 0, "", false, fmt.Errorf("monitor name is required")
	}
	if _, err := c.FetchSnapshot(); err != nil {
		return 0, "", false, err
	}
	return c.GetMonitorValueByNameCached(monitorName)
}

func (c *CoolerControlClient) GetMonitorValueByNameCached(name string) (float64, string, bool, error) {
	monitorName := strings.TrimSpace(name)
	if monitorName == "" {
		return 0, "", false, fmt.Errorf("monitor name is required")
	}
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	if entry, ok := c.valueMap[monitorName]; ok {
		return entry.Value, entry.Unit, true, nil
	}
	return 0, "", false, nil
}

func isSupportedCoolerControlMetric(metric string) bool {
	switch metric {
	case "rpm", "duty", "percent", "freq", "frequency", "watts", "power":
		return true
	default:
		return false
	}
}

func (c *CoolerControlClient) runSSELoop() {
	retryDelay := time.Second
	for {
		err := c.consumeSSE()
		if err != nil {
			if errors.Is(err, io.EOF) {
				c.setLastError(nil)
				retryDelay = time.Second
			}
			if errors.Is(err, errCoolerControlUnauthorized) && c.password != "" {
				if loginErr := c.login(); loginErr != nil {
					c.setLastError(loginErr)
					logWarnModule("coolercontrol", "login failed: %v", loginErr)
				} else {
					c.setLastError(nil)
					retryDelay = time.Second
					continue
				}
			} else if !errors.Is(err, io.EOF) {
				c.setLastError(err)
				logWarnModule("coolercontrol", "sse disconnected: %v", err)
			}
		}

		time.Sleep(retryDelay)
		if retryDelay < 8*time.Second {
			retryDelay *= 2
		}
	}
}

func (c *CoolerControlClient) consumeSSE() error {
	req, err := http.NewRequest(http.MethodGet, c.baseURL+"/sse/status", nil)
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "text/event-stream")

	resp, err := c.streamClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized {
		return errCoolerControlUnauthorized
	}
	if resp.StatusCode != http.StatusOK {
		message, _ := io.ReadAll(io.LimitReader(resp.Body, 256))
		return fmt.Errorf("coolercontrol sse request failed: %d %s", resp.StatusCode, strings.TrimSpace(string(message)))
	}

	scanner := bufio.NewScanner(resp.Body)
	scanner.Buffer(make([]byte, 64*1024), 2*1024*1024)

	eventName := ""
	var dataLines []string
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, ":") {
			continue
		}
		if line == "" {
			if len(dataLines) > 0 {
				if eventName == "" || eventName == "status" {
					payload := strings.Join(dataLines, "\n")
					c.applySSEPayload(payload)
				}
				dataLines = dataLines[:0]
			}
			eventName = ""
			continue
		}
		if strings.HasPrefix(line, "event:") {
			eventName = strings.TrimSpace(line[len("event:"):])
			continue
		}
		if strings.HasPrefix(line, "data:") {
			data := strings.TrimLeft(line[len("data:"):], " ")
			dataLines = append(dataLines, data)
		}
	}

	if err := scanner.Err(); err != nil {
		return err
	}
	return io.EOF
}

func (c *CoolerControlClient) applySSEPayload(payload string) {
	var status coolerControlStatusResponse
	if err := json.Unmarshal([]byte(payload), &status); err != nil {
		logDebugModule("coolercontrol", "ignore invalid sse payload: %v", err)
		return
	}

	snapshot := buildCoolerControlSnapshot(&status)
	c.setSnapshot(snapshot)
	c.setLastError(nil)
}

func (c *CoolerControlClient) getStatus() (*coolerControlStatusResponse, error) {
	status, statusCode, err := c.requestStatus()
	if err == nil {
		return status, nil
	}

	if statusCode == http.StatusUnauthorized && c.password != "" {
		if loginErr := c.login(); loginErr != nil {
			return nil, loginErr
		}
		status, _, retryErr := c.requestStatus()
		if retryErr != nil {
			return nil, retryErr
		}
		return status, nil
	}
	return nil, err
}

func (c *CoolerControlClient) requestStatus() (*coolerControlStatusResponse, int, error) {
	req, err := http.NewRequest(http.MethodGet, c.baseURL+"/status", nil)
	if err != nil {
		return nil, 0, err
	}

	resp, err := c.apiClient.Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		message, _ := io.ReadAll(io.LimitReader(resp.Body, 256))
		return nil, resp.StatusCode, fmt.Errorf("coolercontrol status request failed: %d %s", resp.StatusCode, strings.TrimSpace(string(message)))
	}

	var payload coolerControlStatusResponse
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return nil, resp.StatusCode, fmt.Errorf("failed to decode coolercontrol status: %w", err)
	}
	return &payload, resp.StatusCode, nil
}

func (c *CoolerControlClient) ensureDeviceMeta() map[string]coolerControlDeviceNameMap {
	c.mutex.RLock()
	if len(c.deviceMeta) > 0 {
		copyMap := cloneCoolerControlDeviceMetaMap(c.deviceMeta)
		c.mutex.RUnlock()
		return copyMap
	}
	c.mutex.RUnlock()

	meta, err := c.fetchDeviceMeta()
	if err != nil {
		logDebugModule("coolercontrol", "fetch devices metadata failed: %v", err)
		c.mutex.RLock()
		copyMap := cloneCoolerControlDeviceMetaMap(c.deviceMeta)
		c.mutex.RUnlock()
		return copyMap
	}

	c.mutex.Lock()
	c.deviceMeta = meta
	c.mutex.Unlock()
	return cloneCoolerControlDeviceMetaMap(meta)
}

func (c *CoolerControlClient) fetchDeviceMeta() (map[string]coolerControlDeviceNameMap, error) {
	req, err := http.NewRequest(http.MethodGet, c.baseURL+"/devices", nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.apiClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		message, _ := io.ReadAll(io.LimitReader(resp.Body, 256))
		return nil, fmt.Errorf("coolercontrol devices request failed: %d %s", resp.StatusCode, strings.TrimSpace(string(message)))
	}

	var payload coolerControlDevicesResponse
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return nil, fmt.Errorf("failed to decode coolercontrol devices: %w", err)
	}

	result := make(map[string]coolerControlDeviceNameMap, len(payload.Devices))
	for _, device := range payload.Devices {
		uid := strings.TrimSpace(device.UID)
		if uid == "" {
			continue
		}

		tempLabels := make(map[string]string)
		for raw, meta := range device.Info.Temps {
			key := strings.ToLower(strings.TrimSpace(raw))
			if key == "" {
				continue
			}
			label := strings.TrimSpace(meta.Label)
			if label == "" {
				label = strings.TrimSpace(raw)
			}
			tempLabels[key] = label
		}

		channelLabels := make(map[string]string)
		for raw, meta := range device.Info.Channels {
			key := strings.ToLower(strings.TrimSpace(raw))
			if key == "" {
				continue
			}
			label := strings.TrimSpace(meta.Label)
			if label == "" {
				label = strings.TrimSpace(raw)
			}
			channelLabels[key] = label
		}

		result[uid] = coolerControlDeviceNameMap{
			Type:          strings.TrimSpace(device.Type),
			TypeIndex:     device.TypeIndex,
			Name:          strings.TrimSpace(device.Name),
			TempLabels:    tempLabels,
			ChannelLabels: channelLabels,
		}
	}
	return result, nil
}

func (c *CoolerControlClient) login() error {
	req, err := http.NewRequest(http.MethodPost, c.baseURL+"/login", strings.NewReader("{}"))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.SetBasicAuth(coolerControlSessionUsername, c.password)

	resp, err := c.apiClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		message, _ := io.ReadAll(io.LimitReader(resp.Body, 256))
		return fmt.Errorf("coolercontrol login failed: %d %s", resp.StatusCode, strings.TrimSpace(string(message)))
	}
	return nil
}

func (c *CoolerControlClient) getFreshSnapshot(maxAge time.Duration) *coolerControlSnapshot {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	if c.snapshot == nil {
		return nil
	}
	if maxAge > 0 && time.Since(c.snapshotAt) > maxAge {
		return nil
	}
	return c.snapshot
}

func (c *CoolerControlClient) setSnapshot(snapshot *coolerControlSnapshot) {
	meta := c.ensureDeviceMeta()
	entries := buildCoolerControlMonitorEntries(snapshot, meta)
	options := make([]CoolerControlMonitorOption, 0, len(entries))
	valueMap := make(map[string]coolerControlMonitorEntry, len(entries))
	for _, entry := range entries {
		options = append(options, entry.CoolerControlMonitorOption)
		valueMap[entry.Name] = entry
	}

	c.mutex.Lock()
	c.snapshot = snapshot
	c.snapshotAt = time.Now()
	c.options = options
	c.valueMap = valueMap
	c.mutex.Unlock()

	c.readyOnce.Do(func() {
		close(c.readyCh)
	})
}

func (c *CoolerControlClient) setLastError(err error) {
	c.mutex.Lock()
	c.lastErr = err
	c.mutex.Unlock()
}

func buildCoolerControlSnapshot(status *coolerControlStatusResponse) *coolerControlSnapshot {
	snapshot := &coolerControlSnapshot{
		Devices: make(map[string]coolerControlDeviceSnapshot),
	}

	for _, device := range status.Devices {
		if device.UID == "" || len(device.StatusHistory) == 0 {
			continue
		}

		latest := device.StatusHistory[len(device.StatusHistory)-1]
		deviceSnapshot := coolerControlDeviceSnapshot{
			Type:      strings.TrimSpace(device.Type),
			TypeIndex: device.TypeIndex,
			UID:       strings.TrimSpace(device.UID),
			Temps:     make(map[string]coolerControlTempStatus),
			Channels:  make(map[string]coolerControlChannelStatus),
		}

		for _, temp := range latest.Temps {
			key := strings.ToLower(strings.TrimSpace(temp.Name))
			if key == "" {
				continue
			}
			deviceSnapshot.Temps[key] = temp
		}
		for _, channel := range latest.Channels {
			key := strings.ToLower(strings.TrimSpace(channel.Name))
			if key == "" {
				continue
			}
			deviceSnapshot.Channels[key] = channel
		}
		snapshot.Devices[device.UID] = deviceSnapshot
	}

	return snapshot
}

func buildCoolerControlMonitorEntries(snapshot *coolerControlSnapshot, meta map[string]coolerControlDeviceNameMap) []coolerControlMonitorEntry {
	if snapshot == nil || len(snapshot.Devices) == 0 {
		return []coolerControlMonitorEntry{}
	}

	deviceUIDs := make([]string, 0, len(snapshot.Devices))
	for uid := range snapshot.Devices {
		deviceUIDs = append(deviceUIDs, uid)
	}
	sort.Slice(deviceUIDs, func(i, j int) bool {
		left := snapshot.Devices[deviceUIDs[i]]
		right := snapshot.Devices[deviceUIDs[j]]
		leftType := strings.ToLower(strings.TrimSpace(left.Type))
		rightType := strings.ToLower(strings.TrimSpace(right.Type))
		if leftType != rightType {
			return leftType < rightType
		}
		if left.TypeIndex != right.TypeIndex {
			return left.TypeIndex < right.TypeIndex
		}
		return deviceUIDs[i] < deviceUIDs[j]
	})

	entries := make([]coolerControlMonitorEntry, 0)
	usedNames := make(map[string]int)
	for _, uid := range deviceUIDs {
		device := snapshot.Devices[uid]
		deviceKey := coolerControlDeviceKey(device)
		deviceLabel := coolerControlDeviceLabel(device, meta[uid])
		deviceTypeToken := sanitizeCoolerControlName(device.Type)
		if deviceTypeToken == "unknown" || deviceTypeToken == "" {
			deviceTypeToken = "device"
		}

		tempKeys := make([]string, 0, len(device.Temps))
		for key := range device.Temps {
			tempKeys = append(tempKeys, key)
		}
		sort.Strings(tempKeys)
		for _, tempKey := range tempKeys {
			temp := device.Temps[tempKey]
			displayName := coolerControlResolveTempLabel(tempKey, temp, meta[uid])
			nameToken := coolerControlNormalizeMetricToken(displayName, tempKey, deviceTypeToken)
			entries = append(entries, coolerControlMonitorEntry{
				CoolerControlMonitorOption: CoolerControlMonitorOption{
					Name:  coolerControlUniqueName("coolercontrol_"+deviceKey+"_"+nameToken, usedNames),
					Label: coolerControlBuildShortLabel(deviceLabel, displayName),
					Unit:  "°C",
				},
				Value: temp.Temp,
			})
		}

		channelKeys := make([]string, 0, len(device.Channels))
		for key := range device.Channels {
			channelKeys = append(channelKeys, key)
		}
		sort.Strings(channelKeys)
		for _, channelKey := range channelKeys {
			channel := device.Channels[channelKey]
			channelName := coolerControlResolveChannelLabel(channelKey, channel, meta[uid])
			channelToken := coolerControlNormalizeMetricToken(channelName, channelKey, deviceTypeToken)
			metricCount := coolerControlChannelMetricCount(channel)

			if channel.RPM != nil {
				entries = appendCoolerControlChannelEntry(
					entries,
					usedNames,
					deviceKey,
					deviceLabel,
					channelName,
					channelToken,
					metricCount,
					"rpm",
					"RPM",
					"RPM",
					float64(*channel.RPM),
				)
			}
			if channel.Duty != nil {
				entries = appendCoolerControlChannelEntry(
					entries,
					usedNames,
					deviceKey,
					deviceLabel,
					channelName,
					channelToken,
					metricCount,
					"duty",
					"Duty",
					"%",
					*channel.Duty,
				)
			}
			if channel.Freq != nil {
				entries = appendCoolerControlChannelEntry(
					entries,
					usedNames,
					deviceKey,
					deviceLabel,
					channelName,
					channelToken,
					metricCount,
					"freq",
					"Freq",
					"MHz",
					float64(*channel.Freq),
				)
			}
			if channel.Watts != nil {
				entries = appendCoolerControlChannelEntry(
					entries,
					usedNames,
					deviceKey,
					deviceLabel,
					channelName,
					channelToken,
					metricCount,
					"power",
					"Power",
					"W",
					*channel.Watts,
				)
			}
		}
	}

	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Name < entries[j].Name
	})
	return entries
}

func sanitizeCoolerControlName(name string) string {
	text := strings.ToLower(strings.TrimSpace(name))
	if text == "" {
		return "unknown"
	}
	var builder strings.Builder
	lastUnderscore := false
	for _, ch := range text {
		isAlpha := ch >= 'a' && ch <= 'z'
		isDigit := ch >= '0' && ch <= '9'
		if isAlpha || isDigit {
			builder.WriteRune(ch)
			lastUnderscore = false
			continue
		}
		if !lastUnderscore {
			builder.WriteByte('_')
			lastUnderscore = true
		}
	}
	result := strings.Trim(builder.String(), "_")
	if result == "" {
		return "unknown"
	}
	return result
}

func coolerControlUniqueName(base string, used map[string]int) string {
	if _, exists := used[base]; !exists {
		used[base] = 1
		return base
	}
	used[base]++
	return fmt.Sprintf("%s_%d", base, used[base])
}

func extractCoolerControlChannelMetric(channel coolerControlChannelStatus, metric string) (float64, bool, error) {
	switch metric {
	case "rpm":
		if channel.RPM == nil {
			return 0, false, nil
		}
		return float64(*channel.RPM), true, nil
	case "duty", "percent":
		if channel.Duty == nil {
			return 0, false, nil
		}
		return *channel.Duty, true, nil
	case "freq", "frequency":
		if channel.Freq == nil {
			return 0, false, nil
		}
		return float64(*channel.Freq), true, nil
	case "watts", "power":
		if channel.Watts == nil {
			return 0, false, nil
		}
		return *channel.Watts, true, nil
	default:
		return 0, false, fmt.Errorf("unsupported coolercontrol metric: %s", metric)
	}
}

func coolerControlResolveTempLabel(rawKey string, temp coolerControlTempStatus, meta coolerControlDeviceNameMap) string {
	key := strings.ToLower(strings.TrimSpace(rawKey))
	if key != "" {
		if label := strings.TrimSpace(meta.TempLabels[key]); label != "" {
			return label
		}
	}
	if label := strings.TrimSpace(temp.Name); label != "" {
		return label
	}
	if key != "" {
		return key
	}
	return "temp"
}

func coolerControlResolveChannelLabel(rawKey string, channel coolerControlChannelStatus, meta coolerControlDeviceNameMap) string {
	key := strings.ToLower(strings.TrimSpace(rawKey))
	if key != "" {
		if label := strings.TrimSpace(meta.ChannelLabels[key]); label != "" {
			return label
		}
	}
	if label := strings.TrimSpace(channel.Name); label != "" {
		return label
	}
	if key != "" {
		return key
	}
	return "channel"
}

func coolerControlNormalizeMetricToken(displayName string, fallbackRaw string, deviceTypeToken string) string {
	token := sanitizeCoolerControlName(displayName)
	fallback := sanitizeCoolerControlName(fallbackRaw)
	if token == "unknown" || token == "" {
		token = fallback
	}
	if token == "unknown" || token == "" {
		token = "metric"
	}

	if deviceTypeToken != "" && deviceTypeToken != "unknown" {
		token = strings.TrimPrefix(token, deviceTypeToken+"_")
		switch deviceTypeToken {
		case "cpu":
			token = strings.TrimPrefix(token, "cpu_")
		case "gpu":
			token = strings.TrimPrefix(token, "gpu_")
		}
	}

	token = strings.Trim(token, "_")
	if token == "" {
		token = fallback
	}
	if token == "" || token == "unknown" {
		token = "metric"
	}
	return token
}

func coolerControlChannelMetricCount(channel coolerControlChannelStatus) int {
	count := 0
	if channel.RPM != nil {
		count++
	}
	if channel.Duty != nil {
		count++
	}
	if channel.Freq != nil {
		count++
	}
	if channel.Watts != nil {
		count++
	}
	return count
}

func coolerControlCompactSpaces(text string) string {
	return strings.Join(strings.Fields(strings.TrimSpace(text)), " ")
}

func coolerControlNormalizeMetricText(text string) string {
	raw := coolerControlCompactSpaces(text)
	if raw == "" {
		return ""
	}
	aliases := map[string]string{
		"temp":        "Temp",
		"temperature": "Temp",
		"freq":        "Freq",
		"frequency":   "Freq",
		"load":        "Load",
		"power":       "Power",
		"duty":        "Duty",
		"rpm":         "RPM",
		"fan":         "Fan",
		"pump":        "Pump",
		"avg":         "Avg",
		"average":     "Avg",
		"min":         "Min",
		"max":         "Max",
		"core":        "Core",
		"mem":         "Mem",
		"memory":      "Mem",
		"clock":       "Clock",
	}
	parts := strings.Fields(raw)
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		clean := strings.Trim(part, " _-")
		if clean == "" {
			continue
		}
		if alias, ok := aliases[strings.ToLower(clean)]; ok {
			out = append(out, alias)
			continue
		}
		out = append(out, clean)
	}
	return strings.Join(out, " ")
}

func coolerControlTrimLeadingToken(text string, token string) string {
	parts := strings.Fields(coolerControlCompactSpaces(text))
	if len(parts) == 0 {
		return ""
	}
	if strings.EqualFold(parts[0], token) {
		return strings.Join(parts[1:], " ")
	}
	return strings.Join(parts, " ")
}

func coolerControlExtractCPUModel(deviceLabel string) string {
	match := coolerControlCPUModelRegex.FindStringSubmatch(coolerControlCompactSpaces(deviceLabel))
	if len(match) < 2 {
		return ""
	}
	return strings.ToUpper(coolerControlCompactSpaces(match[1]))
}

func coolerControlExtractGPUModel(deviceLabel string) string {
	match := coolerControlGPUModelRegex.FindStringSubmatch(coolerControlCompactSpaces(deviceLabel))
	if len(match) < 2 {
		return ""
	}
	return strings.ToUpper(coolerControlCompactSpaces(match[1]))
}

func coolerControlGuessKind(deviceLabel string, metricText string) string {
	metricLower := strings.ToLower(coolerControlCompactSpaces(metricText))
	if strings.Contains(metricLower, " gpu ") || strings.HasPrefix(metricLower, "gpu ") || strings.HasSuffix(metricLower, " gpu") || metricLower == "gpu" {
		return "GPU"
	}
	if strings.Contains(metricLower, " cpu ") || strings.HasPrefix(metricLower, "cpu ") || strings.HasSuffix(metricLower, " cpu") || metricLower == "cpu" {
		return "CPU"
	}

	combined := strings.ToLower(coolerControlCompactSpaces(deviceLabel + " " + metricText))
	if strings.Contains(combined, "amdgpu") || strings.Contains(combined, " nvidia") ||
		strings.Contains(combined, "geforce") || strings.Contains(combined, "radeon") ||
		strings.Contains(combined, " rtx") || strings.Contains(combined, " gtx") ||
		strings.Contains(combined, " rx ") || strings.Contains(combined, " gpu") {
		return "GPU"
	}
	if strings.Contains(combined, "ryzen") || strings.Contains(combined, "threadripper") ||
		strings.Contains(combined, "xeon") || strings.Contains(combined, "intel") ||
		strings.Contains(combined, " cpu") {
		return "CPU"
	}
	return ""
}

func coolerControlBuildShortLabel(deviceLabel string, metricText string) string {
	device := coolerControlCompactSpaces(deviceLabel)
	metric := coolerControlNormalizeMetricText(metricText)
	kind := coolerControlGuessKind(device, metric)

	if kind == "CPU" {
		model := coolerControlExtractCPUModel(device)
		tail := coolerControlTrimLeadingToken(metric, "CPU")
		parts := []string{"CPU"}
		if model != "" {
			parts = append(parts, model)
		}
		if tail != "" {
			parts = append(parts, tail)
		}
		return strings.Join(parts, " ")
	}

	if kind == "GPU" {
		model := coolerControlExtractGPUModel(device)
		prefix := ""
		if model != "" {
			prefix = model
		} else {
			deviceLower := strings.ToLower(device)
			switch {
			case strings.Contains(deviceLower, "amdgpu"), strings.Contains(deviceLower, "amd"):
				prefix = "AMD"
			case strings.Contains(deviceLower, "nvidia"):
				prefix = "NVIDIA"
			case strings.Contains(deviceLower, "intel"):
				prefix = "Intel"
			}
		}
		tail := coolerControlTrimLeadingToken(metric, "GPU")
		parts := []string{}
		if prefix != "" {
			parts = append(parts, prefix)
		}
		parts = append(parts, "GPU")
		if tail != "" {
			parts = append(parts, tail)
		}
		return strings.Join(parts, " ")
	}

	if metric != "" {
		return metric
	}
	if device != "" {
		return device
	}
	return "CoolerControl"
}

func BuildShortLabel(deviceLabel string, metricText string) string {
	return coolerControlBuildShortLabel(deviceLabel, metricText)
}

func coolerControlDeviceKey(device coolerControlDeviceSnapshot) string {
	typeName := sanitizeCoolerControlName(device.Type)
	if typeName == "unknown" {
		typeName = "device"
	}
	if device.TypeIndex > 0 {
		return fmt.Sprintf("%s%d", typeName, device.TypeIndex)
	}
	uidSuffix := coolerControlUIDShort(device.UID)
	if uidSuffix == "" {
		return typeName
	}
	return typeName + "_" + uidSuffix
}

func coolerControlDeviceLabel(device coolerControlDeviceSnapshot, meta coolerControlDeviceNameMap) string {
	if strings.TrimSpace(meta.Name) != "" {
		return strings.TrimSpace(meta.Name)
	}
	typeName := strings.TrimSpace(device.Type)
	if typeName == "" {
		typeName = strings.TrimSpace(meta.Type)
	}
	if typeName == "" {
		typeName = "Device"
	}
	if device.TypeIndex > 0 {
		return fmt.Sprintf("%s#%d", typeName, device.TypeIndex)
	}
	uidSuffix := coolerControlUIDShort(device.UID)
	if uidSuffix == "" {
		return typeName
	}
	return typeName + " " + uidSuffix
}

func coolerControlUIDShort(uid string) string {
	text := sanitizeCoolerControlName(uid)
	if text == "unknown" || text == "" {
		return ""
	}
	if len(text) > 8 {
		return text[:8]
	}
	return text
}

func cloneCoolerControlDeviceMetaMap(source map[string]coolerControlDeviceNameMap) map[string]coolerControlDeviceNameMap {
	if len(source) == 0 {
		return map[string]coolerControlDeviceNameMap{}
	}
	result := make(map[string]coolerControlDeviceNameMap, len(source))
	for uid, item := range source {
		tempLabels := make(map[string]string, len(item.TempLabels))
		for key, label := range item.TempLabels {
			tempLabels[key] = label
		}
		channelLabels := make(map[string]string, len(item.ChannelLabels))
		for key, label := range item.ChannelLabels {
			channelLabels[key] = label
		}
		result[uid] = coolerControlDeviceNameMap{
			Type:          item.Type,
			TypeIndex:     item.TypeIndex,
			Name:          item.Name,
			TempLabels:    tempLabels,
			ChannelLabels: channelLabels,
		}
	}
	return result
}

func appendCoolerControlChannelEntry(
	entries []coolerControlMonitorEntry,
	usedNames map[string]int,
	deviceKey string,
	deviceLabel string,
	channelName string,
	channelToken string,
	metricCount int,
	metricKey string,
	metricLabel string,
	unit string,
	value float64,
) []coolerControlMonitorEntry {
	baseName := "coolercontrol_" + deviceKey + "_" + channelToken
	needSuffix := metricCount > 1
	if !needSuffix {
		switch metricKey {
		case "rpm":
			needSuffix = !strings.Contains(channelToken, "rpm")
		case "duty":
			needSuffix = !strings.Contains(channelToken, "duty") && !strings.Contains(channelToken, "load")
		case "freq":
			needSuffix = !strings.Contains(channelToken, "freq")
		case "power":
			needSuffix = !strings.Contains(channelToken, "power") && !strings.Contains(channelToken, "watt")
		default:
			needSuffix = true
		}
	}
	if needSuffix && !strings.Contains(channelToken, metricKey) {
		baseName += "_" + metricKey
	}
	channelLower := strings.ToLower(channelName)
	needMetricLabel := !strings.Contains(channelLower, strings.ToLower(metricLabel))
	if metricKey == "duty" && strings.Contains(channelLower, "load") {
		needMetricLabel = false
	}
	metricText := channelName
	if needMetricLabel {
		metricText += " " + metricLabel
	}
	label := coolerControlBuildShortLabel(deviceLabel, metricText)
	return append(entries, coolerControlMonitorEntry{
		CoolerControlMonitorOption: CoolerControlMonitorOption{
			Name:  coolerControlUniqueName(baseName, usedNames),
			Label: label,
			Unit:  unit,
		},
		Value: value,
	})
}
