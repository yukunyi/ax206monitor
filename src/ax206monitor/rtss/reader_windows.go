//go:build windows

package rtss

import (
	"math"
	"unsafe"

	"golang.org/x/sys/windows"
)

const (
	sharedMemoryName = "RTSSSharedMemoryV2"
	signatureA       = 0x52545353
	signatureB       = 0x53535452
	fileMapRead      = 0x0004
	versionMin       = 0x00020000
	versionFPSRaw    = 0x00020005
	fpsStaleMS       = uint32(2000)
)

var (
	modKernel32                  = windows.NewLazySystemDLL("kernel32.dll")
	modUser32                    = windows.NewLazySystemDLL("user32.dll")
	procOpenFileMappingW         = modKernel32.NewProc("OpenFileMappingW")
	procMapViewOfFile            = modKernel32.NewProc("MapViewOfFile")
	procUnmapViewOfFile          = modKernel32.NewProc("UnmapViewOfFile")
	procCloseHandle              = modKernel32.NewProc("CloseHandle")
	procGetTickCount             = modKernel32.NewProc("GetTickCount")
	procGetForegroundWindow      = modUser32.NewProc("GetForegroundWindow")
	procGetWindowThreadProcessID = modUser32.NewProc("GetWindowThreadProcessId")
)

type sharedMemoryHeader struct {
	Signature    uint32
	Version      uint32
	AppEntrySize uint32
	AppArrOffset uint32
	AppArrSize   uint32
	OSDEntrySize uint32
	OSDArrOffset uint32
	OSDArrSize   uint32
	OSDFrame     uint32
	Busy         uint32
}

type appEntry struct {
	Name                      [260]byte
	ProcessID                 uint32
	Frames                    uint32
	Time0                     uint32
	Time1                     uint32
	Flags                     uint32
	StatFlags                 uint32
	StatFrameTimeBufFramerate uint32
}

func ReadMetrics() (Metrics, bool) {
	name, err := windows.UTF16PtrFromString(sharedMemoryName)
	if err != nil {
		return Metrics{}, false
	}
	handle, _, _ := procOpenFileMappingW.Call(uintptr(fileMapRead), 0, uintptr(unsafe.Pointer(name)))
	if handle == 0 {
		return Metrics{}, false
	}
	defer procCloseHandle.Call(handle)

	view, _, _ := procMapViewOfFile.Call(handle, uintptr(fileMapRead), 0, 0, 0)
	if view == 0 {
		return Metrics{}, false
	}
	defer procUnmapViewOfFile.Call(view)

	header := (*sharedMemoryHeader)(unsafe.Pointer(view))
	if header == nil {
		return Metrics{}, false
	}
	if header.Signature != signatureA && header.Signature != signatureB {
		return Metrics{}, false
	}
	minAppEntrySize := uint32(unsafe.Sizeof(appEntry{}))
	if header.Version < versionMin || header.AppEntrySize < minAppEntrySize || header.AppArrSize == 0 || header.AppArrOffset == 0 {
		return Metrics{}, false
	}
	if header.AppArrSize > 4096 {
		return Metrics{}, false
	}

	foregroundPID := getForegroundPID()
	nowTick := getTickCount()
	metrics := Metrics{ForegroundPID: foregroundPID}
	appBase := uintptr(view) + uintptr(header.AppArrOffset)
	entrySize := uintptr(header.AppEntrySize)

	for i := uint32(0); i < header.AppArrSize; i++ {
		entry := appBase + uintptr(i)*entrySize
		current := (*appEntry)(unsafe.Pointer(entry))
		if current == nil || current.ProcessID == 0 {
			continue
		}
		fps, available := readEntryFPS(current, header.Version, nowTick)
		if !available || fps <= 0 {
			continue
		}
		metrics.ActiveApps++
		if metrics.MaxFPS < fps {
			metrics.MaxFPS = fps
		}
		if foregroundPID != 0 && current.ProcessID == foregroundPID {
			metrics.ForegroundFPS = fps
		}
	}

	if metrics.ForegroundFPS <= 0 {
		metrics.ForegroundFPS = metrics.MaxFPS
	}
	return metrics, true
}

func readEntryFPS(entry *appEntry, version uint32, nowTick uint32) (float64, bool) {
	if entry == nil {
		return 0, false
	}
	if tickDeltaMS(nowTick, entry.Time1) > fpsStaleMS {
		return 0, false
	}

	// RTSS v2.5+ provides direct framerate in 1/10 FPS precision.
	if version >= versionFPSRaw && entry.StatFrameTimeBufFramerate > 0 {
		fps := sanitizeFPS(entry.StatFrameTimeBufFramerate)
		return fps, fps > 0
	}

	// Fallback for older RTSS versions: frames / elapsed_ms * 1000.
	delta := int64(entry.Time1) - int64(entry.Time0)
	if delta <= 0 || entry.Frames == 0 {
		return 0, false
	}
	fps := 1000.0 * float64(entry.Frames) / float64(delta)
	fps = sanitizeFPSFloat(fps)
	if math.IsNaN(fps) || math.IsInf(fps, 0) || fps <= 0 {
		return 0, false
	}
	return fps, true
}

func getTickCount() uint32 {
	value, _, _ := procGetTickCount.Call()
	return uint32(value)
}

func tickDeltaMS(now uint32, past uint32) uint32 {
	return now - past
}

func getForegroundPID() uint32 {
	hwnd, _, _ := procGetForegroundWindow.Call()
	if hwnd == 0 {
		return 0
	}
	var pid uint32
	procGetWindowThreadProcessID.Call(hwnd, uintptr(unsafe.Pointer(&pid)))
	return pid
}
