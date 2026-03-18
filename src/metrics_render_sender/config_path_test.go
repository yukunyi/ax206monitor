package main

import (
	"path/filepath"
	"testing"
)

func TestGetUserConfigPathUsesXDGConfigHome(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", "/tmp/metrics_render_sender-xdg")
	t.Setenv("HOME", "/tmp/metrics_render_sender-home")

	path, err := getUserConfigPath()
	if err != nil {
		t.Fatalf("getUserConfigPath failed: %v", err)
	}

	expected := filepath.Join("/tmp/metrics_render_sender-xdg", "metrics_render_sender", "config.json")
	if path != expected {
		t.Fatalf("expected path %q, got %q", expected, path)
	}
}

func TestGetUserConfigPathFallsBackToHomeConfigDir(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", "")
	t.Setenv("HOME", "/tmp/metrics_render_sender-home")

	path, err := getUserConfigPath()
	if err != nil {
		t.Fatalf("getUserConfigPath failed: %v", err)
	}

	expected := filepath.Join("/tmp/metrics_render_sender-home", ".config", "metrics_render_sender", "config.json")
	if path != expected {
		t.Fatalf("expected path %q, got %q", expected, path)
	}
}
