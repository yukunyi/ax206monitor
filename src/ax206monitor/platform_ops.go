package main

import "ax206monitor/platformops"

type ServiceInstallOptions = platformops.ServiceInstallOptions

func InstallService(options ServiceInstallOptions) error {
	result, err := platformops.InstallService(options)
	if err != nil {
		return err
	}
	if result.UserMode {
		logInfo("Installed user service: %s", result.ServicePath)
		logInfo("Check status: systemctl --user status %s", "ax206monitor")
	} else {
		logInfo("Installed system service: %s", result.ServicePath)
		logInfo("Check status: systemctl status %s", "ax206monitor")
	}
	return nil
}

func UninstallService() error {
	if err := platformops.UninstallService(); err != nil {
		return err
	}
	logInfo("Service uninstalled: %s", "ax206monitor")
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
