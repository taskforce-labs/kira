// Package session persists kira run workflow session state under .workflows/sessions/.
package session

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	yaml "gopkg.in/yaml.v3"
)

// SchemaVersion is the session file schema version written under schema_version.
const SchemaVersion = 1

// Session is the on-disk YAML model for a single workflow run.
type Session struct {
	SchemaVersion int `yaml:"schema_version,omitempty"`

	Path        string `yaml:"path"`
	KiraVersion string `yaml:"kira-version"`
	RunID       string `yaml:"run-id"`
	Attempt     int    `yaml:"attempt"`

	Attempts []AttemptRecord `yaml:"attempts,omitempty"`
	Steps    []StepRecord    `yaml:"steps,omitempty"`
}

// AttemptRecord records a failed run or step attempt.
type AttemptRecord struct {
	Attempt   int            `yaml:"attempt"`
	Name      string         `yaml:"name"`
	StartedAt string         `yaml:"started_at"`
	FailedAt  string         `yaml:"failed_at"`
	Error     map[string]any `yaml:"error,omitempty"`
}

// StepRecord stores successful step.Do output (JSON-compatible under data).
type StepRecord struct {
	Name       string         `yaml:"name"`
	Attempt    int            `yaml:"attempt"`
	StartedAt  string         `yaml:"started_at"`
	FinishedAt string         `yaml:"finished_at"`
	Data       map[string]any `yaml:"data,omitempty"`
}

// Validate checks required fields and types after YAML decode.
func (s *Session) Validate(sessionFileAbs string) error {
	if s == nil {
		return fmt.Errorf("session is nil")
	}
	stem := strings.TrimSuffix(filepath.Base(sessionFileAbs), filepath.Ext(sessionFileAbs))
	if s.RunID == "" {
		return fmt.Errorf("missing required field: run-id")
	}
	if err := ValidateRunID(s.RunID); err != nil {
		return fmt.Errorf("run-id: %w", err)
	}
	if s.RunID != stem {
		return fmt.Errorf("run-id %q must match session filename stem %q", s.RunID, stem)
	}
	if s.SchemaVersion != 0 && s.SchemaVersion != SchemaVersion {
		return fmt.Errorf("unsupported schema_version %d (expected %d)", s.SchemaVersion, SchemaVersion)
	}
	if s.Path == "" {
		return fmt.Errorf("missing required field: path")
	}
	if !filepath.IsAbs(s.Path) {
		return fmt.Errorf("path must be absolute: %q", s.Path)
	}
	if s.KiraVersion == "" {
		return fmt.Errorf("missing required field: kira-version")
	}
	if s.Attempt < 1 {
		return fmt.Errorf("attempt must be >= 1, got %d", s.Attempt)
	}
	for i, a := range s.Attempts {
		if err := a.validate(); err != nil {
			return fmt.Errorf("attempts[%d]: %w", i, err)
		}
	}
	for i, st := range s.Steps {
		if err := st.validate(); err != nil {
			return fmt.Errorf("steps[%d]: %w", i, err)
		}
	}
	return nil
}

func (a AttemptRecord) validate() error {
	if a.Attempt < 1 {
		return fmt.Errorf("attempt must be >= 1, got %d", a.Attempt)
	}
	if strings.TrimSpace(a.Name) == "" {
		return fmt.Errorf("name is required")
	}
	if _, err := parseRFC3339(a.StartedAt); err != nil {
		return fmt.Errorf("started_at: %w", err)
	}
	if _, err := parseRFC3339(a.FailedAt); err != nil {
		return fmt.Errorf("failed_at: %w", err)
	}
	return nil
}

func (st StepRecord) validate() error {
	if strings.TrimSpace(st.Name) == "" {
		return fmt.Errorf("name is required")
	}
	if st.Attempt < 1 {
		return fmt.Errorf("attempt must be >= 1, got %d", st.Attempt)
	}
	if _, err := parseRFC3339(st.StartedAt); err != nil {
		return fmt.Errorf("started_at: %w", err)
	}
	if _, err := parseRFC3339(st.FinishedAt); err != nil {
		return fmt.Errorf("finished_at: %w", err)
	}
	return nil
}

func parseRFC3339(s string) (time.Time, error) {
	if strings.TrimSpace(s) == "" {
		return time.Time{}, fmt.Errorf("timestamp is empty")
	}
	t, err := time.Parse(time.RFC3339Nano, s)
	if err != nil {
		t2, err2 := time.Parse(time.RFC3339, s)
		if err2 != nil {
			return time.Time{}, fmt.Errorf("expected RFC3339 or RFC3339Nano: %w", err)
		}
		return t2, nil
	}
	return t, nil
}

// Load reads and validates a session file. path must be absolute.
func Load(path string) (*Session, error) {
	if !filepath.IsAbs(path) {
		return nil, fmt.Errorf("session path must be absolute: %s", path)
	}
	data, err := readFile(path)
	if err != nil {
		return nil, err
	}
	var s Session
	if err := yaml.Unmarshal(data, &s); err != nil {
		return nil, fmt.Errorf("%s: invalid YAML: %w", path, err)
	}
	if err := s.Validate(path); err != nil {
		return nil, fmt.Errorf("%s: invalid session: %w", path, err)
	}
	return &s, nil
}

// Marshal writes the session to YAML bytes.
func (s *Session) Marshal() ([]byte, error) {
	if s == nil {
		return nil, fmt.Errorf("session is nil")
	}
	if s.SchemaVersion == 0 {
		s.SchemaVersion = SchemaVersion
	}
	out, err := yaml.Marshal(s)
	if err != nil {
		return nil, fmt.Errorf("marshal session: %w", err)
	}
	return out, nil
}

// StepDataJSON returns step data as JSON bytes for generic JSON encoding of step outputs.
func StepDataJSON(data map[string]any) ([]byte, error) {
	if data == nil {
		return []byte("null"), nil
	}
	b, err := json.Marshal(data)
	if err != nil {
		return nil, fmt.Errorf("encode step data: %w", err)
	}
	return b, nil
}
