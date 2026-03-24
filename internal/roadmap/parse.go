package roadmap

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	yaml "gopkg.in/yaml.v3"
)

// LoadFile reads and parses a roadmap YAML file. Path must be under baseDir (e.g. project root).
func LoadFile(baseDir, path string) (*File, error) {
	if err := validateRoadmapPath(baseDir, path); err != nil {
		return nil, err
	}
	// #nosec G304 - path has been validated by validateRoadmapPath
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read roadmap file: %w", err)
	}
	return Parse(data)
}

// ValidateRoadmapPath returns an error if path is not under baseDir (safe for file operations).
func ValidateRoadmapPath(baseDir, path string) error {
	return validateRoadmapPath(baseDir, path)
}

func validateRoadmapPath(baseDir, path string) error {
	cleanPath := filepath.Clean(path)
	if strings.Contains(cleanPath, "..") {
		return fmt.Errorf("invalid roadmap path: %s", path)
	}
	absPath, err := filepath.Abs(cleanPath)
	if err != nil {
		return fmt.Errorf("invalid path: %w", err)
	}
	absBase, err := filepath.Abs(filepath.Clean(baseDir))
	if err != nil {
		return fmt.Errorf("base directory: %w", err)
	}
	baseWithSep := absBase + string(filepath.Separator)
	if absPath != absBase && !strings.HasPrefix(absPath+string(filepath.Separator), baseWithSep) {
		return fmt.Errorf("path outside project directory: %s", path)
	}
	return nil
}

// Parse parses roadmap YAML bytes into a File.
func Parse(data []byte) (*File, error) {
	var f File
	if err := yaml.Unmarshal(data, &f); err != nil {
		return nil, fmt.Errorf("parse roadmap YAML: %w", err)
	}
	return &f, nil
}

// Serialize serializes a File to YAML bytes (canonical group form).
func Serialize(f *File) ([]byte, error) {
	data, err := yaml.Marshal(f)
	if err != nil {
		return nil, fmt.Errorf("serialize roadmap: %w", err)
	}
	return data, nil
}

// WriteBytes writes data to path with permission 0o600 after validating path is under baseDir.
func WriteBytes(baseDir, path string, data []byte) error {
	if err := validateRoadmapPath(baseDir, path); err != nil {
		return err
	}
	if err := os.WriteFile(path, data, 0o600); err != nil {
		return fmt.Errorf("write roadmap file: %w", err)
	}
	return nil
}

// SaveFile writes a File to a path. Path must be under baseDir.
func SaveFile(baseDir, path string, f *File) error {
	data, err := Serialize(f)
	if err != nil {
		return err
	}
	return WriteBytes(baseDir, path, data)
}
