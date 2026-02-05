package commands

import (
	"bytes"
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

func TestDoneCommandRegistration(t *testing.T) {
	// Verify done command is registered under root
	c, _, _ := rootCmd.Find([]string{"done"})
	require.NotNil(t, c, "done command should be registered")
	assert.Equal(t, "done", c.Name())
	assert.Contains(t, c.Use, "work-item-id")
}

func TestDoneCommandArgsAndFlags(t *testing.T) {
	t.Run("fails when work-item-id is missing", func(t *testing.T) {
		tmpDir := t.TempDir()
		require.NoError(t, os.Chdir(tmpDir))
		defer func() { _ = os.Chdir("/") }()
		require.NoError(t, os.MkdirAll(".work/2_doing", 0o700))

		rootCmd.SetArgs([]string{"done"})
		err := rootCmd.Execute()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "accepts 1 arg(s)")
	})

	t.Run("help output contains expected flags", func(t *testing.T) {
		buf := new(bytes.Buffer)
		rootCmd.SetOut(buf)
		rootCmd.SetErr(buf)
		rootCmd.SetArgs([]string{"done", "--help"})
		err := rootCmd.Execute()
		require.NoError(t, err)
		out := buf.String()
		assert.Contains(t, out, "work-item-id")
		assert.Contains(t, out, "merge-strategy")
		assert.Contains(t, out, "no-cleanup")
		assert.Contains(t, out, "force")
		assert.Contains(t, out, "dry-run")
		// Reset so next subtest does not inherit this output
		rootCmd.SetOut(nil)
		rootCmd.SetErr(nil)
	})

	t.Run("dry-run passes after trunk validation", func(t *testing.T) {
		tmpDir := t.TempDir()
		require.NoError(t, os.Chdir(tmpDir))
		defer func() { _ = os.Chdir("/") }()
		require.NoError(t, os.MkdirAll(".work/2_doing", 0o700))
		// #nosec G204 - tmpDir from t.TempDir(), command is fixed
		require.NoError(t, exec.Command("git", "init").Run())
		// #nosec G204 - command is fixed
		require.NoError(t, exec.Command("git", "config", "user.email", "test@example.com").Run())
		// #nosec G204 - command is fixed
		require.NoError(t, exec.Command("git", "config", "user.name", "Test User").Run())
		require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "f"), []byte("x"), 0o600))
		// #nosec G204 - command is fixed
		require.NoError(t, exec.Command("git", "add", "f").Run())
		// #nosec G204 - commit message is fixed
		require.NoError(t, exec.Command("git", "commit", "-m", "init").Run())
		// #nosec G204 - branch name is fixed
		_ = exec.Command("git", "branch", "-m", "main").Run()

		// kira.yml with trunk_branch so validateTrunkBranch passes
		require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "kira.yml"), []byte("version: \"1.0\"\ngit:\n  trunk_branch: main\n"), 0o600))

		rootCmd.SetArgs([]string{"done", "014", "--dry-run"})
		err := rootCmd.Execute()
		require.NoError(t, err)
		// Dry-run with valid trunk succeeds; output may go to cobra's Out
	})
}
