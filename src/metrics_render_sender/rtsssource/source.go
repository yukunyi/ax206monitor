package rtsssource

import (
	"strings"
	"sync"
	"time"

	"metrics_render_sender/rtss"
)

type RTSSMonitorOption struct {
	Name  string `json:"name"`
	Label string `json:"label"`
	Unit  string `json:"unit,omitempty"`
}

type rtssMonitorEntry struct {
	RTSSMonitorOption
	read func(metrics rtss.Metrics) (float64, bool)
}

var rtssMonitorEntries = []rtssMonitorEntry{
	{
		RTSSMonitorOption: RTSSMonitorOption{Name: "rtss_connected", Label: "[RTSS] Connected", Unit: ""},
		read: func(metrics rtss.Metrics) (float64, bool) {
			if metrics.Connected {
				return 1, true
			}
			return 0, true
		},
	},
	{
		RTSSMonitorOption: RTSSMonitorOption{Name: "rtss_fps", Label: "[RTSS] FPS", Unit: "FPS"},
		read: func(metrics rtss.Metrics) (float64, bool) {
			return metrics.ForegroundFPS, true
		},
	},
	{
		RTSSMonitorOption: RTSSMonitorOption{Name: "rtss_frametime_ms", Label: "[RTSS] Frame Time", Unit: "ms"},
		read: func(metrics rtss.Metrics) (float64, bool) {
			return metrics.ForegroundFrameTimeMS(), true
		},
	},
	{
		RTSSMonitorOption: RTSSMonitorOption{Name: "rtss_fps_avg", Label: "[RTSS] FPS Avg", Unit: "FPS"},
		read: func(metrics rtss.Metrics) (float64, bool) {
			return metrics.ForegroundFPSAvg, true
		},
	},
	{
		RTSSMonitorOption: RTSSMonitorOption{Name: "rtss_fps_1p_low", Label: "[RTSS] FPS 1% Low", Unit: "FPS"},
		read: func(metrics rtss.Metrics) (float64, bool) {
			return metrics.ForegroundFPS1PLow, true
		},
	},
	{
		RTSSMonitorOption: RTSSMonitorOption{Name: "rtss_fps_01p_low", Label: "[RTSS] FPS 0.1% Low", Unit: "FPS"},
		read: func(metrics rtss.Metrics) (float64, bool) {
			return metrics.ForegroundFPS01PLow, true
		},
	},
	{
		RTSSMonitorOption: RTSSMonitorOption{Name: "rtss_frametime_min_ms", Label: "[RTSS] Frame Time Min", Unit: "ms"},
		read: func(metrics rtss.Metrics) (float64, bool) {
			return metrics.ForegroundFTMinMS, true
		},
	},
	{
		RTSSMonitorOption: RTSSMonitorOption{Name: "rtss_frametime_avg_ms", Label: "[RTSS] Frame Time Avg", Unit: "ms"},
		read: func(metrics rtss.Metrics) (float64, bool) {
			return metrics.ForegroundFTAvgMS, true
		},
	},
	{
		RTSSMonitorOption: RTSSMonitorOption{Name: "rtss_frametime_max_ms", Label: "[RTSS] Frame Time Max", Unit: "ms"},
		read: func(metrics rtss.Metrics) (float64, bool) {
			return metrics.ForegroundFTMaxMS, true
		},
	},
	{
		RTSSMonitorOption: RTSSMonitorOption{Name: "rtss_frametime_p99_ms", Label: "[RTSS] Frame Time p99", Unit: "ms"},
		read: func(metrics rtss.Metrics) (float64, bool) {
			return metrics.ForegroundFTP99MS, true
		},
	},
	{
		RTSSMonitorOption: RTSSMonitorOption{Name: "rtss_frametime_p999_ms", Label: "[RTSS] Frame Time p99.9", Unit: "ms"},
		read: func(metrics rtss.Metrics) (float64, bool) {
			return metrics.ForegroundFTP999MS, true
		},
	},
	{
		RTSSMonitorOption: RTSSMonitorOption{Name: "rtss_max_fps", Label: "[RTSS] Max FPS", Unit: "FPS"},
		read: func(metrics rtss.Metrics) (float64, bool) {
			return metrics.MaxFPS, true
		},
	},
	{
		RTSSMonitorOption: RTSSMonitorOption{Name: "rtss_active_apps", Label: "[RTSS] Active Apps", Unit: ""},
		read: func(metrics rtss.Metrics) (float64, bool) {
			return float64(metrics.ActiveApps), true
		},
	},
	{
		RTSSMonitorOption: RTSSMonitorOption{Name: "rtss_foreground_pid", Label: "[RTSS] Foreground PID", Unit: ""},
		read: func(metrics rtss.Metrics) (float64, bool) {
			return float64(metrics.ForegroundPID), true
		},
	},
}

