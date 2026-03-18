package main

import "metrics_render_sender/platformops"

type ServiceInstallOptions = platformops.ServiceInstallOptions

func InstallService(options ServiceInstallOptions) error {
	result, err := platformops.InstallService(options)
	if err != nil {
		return err
	}
	if result.UserMode {
		logInfo("Installed user service: %s", result.ServicePath)
		logInfo("Check status: systemctl --user status %s", "metrics_render_sender")
	} else {
		logInfo("Installed system service: %s", result.ServicePath)
		logInfo("Check status: systemctl status %s", "metrics_render_sender")
	}
	return nil
}

func UninstallService() error {
	if err := platformops.UninstallService(); err != nil {
		return err
	}
	logInfo("Service uninstalled: %s", "metrics_render_sender")
	return nil
}

func InstallAX206UdevRule() error {
	result, err := platformops.InstallAX206UdevRule()
	if err != nil {
		return err
	}
	logInfo("AX206 udev rule installed for user %s: %s", result.TargetUser, result.RulePath)
	logInfo("udev rules reloaded. If AX206 is already connected, replug the USB cable once.")
	return nil
}

func resolveUdevRuleTargetUser() (string, error) {
	return platformops.ResolveUdevRuleTargetUser()
}

func buildAX206UdevRuleContent(targetUser string) string {
	return platformops.BuildAX206UdevRuleContent(targetUser)
}

func IsAutoStartEnabled() (bool, error) {
	return platformops.IsAutoStartEnabled()
}

func EnableAutoStart() error {
	return platformops.EnableAutoStart()
}

func DisableAutoStart() error {
	return platformops.DisableAutoStart()
}
