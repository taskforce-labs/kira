// Package steppersist wires session storage to kirarun.StepPersister.
package steppersist

import (
	"fmt"
	"time"

	"kira/internal/kirarun/session"
	"kira/kirarun"
)

// Session wraps a session.Session for kirarun.StepPersister.
type Session struct {
	S *session.Session
}

// GetStepData implements kirarun.StepPersister.
func (p *Session) GetStepData(name string) (map[string]any, bool) {
	if p == nil || p.S == nil {
		return nil, false
	}
	for _, st := range p.S.Steps {
		if st.Name == name {
			if st.Data == nil {
				return map[string]any{}, true
			}
			return cloneMap(st.Data), true
		}
	}
	return nil, false
}

// PutStep implements kirarun.StepPersister.
func (p *Session) PutStep(name string, attempt int, startedAt, finishedAt time.Time, data map[string]any) error {
	if p == nil || p.S == nil {
		return fmt.Errorf("session is nil")
	}
	if name == "" {
		return fmt.Errorf("step name is empty")
	}
	rec := session.StepRecord{
		Name:       name,
		Attempt:    attempt,
		StartedAt:  startedAt.UTC().Format(time.RFC3339Nano),
		FinishedAt: finishedAt.UTC().Format(time.RFC3339Nano),
		Data:       cloneMap(data),
	}
	out := make([]session.StepRecord, 0, len(p.S.Steps)+1)
	replaced := false
	for _, st := range p.S.Steps {
		if st.Name == name {
			out = append(out, rec)
			replaced = true
			continue
		}
		out = append(out, st)
	}
	if !replaced {
		out = append(out, rec)
	}
	p.S.Steps = out
	return nil
}

func cloneMap(m map[string]any) map[string]any {
	if m == nil {
		return nil
	}
	out := make(map[string]any, len(m))
	for k, v := range m {
		out[k] = v
	}
	return out
}

// NewSessionAdapter returns a kirarun.StepPersister backed by the session model.
func NewSessionAdapter(s *session.Session) kirarun.StepPersister {
	return &Session{S: s}
}
