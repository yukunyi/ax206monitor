//go:build windows

package rtss

import (
	"unsafe"

	"golang.org/x/sys/windows"
)

const (
	sharedMemoryName = "RTSSSharedMemoryV2"
	signatureA       = 0x52545353
	signatureB       = 0x53535452
	fileMapRead      = 0x0004
	pidOffset        = 260
	fpsOffset        = 284
)

var (
	modKernel32                  = windows.NewLazySystemDLL("kernel32.dll")
	modUser32                    = windows.NewLazySystemDLL("user32.dll")
	procOpenFileMappingW         = modKernel32.NewProc("OpenFileMappingW")
	procMapViewOfFile            = modKernel32.NewProc("MapViewOfFile")
	procUnmapViewOfFile          = modKernel32.NewProc("UnmapViewOfFile")
	procCloseHandle              = modKernel32.NewProc("CloseHandle")
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
	if header.AppEntrySize < fpsOffset+4 || header.AppArrSize == 0 || header.AppArrOffset == 0 {
		return Metrics{}, false
	}
	if header.AppArrSize > 4096 {
		return Metrics{}, false
	}

	foregroundPID := getForegroundPID()
	metrics := Metrics{ForegroundPID: foregroundPID}
	appBase := uintptr(view) + uintptr(header.AppArrOffset)
	entrySize := uintptr(header.AppEntrySize)

	for i := uint32(0); i < header.AppArrSize; i++ {
		entry := appBase + uintptr(i)*entrySize
		pid := *(*uint32)(unsafe.Pointer(entry + pidOffset))
		fpsRaw := *(*uint32)(unsafe.Pointer(entry + fpsOffset))
		fps := sanitizeFPS(fpsRaw)
		if fps <= 0 {
			continue
		}
		metrics.ActiveApps++
		if metrics.MaxFPS < fps {
			metrics.MaxFPS = fps
		}
		if foregroundPID != 0 && pid == foregroundPID {
			metrics.ForegroundFPS = fps
		}
	}

	if metrics.ForegroundFPS <= 0 {
		metrics.ForegroundFPS = metrics.MaxFPS
	}
	return metrics, true
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
