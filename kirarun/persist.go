package kirarun

import "time"

// StepPersister stores successful step.Do results for resume.
type StepPersister interface {
	// GetStepData returns JSON-compatible data for a completed step, or false if none.
	GetStepData(name string) (data map[string]any, ok bool)
	// PutStep persists a successful step completion.
	PutStep(name string, attempt int, startedAt, finishedAt time.Time, data map[string]any) error
}
