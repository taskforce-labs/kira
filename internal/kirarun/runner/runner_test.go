package runner

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"kira/internal/kirarun/runevents"
	"kira/internal/kirarun/session"
)

func TestDeriveRunID(t *testing.T) {
	ts := time.Date(2006, 1, 2, 15, 4, 5, 0, time.UTC)
	id, err := DeriveRunID("hello.go", ts)
	require.NoError(t, err)
	require.Equal(t, "hello-20060102150405", id)
}

func TestExecuteSuccess(t *testing.T) {
	root := t.TempDir()
	wf := filepath.Join("testdata", "ok.go")
	absWF, err := filepath.Abs(wf)
	require.NoError(t, err)

	var buf, stderr bytes.Buffer
	cfg := &Config{
		ProjectRoot:  root,
		WorkflowPath: absWF,
		KiraVersion:  "test",
		Stdout:       &buf,
		Stderr:       &stderr,
	}
	_, err = cfg.Execute(context.Background())
	require.NoError(t, err)
	require.Contains(t, buf.String(), "run id:")
	human := stderr.String()
	require.Contains(t, human, "[kira-run] starting")
	require.Contains(t, human, "[kira-run] run completed ok")
	iStart := strings.Index(human, "[kira-run] starting")
	iDone := strings.Index(human, "[kira-run] run completed ok")
	require.GreaterOrEqual(t, iStart, 0)
	require.Greater(t, iDone, iStart)

	sessionsDir, err := session.SessionsDir(root)
	require.NoError(t, err)
	entries, err := os.ReadDir(sessionsDir)
	require.NoError(t, err)
	require.Empty(t, entries)
}

func TestExecuteFailNoRetry(t *testing.T) {
	root := t.TempDir()
	wf := filepath.Join("testdata", "fail.go")
	absWF, err := filepath.Abs(wf)
	require.NoError(t, err)

	var stderr bytes.Buffer
	cfg := &Config{
		ProjectRoot:  root,
		WorkflowPath: absWF,
		KiraVersion:  "test",
		Stdout:       io.Discard,
		Stderr:       &stderr,
	}
	_, err = cfg.Execute(context.Background())
	require.Error(t, err)
	require.Contains(t, stderr.String(), "[kira-run] starting")
	require.Contains(t, stderr.String(), "[kira-run] run failed")

	sessDir, err := session.SessionsDir(root)
	require.NoError(t, err)
	entries, err := os.ReadDir(sessDir)
	require.NoError(t, err)
	require.NotEmpty(t, entries)
}

func TestHumanOutputThreeSteps(t *testing.T) {
	root := t.TempDir()
	wf := filepath.Join("testdata", "threestep.go")
	absWF, err := filepath.Abs(wf)
	require.NoError(t, err)

	var stdout, stderr bytes.Buffer
	cfg := &Config{
		ProjectRoot:       root,
		WorkflowPath:      absWF,
		KiraVersion:       "test",
		Stdout:            &stdout,
		Stderr:            &stderr,
		ScriptDisplayName: "threestep",
	}
	_, err = cfg.Execute(context.Background())
	require.NoError(t, err)

	s := stderr.String()
	iStart := strings.Index(s, "[kira-run] starting")
	iA := strings.Index(s, "step start name=a")
	iB := strings.Index(s, "step start name=b")
	iC := strings.Index(s, "step start name=c")
	iDone := strings.Index(s, "[kira-run] run completed ok")
	require.Greater(t, iA, iStart)
	require.Greater(t, iB, iA)
	require.Greater(t, iC, iB)
	require.Greater(t, iDone, iC)
}

func TestHumanOutputResumeSkipsFirstStep(t *testing.T) {
	root := t.TempDir()
	wf := filepath.Join("testdata", "resume_twophase.go")
	absWF, err := filepath.Abs(wf)
	require.NoError(t, err)

	var stdout1, stderr1 bytes.Buffer
	cfg1 := &Config{
		ProjectRoot:       root,
		WorkflowPath:      absWF,
		KiraVersion:       "test",
		Stdout:            &stdout1,
		Stderr:            &stderr1,
		ScriptDisplayName: "resume_twophase",
	}
	_, err = cfg1.Execute(context.Background())
	require.Error(t, err)
	require.Contains(t, stderr1.String(), "[kira-run] run failed")

	const prefix = "run id: "
	out := stdout1.String()
	require.True(t, strings.HasPrefix(out, prefix), out)
	runID := strings.TrimSpace(strings.TrimPrefix(out, prefix))

	marker := filepath.Join(root, ".kira_run_test_phase2")
	require.NoError(t, os.WriteFile(marker, []byte{}, 0o600))

	var stderr2 bytes.Buffer
	cfg2 := &Config{
		ProjectRoot:       root,
		WorkflowPath:      absWF,
		KiraVersion:       "test",
		ResumeID:          runID,
		Stdout:            io.Discard,
		Stderr:            &stderr2,
		ScriptDisplayName: "resume_twophase",
	}
	_, err = cfg2.Execute(context.Background())
	require.NoError(t, err)

	s := stderr2.String()
	require.Contains(t, s, "[kira-run] resuming")
	require.Contains(t, s, "step skip name=first")
	require.Contains(t, s, "reason=already_completed")
	iSkip := strings.Index(s, "step skip name=first")
	iSecond := strings.Index(s, "step start name=second")
	require.Greater(t, iSecond, iSkip)
	require.Contains(t, s, "[kira-run] run completed ok")
}

