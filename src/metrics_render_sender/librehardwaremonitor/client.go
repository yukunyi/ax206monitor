package librehardwaremonitor

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

type LibreHardwareMonitorNode struct {
	ID       int                        `json:"id"`
	Text     string                     `json:"Text"`
	Min      string                     `json:"Min"`
	Value    string                     `json:"Value"`
	Max      string                     `json:"Max"`
	SensorID string                     `json:"SensorId"`
	Type     string                     `json:"Type"`
	Children []LibreHardwareMonitorNode `json:"Children"`
}

type LibreHardwareMonitorSensorSnapshot struct {
	SensorID string
	Name     string
	Type     string
	Unit     string
	Value    float64
}

type LibreHardwareMonitorSensorOption struct {
	SensorID string `json:"sensor_id"`
	Name     string `json:"name"`
	Type     string `json:"type"`
	Unit     string `json:"unit,omitempty"`
}

type LibreHardwareMonitorMonitorOption struct {
	Name  string `json:"name"`
	Label string `json:"label"`
	Unit  string `json:"unit,omitempty"`
}

type LibreHardwareMonitorData struct {
	CPUUsage        float64
	CPUTemp         float64
	CPUFreq         float64
	GPUUsage        float64
	GPUTemp         float64
	GPUFreq         float64
	MemoryUsage     float64
	MemoryUsed      float64
	MemoryTotal     float64
	NetworkUpload   float64
	NetworkDownload float64
	Fans            []FanInfo
	Sensors         map[string]LibreHardwareMonitorSensorSnapshot
	lastUpdate      time.Time
}

type LibreHardwareMonitorClient struct {
	baseURL    string
	username   string
	password   string
	httpClient *http.Client
	data       *LibreHardwareMonitorData
	options    []LibreHardwareMonitorMonitorOption
	sensorMap  map[string]string
	mutex      sync.RWMutex
}

var (
	libreHWMonitorClients   = make(map[string]*LibreHardwareMonitorClient)
	libreHWMonitorClientsMu sync.Mutex
)

func GetLibreHardwareMonitorClient(url, username, password string) *LibreHardwareMonitorClient {
	if strings.TrimSpace(url) == "" {
		url = "http://127.0.0.1:8085"
	}
	url = strings.TrimRight(url, "/")
	username = strings.TrimSpace(username)

	libreHWMonitorClientsMu.Lock()
	defer libreHWMonitorClientsMu.Unlock()

	key := fmt.Sprintf("%s|%s|%s", url, username, password)
	if client, ok := libreHWMonitorClients[key]; ok {
		return client
	}

	client := &LibreHardwareMonitorClient{
		baseURL:  url,
		username: username,
		password: password,
		httpClient: &http.Client{
			Timeout: 5 * time.Second,
		},
		data: &LibreHardwareMonitorData{
			Sensors: make(map[string]LibreHardwareMonitorSensorSnapshot),
		},
		options:   []LibreHardwareMonitorMonitorOption{},
		sensorMap: make(map[string]string),
	}
	libreHWMonitorClients[key] = client
	return client
}

func (c *LibreHardwareMonitorClient) FetchData() error {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	if time.Since(c.data.lastUpdate) < time.Second {
		return nil
	}

	url := c.baseURL + "/data.json"
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return fmt.Errorf("failed to build request for %s: %v", url, err)
	}
	if c.username != "" || c.password != "" {
		req.SetBasicAuth(c.username, c.password)
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to fetch data from %s: %v", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("HTTP error %d from %s", resp.StatusCode, url)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response body: %v", err)
	}

	var root LibreHardwareMonitorNode
	if err := json.Unmarshal(body, &root); err != nil {
		return fmt.Errorf("failed to parse JSON: %v", err)
	}

	c.parseData(&root)
	c.rebuildMonitorOptionsLocked()
	c.data.lastUpdate = time.Now()
	return nil
}

