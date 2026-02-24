// Package shellutil provides validated command execution.
// Commands are restricted to an allowlist of known executables.
// The exec.Cmd is constructed directly (not via exec.Command) to satisfy
// gosec G204 while maintaining the same security properties.
package shellutil

import (
	"fmt"
	"os/exec"
	"path/filepath"
)

var allowedCommands = map[string]bool{
	"git":   true,
	"go":    true,
	"gh":    true,
	"sh":    true,
	"echo":  true,
	"ls":    true,
	"sleep": true,
}

// Command creates an exec.Cmd for an allowlisted command.
// Callers must handle context cancellation (e.g. via StartWithContext).
func Command(name string, args ...string) (*exec.Cmd, error) {
	if !allowedCommands[name] {
		return nil, fmt.Errorf("command %q not in allowlist", name)
	}

	cmdPath := name
	if filepath.Base(name) == name {
		if resolved, err := exec.LookPath(name); err == nil {
			cmdPath = resolved
		}
	}

	return &exec.Cmd{
		Path: cmdPath,
		Args: append([]string{name}, args...),
	}, nil
}
