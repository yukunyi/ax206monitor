//go:build (linux && cgo) || (windows && cgo)

package main

type trayUpdateMenuConfig struct {
	checkTitle     string
	checkEnabled   bool
	upgradeTitle   string
	upgradeEnabled bool
}

func resolveTrayUpdateMenuConfig(state appUpdateState) trayUpdateMenuConfig {
	config := trayUpdateMenuConfig{
		checkTitle:   "Check for Updates",
		checkEnabled: true,
		upgradeTitle: "Upgrade Unavailable",
	}
	switch {
	case !state.Supported:
		config.checkTitle = "Updates Unavailable"
		config.checkEnabled = false
	case state.Installing:
		config.checkTitle = "Checking for Updates"
		config.checkEnabled = false
		config.upgradeTitle = "Updating..."
		config.upgradeEnabled = false
	case state.Checking:
		config.checkTitle = "Checking for Updates..."
		config.checkEnabled = false
	case state.UpdateAvailable:
		config.upgradeTitle = "Upgrade to v" + state.LatestVersion
		config.upgradeEnabled = true
	case state.LatestVersion != "":
		config.upgradeTitle = "Reinstall v" + state.LatestVersion
		config.upgradeEnabled = true
	}
	return config
}
