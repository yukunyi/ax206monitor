//go:build linux

package main

import (
	"bufio"
	"context"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

func startPlatformLockScreenMonitor(onChange func(bool)) (LockScreenMonitor, error) {
	sessionID := resolveLinuxSessionID()
	if sessionID == "" {
		return nil, nil
	}
	detect := func() (bool, bool) {
		return detectLinuxLockedHint(sessionID)
	}
	return startLockPollingMonitor(2*time.Second, detect, onChange), nil
}

func resolveLinuxSessionID() string {
	if value := strings.TrimSpace(os.Getenv("XDG_SESSION_ID")); value != "" {
		return value
	}

	uid := os.Getuid()
	if uid <= 0 {
		return ""
	}
	path := "/run/systemd/users/" + strconv.Itoa(uid)
	file, err := os.Open(path)
	if err != nil {
		return ""
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if !strings.HasPrefix(line, "ACTIVE_SESSIONS=") {
			continue
		}
		values := strings.Fields(strings.TrimSpace(strings.TrimPrefix(line, "ACTIVE_SESSIONS=")))
		for _, value := range values {
			if strings.TrimSpace(value) != "" {
				return strings.TrimSpace(value)
			}
		}
	}
	return ""
}

func detectLinuxLockedHint(sessionID string) (bool, bool) {
	if strings.TrimSpace(sessionID) == "" {
		return false, false
	}
	ctx, cancel := context.WithTimeout(context.Background(), 450*time.Millisecond)
	defer cancel()

	cmd := exec.CommandContext(ctx, "loginctl", "show-session", sessionID, "-p", "LockedHint", "--value")
	output, err := cmd.Output()
	if err != nil {
		return false, false
	}
	value := strings.ToLower(strings.TrimSpace(string(output)))
	switch value {
	case "yes", "true", "1":
		return true, true
	case "no", "false", "0":
		return false, true
	default:
		return false, false
	}
}
