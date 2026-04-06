package steppersist

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"kira/internal/kirarun/session"
	"kira/kirarun"
)

func TestSessionAdapterRoundTrip(t *testing.T) {
	root := t.TempDir()
	script := filepath.Join(root, "w.go")

	s := &session.Session{
		Path:        script,
		KiraVersion: "0",
		RunID:       "run-test",
		Attempt:     1,
	}
	ad := NewSessionAdapter(s)
	run := kirarun.NewRunHandle(1, false, false)
	step := kirarun.NewStep(ad, run)

	type stepOut struct {
		K string `json:"k"`
	}

	_, err := kirarun.Do(step, "s1", func(_ kirarun.StepContext) (stepOut, error) {
		return stepOut{K: "v"}, nil
	})
	require.NoError(t, err)
	require.Len(t, s.Steps, 1)
	require.Equal(t, "v", s.Steps[0].Data["k"])

	step2 := kirarun.NewStep(ad, run)
	v, err := kirarun.Do(step2, "s1", func(_ kirarun.StepContext) (stepOut, error) {
		return stepOut{K: "other"}, nil
	})
	require.NoError(t, err)
	require.Equal(t, "v", v.K)
}
