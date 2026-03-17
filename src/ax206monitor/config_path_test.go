package main

import (
	"path/filepath"
	"testing"
)

func TestGetUserConfigPathUsesXDGConfigHome(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", "/tmp/ax206monitor-xdg")
	t.Setenv("HOME", "/tmp/ax206monitor-home")

	path, err := getUserConfigPath()
	if err != nil {
		t.Fatalf("getUserConfigPath failed: %v", err)
	}

	expected := filepath.Join("/tmp/ax206monitor-xdg", "ax206monitor", "config.json")
	if path != expected {
		t.Fatalf("expected path %q, got %q", expected, path)
	}
}

func TestGetUserConfigPathFallsBackToHomeConfigDir(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", "")
	t.Setenv("HOME", "/tmp/ax206monitor-home")

	path, err := getUserConfigPath()
	if err != nil {
		t.Fatalf("getUserConfigPath failed: %v", err)
	}

	expected := filepath.Join("/tmp/ax206monitor-home", ".config", "ax206monitor", "config.json")
	if path != expected {
		t.Fatalf("expected path %q, got %q", expected, path)
	}
}
