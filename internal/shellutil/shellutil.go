// Package shellutil provides validated command execution.
// Commands are restricted to a minimum allowlist of known executables (git, sh, echo, ls, sleep).
// CommandContext is the single entry point; it checks the allowlist then runs exec.CommandContext.
package shellutil

import (
	"context"
	"fmt"
	"os/exec"
)

// allowedCommands is the minimum set of commands that may be run via CommandContext.
// Only executables actually used by kira via executeCommand/newCommand are listed.
var allowedCommands = map[string]bool{
	"git":   true,
	"sh":    true,
	"echo":  true,
	"ls":    true,
	"sleep": true,
}

// CommandContext creates an exec.Cmd for an allowlisted command with context cancellation support.
// Returns an error if name is not in the allowlist. See .docs/guides/security/golang-secure-coding.md
// for the approved G204 exception and allowlist policy.
func CommandContext(ctx context.Context, name string, args ...string) (*exec.Cmd, error) {
	if !allowedCommands[name] {
		return nil, fmt.Errorf("command %q not in allowlist", name)
	}
	// #nosec G204 -- Centralized exec: name/args are from internal callers only; allowlist above restricts to git, sh, echo, ls, sleep. See .docs/guides/security/golang-secure-coding.md § Approved #nosec exceptions.
	return exec.CommandContext(ctx, name, args...), nil
}
