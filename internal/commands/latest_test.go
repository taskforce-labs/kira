package commands

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRunLatest(t *testing.T) {
	t.Run("validates workspace exists", func(t *testing.T) {
		tmpDir := t.TempDir()
		require.NoError(t, os.Chdir(tmpDir))
		defer func() { _ = os.Chdir("/") }()

		// No .work directory exists
		err := runLatest(nil, nil)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "not a kira workspace")
		assert.Contains(t, err.Error(), "Run 'kira init' first")
	})

	t.Run("returns not implemented message in valid workspace", func(t *testing.T) {
		tmpDir := t.TempDir()
		require.NoError(t, os.Chdir(tmpDir))
		defer func() { _ = os.Chdir("/") }()

		// Create .work directory to make it a valid workspace
		require.NoError(t, os.MkdirAll(".work", 0o700))

		err := runLatest(nil, nil)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "kira latest is not yet implemented")
	})
}