func (c *LibreHardwareMonitorClient) parseData(node *LibreHardwareMonitorNode) {
	c.data.CPUUsage = 0
	c.data.CPUTemp = 0
	c.data.CPUFreq = 0
	c.data.GPUUsage = 0
	c.data.GPUTemp = 0
	c.data.GPUFreq = 0
	c.data.MemoryUsage = 0
	c.data.MemoryUsed = 0
	c.data.MemoryTotal = 0
	c.data.NetworkUpload = 0
	c.data.NetworkDownload = 0
	c.data.Fans = c.data.Fans[:0]
	if c.data.Sensors == nil {
		c.data.Sensors = make(map[string]LibreHardwareMonitorSensorSnapshot)
	}
	for key := range c.data.Sensors {
		delete(c.data.Sensors, key)
	}

	c.parseNode(node)
}

func (c *LibreHardwareMonitorClient) parseNode(node *LibreHardwareMonitorNode) {
	if node.Type != "" && node.Value != "" {
		c.processValue(node)
	}
	for i := range node.Children {
		c.parseNode(&node.Children[i])
	}
}

func (c *LibreHardwareMonitorClient) processValue(node *LibreHardwareMonitorNode) {
	value, ok := parseLibreNumericValue(node.Value)

	sensorID := strings.ToLower(strings.TrimSpace(node.SensorID))
	nodeText := strings.ToLower(strings.TrimSpace(node.Text))
	nodeType := strings.TrimSpace(node.Type)

	if sensorID != "" && ok {
		c.data.Sensors[sensorID] = LibreHardwareMonitorSensorSnapshot{
			SensorID: sensorID,
			Name:     strings.TrimSpace(node.Text),
			Type:     nodeType,
			Unit:     parseLibreUnit(node.Value),
			Value:    value,
		}
	}

	if !ok {
		return
	}

	switch nodeType {
	case "Load":
		if strings.Contains(sensorID, "/intelcpu/") || strings.Contains(sensorID, "/amdcpu/") {
			if strings.Contains(nodeText, "cpu total") {
				c.data.CPUUsage = value
			}
		} else if strings.Contains(sensorID, "/gpu-nvidia/") || strings.Contains(sensorID, "/gpu-amd/") {
			if strings.Contains(nodeText, "gpu core") {
				c.data.GPUUsage = value
			}
		} else if strings.Contains(sensorID, "/ram/") {
			if strings.Contains(nodeText, "memory") && !strings.Contains(nodeText, "virtual") {
				c.data.MemoryUsage = value
			}
		}
	case "Temperature":
		if strings.Contains(sensorID, "/intelcpu/") || strings.Contains(sensorID, "/amdcpu/") {
			if strings.Contains(nodeText, "core max") || strings.Contains(nodeText, "cpu package") {
				c.data.CPUTemp = value
			}
		} else if strings.Contains(sensorID, "/gpu-nvidia/") || strings.Contains(sensorID, "/gpu-amd/") {
			if strings.Contains(nodeText, "gpu core") {
				c.data.GPUTemp = value
			}
		}
	case "Clock":
		if strings.Contains(sensorID, "/intelcpu/") || strings.Contains(sensorID, "/amdcpu/") {
			if strings.Contains(nodeText, "cpu core #1") {
				c.data.CPUFreq = value
			}
		} else if strings.Contains(sensorID, "/gpu-nvidia/") || strings.Contains(sensorID, "/gpu-amd/") {
			if strings.Contains(nodeText, "gpu core") {
				c.data.GPUFreq = value
			}
		}
	case "Data":
		if strings.Contains(sensorID, "/ram/") {
			if strings.Contains(nodeText, "memory used") && !strings.Contains(nodeText, "virtual") {
				c.data.MemoryUsed = value
			} else if strings.Contains(nodeText, "memory available") && !strings.Contains(nodeText, "virtual") {
				if c.data.MemoryUsed > 0 {
					c.data.MemoryTotal = c.data.MemoryUsed + value
				}
			}
		}
	case "Fan":
		c.data.Fans = append(c.data.Fans, FanInfo{
			Name:  strings.TrimSpace(node.Text),
			Speed: int(value),
			Index: len(c.data.Fans) + 1,
		})
	case "Throughput":
		if strings.Contains(sensorID, "/nic/") {
			if strings.Contains(nodeText, "upload speed") {
				if strings.Contains(node.Value, "KB/s") {
					c.data.NetworkUpload = value / 1024
				} else {
					c.data.NetworkUpload = value
				}
			} else if strings.Contains(nodeText, "download speed") {
				if strings.Contains(node.Value, "KB/s") {
					c.data.NetworkDownload = value / 1024
				} else {
					c.data.NetworkDownload = value
				}
			}
		}
	}
}

