package main

import (
	"sync"
	"time"

	"github.com/shirou/gopsutil/v3/mem"
)

type memorySnapshot struct {
	at      time.Time
	virtual *mem.VirtualMemoryStat
	swap    *mem.SwapMemoryStat
}

var (
	memorySnapshotMu    sync.RWMutex
	memorySnapshotCache memorySnapshot
)

func fetchMemorySnapshot(maxAge time.Duration) error {
	now := time.Now()

	memorySnapshotMu.RLock()
	if !memorySnapshotCache.at.IsZero() && now.Sub(memorySnapshotCache.at) <= maxAge {
		memorySnapshotMu.RUnlock()
		return nil
	}
	memorySnapshotMu.RUnlock()

	virtualStat, virtualErr := mem.VirtualMemory()
	swapStat, swapErr := mem.SwapMemory()

	memorySnapshotMu.Lock()
	memorySnapshotCache.at = now
	if virtualErr == nil && virtualStat != nil {
		virtualCopy := *virtualStat
		memorySnapshotCache.virtual = &virtualCopy
	} else {
		memorySnapshotCache.virtual = nil
	}
	if swapErr == nil && swapStat != nil {
		swapCopy := *swapStat
		memorySnapshotCache.swap = &swapCopy
	} else {
		memorySnapshotCache.swap = nil
	}
	memorySnapshotMu.Unlock()

	if virtualErr != nil {
		return virtualErr
	}
	return swapErr
}

func getVirtualMemorySnapshot() (*mem.VirtualMemoryStat, bool) {
	memorySnapshotMu.RLock()
	defer memorySnapshotMu.RUnlock()
	if memorySnapshotCache.virtual == nil {
		return nil, false
	}
	copyVal := *memorySnapshotCache.virtual
	return &copyVal, true
}

func getSwapMemorySnapshot() (*mem.SwapMemoryStat, bool) {
	memorySnapshotMu.RLock()
	defer memorySnapshotMu.RUnlock()
	if memorySnapshotCache.swap == nil {
		return nil, false
	}
	copyVal := *memorySnapshotCache.swap
	return &copyVal, true
}
