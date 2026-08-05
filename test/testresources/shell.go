package testresources

import (
	"os/exec"
	"strings"
)

// shellOutput runs a shell command and returns trimmed stdout, empty string on error.
func shellOutput(cmd string) string {
	out, err := exec.Command("sh", "-c", cmd).Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}
