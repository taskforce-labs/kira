package commands

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func gitConfigForCleanTreeTest(t *testing.T, dir string) {
	t.Helper()
	// #nosec G204 -- test helper, dir is test temp path
	cmd := exec.Command("git", "config", "user.email", "test@example.com")
	cmd.Dir = dir
	require.NoError(t, cmd.Run())
	// #nosec G204 -- test helper, dir is test temp path
	cmd = exec.Command("git", "config", "user.name", "Test User")
	cmd.Dir = dir
	require.NoError(t, cmd.Run())
}

func initGitRepo(t *testing.T, dir string) {
	t.Helper()
	// #nosec G204 -- test helper, dir is test temp path
	cmd := exec.Command("git", "init")
	cmd.Dir = dir
	require.NoError(t, cmd.Run())
	gitConfigForCleanTreeTest(t, dir)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "f.txt"), []byte("x"), 0o600))
	// #nosec G204 -- test helper
	cmd = exec.Command("git", "add", "f.txt")
	cmd.Dir = dir
	require.NoError(t, cmd.Run())
	// #nosec G204 -- test helper
	cmd = exec.Command("git", "commit", "-m", "initial")
	cmd.Dir = dir
	require.NoError(t, cmd.Run())
}

func TestHasUncommitted(t *testing.T) {
	t.Run("returns false when dryRun", func(t *testing.T) {
		dir := t.TempDir()
		dirty, err := HasUncommitted(dir, true)
		require.NoError(t, err)
		assert.False(t, dirty)
	})

	t.Run("returns false for clean repo", func(t *testing.T) {
		dir := t.TempDir()
		// #nosec G204 -- test helper
		cmd := exec.Command("git", "init")
		cmd.Dir = dir
		require.NoError(t, cmd.Run())
		gitConfigForCleanTreeTest(t, dir)
		require.NoError(t, os.WriteFile(filepath.Join(dir, "f.txt"), []byte("x"), 0o600))
		// #nosec G204 -- test helper
		cmd = exec.Command("git", "add", "f.txt")
		cmd.Dir = dir
		require.NoError(t, cmd.Run())
		// #nosec G204 -- test helper
		cmd = exec.Command("git", "commit", "-m", "initial")
		cmd.Dir = dir
		require.NoError(t, cmd.Run())

		dirty, err := HasUncommitted(dir, false)
		require.NoError(t, err)
		assert.False(t, dirty)
	})

	t.Run("returns true when uncommitted changes", func(t *testing.T) {
		dir := t.TempDir()
		// #nosec G204 -- test helper
		cmd := exec.Command("git", "init")
		cmd.Dir = dir
		require.NoError(t, cmd.Run())
		gitConfigForCleanTreeTest(t, dir)
		require.NoError(t, os.WriteFile(filepath.Join(dir, "f.txt"), []byte("x"), 0o600))
		// #nosec G204 -- test helper
		cmd = exec.Command("git", "add", "f.txt")
		cmd.Dir = dir
		require.NoError(t, cmd.Run())
		// #nosec G204 -- test helper
		cmd = exec.Command("git", "commit", "-m", "initial")
		cmd.Dir = dir
		require.NoError(t, cmd.Run())
		require.NoError(t, os.WriteFile(filepath.Join(dir, "dirty.txt"), []byte("y"), 0o600))

		dirty, err := HasUncommitted(dir, false)
		require.NoError(t, err)
		assert.True(t, dirty)
	})
}

