//go:build windows

package main

import (
	"strings"

	"golang.org/x/sys/windows"
)

func detectDiskInfoByWindows() []*DiskInfo {
	names := enumerateWindowsFixedDriveNames()
	disks := make([]*DiskInfo, 0, len(names))
	for _, name := range names {
		diskInfo := buildWindowsDiskInfo(name)
		if diskInfo == nil {
			continue
		}
		disks = append(disks, diskInfo)
	}
	return disks
}

func enumerateWindowsFixedDriveNames() []string {
	buf := make([]uint16, 254)
	n, err := windows.GetLogicalDriveStrings(uint32(len(buf)), &buf[0])
	if err != nil || n == 0 {
		return []string{}
	}

	names := make([]string, 0, 8)
	current := make([]uint16, 0, 8)
	for _, ch := range buf[:n] {
		if ch != 0 {
			current = append(current, ch)
			continue
		}
		if len(current) == 0 {
			continue
		}
		path := windows.UTF16ToString(current)
		current = current[:0]
		path = strings.TrimSpace(path)
		if path == "" {
			continue
		}
		ptr, err := windows.UTF16PtrFromString(path)
		if err != nil {
			continue
		}
		if windows.GetDriveType(ptr) != windows.DRIVE_FIXED {
			continue
		}
		names = append(names, strings.TrimSuffix(path, `\`))
	}
	return names
}

func buildWindowsDiskInfo(name string) *DiskInfo {
	trimmed := strings.TrimSpace(name)
	if trimmed == "" {
		return nil
	}
	ptr, err := windows.UTF16PtrFromString(trimmed + `\`)
	if err != nil {
		return nil
	}
	var freeBytesAvailable uint64
	var totalBytes uint64
	var totalFreeBytes uint64
	if err := windows.GetDiskFreeSpaceEx(ptr, &freeBytesAvailable, &totalBytes, &totalFreeBytes); err != nil {
		return nil
	}

	info := &DiskInfo{
		Name:      trimmed,
		Model:     inferDiskModel(trimmed),
		Size:      int64(totalBytes / (1024 * 1024 * 1024)),
		Used:      int64((totalBytes - totalFreeBytes) / (1024 * 1024 * 1024)),
		Available: int64(totalFreeBytes / (1024 * 1024 * 1024)),
	}
	if totalBytes > 0 {
		info.Usage = float64(totalBytes-totalFreeBytes) * 100 / float64(totalBytes)
	}
	return info
}
