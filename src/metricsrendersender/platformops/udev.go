package platformops

import (
	"fmt"
	"os"
	"os/user"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
)

const (
	AX206UdevRulePath = "/etc/udev/rules.d/99-metricsrendersender.rules"
	ax206VendorID     = "1908"
	ax206ProductID    = "0102"
)

var udevUsernamePattern = regexp.MustCompile(`^[a-zA-Z0-9._-]+$`)

type UdevInstallResult struct {
	TargetUser string
	RulePath   string
}

func InstallAX206UdevRule() (UdevInstallResult, error) {
	if runtime.GOOS != "linux" {
		return UdevInstallResult{}, fmt.Errorf("--add-udev-rule only supports Linux")
	}

	rootMode, err := isRootUser()
	if err != nil {
		return UdevInstallResult{}, err
	}
	if !rootMode {
		return UdevInstallResult{}, fmt.Errorf(
			"--add-udev-rule requires sudo, please run: %s",
			BuildSudoAddUdevRuleHint(),
		)
	}

	targetUser, err := ResolveUdevRuleTargetUser()
	if err != nil {
		return UdevInstallResult{}, err
	}

	ruleContent := BuildAX206UdevRuleContent(targetUser)
	if err := writeAX206UdevRule(ruleContent); err != nil {
		return UdevInstallResult{}, err
	}

	if err := runCommand("udevadm", "control", "--reload-rules"); err != nil {
		return UdevInstallResult{}, fmt.Errorf("failed to reload udev rules: %w", err)
	}
	if err := runCommand(
		"udevadm", "trigger",
		"--subsystem-match=usb",
		"--attr-match=idVendor="+ax206VendorID,
		"--attr-match=idProduct="+ax206ProductID,
	); err != nil {
		return UdevInstallResult{}, fmt.Errorf("failed to trigger udev for AX206 device: %w", err)
	}

	return UdevInstallResult{TargetUser: targetUser, RulePath: AX206UdevRulePath}, nil
}

func BuildSudoAddUdevRuleHint() string {
	base := filepath.Base(strings.TrimSpace(os.Args[0]))
	if base == "" || base == "." || base == string(filepath.Separator) {
		base = "metricsrendersender"
	}
	return fmt.Sprintf("sudo %s --add-udev-rule", base)
}

func ResolveUdevRuleTargetUser() (string, error) {
	if sudoUser := strings.TrimSpace(os.Getenv("SUDO_USER")); sudoUser != "" && sudoUser != "root" {
		if !udevUsernamePattern.MatchString(sudoUser) {
			return "", fmt.Errorf("invalid SUDO_USER value: %q", sudoUser)
		}
		return sudoUser, nil
	}

	currentUser, err := user.Current()
	if err != nil {
		return "", fmt.Errorf("failed to detect current user: %w", err)
	}
	name := strings.TrimSpace(currentUser.Username)
	if !udevUsernamePattern.MatchString(name) {
		return "", fmt.Errorf(
			"invalid current username %q, please run with sudo and set SUDO_USER",
			name,
		)
	}
	return name, nil
}

func BuildAX206UdevRuleContent(targetUser string) string {
	return fmt.Sprintf(
		"# Added by metricsrendersender --add-udev-rule\n"+
			`SUBSYSTEM=="usb", ATTR{idVendor}=="%s", ATTR{idProduct}=="%s", OWNER="%s", MODE="0660"`+"\n",
		ax206VendorID, ax206ProductID, targetUser,
	)
}

func writeAX206UdevRule(content string) error {
	existing, err := os.ReadFile(AX206UdevRulePath)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to read existing udev rule: %w", err)
	}
	if string(existing) == content {
		return nil
	}
	if err := os.WriteFile(AX206UdevRulePath, []byte(content), 0o644); err != nil {
		return fmt.Errorf("failed to write udev rule %s: %w", AX206UdevRulePath, err)
	}
	return nil
}
