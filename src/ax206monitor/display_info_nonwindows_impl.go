//go:build !windows

package main

func detectPrimaryDisplayInfoWindowsImpl() (int, int, float64, bool) {
	return 0, 0, 0, false
}
