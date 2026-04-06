package kirarun

// RunHandle exposes read-only facts about the current workflow invocation.
type RunHandle struct {
	attempt       int
	resume        bool
	ignoreAttempt bool
}

// Attempt returns the 1-based attempt index for this process invocation.
func (r RunHandle) Attempt() int {
	return r.attempt
}

// IsResume is true when the run was started with --resume.
func (r RunHandle) IsResume() bool {
	return r.resume
}

// IgnoreAttemptLimit is true when --ignore-attempt-limit was set for this invocation only.
func (r RunHandle) IgnoreAttemptLimit() bool {
	return r.ignoreAttempt
}

// NewRunHandle constructs a RunHandle for tests and the host runner.
func NewRunHandle(attempt int, resume, ignoreAttemptLimit bool) RunHandle {
	if attempt < 1 {
		attempt = 1
	}
	return RunHandle{
		attempt:       attempt,
		resume:        resume,
		ignoreAttempt: ignoreAttemptLimit,
	}
}
