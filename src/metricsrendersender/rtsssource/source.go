package rtsssource

import (
	"strings"
	"sync"
	"time"

	"metricsrendersender/rtss"
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
		RTSSMonitorOption: RTSSMonitorOption{Name: "rtss_fps", Label: "[RTSS] FPS", Unit: "FPS"},
		read: func(metrics rtss.Metrics) (float64, bool) {
			if metrics.ForegroundFPS <= 0 {
				return 0, false
			}
			return metrics.ForegroundFPS, true
		},
	},
	{
		RTSSMonitorOption: RTSSMonitorOption{Name: "rtss_frametime_ms", Label: "[RTSS] Frame Time", Unit: "ms"},
		read: func(metrics rtss.Metrics) (float64, bool) {
			value := metrics.ForegroundFrameTimeMS()
			if value <= 0 {
				return 0, false
			}
			return value, true
		},
	},
	{
		RTSSMonitorOption: RTSSMonitorOption{Name: "rtss_max_fps", Label: "[RTSS] Max FPS", Unit: "FPS"},
		read: func(metrics rtss.Metrics) (float64, bool) {
			if metrics.MaxFPS <= 0 {
				return 0, false
			}
			return metrics.MaxFPS, true
		},
	},
	{
		RTSSMonitorOption: RTSSMonitorOption{Name: "rtss_active_apps", Label: "[RTSS] Active Apps", Unit: ""},
		read: func(metrics rtss.Metrics) (float64, bool) {
			if metrics.ActiveApps < 0 {
				return 0, false
			}
			return float64(metrics.ActiveApps), true
		},
	},
	{
		RTSSMonitorOption: RTSSMonitorOption{Name: "rtss_foreground_pid", Label: "[RTSS] Foreground PID", Unit: ""},
		read: func(metrics rtss.Metrics) (float64, bool) {
			if metrics.ForegroundPID == 0 {
				return 0, false
			}
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
		return 0, entry.Unit, false, nil
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
		return 0, entry.Unit, false, nil
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
