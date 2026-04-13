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

// humanLines returns stderr lines that contain kira-run progress output.
func humanLines(stderr string) []string {
	var out []string
	for _, line := range strings.Split(stderr, "\n") {
		if strings.Contains(line, "[kira-run]") {
			out = append(out, line)
		}
	}
	return out
}

func lineIndex(lines []string, subs ...string) int {
outer:
	for i, line := range lines {
		for _, sub := range subs {
			if !strings.Contains(line, sub) {
				continue outer
			}
		}
		return i
	}
	return -1
}

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
	lines := humanLines(human)
	require.NotEmpty(t, lines)
	require.Contains(t, lines[0], "run_start")
	last := lines[len(lines)-1]
	require.Contains(t, last, "run_completed")
	require.Contains(t, last, "session=removed")

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
	out := stderr.String()
	require.Contains(t, out, "run_start")
	require.Contains(t, out, "run_failed")

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

	lines := humanLines(stderr.String())
	require.GreaterOrEqual(t, len(lines), 8)
	require.Contains(t, lines[0], "run_start")
	require.Contains(t, lines[1], "step_start")
	require.Contains(t, lines[1], "step=a")
	require.Contains(t, lines[2], "step_done")
	require.Contains(t, lines[2], "step=a")
	require.Contains(t, lines[3], "step_start")
	require.Contains(t, lines[3], "step=b")
	require.Contains(t, lines[7], "run_completed")
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
	require.Contains(t, stderr1.String(), "run_failed")

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

	lines := humanLines(stderr2.String())
	require.GreaterOrEqual(t, lineIndex(lines, "run_resume"), 0)
	require.GreaterOrEqual(t, lineIndex(lines, "step_skip", "step=first"), 0)
	require.Contains(t, strings.Join(lines, "\n"), "reason=already_completed")
	iSkip := lineIndex(lines, "step_skip", "step=first")
	iSecond := lineIndex(lines, "step_start", "step=second")
	require.Greater(t, iSecond, iSkip)
	require.GreaterOrEqual(t, lineIndex(lines, "run_completed"), 0)
}

func TestHumanOutputAutoRetry(t *testing.T) {
	root := t.TempDir()
	wf := filepath.Join("testdata", "autoretry.go")
	absWF, err := filepath.Abs(wf)
	require.NoError(t, err)

	var stdout, stderr bytes.Buffer
	// Avoid "retry" in workflow= so tests can substring-match the retry event line.
	cfg := &Config{
		ProjectRoot:       root,
		WorkflowPath:      absWF,
		KiraVersion:       "test",
		AutoRetry:         true,
		Stdout:            &stdout,
		Stderr:            &stderr,
		ScriptDisplayName: "bounce",
	}
	_, err = cfg.Execute(context.Background())
	require.NoError(t, err)

	lines := humanLines(stderr.String())
	iStart := lineIndex(lines, "run_start")
	iFail := lineIndex(lines, "run_failed")
	iRetry := lineIndex(lines, "retry")
	iDone := lineIndex(lines, "run_completed")
	require.Less(t, iStart, iFail)
	require.Less(t, iFail, iRetry)
	require.Less(t, iRetry, iDone)
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

	lines := humanLines(stderr.String())
	iNotice := lineIndex(lines, "flag_notice")
	iStart := lineIndex(lines, "run_start")
	require.GreaterOrEqual(t, iNotice, 0)
	require.GreaterOrEqual(t, iStart, 0)
	require.Less(t, iNotice, iStart)
	require.Contains(t, lines[iNotice], "ignore_attempt_limit")
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
