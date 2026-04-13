package runner

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"kira/internal/kirarun/runevents"
	"kira/internal/kirarun/session"
	"kira/internal/kirarun/steppersist"
	"kira/internal/kirarun/yaegi"
	"kira/kirarun"

	"github.com/traefik/yaegi/interp"
)

// Config drives one runner invocation.
type Config struct {
	ProjectRoot        string
	WorkflowPath       string
	KiraVersion        string
	ResumeID           string
	AutoRetry          bool
	IgnoreAttemptLimit bool
	// ScriptDisplayName overrides the basename used in DeriveRunID (optional).
	ScriptDisplayName string
	Stdout            io.Writer
	Stderr            io.Writer
	// InterpArgs is passed to the Yaegi interpreter as os.Args for the workflow.
	InterpArgs []string
}

func (c *Config) stdout() io.Writer {
	if c != nil && c.Stdout != nil {
		return c.Stdout
	}
	return os.Stdout
}

func (c *Config) stderr() io.Writer {
	if c != nil && c.Stderr != nil {
		return c.Stderr
	}
	return os.Stderr
}

func (c *Config) workflowLabel(wfAbs string) string {
	if c != nil && c.ScriptDisplayName != "" {
		return c.ScriptDisplayName
	}
	return wfAbs
}

// Execute loads the workflow, manages the session file, and invokes Run until success, fatal error, or ctx done.
// For a new run it prints the derived run id to stdout and returns it.
func (c *Config) Execute(ctx context.Context) (runID string, err error) {
	if c == nil {
		return "", fmt.Errorf("nil runner config")
	}
	wfAbs, err := filepath.Abs(c.WorkflowPath)
	if err != nil {
		return "", fmt.Errorf("workflow path: %w", err)
	}
	root, err := filepath.Abs(c.ProjectRoot)
	if err != nil {
		return "", fmt.Errorf("project root: %w", err)
	}

	i, err := yaegi.LoadWorkflow(wfAbs, c.InterpArgs)
	if err != nil {
		return "", err
	}

	runID, err = c.resolveRunID(wfAbs)
	if err != nil {
		return "", err
	}

	sessPath, err := session.FilePath(root, runID)
	if err != nil {
		return "", err
	}
	lockPath, err := session.LockFilePath(sessPath)
	if err != nil {
		return "", err
	}

	lk, err := session.TryLock(lockPath)
	if err != nil {
		return "", err
	}
	defer func() {
		_ = lk.Unlock()
		_ = os.Remove(lockPath)
	}()

	cliResume := c.ResumeID != ""
	if !cliResume {
		if _, statErr := os.Stat(sessPath); statErr == nil {
			return "", fmt.Errorf("session file already exists for run id %q; use --resume %s", runID, runID)
		}
		_, _ = fmt.Fprintf(c.stdout(), "run id: %s\n", runID)
	}

	sess, err := c.loadSession(cliResume, sessPath, wfAbs, runID)
	if err != nil {
		return runID, err
	}

	bus := runevents.NewBus()
	bus.AddSink(runevents.NewHumanSink(c.stderr()))

	return c.runLoop(ctx, i, root, sessPath, sess, cliResume, bus, c.workflowLabel(wfAbs))
}

func (c *Config) loadSession(cliResume bool, sessPath, wfAbs, runID string) (*session.Session, error) {
	if cliResume {
		sess, err := session.Load(sessPath)
		if err != nil {
			return nil, err
		}
		if sess.Path != wfAbs {
			return nil, fmt.Errorf("session %q is for workflow %q, not %q", runID, sess.Path, wfAbs)
		}
		return sess, nil
	}
	return &session.Session{
		Path:        wfAbs,
		KiraVersion: c.KiraVersion,
		RunID:       runID,
		Attempt:     1,
	}, nil
}

