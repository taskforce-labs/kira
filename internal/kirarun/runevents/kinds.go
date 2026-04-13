// Package runevents implements versioned runner progress events for kira run.
package runevents

// SchemaVersion is bumped when JSON field names or event kinds change incompatibly.
const SchemaVersion = 1

// Event kinds (normative JSON "event" values).
const (
	KindRunStart     = "run_start"
	KindRunResume    = "run_resume"
	KindStepSkip     = "step_skip"
	KindStepStart    = "step_start"
	KindStepDone     = "step_done"
	KindRetry        = "retry"
	KindRunFailed    = "run_failed"
	KindRunCompleted = "run_completed"
	KindFlagNotice   = "flag_notice"
)

// SourceRunner is the JSON "source" field for events emitted by the kira run host.
const SourceRunner = "runner"
