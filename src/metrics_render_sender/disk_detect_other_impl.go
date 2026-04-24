//go:build !windows

package main

func detectDiskInfoByWindows() []*DiskInfo {
	return []*DiskInfo{}
}