func TestStash(t *testing.T) {
	t.Run("stashes changes", func(t *testing.T) {
		dir := t.TempDir()
		// #nosec G204 -- test helper
		cmd := exec.Command("git", "init")
		cmd.Dir = dir
		require.NoError(t, cmd.Run())
		gitConfigForCleanTreeTest(t, dir)
		require.NoError(t, os.WriteFile(filepath.Join(dir, "f.txt"), []byte("x"), 0o600))
		// #nosec G204 -- test helper
		cmd = exec.Command("git", "add", "f.txt")
		cmd.Dir = dir
		require.NoError(t, cmd.Run())
		// #nosec G204 -- test helper
		cmd = exec.Command("git", "commit", "-m", "initial")
		cmd.Dir = dir
		require.NoError(t, cmd.Run())
		require.NoError(t, os.WriteFile(filepath.Join(dir, "dirty.txt"), []byte("y"), 0o600))

		err := Stash(dir, "test stash message")
		require.NoError(t, err)

		dirty, err := HasUncommitted(dir, false)
		require.NoError(t, err)
		assert.False(t, dirty)
	})

	t.Run("no error when nothing to stash", func(t *testing.T) {
		dir := t.TempDir()
		// #nosec G204 -- test helper
		cmd := exec.Command("git", "init")
		cmd.Dir = dir
		require.NoError(t, cmd.Run())
		gitConfigForCleanTreeTest(t, dir)
		require.NoError(t, os.WriteFile(filepath.Join(dir, "f.txt"), []byte("x"), 0o600))
		// #nosec G204 -- test helper
		cmd = exec.Command("git", "add", "f.txt")
		cmd.Dir = dir
		require.NoError(t, cmd.Run())
		// #nosec G204 -- test helper
		cmd = exec.Command("git", "commit", "-m", "initial")
		cmd.Dir = dir
		require.NoError(t, cmd.Run())

		err := Stash(dir, "test")
		require.NoError(t, err)
	})
}

func TestPop(t *testing.T) {
	t.Run("pops stash", func(t *testing.T) {
		dir := t.TempDir()
		// #nosec G204 -- test helper
		cmd := exec.Command("git", "init")
		cmd.Dir = dir
		require.NoError(t, cmd.Run())
		gitConfigForCleanTreeTest(t, dir)
		require.NoError(t, os.WriteFile(filepath.Join(dir, "f.txt"), []byte("x"), 0o600))
		// #nosec G204 -- test helper
		cmd = exec.Command("git", "add", "f.txt")
		cmd.Dir = dir
		require.NoError(t, cmd.Run())
		// #nosec G204 -- test helper
		cmd = exec.Command("git", "commit", "-m", "initial")
		cmd.Dir = dir
		require.NoError(t, cmd.Run())
		require.NoError(t, os.WriteFile(filepath.Join(dir, "dirty.txt"), []byte("y"), 0o600))
		// #nosec G204 -- test helper
		cmd = exec.Command("git", "stash", "push", "-m", "test", "--include-untracked")
		cmd.Dir = dir
		require.NoError(t, cmd.Run())

		err := Pop(dir)
		require.NoError(t, err)

		dirty, err := HasUncommitted(dir, false)
		require.NoError(t, err)
		assert.True(t, dirty)
	})

	t.Run("no error when no stash", func(t *testing.T) {
		dir := t.TempDir()
		// #nosec G204 -- test helper
		cmd := exec.Command("git", "init")
		cmd.Dir = dir
		require.NoError(t, cmd.Run())
		gitConfigForCleanTreeTest(t, dir)
		require.NoError(t, os.WriteFile(filepath.Join(dir, "f.txt"), []byte("x"), 0o600))
		// #nosec G204 -- test helper
		cmd = exec.Command("git", "add", "f.txt")
		cmd.Dir = dir
		require.NoError(t, cmd.Run())
		// #nosec G204 -- test helper
		cmd = exec.Command("git", "commit", "-m", "initial")
		cmd.Dir = dir
		require.NoError(t, cmd.Run())

		err := Pop(dir)
		require.NoError(t, err)
	})
}

