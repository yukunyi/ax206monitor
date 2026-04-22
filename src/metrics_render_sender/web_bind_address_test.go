package main

import (
	"path/filepath"
	"testing"
)

func TestNormalizeWebBindHostDefaultsToLoopback(t *testing.T) {
	if got := normalizeWebBindHost(""); got != defaultWebBindHost {
		t.Fatalf("expected default host %q, got %q", defaultWebBindHost, got)
	}
	if got := normalizeWebBindHost("127.0.0.0"); got != defaultWebBindHost {
		t.Fatalf("expected invalid loopback variant to normalize to %q, got %q", defaultWebBindHost, got)
	}
}

func TestLoadWebBindHostUsesDefaultWhenFileMissing(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", "/tmp/metrics_render_sender-bind-xdg-missing")
	t.Setenv("HOME", "/tmp/metrics_render_sender-bind-home-missing")

	host, err := loadWebBindHost()
	if err != nil {
		t.Fatalf("loadWebBindHost failed: %v", err)
	}
	if host != defaultWebBindHost {
		t.Fatalf("expected default host %q, got %q", defaultWebBindHost, host)
	}
}

func TestSaveAndLoadWebBindHostRoundTrip(t *testing.T) {
	root := "/tmp/metrics_render_sender-bind-xdg-roundtrip"
	t.Setenv("XDG_CONFIG_HOME", root)
	t.Setenv("HOME", "/tmp/metrics_render_sender-bind-home-roundtrip")

	if err := saveWebBindHost(publicWebBindHost); err != nil {
		t.Fatalf("saveWebBindHost failed: %v", err)
	}

	host, err := loadWebBindHost()
	if err != nil {
		t.Fatalf("loadWebBindHost failed: %v", err)
	}
	if host != publicWebBindHost {
		t.Fatalf("expected host %q, got %q", publicWebBindHost, host)
	}

	path, err := getWebBindHostPath()
	if err != nil {
		t.Fatalf("getWebBindHostPath failed: %v", err)
	}
	expected := filepath.Join(root, "metrics_render_sender", webBindHostFile)
	if path != expected {
		t.Fatalf("expected path %q, got %q", expected, path)
	}
}
