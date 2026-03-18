//go:build !linux && !windows

package main

func startPlatformLockScreenMonitor(onChange func(bool)) (LockScreenMonitor, error) {
	_ = onChange
	return nil, nil
}
