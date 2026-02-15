// Package commands provides optional ANSI color helpers for slice command output.
// Color is disabled when NO_COLOR is set (https://no-color.org) or stdout is not a TTY.
package commands

import (
	"os"
	"strings"
)

// ANSI escape codes (no dependency; safe for terminal output).
const (
	ansiReset  = "\033[0m"
	ansiBold   = "\033[1m"
	ansiDim    = "\033[2m"
	ansiCyan   = "\033[36m"
	ansiGreen  = "\033[32m"
	ansiYellow = "\033[33m"
	ansiRed    = "\033[31m"
)

// sliceColorEnabled returns true when color output is allowed:
// stdout is a TTY and NO_COLOR is not set.
func sliceColorEnabled() bool {
	if strings.TrimSpace(os.Getenv("NO_COLOR")) != "" {
		return false
	}
	fi, err := os.Stdout.Stat()
	if err != nil {
		return false
	}
	return (fi.Mode() & os.ModeCharDevice) != 0
}

// sliceNameStyle returns the slice name with optional cyan bold styling.
func sliceNameStyle(s string) string {
	if !sliceColorEnabled() {
		return s
	}
	return ansiCyan + ansiBold + s + ansiReset
}

// taskIDStyle returns the task ID (e.g. T001) with optional yellow styling.
func taskIDStyle(s string) string {
	if !sliceColorEnabled() {
		return s
	}
	return ansiYellow + s + ansiReset
}

// taskBoxStyle returns [x] or [ ] with optional green/dim styling.
func taskBoxStyle(done bool) string {
	if done {
		if sliceColorEnabled() {
			return ansiGreen + "[x]" + ansiReset
		}
		return "[x]"
	}
	if sliceColorEnabled() {
		return ansiDim + "[ ]" + ansiReset
	}
	return "[ ]"
}

// labelStyle returns a label (e.g. "Slice:", "Task:") with optional dim styling.
func labelStyle(s string) string {
	if !sliceColorEnabled() {
		return s
	}
	return ansiDim + s + ansiReset
}

// successStyle returns text (e.g. "done", "valid") with optional green styling.
func successStyle(s string) string {
	if !sliceColorEnabled() {
		return s
	}
	return ansiGreen + s + ansiReset
}

// errorStyle returns text with optional red styling (e.g. lint rule names).
func errorStyle(s string) string {
	if !sliceColorEnabled() {
		return s
	}
	return ansiRed + s + ansiReset
}

// checkNameStyle returns the check name with optional cyan bold styling.
func checkNameStyle(s string) string {
	if !sliceColorEnabled() {
		return s
	}
	return ansiCyan + ansiBold + s + ansiReset
}

// checkDescStyle returns the check description with optional dim styling.
func checkDescStyle(s string) string {
	if !sliceColorEnabled() {
		return s
	}
	return ansiDim + s + ansiReset
}

// pathStyle returns a file path with optional cyan styling.
func pathStyle(s string) string {
	if !sliceColorEnabled() {
		return s
	}
	return ansiCyan + s + ansiReset
}

// itemNameStyle returns an item name (skill/command) with optional bold styling.
func itemNameStyle(s string) string {
	if !sliceColorEnabled() {
		return s
	}
	return ansiBold + s + ansiReset
}

// warningStyle returns warning text with optional yellow styling.
func warningStyle(s string) string {
	if !sliceColorEnabled() {
		return s
	}
	return ansiYellow + s + ansiReset
}

// promptStyle returns prompt text with optional dim styling.
func promptStyle(s string) string {
	if !sliceColorEnabled() {
		return s
	}
	return ansiDim + s + ansiReset
}
