package commands

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"kira/internal/config"
)

func TestValidateTrunkBranch(t *testing.T) {
	t.Run("returns nil when on trunk branch", func(t *testing.T) {
		tmpDir := t.TempDir()
		require.NoError(t, os.Chdir(tmpDir))
		defer func() { _ = os.Chdir("/") }()

		// Create git repo and make initial commit on main
		// #nosec G204 - tmpDir is from t.TempDir(), command is fixed
		require.NoError(t, exec.Command("git", "init").Run())
		// #nosec G204 - command is fixed
		require.NoError(t, exec.Command("git", "config", "user.email", "test@example.com").Run())
		// #nosec G204 - command is fixed
		require.NoError(t, exec.Command("git", "config", "user.name", "Test User").Run())
		require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "f"), []byte("x"), 0o600))
		// #nosec G204 - command is fixed
		require.NoError(t, exec.Command("git", "add", "f").Run())
		// #nosec G204 - commit message is fixed test data
		require.NoError(t, exec.Command("git", "commit", "-m", "init").Run())
		// Ensure branch is named main (git init may create master)
		// #nosec G204 - branch name is fixed
		_ = exec.Command("git", "branch", "-m", "main").Run()

		cfg, err := config.LoadConfig()
		require.NoError(t, err)
		if cfg.Git == nil {
			cfg.Git = &config.GitConfig{}
		}
		cfg.Git.TrunkBranch = "main"

		err = validateTrunkBranch(cfg)
		assert.NoError(t, err)
	})

	t.Run("returns error when on feature branch", func(t *testing.T) {
		tmpDir := t.TempDir()
		require.NoError(t, os.Chdir(tmpDir))
		defer func() { _ = os.Chdir("/") }()

		// #nosec G204 - tmpDir is from t.TempDir(), command is fixed
		require.NoError(t, exec.Command("git", "init").Run())
		// #nosec G204 - command is fixed
		require.NoError(t, exec.Command("git", "config", "user.email", "test@example.com").Run())
		// #nosec G204 - command is fixed
		require.NoError(t, exec.Command("git", "config", "user.name", "Test User").Run())
		require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "f"), []byte("x"), 0o600))
		// #nosec G204 - command is fixed
		require.NoError(t, exec.Command("git", "add", "f").Run())
		// #nosec G204 - commit message is fixed test data
		require.NoError(t, exec.Command("git", "commit", "-m", "init").Run())
		// #nosec G204 - branch name is fixed
		_ = exec.Command("git", "branch", "-m", "main").Run()
		// #nosec G204 - branch name is fixed
		require.NoError(t, exec.Command("git", "checkout", "-b", "014-feature").Run())

		cfg, err := config.LoadConfig()
		require.NoError(t, err)
		if cfg.Git == nil {
			cfg.Git = &config.GitConfig{}
		}
		cfg.Git.TrunkBranch = "main"

		err = validateTrunkBranch(cfg)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "cannot run 'kira done' on a feature branch")
		assert.Contains(t, err.Error(), "Check out the trunk branch (main) first")
	})
}