func (c *Config) runLoop(
	ctx context.Context,
	i *interp.Interpreter,
	root, sessPath string,
	sess *session.Session,
	cliResume bool,
	bus *runevents.Bus,
	workflow string,
) (string, error) {
	runID := sess.RunID
	loop := 0
	for {
		loop++
		select {
		case <-ctx.Done():
			bus.Emit(runevents.Event{
				Event:   runevents.KindRunFailed,
				RunID:   runID,
				Attempt: sess.Attempt,
				Error:   ctx.Err().Error(),
			})
			return runID, ctx.Err()
		default:
		}

		if err := session.Save(root, sess); err != nil {
			return runID, err
		}

		if c.IgnoreAttemptLimit && loop == 1 {
			bus.Emit(runevents.Event{Event: runevents.KindFlagNotice, Flag: "ignore_attempt_limit"})
		}
		if loop == 1 {
			if cliResume {
				n, names := completedStepSummary(sess.Steps)
				bus.Emit(runevents.Event{
					Event:          runevents.KindRunResume,
					RunID:          runID,
					Attempt:        sess.Attempt,
					Workflow:       workflow,
					Completed:      n,
					CompletedSteps: names,
				})
			} else {
				bus.Emit(runevents.Event{
					Event:    runevents.KindRunStart,
					RunID:    runID,
					Attempt:  sess.Attempt,
					Workflow: workflow,
				})
			}
		} else if c.AutoRetry {
			bus.Emit(runevents.Event{
				Event:   runevents.KindRetry,
				RunID:   runID,
				Attempt: sess.Attempt,
			})
		}

		runHandle := kirarun.NewRunHandle(sess.Attempt, cliResume, c.IgnoreAttemptLimit)
		pers := steppersist.NewSessionAdapter(sess)
		prog := &kirarun.StepProgress{
			StepSkipped: func(name, reason string) {
				bus.Emit(runevents.Event{
					Event:    runevents.KindStepSkip,
					RunID:    runID,
					Attempt:  sess.Attempt,
					Workflow: workflow,
					Step:     name,
					Reason:   reason,
				})
			},
			StepStarted: func(name string) {
				bus.Emit(runevents.Event{
					Event:    runevents.KindStepStart,
					RunID:    runID,
					Attempt:  sess.Attempt,
					Workflow: workflow,
					Step:     name,
				})
			},
			StepDone: func(name string) {
				bus.Emit(runevents.Event{
					Event:    runevents.KindStepDone,
					RunID:    runID,
					Attempt:  sess.Attempt,
					Workflow: workflow,
					Step:     name,
				})
			},
		}
		step := kirarun.NewStep(pers, runHandle, kirarun.WithStepProgress(prog))
		kctx := &kirarun.Context{
			Workspace: kirarun.NewWorkspace(root),
			Log:       kirarun.NewLogger(c.stderr()),
			Run:       runHandle,
		}

		started := time.Now().UTC()
		runErr := yaegi.InvokeRun(i, kctx, step, kirarun.Agents{})
		finished := time.Now().UTC()

		if runErr == nil {
			bus.Emit(runevents.Event{Event: runevents.KindRunCompleted, RunID: runID})
			if err := session.Remove(sessPath); err != nil {
				return runID, err
			}
			return runID, nil
		}

		bus.Emit(runevents.Event{
			Event:   runevents.KindRunFailed,
			RunID:   runID,
			Attempt: sess.Attempt,
			Error:   runErr.Error(),
		})

		sess.Attempts = append(sess.Attempts, session.AttemptRecord{
			Attempt:   sess.Attempt,
			Name:      "run",
			StartedAt: started.Format(time.RFC3339Nano),
			FailedAt:  finished.Format(time.RFC3339Nano),
			Error:     map[string]any{"message": runErr.Error()},
		})

		if !c.AutoRetry {
			sess.Attempt++
			if saveErr := session.Save(root, sess); saveErr != nil {
				return runID, fmt.Errorf("%w (also failed to save session: %v)", runErr, saveErr)
			}
			return runID, runErr
		}

		sess.Attempt++
		if saveErr := session.Save(root, sess); saveErr != nil {
			return runID, fmt.Errorf("%w (also failed to save session: %v)", runErr, saveErr)
		}
	}
}

func completedStepSummary(steps []session.StepRecord) (int, []string) {
	if len(steps) == 0 {
		return 0, nil
	}
	names := make([]string, 0, len(steps))
	for _, st := range steps {
		names = append(names, st.Name)
	}
	return len(names), names
}

func (c *Config) resolveRunID(wfAbs string) (string, error) {
	if c.ResumeID != "" {
		if err := session.ValidateRunID(c.ResumeID); err != nil {
			return "", err
		}
		return c.ResumeID, nil
	}
	name := c.ScriptDisplayName
	if name == "" {
		name = wfAbs
	}
	return DeriveRunID(name, time.Now())
}
