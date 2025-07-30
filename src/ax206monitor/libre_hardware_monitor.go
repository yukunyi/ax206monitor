package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
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
	lastUpdate      time.Time
}

type LibreHardwareMonitorClient struct {
	baseURL    string
	httpClient *http.Client
	data       *LibreHardwareMonitorData
	mutex      sync.RWMutex
}

var (
	libreHWMonitorClient *LibreHardwareMonitorClient
	libreHWMonitorOnce   sync.Once
)

func GetLibreHardwareMonitorClient(url string) *LibreHardwareMonitorClient {
	libreHWMonitorOnce.Do(func() {
		if url == "" {
			url = "http://127.0.0.1:8085"
		}
		libreHWMonitorClient = &LibreHardwareMonitorClient{
			baseURL: url,
			httpClient: &http.Client{
				Timeout: 5 * time.Second,
			},
			data: &LibreHardwareMonitorData{},
		}
	})
	return libreHWMonitorClient
}

func (c *LibreHardwareMonitorClient) FetchData() error {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	// Check if data is still fresh (within 1 second)
	if time.Since(c.data.lastUpdate) < time.Second {
		return nil
	}

	url := c.baseURL + "/data.json"
	resp, err := c.httpClient.Get(url)
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
	c.data.lastUpdate = time.Now()
	return nil
}

func (c *LibreHardwareMonitorClient) parseData(node *LibreHardwareMonitorNode) {

	c.data.Fans = c.data.Fans[:0]

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
	value := c.parseNumericValue(node.Value)
	if value == 0 {
		return
	}

	sensorID := strings.ToLower(node.SensorID)
	nodeText := strings.ToLower(node.Text)

	switch node.Type {
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
				// Total memory = used + available
				if c.data.MemoryUsed > 0 {
					c.data.MemoryTotal = c.data.MemoryUsed + value
				}
			}
		}

	case "Fan":
		fanInfo := FanInfo{
			Name:  node.Text,
			Speed: int(value),
			Index: len(c.data.Fans) + 1,
		}
		c.data.Fans = append(c.data.Fans, fanInfo)

	case "Throughput":
		if strings.Contains(sensorID, "/nic/") {
			if strings.Contains(nodeText, "upload speed") {
				// Convert from KB/s to MB/s if needed
				if strings.Contains(node.Value, "KB/s") {
					c.data.NetworkUpload = value / 1024
				} else {
					c.data.NetworkUpload = value
				}
			} else if strings.Contains(nodeText, "download speed") {
				// Convert from KB/s to MB/s if needed
				if strings.Contains(node.Value, "KB/s") {
					c.data.NetworkDownload = value / 1024
				} else {
					c.data.NetworkDownload = value
				}
			}
		}
	}
}

// parseNumericValue extracts numeric value from string with units
func (c *LibreHardwareMonitorClient) parseNumericValue(valueStr string) float64 {
	// Remove common units and parse
	valueStr = strings.TrimSpace(valueStr)
	valueStr = strings.Replace(valueStr, " °C", "", -1)
	valueStr = strings.Replace(valueStr, " %", "", -1)
	valueStr = strings.Replace(valueStr, " MHz", "", -1)
	valueStr = strings.Replace(valueStr, " GB", "", -1)
	valueStr = strings.Replace(valueStr, " MB", "", -1)
	valueStr = strings.Replace(valueStr, " KB/s", "", -1)
	valueStr = strings.Replace(valueStr, " MB/s", "", -1)
	valueStr = strings.Replace(valueStr, " RPM", "", -1)
	valueStr = strings.Replace(valueStr, " V", "", -1)
	valueStr = strings.Replace(valueStr, " W", "", -1)

	if value, err := strconv.ParseFloat(valueStr, 64); err == nil {
		return value
	}
	return 0
}

// GetData returns the current monitoring data
func (c *LibreHardwareMonitorClient) GetData() *LibreHardwareMonitorData {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	// Return a copy to avoid race conditions
	dataCopy := *c.data
	fansCopy := make([]FanInfo, len(c.data.Fans))
	copy(fansCopy, c.data.Fans)
	dataCopy.Fans = fansCopy

	return &dataCopy
}
