//go:build windows

package main

import (
	"strings"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"
)

const (
	desktopReadObjects  = 0x0001
	desktopSwitchGlobal = 0x0100
	userObjectNameClass = 2
)

var (
	lockUser32DLL                     = windows.NewLazySystemDLL("user32.dll")
	procOpenInputDesktop              = lockUser32DLL.NewProc("OpenInputDesktop")
	procCloseDesktop                  = lockUser32DLL.NewProc("CloseDesktop")
	procGetUserObjectInformationWLock = lockUser32DLL.NewProc("GetUserObjectInformationW")
)

func startPlatformLockScreenMonitor(onChange func(bool)) (LockScreenMonitor, error) {
	return startLockPollingMonitor(time.Second, detectWindowsLockScreenState, onChange), nil
}

func detectWindowsLockScreenState() (bool, bool) {
	access := uintptr(desktopReadObjects | desktopSwitchGlobal)
	handle, _, callErr := procOpenInputDesktop.Call(0, 0, access)
	if handle == 0 {
		if errno, ok := callErr.(windows.Errno); ok && errno == windows.ERROR_ACCESS_DENIED {
			return true, true
		}
		return false, false
	}
	defer procCloseDesktop.Call(handle)

	var needed uint32
	procGetUserObjectInformationWLock.Call(
		handle,
		uintptr(userObjectNameClass),
		0,
		0,
		uintptr(unsafe.Pointer(&needed)),
	)
	if needed == 0 {
		return false, false
	}
	buf := make([]uint16, int(needed/2)+1)
	ret, _, _ := procGetUserObjectInformationWLock.Call(
		handle,
		uintptr(userObjectNameClass),
		uintptr(unsafe.Pointer(&buf[0])),
		uintptr(needed),
		uintptr(unsafe.Pointer(&needed)),
	)
	if ret == 0 {
		return false, false
	}
	desktopName := strings.ToLower(strings.TrimSpace(windows.UTF16ToString(buf)))
	if desktopName == "" {
		return false, false
	}
	return desktopName != "default", true
}
