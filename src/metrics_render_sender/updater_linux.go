//go:build linux

package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"syscall"
)

func scheduleUpdateAndRestart(packageRoot, executablePath string, currentArgs []string, currentPID int) error {
	expectedName, err := expectedPackageExecutableName("linux")
	if err != nil {
		return err
	}
	execDir := filepath.Dir(executablePath)
	currentBase := filepath.Base(executablePath)
	sourceExecutablePath := filepath.Join(packageRoot, expectedName)
	if _, err := os.Stat(sourceExecutablePath); err != nil {
		return fmt.Errorf("package executable missing: %w", err)
	}

	entries, err := os.ReadDir(packageRoot)
	if err != nil {
		return fmt.Errorf("read package root failed: %w", err)
	}
	for _, entry := range entries {
		sourcePath := filepath.Join(packageRoot, entry.Name())
		if entry.IsDir() {
			continue
		}
		targetName := entry.Name()
		info, err := entry.Info()
		if err != nil {
			return fmt.Errorf("read file info failed: %w", err)
		}
		targetPath := filepath.Join(execDir, targetName)
		if targetName == expectedName {
			tempExecutablePath := filepath.Join(execDir, "."+currentBase+".update")
			if err := copyFileWithMode(sourcePath, tempExecutablePath, info.Mode()); err != nil {
				return err
			}
			if err := os.Rename(tempExecutablePath, executablePath); err != nil {
				_ = os.Remove(tempExecutablePath)
				return fmt.Errorf("replace executable failed: %w", err)
			}
			continue
		}
		if err := copyFileWithMode(sourcePath, targetPath, info.Mode()); err != nil {
			return err
		}
	}

	scriptPath := filepath.Join(os.TempDir(), fmt.Sprintf("mrs-restart-%d.sh", currentPID))
	scriptContent := "#!/bin/sh\n" +
		"pid=\"$1\"\n" +
		"shift\n" +
		"while kill -0 \"$pid\" 2>/dev/null; do\n" +
		"  sleep 1\n" +
		"done\n" +
		"exec \"$@\"\n"
	if err := os.WriteFile(scriptPath, []byte(scriptContent), 0o700); err != nil {
		return fmt.Errorf("write restart helper failed: %w", err)
	}

	commandArgs := []string{scriptPath, strconv.Itoa(currentPID), executablePath}
	commandArgs = append(commandArgs, currentArgs...)
	command := exec.Command("/bin/sh", commandArgs...)
	command.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
	if err := command.Start(); err != nil {
		return fmt.Errorf("start restart helper failed: %w", err)
	}
	return nil
}