func TestRunWithCleanTree(t *testing.T) {
	t.Run("clean tree runs fn without stash", func(t *testing.T) {
		dir := t.TempDir()
		// #nosec G204 -- test helper
		cmd := exec.Command("git", "init")
		cmd.Dir = dir
		require.NoError(t, cmd.Run())
		gitConfigForCleanTreeTest(t, dir)
		initGitRepo(t, dir)

		called := false
		hadStash, err := RunWithCleanTree(dir, "test", "repo", false, func() error {
			called = true
			return nil
		})
		require.NoError(t, err)
		assert.True(t, called)
		assert.False(t, hadStash)
	})

	t.Run("dirty tree stashes runs fn and pops", func(t *testing.T) {
		dir := t.TempDir()
		// #nosec G204 -- test helper
		cmd := exec.Command("git", "init")
		cmd.Dir = dir
		require.NoError(t, cmd.Run())
		gitConfigForCleanTreeTest(t, dir)
		initGitRepo(t, dir)
		require.NoError(t, os.WriteFile(filepath.Join(dir, "dirty.txt"), []byte("y"), 0o600))

		called := false
		hadStash, err := RunWithCleanTree(dir, "test", "repo", false, func() error {
			called = true
			return nil
		})
		require.NoError(t, err)
		assert.True(t, called)
		assert.True(t, hadStash)
		dirty, err := HasUncommitted(dir, false)
		require.NoError(t, err)
		assert.True(t, dirty, "stash should have been popped and changes restored")
	})

	t.Run("noPopStash leaves stash in place", func(t *testing.T) {
		dir := t.TempDir()
		// #nosec G204 -- test helper
		cmd := exec.Command("git", "init")
		cmd.Dir = dir
		require.NoError(t, cmd.Run())
		gitConfigForCleanTreeTest(t, dir)
		initGitRepo(t, dir)
		require.NoError(t, os.WriteFile(filepath.Join(dir, "dirty.txt"), []byte("y"), 0o600))

		hadStash, err := RunWithCleanTree(dir, "test", "repo", true, func() error { return nil })
		require.NoError(t, err)
		assert.True(t, hadStash)
		dirty, err := HasUncommitted(dir, false)
		require.NoError(t, err)
		assert.False(t, dirty, "stash should not have been popped")
	})

	t.Run("restore on failure pops stash", func(t *testing.T) {
		dir := t.TempDir()
		// #nosec G204 -- test helper
		cmd := exec.Command("git", "init")
		cmd.Dir = dir
		require.NoError(t, cmd.Run())
		gitConfigForCleanTreeTest(t, dir)
		initGitRepo(t, dir)
		require.NoError(t, os.WriteFile(filepath.Join(dir, "dirty.txt"), []byte("y"), 0o600))

		opErr := errors.New("operation failed")
		hadStash, err := RunWithCleanTree(dir, "test", "repo", false, func() error { return opErr })
		assert.ErrorIs(t, err, opErr)
		assert.True(t, hadStash)
		dirty, err := HasUncommitted(dir, false)
		require.NoError(t, err)
		assert.True(t, dirty, "stash should have been restored on failure")
	})

	t.Run("ErrKeepStashOnFailure does not pop", func(t *testing.T) {
		dir := t.TempDir()
		// #nosec G204 -- test helper
		cmd := exec.Command("git", "init")
		cmd.Dir = dir
		require.NoError(t, cmd.Run())
		gitConfigForCleanTreeTest(t, dir)
		initGitRepo(t, dir)
		require.NoError(t, os.WriteFile(filepath.Join(dir, "dirty.txt"), []byte("y"), 0o600))

		innerErr := errors.New("rebase conflict")
		keepErr := fmt.Errorf("%w: %w", ErrKeepStashOnFailure, innerErr)
		hadStash, err := RunWithCleanTree(dir, "test", "repo", false, func() error { return keepErr })
		assert.True(t, errors.Is(err, ErrKeepStashOnFailure))
		assert.True(t, hadStash)
		dirty, err := HasUncommitted(dir, false)
		require.NoError(t, err)
		assert.False(t, dirty, "stash should be kept when callback returns ErrKeepStashOnFailure")
	})
}
