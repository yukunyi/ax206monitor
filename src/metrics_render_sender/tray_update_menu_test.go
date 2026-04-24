package main

import "testing"

func TestResolveTrayUpdateMenuConfigSeparatesCheckAndUpgrade(t *testing.T) {
	cfg := resolveTrayUpdateMenuConfig(appUpdateState{Supported: true, LatestVersion: "1.2.3", UpdateAvailable: true})
	if cfg.checkTitle != "Check for Updates" || !cfg.checkEnabled {
		t.Fatalf("unexpected check menu config: %#v", cfg)
	}
	if cfg.upgradeTitle != "Upgrade to v1.2.3" || !cfg.upgradeEnabled {
		t.Fatalf("unexpected upgrade menu config: %#v", cfg)
	}
}
