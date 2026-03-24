//go:build windows

package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
)

func scheduleUpdateAndRestart(packageRoot, executablePath string, currentArgs []string, currentPID int) error {
	expectedName, err := expectedPackageExecutableName("windows")
	if err != nil {
		return err
	}
	if _, err := os.Stat(filepath.Join(packageRoot, expectedName)); err != nil {
		return fmt.Errorf("package executable missing: %w", err)
	}

	execDir := filepath.Dir(executablePath)
	currentBase := filepath.Base(executablePath)
	scriptPath := filepath.Join(filepath.Dir(packageRoot), "apply_update.cmd")

	var script strings.Builder
	script.WriteString("@echo off\r\n")
	script.WriteString("setlocal enableextensions\r\n")
	script.WriteString(fmt.Sprintf("set \"PID=%d\"\r\n", currentPID))
	script.WriteString(fmt.Sprintf("set \"SRC=%s\"\r\n", escapeBatchValue(packageRoot)))
	script.WriteString(fmt.Sprintf("set \"DST=%s\"\r\n", escapeBatchValue(execDir)))
	script.WriteString(fmt.Sprintf("set \"TARGET_EXE=%s\"\r\n", escapeBatchValue(executablePath)))
	script.WriteString(fmt.Sprintf("set \"PKG_EXE=%s\"\r\n", escapeBatchValue(expectedName)))
	script.WriteString(":wait_loop\r\n")
	script.WriteString("tasklist /FI \"PID eq %PID%\" | findstr /R /C:\"[ ]%PID%[ ]\" >nul\r\n")
	script.WriteString("if %errorlevel%==0 (\r\n")
	script.WriteString("  timeout /T 1 /NOBREAK >nul\r\n")
	script.WriteString("  goto wait_loop\r\n")
	script.WriteString(")\r\n")
	script.WriteString("xcopy \"%SRC%\\*\" \"%DST%\\\" /E /I /Y /Q >nul\r\n")
	if !strings.EqualFold(currentBase, expectedName) {
		script.WriteString(fmt.Sprintf("copy /Y \"%%DST%%\\%s\" \"%%TARGET_EXE%%\" >nul\r\n", expectedName))
	}
	script.WriteString(fmt.Sprintf("start \"\" \"%%TARGET_EXE%%\"%s\r\n", formatWindowsStartArgs(currentArgs)))
	script.WriteString("rmdir /S /Q \"%SRC%\" >nul 2>nul\r\n")
	script.WriteString("del \"%~f0\" >nul 2>nul\r\n")

	if err := os.WriteFile(scriptPath, []byte(script.String()), 0o700); err != nil {
		return fmt.Errorf("write update helper failed: %w", err)
	}

	command := exec.Command("cmd", "/C", scriptPath)
	command.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	if err := command.Start(); err != nil {
		return fmt.Errorf("start update helper failed: %w", err)
	}
	return nil
}

func formatWindowsStartArgs(args []string) string {
	if len(args) == 0 {
		return ""
	}
	parts := make([]string, 0, len(args))
	for _, arg := range args {
		parts = append(parts, syscall.EscapeArg(arg))
	}
	return " " + strings.Join(parts, " ")
}

func escapeBatchValue(value string) string {
	escaped := strings.ReplaceAll(value, "%", "%%")
	return strings.ReplaceAll(escaped, "\"", "\"\"")
}
