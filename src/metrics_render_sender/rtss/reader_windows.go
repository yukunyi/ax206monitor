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
}

type appEntry struct {
	ProcessID            uint32
	Name                 [260]byte
	Flags                uint32
	Time0                uint32
	Time1                uint32
	Frames               uint32
	FrameTime            uint32
	StatFlags            uint32
	StatTime0            uint32
	StatTime1            uint32
	StatFrames           uint32
	StatCount            uint32
	StatFramerateMin     uint32
	StatFramerateAvg     uint32
	StatFramerateMax     uint32
	OSDX                 uint32
	OSDY                 uint32
	OSDPixel             uint32
	OSDColor             uint32
	OSDFrame             uint32
	ScreenCaptureFlags   uint32
	ScreenCapturePath    [260]byte
	OSDBgndColor         uint32
	VideoCaptureFlags    uint32
	VideoCapturePath     [260]byte
	VideoFramerate       uint32
	VideoFramesize       uint32
	VideoFormat          uint32
	VideoQuality         uint32
	VideoCaptureThreads  uint32
	ScreenCaptureQuality uint32
	ScreenCaptureThreads uint32
	AudioCaptureFlags    uint32
	VideoCaptureFlagsEx  uint32
	AudioCaptureFlags2   uint32
	StatFrameTimeMin     uint32
	StatFrameTimeAvg     uint32
	StatFrameTimeMax     uint32
	StatFrameTimeCount   uint32
	StatFrameTimeBuf     [1024]uint32
	StatFrameTimeBufPos  uint32
	// RTSS v2.5+ exposes direct FPS in 1/10 precision.
	StatFrameTimeBufFramerate uint32
}

var (
	minAppEntrySize = uint32(unsafe.Offsetof(appEntry{}.FrameTime) + unsafe.Sizeof(appEntry{}.FrameTime))
	rawFPSFieldEnd  = uint32(unsafe.Offsetof(appEntry{}.StatFrameTimeBufFramerate) + unsafe.Sizeof(appEntry{}.StatFrameTimeBufFramerate))
)

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
		fps, available := readEntryFPS(current, header.Version, header.AppEntrySize, nowTick)
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

func readEntryFPS(entry *appEntry, version uint32, appEntrySize uint32, nowTick uint32) (float64, bool) {
	if entry == nil {
		return 0, false
	}
	if entry.Time1 > 0 && tickDeltaMS(nowTick, entry.Time1) > fpsStaleMS {
		return 0, false
	}

	// RTSS v2.5+ provides direct framerate in 1/10 FPS precision.
	if version >= versionFPSRaw && appEntrySize >= rawFPSFieldEnd && entry.StatFrameTimeBufFramerate > 0 {
		fps := sanitizeFPS(entry.StatFrameTimeBufFramerate)
		return fps, fps > 0
	}

	// dwFrameTime is in microseconds per frame.
	if entry.FrameTime > 0 {
		fps := sanitizeFPSFloat(1000000.0 / float64(entry.FrameTime))
		if fps > 0 {
			return fps, true
		}
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
