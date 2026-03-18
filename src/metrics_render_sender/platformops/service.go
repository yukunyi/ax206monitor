package platformops

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
)

const serviceName = "metrics_render_sender"

type ServiceInstallOptions struct {
}

type ServiceInstallResult struct {
	ServicePath string
	UserMode    bool
}

func InstallService(_ ServiceInstallOptions) (ServiceInstallResult, error) {
	if runtime.GOOS != "linux" {
		return ServiceInstallResult{}, fmt.Errorf("--install only supports Linux systemd")
	}

	execPath, err := os.Executable()
	if err != nil {
		return ServiceInstallResult{}, fmt.Errorf("failed to resolve executable path: %w", err)
	}
	execPath, err = filepath.EvalSymlinks(execPath)
	if err != nil {
		execPath = filepath.Clean(execPath)
	}

	rootMode, err := isRootUser()
	if err != nil {
		return ServiceInstallResult{}, err
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return ServiceInstallResult{}, fmt.Errorf("failed to resolve home directory: %w", err)
	}

	targetBin, servicePath, wantedBy := resolveServicePaths(rootMode, homeDir)
	if err := os.MkdirAll(filepath.Dir(targetBin), 0o755); err != nil {
		return ServiceInstallResult{}, fmt.Errorf("failed to create binary directory: %w", err)
	}
	if err := copyExecutable(execPath, targetBin); err != nil {
		return ServiceInstallResult{}, err
	}

	if err := os.MkdirAll(filepath.Dir(servicePath), 0o755); err != nil {
		return ServiceInstallResult{}, fmt.Errorf("failed to create service directory: %w", err)
	}

	serviceContent := buildServiceContent(targetBin, homeDir, wantedBy)
	if err := os.WriteFile(servicePath, []byte(serviceContent), 0o644); err != nil {
		return ServiceInstallResult{}, fmt.Errorf("failed to write service file: %w", err)
	}

	if rootMode {
		if err := runCommand("systemctl", "daemon-reload"); err != nil {
			return ServiceInstallResult{}, err
		}
		if err := runCommand("systemctl", "enable", "--now", serviceName+".service"); err != nil {
			return ServiceInstallResult{}, err
		}
	} else {
		if err := runCommand("systemctl", "--user", "daemon-reload"); err != nil {
			return ServiceInstallResult{}, err
		}
		if err := runCommand("systemctl", "--user", "enable", "--now", serviceName+".service"); err != nil {
			return ServiceInstallResult{}, err
		}
	}

	return ServiceInstallResult{ServicePath: servicePath, UserMode: !rootMode}, nil
}

func UninstallService() error {
	if runtime.GOOS != "linux" {
		return fmt.Errorf("--uninstall only supports Linux systemd")
	}

	rootMode, err := isRootUser()
	if err != nil {
		return err
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to resolve home directory: %w", err)
	}
	targetBin, servicePath, _ := resolveServicePaths(rootMode, homeDir)

	if rootMode {
		_ = runCommand("systemctl", "disable", "--now", serviceName+".service")
		_ = os.Remove(servicePath)
		if err := runCommand("systemctl", "daemon-reload"); err != nil {
			return err
		}
	} else {
		_ = runCommand("systemctl", "--user", "disable", "--now", serviceName+".service")
		_ = os.Remove(servicePath)
		if err := runCommand("systemctl", "--user", "daemon-reload"); err != nil {
			return err
		}
	}

	_ = os.Remove(targetBin)
	return nil
}

func resolveServicePaths(rootMode bool, homeDir string) (binPath, servicePath, wantedBy string) {
	if rootMode {
		return "/usr/local/bin/metrics_render_sender", "/etc/systemd/system/metrics_render_sender.service", "multi-user.target"
	}
	return filepath.Join(homeDir, ".local", "bin", "metrics_render_sender"),
		filepath.Join(homeDir, ".config", "systemd", "user", "metrics_render_sender.service"),
		"default.target"
}

func buildServiceContent(binaryPath, homeDir, wantedBy string) string {
	return fmt.Sprintf(`[Unit]
Description=MetricsRenderSender
After=network.target

[Service]
Type=simple
ExecStart=%s
Restart=always
RestartSec=5
Environment=HOME=%s

[Install]
WantedBy=%s
`, binaryPath, homeDir, wantedBy)
}

func copyExecutable(sourcePath, targetPath string) error {
	source, err := os.Open(sourcePath)
	if err != nil {
		return fmt.Errorf("failed to open source executable: %w", err)
	}
	defer source.Close()

	target, err := os.Create(targetPath)
	if err != nil {
		return fmt.Errorf("failed to create target executable: %w", err)
	}
	defer target.Close()

	if _, err := io.Copy(target, source); err != nil {
		return fmt.Errorf("failed to copy executable: %w", err)
	}
	if err := target.Chmod(0o755); err != nil {
		return fmt.Errorf("failed to chmod executable: %w", err)
	}
	return nil
}
