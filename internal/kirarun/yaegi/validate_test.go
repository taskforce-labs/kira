package yaegi

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestValidateWorkflowGood(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "wf.go")
	src := `package main

import (
	"kira/kirarun"
)

func Run(ctx *kirarun.Context, step *kirarun.Step, agents kirarun.Agents) error {
	_ = ctx
	_ = step
	_ = agents
	return nil
}
`
	require.NoError(t, os.WriteFile(p, []byte(src), 0o600))
	require.NoError(t, ValidateWorkflow(p))
}

func TestValidateWorkflowBadSig(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "wf.go")
	src := `package main

import "kira/kirarun"

func Run(ctx *kirarun.Context) error {
	return nil
}
`
	require.NoError(t, os.WriteFile(p, []byte(src), 0o600))
	err := ValidateWorkflow(p)
	require.Error(t, err)
}
