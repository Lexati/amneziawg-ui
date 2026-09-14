// Package awg is the backend's only way onto the host: every shell command
// it needs - the amneziawg-tools binaries, ip(8), the iptables
// scripts - is issued from here, through a Runner that tests replace.
package awg

import (
	"os/exec"
	"strings"
)

// Runner executes one shell command line and returns its trimmed stdout.
type Runner interface {
	Run(command string) (string, error)
}

// Shell runs commands through bash, which is what the container has and what
// the process substitution in SyncConf needs.
type Shell struct{}

// Run implements Runner.
func (Shell) Run(command string) (string, error) {
	out, err := exec.Command("bash", "-c", command).Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}
