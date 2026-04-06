package session

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
)

var runIDPattern = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9._-]*$`)

// ValidateRunID ensures run-id is safe to use as a single path segment (session filename stem).
func ValidateRunID(id string) error {
	if id == "" {
		return fmt.Errorf("run-id is empty")
	}
	if len(id) > 240 {
		return fmt.Errorf("run-id is too long")
	}
	if strings.Contains(id, string(filepath.Separator)) || strings.ContainsRune(id, '/') || strings.ContainsRune(id, '\\') {
		return fmt.Errorf("run-id must not contain path separators")
	}
	if strings.Contains(id, "..") {
		return fmt.Errorf("run-id must not contain '..'")
	}
	if !runIDPattern.MatchString(id) {
		return fmt.Errorf("run-id has invalid characters (use letters, digits, '.', '_', '-', starting with alphanumeric)")
	}
	return nil
}
