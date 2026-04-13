package runevents

import (
	"fmt"
	"io"
	"strings"
)

// HumanSink writes PRD-style lines to w (typically stderr).
type HumanSink struct {
	w io.Writer
}

// NewHumanSink builds a sink that writes [kira-run] lines to w.
func NewHumanSink(w io.Writer) *HumanSink {
	return &HumanSink{w: w}
}

// Emit implements Sink.
func (h *HumanSink) Emit(e Event) error {
	if h == nil || h.w == nil {
		return nil
	}
	line := formatHuman(e)
	if line == "" {
		return nil
	}
	_, err := io.WriteString(h.w, line)
	return err
}

func formatHuman(e Event) string {
	switch e.Event {
	case KindRunStart:
		return fmt.Sprintf("[kira-run] starting workflow=%s run-id=%s attempt=%d\n", e.Workflow, e.RunID, e.Attempt)
	case KindRunResume:
		names := strings.Join(e.CompletedSteps, ", ")
		if names == "" {
			names = "(none)"
		}
		return fmt.Sprintf("[kira-run] resuming run-id=%s attempt=%d completed=%d (%s)\n", e.RunID, e.Attempt, e.Completed, names)
	case KindStepSkip:
		return fmt.Sprintf("[kira-run] step skip name=%s reason=%s\n", e.Step, e.Reason)
	case KindStepStart:
		return fmt.Sprintf("[kira-run] step start name=%s\n", e.Step)
	case KindStepDone:
		return fmt.Sprintf("[kira-run] step done name=%s\n", e.Step)
	case KindRetry:
		return fmt.Sprintf("[kira-run] retry run-id=%s attempt=%d\n", e.RunID, e.Attempt)
	case KindRunFailed:
		return fmt.Sprintf("[kira-run] run failed run-id=%s attempt=%d error=%s\n", e.RunID, e.Attempt, e.Error)
	case KindRunCompleted:
		return fmt.Sprintf("[kira-run] run completed ok run-id=%s (session removed)\n", e.RunID)
	case KindFlagNotice:
		return "[kira-run] notice flag=ignore_attempt_limit (persisted attempt unchanged; script may bypass attempt cap)\n"
	default:
		return ""
	}
}
