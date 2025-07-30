package main

import (
	"fmt"
	"io/ioutil"
	"strconv"
	"strings"
	"time"
)

// Common monitoring utilities to reduce code duplication

// readSysFile reads a system file and returns its content as string
func readSysFile(path string) (string, error) {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(data)), nil
}

// readSysFileInt reads a system file and returns its content as integer
func readSysFileInt(path string) (int, error) {
	content, err := readSysFile(path)
	if err != nil {
		return 0, err
	}
	return strconv.Atoi(content)
}

// readSysFileFloat reads a system file and returns its content as float64
func readSysFileFloat(path string) (float64, error) {
	content, err := readSysFile(path)
	if err != nil {
		return 0, err
	}
	return strconv.ParseFloat(content, 64)
}

// validateTemperature checks if temperature is within reasonable range
func validateTemperature(temp float64, minTemp, maxTemp float64) bool {
	return temp >= minTemp && temp <= maxTemp
}

// validateFrequency checks if frequency is within reasonable range
func validateFrequency(freq float64, minFreq, maxFreq float64) bool {
	return freq >= minFreq && freq <= maxFreq
}

// findHwmonSensor finds hwmon sensor by name patterns
func findHwmonSensor(namePatterns []string, tempFile string) (string, float64, error) {
	hwmonDirs, err := ioutil.ReadDir("/sys/class/hwmon")
	if err != nil {
		return "", 0, err
	}

	for _, hwmon := range hwmonDirs {
		hwmonPath := fmt.Sprintf("/sys/class/hwmon/%s", hwmon.Name())

		nameBytes, err := ioutil.ReadFile(hwmonPath + "/name")
		if err != nil {
			continue
		}
		hwmonName := strings.TrimSpace(string(nameBytes))

		// Check if this hwmon matches any of the patterns
		matched := false
		for _, pattern := range namePatterns {
			if strings.Contains(strings.ToLower(hwmonName), strings.ToLower(pattern)) {
				matched = true
				break
			}
		}

		if matched {
			tempPath := hwmonPath + "/" + tempFile
			if tempInt, err := readSysFileInt(tempPath); err == nil {
				temp := float64(tempInt) / 1000.0
				return tempPath, temp, nil
			}
		}
	}

	return "", 0, fmt.Errorf("sensor not found")
}

// CachedSensorPath represents a cached sensor path with validation
type CachedSensorPath struct {
	path        string
	lastCheck   time.Time
	checkPeriod time.Duration
}

// NewCachedSensorPath creates a new cached sensor path
func NewCachedSensorPath(checkPeriod time.Duration) *CachedSensorPath {
	return &CachedSensorPath{
		checkPeriod: checkPeriod,
	}
}

// GetValue reads value from cached path or rescans if needed
func (c *CachedSensorPath) GetValue(namePatterns []string, tempFile string,
	minVal, maxVal float64) (float64, error) {

	now := time.Now()

	// Try cached path first if it's recent
	if c.path != "" && now.Sub(c.lastCheck) < c.checkPeriod {
		if tempInt, err := readSysFileInt(c.path); err == nil {
			temp := float64(tempInt) / 1000.0
			if temp >= minVal && temp <= maxVal {
				return temp, nil
			}
		}
		// Path is invalid, clear it
		c.path = ""
	}

	// Rescan for sensor
	path, temp, err := findHwmonSensor(namePatterns, tempFile)
	if err != nil {
		return 0, err
	}

	if temp >= minVal && temp <= maxVal {
		c.path = path
		c.lastCheck = now
		return temp, nil
	}

	return 0, fmt.Errorf("temperature out of range: %.1f", temp)
}

// MonitorDataCache provides a simple cache for monitor data
type MonitorDataCache struct {
	data      map[string]interface{}
	timestamp time.Time
	ttl       time.Duration
}

// NewMonitorDataCache creates a new monitor data cache
func NewMonitorDataCache(ttl time.Duration) *MonitorDataCache {
	return &MonitorDataCache{
		data: make(map[string]interface{}),
		ttl:  ttl,
	}
}

// Get retrieves cached data if still valid
func (c *MonitorDataCache) Get(key string) (interface{}, bool) {
	if time.Since(c.timestamp) > c.ttl {
		return nil, false
	}

	value, exists := c.data[key]
	return value, exists
}

// Set stores data in cache
func (c *MonitorDataCache) Set(key string, value interface{}) {
	if time.Since(c.timestamp) > c.ttl {
		// Clear old data
		c.data = make(map[string]interface{})
		c.timestamp = time.Now()
	}
	c.data[key] = value
}

// SetMultiple stores multiple values in cache
func (c *MonitorDataCache) SetMultiple(values map[string]interface{}) {
	if time.Since(c.timestamp) > c.ttl {
		// Clear old data
		c.data = make(map[string]interface{})
		c.timestamp = time.Now()
	}

	for key, value := range values {
		c.data[key] = value
	}
}

// IsValid checks if cache is still valid
func (c *MonitorDataCache) IsValid() bool {
	return time.Since(c.timestamp) <= c.ttl
}

// Clear clears the cache
func (c *MonitorDataCache) Clear() {
	c.data = make(map[string]interface{})
	c.timestamp = time.Time{}
}

// Common sensor name patterns
var (
	CPUSensorPatterns  = []string{"k10temp", "coretemp", "zenpower", "cpu", "package"}
	GPUSensorPatterns  = []string{"nouveau", "amdgpu", "radeon", "i915"}
	DiskSensorPatterns = []string{"nvme", "sata", "ata", "scsi", "drivetemp"}
)

// Common temperature ranges
const (
	CPUTempMin  = 20.0
	CPUTempMax  = 150.0
	GPUTempMin  = 0.0
	GPUTempMax  = 120.0
	DiskTempMin = 0.0
	DiskTempMax = 100.0
)

// Common frequency ranges (MHz)
const (
	CPUFreqMin = 100.0
	CPUFreqMax = 10000.0
	GPUFreqMin = 100.0
	GPUFreqMax = 5000.0
)