func TestHumanOutputAutoRetry(t *testing.T) {
	root := t.TempDir()
	wf := filepath.Join("testdata", "autoretry.go")
	absWF, err := filepath.Abs(wf)
	require.NoError(t, err)

	var stdout, stderr bytes.Buffer
	cfg := &Config{
		ProjectRoot:       root,
		WorkflowPath:      absWF,
		KiraVersion:       "test",
		AutoRetry:         true,
		Stdout:            &stdout,
		Stderr:            &stderr,
		ScriptDisplayName: "autoretry",
	}
	_, err = cfg.Execute(context.Background())
	require.NoError(t, err)

	s := stderr.String()
	iStart := strings.Index(s, "[kira-run] starting")
	iFail := strings.Index(s, "[kira-run] run failed")
	iRetry := strings.Index(s, "[kira-run] retry")
	iDone := strings.Index(s, "[kira-run] run completed ok")
	require.Greater(t, iFail, iStart)
	require.Greater(t, iRetry, iFail)
	require.Greater(t, iDone, iRetry)
}

func TestHumanOutputIgnoreAttemptLimitNotice(t *testing.T) {
	root := t.TempDir()
	wf := filepath.Join("testdata", "ok.go")
	absWF, err := filepath.Abs(wf)
	require.NoError(t, err)

	var stdout, stderr bytes.Buffer
	cfg := &Config{
		ProjectRoot:        root,
		WorkflowPath:       absWF,
		KiraVersion:        "test",
		IgnoreAttemptLimit: true,
		Stdout:             &stdout,
		Stderr:             &stderr,
	}
	_, err = cfg.Execute(context.Background())
	require.NoError(t, err)

	s := stderr.String()
	iNotice := strings.Index(s, "ignore_attempt_limit")
	iStart := strings.Index(s, "[kira-run] starting")
	require.GreaterOrEqual(t, iNotice, 0)
	require.Greater(t, iStart, iNotice)
}

func TestJSONLProgressThreeStep(t *testing.T) {
	root := t.TempDir()
	eventsFile := filepath.Join(root, "events.jsonl")
	wf := filepath.Join("testdata", "threestep.go")
	absWF, err := filepath.Abs(wf)
	require.NoError(t, err)

	var stdout, stderr bytes.Buffer
	cfg := &Config{
		ProjectRoot:       root,
		WorkflowPath:      absWF,
		KiraVersion:       "test",
		Stdout:            &stdout,
		Stderr:            &stderr,
		RunEventsPath:     eventsFile,
		ScriptDisplayName: "threestep",
	}
	_, err = cfg.Execute(context.Background())
	require.NoError(t, err)

	raw, err := os.ReadFile(eventsFile) // #nosec G304 -- test reads JSONL it just wrote under t.TempDir
	require.NoError(t, err)
	lines := nonEmptyLines(raw)
	require.NotEmpty(t, lines)

	var kinds []string
	var runID string
	for _, line := range lines {
		var ev runevents.Event
		require.NoError(t, json.Unmarshal(line, &ev))
		require.Equal(t, runevents.SchemaVersion, ev.SchemaVersion)
		require.Equal(t, runevents.SourceRunner, ev.Source)
		require.NotEmpty(t, ev.Event)
		require.NotEmpty(t, ev.Ts)
		if ev.RunID != "" {
			if runID == "" {
				runID = ev.RunID
			} else {
				require.Equal(t, runID, ev.RunID, "event=%s", ev.Event)
			}
		}
		kinds = append(kinds, ev.Event)
	}
	require.NotEmpty(t, runID)
	require.Equal(t, runevents.KindRunStart, kinds[0])
	require.Equal(t, runevents.KindRunCompleted, kinds[len(kinds)-1])
	require.Contains(t, kinds, runevents.KindStepStart)
	require.Contains(t, kinds, runevents.KindStepDone)
}

func TestHumanAndJSONLSameEventCount(t *testing.T) {
	root := t.TempDir()
	eventsFile := filepath.Join(root, "events.jsonl")
	wf := filepath.Join("testdata", "threestep.go")
	absWF, err := filepath.Abs(wf)
	require.NoError(t, err)

	var stdout, stderr bytes.Buffer
	cfg := &Config{
		ProjectRoot:       root,
		WorkflowPath:      absWF,
		KiraVersion:       "test",
		Stdout:            &stdout,
		Stderr:            &stderr,
		RunEventsPath:     eventsFile,
		ScriptDisplayName: "threestep",
	}
	_, err = cfg.Execute(context.Background())
	require.NoError(t, err)

	raw, err := os.ReadFile(eventsFile) // #nosec G304 -- test reads JSONL it just wrote under t.TempDir
	require.NoError(t, err)
	jsonCount := len(nonEmptyLines(raw))
	humanCount := strings.Count(stderr.String(), "[kira-run]")
	require.Equal(t, jsonCount, humanCount)
}

func nonEmptyLines(data []byte) [][]byte {
	var out [][]byte
	for _, line := range bytes.Split(data, []byte("\n")) {
		if len(bytes.TrimSpace(line)) > 0 {
			out = append(out, line)
		}
	}
	return out
}
