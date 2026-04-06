package session

import (
	"fmt"
	"os"
	"path/filepath"
)

const (
	sessionFileMode = 0o600
	sessionDirMode  = 0o700
)

// readFile reads a session file after basic sanity checks.
func readFile(path string) ([]byte, error) {
	if !filepath.IsAbs(path) {
		return nil, fmt.Errorf("session path must be absolute: %s", path)
	}
	clean := filepath.Clean(path)
	if filepath.Base(clean) == "." || filepath.Base(clean) == ".." {
		return nil, fmt.Errorf("invalid session path: %s", path)
	}
	// #nosec G304 — path validated as absolute clean session file under expected dirs by caller
	return os.ReadFile(clean)
}

// Save writes session atomically: temp file in same directory then rename.
func Save(projectRootAbs string, s *Session) error {
	if s == nil {
		return fmt.Errorf("session is nil")
	}
	path, err := FilePath(projectRootAbs, s.RunID)
	if err != nil {
		return err
	}
	if err := s.Validate(path); err != nil {
		return fmt.Errorf("validate before save: %w", err)
	}
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, sessionDirMode); err != nil {
		return fmt.Errorf("create sessions directory %s: %w", dir, err)
	}
	data, err := s.Marshal()
	if err != nil {
		return err
	}
	tmp, err := os.CreateTemp(dir, ".session-*.tmp")
	if err != nil {
		return fmt.Errorf("create temp session file: %w", err)
	}
	tmpPath := tmp.Name()
	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		_ = os.Remove(tmpPath)
		return fmt.Errorf("write temp session: %w", err)
	}
	if err := tmp.Close(); err != nil {
		_ = os.Remove(tmpPath)
		return fmt.Errorf("close temp session: %w", err)
	}
	if err := os.Chmod(tmpPath, sessionFileMode); err != nil {
		_ = os.Remove(tmpPath)
		return fmt.Errorf("chmod temp session: %w", err)
	}
	if err := os.Rename(tmpPath, path); err != nil {
		_ = os.Remove(tmpPath)
		return fmt.Errorf("replace session file %s: %w", path, err)
	}
	return nil
}

// Remove deletes the session file if it exists.
func Remove(sessionFileAbs string) error {
	if !filepath.IsAbs(sessionFileAbs) {
		return fmt.Errorf("session path must be absolute: %s", sessionFileAbs)
	}
	err := os.Remove(sessionFileAbs)
	if err != nil && os.IsNotExist(err) {
		return nil
	}
	return err
}
