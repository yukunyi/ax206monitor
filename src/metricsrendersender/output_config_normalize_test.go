package main

import (
	"sync"
	"testing"
)

var initNormalizeOutputConfigTests sync.Once

func initNormalizeOutputConfigTestDeps() {
	initNormalizeOutputConfigTests.Do(func() {
		initLogger()
	})
}

func TestNormalizeMonitorConfigPrefersExplicitEmptyOutputs(t *testing.T) {
	initNormalizeOutputConfigTestDeps()

	cfg := &MonitorConfig{
		Name:        "test",
		Width:       480,
		Height:      320,
		Outputs:     []OutputConfig{},
		OutputTypes: []string{outputTypeAX206USB},
	}

	normalizeMonitorConfig(cfg)

	if len(cfg.Outputs) != 0 {
		t.Fatalf("expected no outputs, got %#v", cfg.Outputs)
	}
	if len(cfg.OutputTypes) != 0 {
		t.Fatalf("expected no output types, got %#v", cfg.OutputTypes)
	}
}

func TestNormalizeMonitorConfigFallsBackToOutputTypesWhenOutputsMissing(t *testing.T) {
	initNormalizeOutputConfigTestDeps()

	cfg := &MonitorConfig{
		Name:        "test",
		Width:       480,
		Height:      320,
		OutputTypes: []string{outputTypeAX206USB},
	}

	normalizeMonitorConfig(cfg)

	if len(cfg.Outputs) != 1 {
		t.Fatalf("expected 1 output, got %#v", cfg.Outputs)
	}
	if cfg.Outputs[0].Type != outputTypeAX206USB {
		t.Fatalf("expected output type %q, got %#v", outputTypeAX206USB, cfg.Outputs[0])
	}
	if len(cfg.OutputTypes) != 1 || cfg.OutputTypes[0] != outputTypeAX206USB {
		t.Fatalf("expected output types [%q], got %#v", outputTypeAX206USB, cfg.OutputTypes)
	}
}

func TestNormalizeMonitorConfigPreservesDisabledOutputs(t *testing.T) {
	initNormalizeOutputConfigTestDeps()

	enabled := false
	cfg := &MonitorConfig{
		Name:   "test",
		Width:  480,
		Height: 320,
		Outputs: []OutputConfig{
			{
				Type:        outputTypeAX206USB,
				Enabled:     &enabled,
				ReconnectMS: 1500,
			},
		},
	}

	normalizeMonitorConfig(cfg)

	if len(cfg.Outputs) != 1 {
		t.Fatalf("expected 1 output, got %#v", cfg.Outputs)
	}
	if cfg.Outputs[0].Enabled == nil || *cfg.Outputs[0].Enabled {
		t.Fatalf("expected disabled output, got %#v", cfg.Outputs[0])
	}
	if cfg.Outputs[0].ReconnectMS != 1500 {
		t.Fatalf("expected reconnect_ms preserved, got %#v", cfg.Outputs[0])
	}
	if len(cfg.OutputTypes) != 0 {
		t.Fatalf("expected no enabled output types, got %#v", cfg.OutputTypes)
	}
}

func TestNormalizeMonitorConfigRemovesCoolerControlLegacyUsername(t *testing.T) {
	initNormalizeOutputConfigTestDeps()

	cfg := &MonitorConfig{
		Name:   "test",
		Width:  480,
		Height: 320,
		CollectorConfig: map[string]CollectorConfig{
			collectorCoolerControl: {
				Options: map[string]interface{}{
					"url":      defaultCoolerControlURL,
					"username": "legacy-user",
					"password": "secret",
				},
			},
		},
	}

	normalizeMonitorConfig(cfg)

	collector := cfg.CollectorConfig[collectorCoolerControl]
	if _, exists := collector.Options["username"]; exists {
		t.Fatalf("expected coolercontrol legacy username removed, got %#v", collector.Options)
	}
	if got := collector.Options["password"]; got != "secret" {
		t.Fatalf("expected coolercontrol password preserved, got %#v", got)
	}
}
