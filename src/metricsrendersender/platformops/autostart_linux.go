//go:build linux

package platformops

import (
	"fmt"
	"os"
	"path/filepath"
)

const linuxAutoStartFile = "metricsrendersender.desktop"

func IsAutoStartEnabled() (bool, error) {
	path, err := autoStartDesktopPath()
	if err != nil {
		return false, err
	}
	_, err = os.Stat(path)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, fmt.Errorf("check autostart failed: %w", err)
}

func EnableAutoStart() error {
	execPath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("resolve executable failed: %w", err)
	}
	execPath, err = filepath.EvalSymlinks(execPath)
	if err != nil {
		execPath = filepath.Clean(execPath)
	}

	desktopPath, err := autoStartDesktopPath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(desktopPath), 0o755); err != nil {
		return fmt.Errorf("create autostart directory failed: %w", err)
	}

	content := fmt.Sprintf(`[Desktop Entry]
Type=Application
Name=MetricsRenderSender
Comment=MetricsRenderSender
Exec=%q
Terminal=false
X-GNOME-Autostart-enabled=true
`, execPath)
	if err := os.WriteFile(desktopPath, []byte(content), 0o644); err != nil {
		return fmt.Errorf("write autostart file failed: %w", err)
	}
	return nil
}

func DisableAutoStart() error {
	desktopPath, err := autoStartDesktopPath()
	if err != nil {
		return err
	}
	if err := os.Remove(desktopPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("remove autostart file failed: %w", err)
	}
	return nil
}

func autoStartDesktopPath() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("resolve home directory failed: %w", err)
	}
	return filepath.Join(homeDir, ".config", "autostart", linuxAutoStartFile), nil
}