var rtssAliasNames = map[string]string{
	"gpu_fps": "rtss_fps",
}

type RTSSClient struct {
	mu        sync.RWMutex
	metrics   rtss.Metrics
	metricsAt time.Time
	connected bool
}

var (
	rtssClientOnce sync.Once
	rtssClientInst *RTSSClient
)

func GetRTSSClient() *RTSSClient {
	rtssClientOnce.Do(func() {
		rtssClientInst = &RTSSClient{}
	})
	return rtssClientInst
}

func (c *RTSSClient) ListMonitorOptions() []RTSSMonitorOption {
	items := make([]RTSSMonitorOption, 0, len(rtssMonitorEntries))
	for _, entry := range rtssMonitorEntries {
		items = append(items, entry.RTSSMonitorOption)
	}
	return items
}

func (c *RTSSClient) GetMonitorValueByName(name string) (float64, string, bool, error) {
	normalized := normalizeRTSSMonitorName(name)
	entry, ok := findRTSSEntry(normalized)
	if !ok {
		return 0, "", false, nil
	}

	metrics, connected := c.getFreshMetrics(250 * time.Millisecond)
	if !connected {
		// Keep monitor values stable even when RTSS is unavailable.
		return 0, entry.Unit, true, nil
	}
	value, available := entry.read(metrics)
	return value, entry.Unit, available, nil
}

func (c *RTSSClient) RefreshMetrics(maxAge time.Duration) bool {
	_, connected := c.getFreshMetrics(maxAge)
	return connected
}

func (c *RTSSClient) GetMonitorValueByNameCached(name string) (float64, string, bool, error) {
	normalized := normalizeRTSSMonitorName(name)
	entry, ok := findRTSSEntry(normalized)
	if !ok {
		return 0, "", false, nil
	}
	c.mu.RLock()
	metrics := c.metrics
	connected := c.connected
	c.mu.RUnlock()
	if !connected {
		// Keep monitor values stable even when RTSS is unavailable.
		return 0, entry.Unit, true, nil
	}
	value, available := entry.read(metrics)
	return value, entry.Unit, available, nil
}

func (c *RTSSClient) getFreshMetrics(maxAge time.Duration) (rtss.Metrics, bool) {
	now := time.Now()
	c.mu.RLock()
	if c.connected && !c.metricsAt.IsZero() && now.Sub(c.metricsAt) <= maxAge {
		metrics := c.metrics
		c.mu.RUnlock()
		return metrics, true
	}
	cached := c.metrics
	cachedAt := c.metricsAt
	hadCache := c.connected
	c.mu.RUnlock()

	metrics, connected := rtss.ReadMetrics()

	c.mu.Lock()
	defer c.mu.Unlock()
	if connected {
		c.metrics = metrics
		c.metricsAt = now
		c.connected = true
		return metrics, true
	}

	if hadCache && !cachedAt.IsZero() && now.Sub(cachedAt) <= 2*time.Second {
		return cached, true
	}
	c.connected = false
	return rtss.Metrics{}, false
}

func findRTSSEntry(name string) (rtssMonitorEntry, bool) {
	for _, entry := range rtssMonitorEntries {
		if entry.Name == name {
			return entry, true
		}
	}
	return rtssMonitorEntry{}, false
}

func normalizeRTSSMonitorName(name string) string {
	trimmed := strings.TrimSpace(name)
	if alias, ok := rtssAliasNames[trimmed]; ok {
		return alias
	}
	return trimmed
}
