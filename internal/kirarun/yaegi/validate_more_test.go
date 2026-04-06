package yaegi

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestValidateWorkflowMissingRun(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "wf.go")
	src := `package main
import "kira/kirarun"
func NotRun() {}
`
	require.NoError(t, os.WriteFile(p, []byte(src), 0o600))
	err := ValidateWorkflow(p)
	require.Error(t, err)
	require.Contains(t, err.Error(), "Run")
}
