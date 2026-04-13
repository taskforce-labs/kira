package kirarun

// StepProgress carries optional callbacks for runner-owned progress (stderr / JSONL).
// Workflows do not receive this type from the interpreter surface.
type StepProgress struct {
	StepSkipped func(name, reason string)
	StepStarted func(name string)
	StepDone    func(name string)
}

// StepOption configures NewStep.
type StepOption func(*Step)

// WithStepProgress registers progress callbacks.
func WithStepProgress(p *StepProgress) StepOption {
	return func(s *Step) {
		s.progress = p
	}
}
