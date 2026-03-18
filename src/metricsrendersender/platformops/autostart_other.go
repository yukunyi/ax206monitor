//go:build !linux && !windows

package platformops

import "fmt"

func IsAutoStartEnabled() (bool, error) {
	return false, fmt.Errorf("autostart unsupported on this platform")
}

func EnableAutoStart() error {
	return fmt.Errorf("autostart unsupported on this platform")
}

func DisableAutoStart() error {
	return fmt.Errorf("autostart unsupported on this platform")
}
