package session

import (
	"fmt"
	"path/filepath"
	"strings"
)

const (
	// DefaultWorkflowsDir is the default workflows root relative to project root.
	DefaultWorkflowsDir = ".workflows"
	// SessionsDirName is the directory under workflows root holding session files.
	SessionsDirName = "sessions"
)

// SessionsDir returns absolute path to .workflows/sessions given project root (absolute).
func SessionsDir(projectRootAbs string) (string, error) {
	if strings.TrimSpace(projectRootAbs) == "" {
		return "", fmt.Errorf("project root is empty")
	}
	if !filepath.IsAbs(projectRootAbs) {
		return "", fmt.Errorf("project root must be absolute: %s", projectRootAbs)
	}
	root := filepath.Clean(projectRootAbs)
	return filepath.Join(root, DefaultWorkflowsDir, SessionsDirName), nil
}

// FilePath returns absolute path to .workflows/sessions/<run-id>.yml.
func FilePath(projectRootAbs, runID string) (string, error) {
	if err := ValidateRunID(runID); err != nil {
		return "", fmt.Errorf("run-id: %w", err)
	}
	dir, err := SessionsDir(projectRootAbs)
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, runID+".yml"), nil
}

// LockFilePath returns absolute path to the lock file adjacent to the session file.
func LockFilePath(sessionFileAbs string) (string, error) {
	if !filepath.IsAbs(sessionFileAbs) {
		return "", fmt.Errorf("session path must be absolute: %s", sessionFileAbs)
	}
	if filepath.Ext(sessionFileAbs) != ".yml" && filepath.Ext(sessionFileAbs) != ".yaml" {
		return "", fmt.Errorf("session file must end with .yml or .yaml: %s", sessionFileAbs)
	}
	return strings.TrimSuffix(sessionFileAbs, filepath.Ext(sessionFileAbs)) + ".lock", nil
}
