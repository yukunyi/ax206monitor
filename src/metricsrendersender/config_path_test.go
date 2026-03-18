package main

import (
	"path/filepath"
	"testing"
)

func TestGetUserConfigPathUsesXDGConfigHome(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", "/tmp/metricsrendersender-xdg")
	t.Setenv("HOME", "/tmp/metricsrendersender-home")

	path, err := getUserConfigPath()
	if err != nil {
		t.Fatalf("getUserConfigPath failed: %v", err)
	}

	expected := filepath.Join("/tmp/metricsrendersender-xdg", "metricsrendersender", "config.json")
	if path != expected {
		t.Fatalf("expected path %q, got %q", expected, path)
	}
}

func TestGetUserConfigPathFallsBackToHomeConfigDir(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", "")
	t.Setenv("HOME", "/tmp/metricsrendersender-home")

	path, err := getUserConfigPath()
	if err != nil {
		t.Fatalf("getUserConfigPath failed: %v", err)
	}

	expected := filepath.Join("/tmp/metricsrendersender-home", ".config", "metricsrendersender", "config.json")
	if path != expected {
		t.Fatalf("expected path %q, got %q", expected, path)
	}
}
