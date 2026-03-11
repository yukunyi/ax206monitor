//go:build linux

package main

// Linux lock-state detection is intentionally disabled in monitor runtime
// because subprocess-based probing is forbidden for monitoring paths.
func startPlatformLockScreenMonitor(onChange func(bool)) (LockScreenMonitor, error) {
	return nil, nil
}
