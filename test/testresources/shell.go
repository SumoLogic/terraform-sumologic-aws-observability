package testresources

import (
	"os/exec"
	"strings"
)

// shellOutput runs a shell command and returns trimmed stdout, empty string on error.
func shellOutput(cmd string) string {
	out, err := exec.Command("sh", "-c", cmd).CombinedOutput()
	if err != nil {
		return strings.TrimSpace(string(out))
	}
	return strings.TrimSpace(string(out))
}

// ShellOutput is the exported variant of shellOutput for use by test helpers.
func ShellOutput(cmd string) string { return shellOutput(cmd) }
