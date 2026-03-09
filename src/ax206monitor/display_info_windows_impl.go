//go:build windows

package main

import "golang.org/x/sys/windows"

const (
	smCXScreen = 0
	smCYScreen = 1
	vrRefresh  = 116
)

var (
	user32DLL           = windows.NewLazySystemDLL("user32.dll")
	gdi32DLL            = windows.NewLazySystemDLL("gdi32.dll")
	procGetSystemMetric = user32DLL.NewProc("GetSystemMetrics")
	procGetDC           = user32DLL.NewProc("GetDC")
	procReleaseDC       = user32DLL.NewProc("ReleaseDC")
	procGetDeviceCaps   = gdi32DLL.NewProc("GetDeviceCaps")
)

func detectPrimaryDisplayInfoWindowsImpl() (int, int, float64, bool) {
	width, _, _ := procGetSystemMetric.Call(uintptr(smCXScreen))
	height, _, _ := procGetSystemMetric.Call(uintptr(smCYScreen))
	if width == 0 || height == 0 {
		return 0, 0, 0, false
	}

	hdc, _, _ := procGetDC.Call(0)
	if hdc == 0 {
		return int(width), int(height), 0, true
	}
	defer procReleaseDC.Call(0, hdc)

	refreshRaw, _, _ := procGetDeviceCaps.Call(hdc, uintptr(vrRefresh))
	refresh := float64(refreshRaw)
	if refresh <= 1 {
		refresh = 0
	}
	return int(width), int(height), refresh, true
}
