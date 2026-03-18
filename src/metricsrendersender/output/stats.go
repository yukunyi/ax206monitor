package output

import (
	"sort"
	"sync"
	"time"
)

type OutputHandlerRuntimeStats struct {
	Type   string `json:"type"`
	Calls  int64  `json:"calls"`
	Errors int64  `json:"errors"`
	LastMS int64  `json:"last_ms"`
	MaxMS  int64  `json:"max_ms"`
	AvgMS  int64  `json:"avg_ms"`
}

type OutputRuntimeStats struct {
	Calls    int64                                `json:"calls"`
	Errors   int64                                `json:"errors"`
	LastMS   int64                                `json:"last_ms"`
	MaxMS    int64                                `json:"max_ms"`
	AvgMS    int64                                `json:"avg_ms"`
	Handlers map[string]OutputHandlerRuntimeStats `json:"handlers"`
}

type AX206DeviceFrameRuntimeStats struct {
	Calls  int64 `json:"calls"`
	Errors int64 `json:"errors"`
	LastMS int64 `json:"last_ms"`
	MaxMS  int64 `json:"max_ms"`
	AvgMS  int64 `json:"avg_ms"`
}

type TCPPushAvailabilityStats struct {
	Type              string `json:"type"`
	Connected         bool   `json:"connected"`
	ProbeMode         bool   `json:"probe_mode"`
	Available         bool   `json:"available"`
	ShouldSendFrame   bool   `json:"should_send_frame"`
	CanSend           bool   `json:"can_send"`
	UserPriority      int    `json:"user_priority,omitempty"`
	HighestPriority   int    `json:"highest_priority,omitempty"`
	ActivePriority    int    `json:"active_priority,omitempty"`
	ActiveSessionID   string `json:"active_session_id,omitempty"`
	ActiveUser        string `json:"active_user,omitempty"`
	LowerPriorityMode string `json:"lower_priority_mode,omitempty"`
	Reason            string `json:"reason,omitempty"`
	LastStatusCode    int    `json:"last_status_code,omitempty"`
	LastStage         string `json:"last_stage,omitempty"`
	UpdatedAt         string `json:"updated_at,omitempty"`
}

type outputRuntimeAccumulator struct {
	calls   int64
	errors  int64
	lastNS  int64
	maxNS   int64
	totalNS int64
}

var (
	outputRuntimeMu     sync.RWMutex
	outputRuntimeTotal  outputRuntimeAccumulator
	outputRuntimeByType = make(map[string]*outputRuntimeAccumulator)
	ax206DeviceRuntime  outputRuntimeAccumulator
	httpPushByType      = make(map[string]*outputRuntimeAccumulator)
	tcpPushByType       = make(map[string]*outputRuntimeAccumulator)
	tcpPushAvailability = make(map[string]TCPPushAvailabilityStats)
)

func recordOutputRuntime(typeName string, duration time.Duration, err error) {
	if duration < 0 {
		duration = 0
	}
	typeName = normalizeTypeName(typeName)
	durationNS := duration.Nanoseconds()

	outputRuntimeMu.Lock()
	defer outputRuntimeMu.Unlock()

	outputRuntimeTotal.calls++
	outputRuntimeTotal.lastNS = durationNS
	outputRuntimeTotal.totalNS += durationNS
	if durationNS > outputRuntimeTotal.maxNS {
		outputRuntimeTotal.maxNS = durationNS
	}
	if err != nil {
		outputRuntimeTotal.errors++
	}

	entry := outputRuntimeByType[typeName]
	if entry == nil {
		entry = &outputRuntimeAccumulator{}
		outputRuntimeByType[typeName] = entry
	}
	entry.calls++
	entry.lastNS = durationNS
	entry.totalNS += durationNS
	if durationNS > entry.maxNS {
		entry.maxNS = durationNS
	}
	if err != nil {
		entry.errors++
	}
}

func normalizeTypeName(typeName string) string {
	switch typeName {
	case "":
		return "unknown"
	default:
		return typeName
	}
}

func toMillis(ns int64) int64 {
	if ns <= 0 {
		return 0
	}
	return ns / int64(time.Millisecond)
}

func avgMillis(totalNS, calls int64) int64 {
	if calls <= 0 || totalNS <= 0 {
		return 0
	}
	return toMillis(totalNS / calls)
}

func GetRuntimeStats() OutputRuntimeStats {
	outputRuntimeMu.RLock()
	defer outputRuntimeMu.RUnlock()

	handlers := make(map[string]OutputHandlerRuntimeStats, len(outputRuntimeByType))
	keys := make([]string, 0, len(outputRuntimeByType))
	for key := range outputRuntimeByType {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		entry := outputRuntimeByType[key]
		if entry == nil {
			continue
		}
		handlers[key] = OutputHandlerRuntimeStats{
			Type:   key,
			Calls:  entry.calls,
			Errors: entry.errors,
			LastMS: toMillis(entry.lastNS),
			MaxMS:  toMillis(entry.maxNS),
			AvgMS:  avgMillis(entry.totalNS, entry.calls),
		}
	}

	return OutputRuntimeStats{
		Calls:    outputRuntimeTotal.calls,
		Errors:   outputRuntimeTotal.errors,
		LastMS:   toMillis(outputRuntimeTotal.lastNS),
		MaxMS:    toMillis(outputRuntimeTotal.maxNS),
		AvgMS:    avgMillis(outputRuntimeTotal.totalNS, outputRuntimeTotal.calls),
		Handlers: handlers,
	}
}

