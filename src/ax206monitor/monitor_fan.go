package main

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
)

type FanMonitor struct {
	*BaseMonitorItem
	fanIndex int
}

func NewFanMonitor(fanIndex int, fanName string) *FanMonitor {
	name := fmt.Sprintf("fan_%d", fanIndex)
	label := fanName
	if label == "" {
		label = fmt.Sprintf("Fan %d", fanIndex)
	}

	return &FanMonitor{
		BaseMonitorItem: NewBaseMonitorItem(name, label, 0, 0, "RPM", 0),
		fanIndex:        fanIndex,
	}
}

func (f *FanMonitor) Update() error {
	fans := GetAvailableFans()
	if f.fanIndex < len(fans) {
		f.SetValue(fans[f.fanIndex].Speed)
		f.SetAvailable(true)
	} else {
		f.SetAvailable(false)
	}
	return nil
}

func GetAvailableFans() []FanInfo {
	if runtime.GOOS == "windows" {
		return getWindowsFanInfo()
	}
	return getLinuxFanInfo()
}

func getWindowsFanInfo() []FanInfo {
	return []FanInfo{}
}

func getLinuxFanInfo() []FanInfo {
	if runtime.GOOS != "linux" {
		return []FanInfo{}
	}

	fans := []FanInfo{}

	hwmonDirs := []string{"/sys/class/hwmon"}
	for _, hwmonDir := range hwmonDirs {
		if entries, err := os.ReadDir(hwmonDir); err == nil {
			for _, entry := range entries {
				if entry.IsDir() {
					hwmonPath := filepath.Join(hwmonDir, entry.Name())
					fanFiles, _ := filepath.Glob(filepath.Join(hwmonPath, "fan*_input"))

					for _, fanFile := range fanFiles {
						if data, err := os.ReadFile(fanFile); err == nil {
							if speed, err := strconv.Atoi(strings.TrimSpace(string(data))); err == nil && speed > 0 {
								fanName := fmt.Sprintf("Fan %s", strings.TrimSuffix(strings.TrimPrefix(filepath.Base(fanFile), "fan"), "_input"))

								labelFile := strings.Replace(fanFile, "_input", "_label", 1)
								if labelData, err := os.ReadFile(labelFile); err == nil {
									fanName = strings.TrimSpace(string(labelData))
								}

								fans = append(fans, FanInfo{Name: fanName, Speed: speed})
							}
						}
					}
				}
			}
		}
	}

	return fans
}