func parseLibreNumericValue(valueStr string) (float64, bool) {
	text := strings.TrimSpace(valueStr)
	if text == "" {
		return 0, false
	}
	replacer := strings.NewReplacer(
		"°C", "", "%", "", "MHz", "", "GHz", "", "kHz", "", "Hz", "",
		"GB", "", "MB", "", "KB/s", "", "MB/s", "", "GB/s", "",
		"RPM", "", "V", "", "W", "",
	)
	text = strings.TrimSpace(replacer.Replace(text))
	if text == "" {
		return 0, false
	}
	fields := strings.Fields(text)
	if len(fields) == 0 {
		return 0, false
	}
	number := strings.TrimSpace(fields[0])
	value, err := strconv.ParseFloat(number, 64)
	if err != nil {
		return 0, false
	}
	return value, true
}

func parseLibreUnit(valueStr string) string {
	text := strings.TrimSpace(valueStr)
	if text == "" {
		return ""
	}
	fields := strings.Fields(text)
	if len(fields) < 2 {
		return ""
	}
	return strings.Join(fields[1:], " ")
}

func (c *LibreHardwareMonitorClient) GetSensorValue(sensorID string) (float64, bool, error) {
	key := strings.ToLower(strings.TrimSpace(sensorID))
	if key == "" {
		return 0, false, fmt.Errorf("sensor_id is required")
	}
	if err := c.FetchData(); err != nil {
		return 0, false, err
	}
	data := c.GetData()
	if data == nil {
		return 0, false, nil
	}
	sensor, ok := data.Sensors[key]
	if !ok {
		return 0, false, nil
	}
	return sensor.Value, true, nil
}

func (c *LibreHardwareMonitorClient) ListSensorOptions() ([]LibreHardwareMonitorSensorOption, error) {
	if err := c.FetchData(); err != nil {
		return nil, err
	}
	data := c.GetData()
	if data == nil {
		return nil, nil
	}
	result := make([]LibreHardwareMonitorSensorOption, 0, len(data.Sensors))
	for _, sensor := range data.Sensors {
		result = append(result, LibreHardwareMonitorSensorOption{
			SensorID: sensor.SensorID,
			Name:     sensor.Name,
			Type:     sensor.Type,
			Unit:     sensor.Unit,
		})
	}
	sort.Slice(result, func(i, j int) bool {
		if result[i].Type != result[j].Type {
			return result[i].Type < result[j].Type
		}
		if result[i].Name != result[j].Name {
			return result[i].Name < result[j].Name
		}
		return result[i].SensorID < result[j].SensorID
	})
	return result, nil
}

func (c *LibreHardwareMonitorClient) ListMonitorOptions() ([]LibreHardwareMonitorMonitorOption, error) {
	if err := c.FetchData(); err != nil {
		return nil, err
	}
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	if len(c.options) == 0 {
		return []LibreHardwareMonitorMonitorOption{}, nil
	}
	options := make([]LibreHardwareMonitorMonitorOption, len(c.options))
	copy(options, c.options)
	return options, nil
}

func (c *LibreHardwareMonitorClient) GetMonitorValueByName(name string) (float64, string, bool, error) {
	monitorName := strings.TrimSpace(name)
	if monitorName == "" {
		return 0, "", false, fmt.Errorf("monitor name is required")
	}
	if err := c.FetchData(); err != nil {
		return 0, "", false, err
	}
	return c.GetMonitorValueByNameCached(monitorName)
}

func (c *LibreHardwareMonitorClient) GetMonitorValueByNameCached(name string) (float64, string, bool, error) {
	monitorName := strings.TrimSpace(name)
	if monitorName == "" {
		return 0, "", false, fmt.Errorf("monitor name is required")
	}
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	sensorID, ok := c.sensorMap[monitorName]
	if !ok || sensorID == "" {
		return 0, "", false, nil
	}
	sensor, exists := c.data.Sensors[sensorID]
	if !exists {
		return 0, "", false, nil
	}
	return sensor.Value, sensor.Unit, true, nil
}

