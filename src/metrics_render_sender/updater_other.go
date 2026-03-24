//go:build !linux && !windows

package main

import "fmt"

func scheduleUpdateAndRestart(packageRoot, executablePath string, currentArgs []string, currentPID int) error {
	_ = packageRoot
	_ = executablePath
	_ = currentArgs
	_ = currentPID
	return fmt.Errorf("auto update unsupported on this platform")
}
