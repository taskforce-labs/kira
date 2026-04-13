package runevents

import "time"

// Event is the canonical payload for human and JSONL renderers.
type Event struct {
	SchemaVersion  int      `json:"schema_version"`
	Event          string   `json:"event"`
	Ts             string   `json:"ts"`
	Source         string   `json:"source"`
	RunID          string   `json:"run_id,omitempty"`
	Attempt        int      `json:"attempt,omitempty"`
	Workflow       string   `json:"workflow,omitempty"`
	Step           string   `json:"step,omitempty"`
	Reason         string   `json:"reason,omitempty"`
	Flag           string   `json:"flag,omitempty"`
	Error          string   `json:"error,omitempty"`
	Completed      int      `json:"completed,omitempty"`
	CompletedSteps []string `json:"completed_steps,omitempty"`
}

func (e Event) withDefaults() Event {
	if e.SchemaVersion == 0 {
		e.SchemaVersion = SchemaVersion
	}
	if e.Source == "" {
		e.Source = SourceRunner
	}
	if e.Ts == "" {
		e.Ts = time.Now().UTC().Format(time.RFC3339)
	}
	return e
}
