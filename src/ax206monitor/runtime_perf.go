package main

import (
	"sync/atomic"
	"time"
)

type renderRuntimeStats struct {
	Calls  int64
	LastMS int64
	MaxMS  int64
	AvgMS  int64
}

var (
	renderRuntimeCalls   int64
	renderRuntimeLastNS  int64
	renderRuntimeMaxNS   int64
	renderRuntimeTotalNS int64
)

func recordRenderDuration(duration time.Duration) {
	if duration < 0 {
		duration = 0
	}
	durationNS := duration.Nanoseconds()
	atomic.AddInt64(&renderRuntimeCalls, 1)
	atomic.StoreInt64(&renderRuntimeLastNS, durationNS)
	atomic.AddInt64(&renderRuntimeTotalNS, durationNS)
	for {
		current := atomic.LoadInt64(&renderRuntimeMaxNS)
		if durationNS <= current {
			break
		}
		if atomic.CompareAndSwapInt64(&renderRuntimeMaxNS, current, durationNS) {
			break
		}
	}
}

func renderRuntimeSnapshot() renderRuntimeStats {
	calls := atomic.LoadInt64(&renderRuntimeCalls)
	lastNS := atomic.LoadInt64(&renderRuntimeLastNS)
	maxNS := atomic.LoadInt64(&renderRuntimeMaxNS)
	totalNS := atomic.LoadInt64(&renderRuntimeTotalNS)
	avgNS := int64(0)
	if calls > 0 && totalNS > 0 {
		avgNS = totalNS / calls
	}
	return renderRuntimeStats{
		Calls:  calls,
		LastMS: lastNS / int64(time.Millisecond),
		MaxMS:  maxNS / int64(time.Millisecond),
		AvgMS:  avgNS / int64(time.Millisecond),
	}
}
