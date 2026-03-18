//go:build windows

package platformops

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const windowsAutoStartFile = "metricsrendersender.cmd"

func IsAutoStartEnabled() (bool, error) {
	path, err := autoStartScriptPath()
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

	scriptPath, err := autoStartScriptPath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(scriptPath), 0o755); err != nil {
		return fmt.Errorf("create startup directory failed: %w", err)
	}

	escaped := strings.ReplaceAll(execPath, `"`, `""`)
	content := "@echo off\r\nstart \"\" \"" + escaped + "\"\r\n"
	if err := os.WriteFile(scriptPath, []byte(content), 0o644); err != nil {
		return fmt.Errorf("write startup script failed: %w", err)
	}
	return nil
}

func DisableAutoStart() error {
	scriptPath, err := autoStartScriptPath()
	if err != nil {
		return err
	}
	if err := os.Remove(scriptPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("remove startup script failed: %w", err)
	}
	return nil
}

func autoStartScriptPath() (string, error) {
	appData := strings.TrimSpace(os.Getenv("APPDATA"))
	if appData == "" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("resolve home directory failed: %w", err)
		}
		appData = filepath.Join(homeDir, "AppData", "Roaming")
	}
	return filepath.Join(appData, "Microsoft", "Windows", "Start Menu", "Programs", "Startup", windowsAutoStartFile), nil
}
