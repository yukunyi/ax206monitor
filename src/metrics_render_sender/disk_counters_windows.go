//go:build windows

package main

import (
	"fmt"
	"sync"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"
)

const (
	ioctlDiskPerformance           = 0x70020
	windowsDiskHandleRefreshPeriod = int64(time.Minute)
)

type windowsDiskPerformance struct {
	BytesRead           int64
	BytesWritten        int64
	ReadTime            int64
	WriteTime           int64
	IdleTime            int64
	ReadCount           uint32
	WriteCount          uint32
	QueueDepth          uint32
	SplitCount          uint32
	QueryTime           int64
	StorageDeviceNumber uint32
	StorageManagerName  [8]uint16
	alignmentPadding    uint32
}

type windowsDiskHandleState struct {
	mu            sync.Mutex
	handles       map[string]windows.Handle
	lastRefreshNS int64
}

var windowsDiskHandles = windowsDiskHandleState{
	handles: make(map[string]windows.Handle),
}

func readPlatformDiskCounters() (map[string]diskCounterSample, error) {
	windowsDiskHandles.mu.Lock()
	defer windowsDiskHandles.mu.Unlock()

	nowNS := unixNowNS()
	if nowNS-windowsDiskHandles.lastRefreshNS >= windowsDiskHandleRefreshPeriod || len(windowsDiskHandles.handles) == 0 {
		refreshWindowsDiskHandles()
		windowsDiskHandles.lastRefreshNS = nowNS
	}

	result := make(map[string]diskCounterSample, len(windowsDiskHandles.handles))
	for name, handle := range windowsDiskHandles.handles {
		sample, err := readWindowsDiskPerformance(name, handle)
		if err != nil {
			_ = windows.CloseHandle(handle)
			delete(windowsDiskHandles.handles, name)
			reopened, reopenErr := openWindowsDiskHandle(name)
			if reopenErr != nil {
				continue
			}
			windowsDiskHandles.handles[name] = reopened
			sample, err = readWindowsDiskPerformance(name, reopened)
			if err != nil {
				_ = windows.CloseHandle(reopened)
				delete(windowsDiskHandles.handles, name)
				continue
			}
			result[name] = sample
			continue
		}
		result[name] = sample
	}
	return result, nil
}

func refreshWindowsDiskHandles() {
	live := enumerateWindowsFixedDrives()
	for name, handle := range windowsDiskHandles.handles {
		if _, ok := live[name]; ok {
			continue
		}
		_ = windows.CloseHandle(handle)
		delete(windowsDiskHandles.handles, name)
	}
	for name := range live {
		if _, ok := windowsDiskHandles.handles[name]; ok {
			continue
		}
		handle, err := openWindowsDiskHandle(name)
		if err != nil {
			continue
		}
		windowsDiskHandles.handles[name] = handle
	}
}

func enumerateWindowsFixedDrives() map[string]struct{} {
	result := make(map[string]struct{})
	buf := make([]uint16, 254)
	n, err := windows.GetLogicalDriveStrings(uint32(len(buf)), &buf[0])
	if err != nil || n == 0 {
		return result
	}
	for _, value := range buf[:n] {
		if value < 'A' || value > 'Z' {
			continue
		}
		path := string(rune(value)) + ":"
		typePath, err := windows.UTF16PtrFromString(path)
		if err != nil {
			continue
		}
		if windows.GetDriveType(typePath) != windows.DRIVE_FIXED {
			continue
		}
		result[path] = struct{}{}
	}
	return result
}

func openWindowsDiskHandle(name string) (windows.Handle, error) {
	devicePath, err := windows.UTF16PtrFromString(fmt.Sprintf(`\\.\%s`, name))
	if err != nil {
		return 0, err
	}
	return windows.CreateFile(
		devicePath,
		0,
		windows.FILE_SHARE_READ|windows.FILE_SHARE_WRITE,
		nil,
		windows.OPEN_EXISTING,
		0,
		0,
	)
}

func readWindowsDiskPerformance(name string, handle windows.Handle) (diskCounterSample, error) {
	var perf windowsDiskPerformance
	var returned uint32
	err := windows.DeviceIoControl(
		handle,
		ioctlDiskPerformance,
		nil,
		0,
		(*byte)(unsafe.Pointer(&perf)),
		uint32(unsafe.Sizeof(perf)),
		&returned,
		nil,
	)
	if err != nil {
		return diskCounterSample{}, err
	}

	busyRaw := perf.QueryTime - perf.IdleTime
	if busyRaw < 0 {
		busyRaw = perf.ReadTime + perf.WriteTime
	}
	return diskCounterSample{
		Name:        name,
		ReadBytes:   uint64(perf.BytesRead),
		WriteBytes:  uint64(perf.BytesWritten),
		ReadCount:   uint64(perf.ReadCount),
		WriteCount:  uint64(perf.WriteCount),
		ReadTimeMS:  windowsDiskTimeToMS(perf.ReadTime),
		WriteTimeMS: windowsDiskTimeToMS(perf.WriteTime),
		BusyTimeMS:  windowsDiskTimeToMS(busyRaw),
		QueueDepth:  float64(perf.QueueDepth),
	}, nil
}

func windowsDiskTimeToMS(value int64) float64 {
	if value <= 0 {
		return 0
	}
	return float64(value) / 10000.0
}

func unixNowNS() int64 {
	return time.Now().UnixNano()
}
