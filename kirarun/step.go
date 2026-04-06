package kirarun

import (
	"encoding/json"
	"fmt"
	"time"
)

// StepContext is passed to step.Do callbacks (reserved for future use).
type StepContext struct{}

// Step implements idempotent step.Do for a workflow run.
type Step struct {
	persister StepPersister
	run       RunHandle
	ctx       StepContext
}

// NewStep builds a Step bound to persistence and the current attempt counter.
func NewStep(persister StepPersister, run RunHandle) *Step {
	return &Step{persister: persister, run: run}
}

// Do runs fn once per successful completion for name, persisting JSON object output.
// Go does not support generic methods; use kirarun.Do(step, name, fn) instead of a method on Step.
func Do[T any](s *Step, name string, fn func(StepContext) (T, error)) (T, error) {
	var zero T
	if s == nil {
		return zero, fmt.Errorf("step: nil Step")
	}
	if name == "" {
		return zero, fmt.Errorf("Do: name is required")
	}
	if s.persister == nil {
		return zero, fmt.Errorf("Do(%q): no persister configured", name)
	}
	if raw, ok := s.persister.GetStepData(name); ok {
		out, err := decodeStepData[T](name, raw)
		if err != nil {
			return zero, err
		}
		return out, nil
	}

	started := time.Now().UTC()
	out, err := fn(s.ctx)
	if err != nil {
		return zero, err
	}
	finished := time.Now().UTC()
	data, err := toMap(out)
	if err != nil {
		return zero, fmt.Errorf("Do(%q): %w", name, err)
	}
	if err := s.persister.PutStep(name, s.run.Attempt(), started, finished, data); err != nil {
		return zero, fmt.Errorf("Do(%q): persist: %w", name, err)
	}
	return out, nil
}

func decodeStepData[T any](name string, raw map[string]any) (T, error) {
	var zero T
	b, err := json.Marshal(raw)
	if err != nil {
		return zero, fmt.Errorf("step %q: encode stored data: %w", name, err)
	}
	var out T
	if err := json.Unmarshal(b, &out); err != nil {
		return zero, fmt.Errorf("step %q: stored data does not match expected type on resume: %w", name, err)
	}
	return out, nil
}

func toMap(v any) (map[string]any, error) {
	b, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}
	var m map[string]any
	if err := json.Unmarshal(b, &m); err != nil {
		return nil, fmt.Errorf("step data must JSON-serialize to an object (use a struct or map[string]any): %w", err)
	}
	return m, nil
}