func recordAX206DeviceFrameRuntime(duration time.Duration, err error) {
	if duration < 0 {
		duration = 0
	}
	durationNS := duration.Nanoseconds()

	outputRuntimeMu.Lock()
	defer outputRuntimeMu.Unlock()

	ax206DeviceRuntime.calls++
	ax206DeviceRuntime.lastNS = durationNS
	ax206DeviceRuntime.totalNS += durationNS
	if durationNS > ax206DeviceRuntime.maxNS {
		ax206DeviceRuntime.maxNS = durationNS
	}
	if err != nil {
		ax206DeviceRuntime.errors++
	}
}

func GetAX206DeviceFrameRuntimeStats() AX206DeviceFrameRuntimeStats {
	outputRuntimeMu.RLock()
	defer outputRuntimeMu.RUnlock()
	return AX206DeviceFrameRuntimeStats{
		Calls:  ax206DeviceRuntime.calls,
		Errors: ax206DeviceRuntime.errors,
		LastMS: toMillis(ax206DeviceRuntime.lastNS),
		MaxMS:  toMillis(ax206DeviceRuntime.maxNS),
		AvgMS:  avgMillis(ax206DeviceRuntime.totalNS, ax206DeviceRuntime.calls),
	}
}

func recordHTTPPushRuntime(typeName string, duration time.Duration, err error) {
	if duration < 0 {
		duration = 0
	}
	typeName = normalizeTypeName(typeName)
	durationNS := duration.Nanoseconds()

	outputRuntimeMu.Lock()
	defer outputRuntimeMu.Unlock()

	entry := httpPushByType[typeName]
	if entry == nil {
		entry = &outputRuntimeAccumulator{}
		httpPushByType[typeName] = entry
	}
	entry.calls++
	entry.lastNS = durationNS
	entry.totalNS += durationNS
	if durationNS > entry.maxNS {
		entry.maxNS = durationNS
	}
	if err != nil {
		entry.errors++
	}
}

func GetHTTPPushRuntimeStats() map[string]OutputHandlerRuntimeStats {
	outputRuntimeMu.RLock()
	defer outputRuntimeMu.RUnlock()

	result := make(map[string]OutputHandlerRuntimeStats, len(httpPushByType))
	keys := make([]string, 0, len(httpPushByType))
	for key := range httpPushByType {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		entry := httpPushByType[key]
		if entry == nil {
			continue
		}
		result[key] = OutputHandlerRuntimeStats{
			Type:   key,
			Calls:  entry.calls,
			Errors: entry.errors,
			LastMS: toMillis(entry.lastNS),
			MaxMS:  toMillis(entry.maxNS),
			AvgMS:  avgMillis(entry.totalNS, entry.calls),
		}
	}
	return result
}

func recordTCPPushRuntime(typeName string, duration time.Duration, err error) {
	if duration < 0 {
		duration = 0
	}
	typeName = normalizeTypeName(typeName)
	durationNS := duration.Nanoseconds()

	outputRuntimeMu.Lock()
	defer outputRuntimeMu.Unlock()

	entry := tcpPushByType[typeName]
	if entry == nil {
		entry = &outputRuntimeAccumulator{}
		tcpPushByType[typeName] = entry
	}
	entry.calls++
	entry.lastNS = durationNS
	entry.totalNS += durationNS
	if durationNS > entry.maxNS {
		entry.maxNS = durationNS
	}
	if err != nil {
		entry.errors++
	}
}

func GetTCPPushRuntimeStats() map[string]OutputHandlerRuntimeStats {
	outputRuntimeMu.RLock()
	defer outputRuntimeMu.RUnlock()

	result := make(map[string]OutputHandlerRuntimeStats, len(tcpPushByType))
	keys := make([]string, 0, len(tcpPushByType))
	for key := range tcpPushByType {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		entry := tcpPushByType[key]
		if entry == nil {
			continue
		}
		result[key] = OutputHandlerRuntimeStats{
			Type:   key,
			Calls:  entry.calls,
			Errors: entry.errors,
			LastMS: toMillis(entry.lastNS),
			MaxMS:  toMillis(entry.maxNS),
			AvgMS:  avgMillis(entry.totalNS, entry.calls),
		}
	}
	return result
}

func RecordTCPPushAvailabilityStats(stats TCPPushAvailabilityStats) {
	typeName := normalizeTypeName(stats.Type)
	stats.Type = typeName
	stats.UpdatedAt = time.Now().Format(time.RFC3339Nano)

	outputRuntimeMu.Lock()
	defer outputRuntimeMu.Unlock()
	tcpPushAvailability[typeName] = stats
}

func GetTCPPushAvailabilityStats() map[string]TCPPushAvailabilityStats {
	outputRuntimeMu.RLock()
	defer outputRuntimeMu.RUnlock()

	result := make(map[string]TCPPushAvailabilityStats, len(tcpPushAvailability))
	keys := make([]string, 0, len(tcpPushAvailability))
	for key := range tcpPushAvailability {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		result[key] = tcpPushAvailability[key]
	}
	return result
}
