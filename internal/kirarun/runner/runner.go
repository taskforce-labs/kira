package runner

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

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
}

func (c *Config) stdout() io.Writer {
	if c != nil && c.Stdout != nil {
		return c.Stdout
	}
	return os.Stdout
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

	i, err := yaegi.LoadWorkflow(wfAbs)
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

	return c.runLoop(ctx, i, root, sessPath, sess, cliResume)
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
) (string, error) {
	runID := sess.RunID
	for {
		select {
		case <-ctx.Done():
			return runID, ctx.Err()
		default:
		}

		if err := session.Save(root, sess); err != nil {
			return runID, err
		}

		runHandle := kirarun.NewRunHandle(sess.Attempt, cliResume, c.IgnoreAttemptLimit)
		pers := steppersist.NewSessionAdapter(sess)
		step := kirarun.NewStep(pers, runHandle)
		kctx := &kirarun.Context{
			Workspace: kirarun.NewWorkspace(root),
			Log:       kirarun.NewLogger(os.Stderr),
			Run:       runHandle,
		}

		started := time.Now().UTC()
		runErr := yaegi.InvokeRun(i, kctx, step, kirarun.Agents{})
		finished := time.Now().UTC()

		if runErr == nil {
			if err := session.Remove(sessPath); err != nil {
				return runID, err
			}
			return runID, nil
		}

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