func (c *LibreHardwareMonitorClient) GetData() *LibreHardwareMonitorData {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	dataCopy := *c.data
	fansCopy := make([]FanInfo, len(c.data.Fans))
	copy(fansCopy, c.data.Fans)
	dataCopy.Fans = fansCopy
	sensorsCopy := make(map[string]LibreHardwareMonitorSensorSnapshot, len(c.data.Sensors))
	for key, sensor := range c.data.Sensors {
		sensorsCopy[key] = sensor
	}
	dataCopy.Sensors = sensorsCopy

	return &dataCopy
}

func (c *LibreHardwareMonitorClient) rebuildMonitorOptionsLocked() {
	keys := make([]string, 0, len(c.data.Sensors))
	for sensorID := range c.data.Sensors {
		if strings.TrimSpace(sensorID) == "" {
			continue
		}
		keys = append(keys, sensorID)
	}
	sort.Strings(keys)

	options := make([]LibreHardwareMonitorMonitorOption, 0, len(keys))
	sensorMap := make(map[string]string, len(keys))
	usedNames := make(map[string]int)
	for _, sensorID := range keys {
		sensor := c.data.Sensors[sensorID]
		base := libreUniqueName("libre_"+sanitizeLibreName(sensorID), usedNames)
		label := buildLibreMonitorLabel(sensor)
		options = append(options, LibreHardwareMonitorMonitorOption{
			Name:  base,
			Label: label,
			Unit:  sensor.Unit,
		})
		sensorMap[base] = sensorID
	}
	c.options = options
	c.sensorMap = sensorMap
}

func sanitizeLibreName(name string) string {
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

func libreUniqueName(base string, used map[string]int) string {
	if _, exists := used[base]; !exists {
		used[base] = 1
		return base
	}
	used[base]++
	return fmt.Sprintf("%s_%d", base, used[base])
}

func buildLibreMonitorLabel(sensor LibreHardwareMonitorSensorSnapshot) string {
	parts := make([]string, 0, 4)
	if device := libreDeviceFromSensorID(sensor.SensorID); device != "" {
		parts = append(parts, device)
	}
	if strings.TrimSpace(sensor.Type) != "" {
		parts = append(parts, sensor.Type)
	}
	if strings.TrimSpace(sensor.Name) != "" {
		parts = append(parts, sensor.Name)
	} else if strings.TrimSpace(sensor.SensorID) != "" {
		parts = append(parts, sensor.SensorID)
	}
	if len(parts) == 0 {
		return "Monitor"
	}
	return strings.Join(parts, " ")
}

func libreDeviceFromSensorID(sensorID string) string {
	trimmed := strings.Trim(strings.TrimSpace(sensorID), "/")
	if trimmed == "" {
		return ""
	}
	segments := strings.Split(trimmed, "/")
	if len(segments) == 0 {
		return ""
	}
	first := libreHumanizeSegment(segments[0])
	if len(segments) >= 2 && segments[1] != "" && !isNumericString(segments[1]) {
		second := libreHumanizeSegment(segments[1])
		if second != "" && !strings.EqualFold(second, first) {
			return first + " " + second
		}
	}
	return first
}

func libreHumanizeSegment(segment string) string {
	text := strings.TrimSpace(segment)
	if text == "" {
		return ""
	}
	switch strings.ToLower(text) {
	case "amdcpu":
		return "AMD CPU"
	case "intelcpu":
		return "Intel CPU"
	case "gpu-nvidia":
		return "NVIDIA GPU"
	case "gpu-amd":
		return "AMD GPU"
	case "ram":
		return "Memory"
	case "nic":
		return "Network"
	case "lpc":
		return "LPC"
	}
	return strings.ReplaceAll(text, "_", " ")
}

func isNumericString(value string) bool {
	if value == "" {
		return false
	}
	for _, ch := range value {
		if ch < '0' || ch > '9' {
			return false
		}
	}
	return true
}
