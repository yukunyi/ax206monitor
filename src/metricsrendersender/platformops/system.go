package platformops

import (
	"fmt"
	"os/exec"
	"os/user"
	"strings"
)

func runCommand(name string, args ...string) error {
	command := exec.Command(name, args...)
	output, err := command.CombinedOutput()
	if err != nil {
		text := strings.TrimSpace(string(output))
		if text == "" {
			return fmt.Errorf("%s failed: %w", strings.Join(append([]string{name}, args...), " "), err)
		}
		return fmt.Errorf("%s failed: %w (%s)", strings.Join(append([]string{name}, args...), " "), err, text)
	}
	return nil
}

func isRootUser() (bool, error) {
	currentUser, err := user.Current()
	if err != nil {
		return false, fmt.Errorf("failed to detect current user: %w", err)
	}
	return currentUser.Uid == "0", nil
}
