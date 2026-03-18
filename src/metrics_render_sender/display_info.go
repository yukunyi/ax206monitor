package main

import (
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

type displayInfoCacheState struct {
	mu      sync.RWMutex
	at      time.Time
	ok      bool
	width   int
	height  int
	refresh float64
}

var (
	displayInfoCache displayInfoCacheState
	displayUpdating  int32
)

func getDisplayInfoSnapshot(maxAge time.Duration) (string, string, bool) {
	now := time.Now()
	displayInfoCache.mu.RLock()
	cachedAt := displayInfoCache.at
	cachedOK := displayInfoCache.ok
	width := displayInfoCache.width
	height := displayInfoCache.height
	refresh := displayInfoCache.refresh
	displayInfoCache.mu.RUnlock()

	if maxAge <= 0 {
		maxAge = 30 * time.Second
	}
	if cachedOK && !cachedAt.IsZero() && now.Sub(cachedAt) <= maxAge {
		return formatDisplayResolution(width, height), formatDisplayRefresh(refresh), true
	}
	triggerDisplayInfoRefresh()
	if cachedOK {
		return formatDisplayResolution(width, height), formatDisplayRefresh(refresh), true
	}
	return "", "", false
}

func triggerDisplayInfoRefresh() {
	if !atomic.CompareAndSwapInt32(&displayUpdating, 0, 1) {
		return
	}
	go func() {
		defer atomic.StoreInt32(&displayUpdating, 0)
		width, height, refresh, ok := detectPrimaryDisplayInfo()
		displayInfoCache.mu.Lock()
		displayInfoCache.at = time.Now()
		displayInfoCache.ok = ok
		if ok {
			displayInfoCache.width = width
			displayInfoCache.height = height
			displayInfoCache.refresh = refresh
		}
		displayInfoCache.mu.Unlock()
	}()
}

func detectPrimaryDisplayInfo() (int, int, float64, bool) {
	switch runtime.GOOS {
	case "windows":
		return detectPrimaryDisplayInfoWindows()
	case "linux":
		return detectPrimaryDisplayInfoLinux()
	default:
		return 0, 0, 0, false
	}
}

func detectPrimaryDisplayInfoWindows() (int, int, float64, bool) {
	return detectPrimaryDisplayInfoWindowsImpl()
}

func detectPrimaryDisplayInfoLinux() (int, int, float64, bool) {
	if width, height, ok := detectDisplayByDRMMode(); ok {
		return width, height, 0, true
	}
	return 0, 0, 0, false
}

func detectDisplayByDRMMode() (int, int, bool) {
	matches, err := filepath.Glob("/sys/class/drm/*/modes")
	if err != nil || len(matches) == 0 {
		return 0, 0, false
	}
	for _, path := range matches {
		data, readErr := os.ReadFile(path)
		if readErr != nil {
			continue
		}
		lines := strings.Split(strings.TrimSpace(string(data)), "\n")
		for _, line := range lines {
			text := strings.TrimSpace(line)
			if text == "" {
				continue
			}
			dims := strings.SplitN(text, "x", 2)
			if len(dims) != 2 {
				continue
			}
			width, errW := strconv.Atoi(dims[0])
			height, errH := strconv.Atoi(dims[1])
			if errW != nil || errH != nil || width <= 0 || height <= 0 {
				continue
			}
			return width, height, true
		}
	}
	return 0, 0, false
}

func formatDisplayResolution(width, height int) string {
	if width <= 0 || height <= 0 {
		return "-"
	}
	return itoa(width) + "x" + itoa(height)
}

func formatDisplayRefresh(refresh float64) string {
	if refresh <= 0 {
		return "-"
	}
	if refresh >= 100 || refresh == float64(int64(refresh)) {
		return itoa(int(refresh+0.5)) + "Hz"
	}
	return strings.TrimRight(strings.TrimRight(strconv.FormatFloat(refresh, 'f', 2, 64), "0"), ".") + "Hz"
}
