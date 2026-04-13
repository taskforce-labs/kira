package runevents

import (
	"fmt"
	"io"
	"strings"
)

const (
	humanPrefix = "[kira-run]  "
	// humanKindW is the fixed column width for the event kind (matches JSON "event" strings).
	humanKindW = 14
)

// HumanSink writes aligned, scannable lines to w (typically stderr).
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

func padKind(kind string) string {
	if len(kind) >= humanKindW {
		return kind
	}
	return kind + strings.Repeat(" ", humanKindW-len(kind))
}

func formatHuman(e Event) string {
	kind := padKind(e.Event)
	switch e.Event {
	case KindRunStart:
		return fmt.Sprintf("%s%sworkflow=%s  run-id=%s  attempt=%d\n",
			humanPrefix, kind, e.Workflow, e.RunID, e.Attempt)
	case KindRunResume:
		names := strings.Join(e.CompletedSteps, ", ")
		if names == "" {
			names = "(none)"
		}
		return fmt.Sprintf("%s%srun-id=%s  attempt=%d  completed=%d  steps=%s\n",
			humanPrefix, kind, e.RunID, e.Attempt, e.Completed, names)
	case KindStepSkip:
		return fmt.Sprintf("%s%sstep=%s  reason=%s\n", humanPrefix, kind, e.Step, e.Reason)
	case KindStepStart:
		return fmt.Sprintf("%s%sstep=%s\n", humanPrefix, kind, e.Step)
	case KindStepDone:
		return fmt.Sprintf("%s%sstep=%s\n", humanPrefix, kind, e.Step)
	case KindRetry:
		return fmt.Sprintf("%s%srun-id=%s  attempt=%d\n", humanPrefix, kind, e.RunID, e.Attempt)
	case KindRunFailed:
		return fmt.Sprintf("%s%srun-id=%s  attempt=%d  error=%s\n",
			humanPrefix, kind, e.RunID, e.Attempt, e.Error)
	case KindRunCompleted:
		return fmt.Sprintf("%s%srun-id=%s  session=removed\n", humanPrefix, kind, e.RunID)
	case KindFlagNotice:
		return fmt.Sprintf("%s%sflag=ignore_attempt_limit  detail=persisted_attempt_unchanged_bypass_cap_allowed\n",
			humanPrefix, kind)
	default:
		return ""
	}
}
