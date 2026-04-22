package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

const (
	defaultWebBindHost = "127.0.0.1"
	publicWebBindHost  = "0.0.0.0"
	webBindHostFile    = "web-bind-host"
)

func normalizeWebBindHost(raw string) string {
	switch strings.TrimSpace(raw) {
	case publicWebBindHost:
		return publicWebBindHost
	default:
		return defaultWebBindHost
	}
}

func getWebBindHostPath() (string, error) {
	configDir, err := getUserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(configDir, webBindHostFile), nil
}

func loadWebBindHost() (string, error) {
	path, err := getWebBindHostPath()
	if err != nil {
		return defaultWebBindHost, err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return defaultWebBindHost, nil
		}
		return defaultWebBindHost, fmt.Errorf("read web bind host failed: %w", err)
	}
	return normalizeWebBindHost(string(data)), nil
}

func saveWebBindHost(host string) error {
	path, err := getWebBindHostPath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create web bind host directory failed: %w", err)
	}
	value := normalizeWebBindHost(host)
	if err := os.WriteFile(path, []byte(value+"\n"), 0o644); err != nil {
		return fmt.Errorf("write web bind host failed: %w", err)
	}
	return nil
}

func buildWebListenAddr(host string, port int) string {
	return normalizeWebBindHost(host) + ":" + strconv.Itoa(port)
}

func buildWebAccessURL(port int) string {
	return "http://" + defaultWebBindHost + ":" + strconv.Itoa(port)
}
