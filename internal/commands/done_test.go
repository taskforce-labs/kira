package commands

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/google/go-github/v61/github"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"kira/internal/config"
)

// resetHelpFlag clears the help flag on cmd and its children so a previous test's --help doesn't affect the next Execute().
func resetHelpFlag(cmd *cobra.Command) {
	for c := cmd; c != nil; c = c.Parent() {
		if f := c.Flags().Lookup("help"); f != nil {
			_ = f.Value.Set("false")
		}
	}
	for _, c := range cmd.Commands() {
		resetHelpFlag(c)
	}
}

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

func TestDoneWorkItemAndPRResolution(t *testing.T) {
	origDir, err := os.Getwd()
	require.NoError(t, err)
	defer func() { _ = os.Chdir(origDir) }()

	t.Run("fails when work item ID invalid", func(t *testing.T) {
		tmpDir := t.TempDir()
		require.NoError(t, os.Chdir(tmpDir))
		defer func() { _ = os.Chdir(origDir) }()
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
		require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "kira.yml"), []byte("version: \"1.0\"\ngit:\n  trunk_branch: main\n"), 0o600))

		cwd, err := os.Getwd()
		require.NoError(t, err)
		canonTmp, _ := filepath.EvalSymlinks(tmpDir)
		canonCwd, _ := filepath.EvalSymlinks(cwd)
		require.Equal(t, canonTmp, canonCwd, "test must run in tmpDir so config and .work are correct")

		rootCmd.SetArgs([]string{"done", "abc"})
		resetHelpFlag(rootCmd)
		execErr := rootCmd.Execute()
		require.Error(t, execErr, "Execute() should fail for invalid work item ID (cwd: %s)", tmpDir)
		assert.Contains(t, execErr.Error(), "invalid work item ID")
	})

	t.Run("fails when work item not found", func(t *testing.T) {
		tmpDir := t.TempDir()
		require.NoError(t, os.Chdir(tmpDir))
		defer func() { _ = os.Chdir(origDir) }()
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
		require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "kira.yml"), []byte("version: \"1.0\"\ngit:\n  trunk_branch: main\n"), 0o600))

		cwd, err := os.Getwd()
		require.NoError(t, err)
		canonTmp, _ := filepath.EvalSymlinks(tmpDir)
		canonCwd, _ := filepath.EvalSymlinks(cwd)
		require.Equal(t, canonTmp, canonCwd, "test must run in tmpDir so config and .work are correct")

		rootCmd.SetArgs([]string{"done", "999"})
		resetHelpFlag(rootCmd)
		execErr := rootCmd.Execute()
		require.Error(t, execErr, "Execute() should fail when work item not found (cwd: %s)", tmpDir)
		assert.Contains(t, execErr.Error(), "work item 999 not found")
	})

	t.Run("succeeds with non-GitHub remote in dry-run", func(t *testing.T) {
		tmpDir := t.TempDir()
		require.NoError(t, os.Chdir(tmpDir))
		defer func() { _ = os.Chdir(origDir) }()
		require.NoError(t, os.MkdirAll(".work/2_doing", 0o700))
		require.NoError(t, os.WriteFile(filepath.Join(tmpDir, ".work/2_doing/014-test.prd.md"), []byte("---\nid: 014\ntitle: Test\nstatus: doing\nkind: prd\ncreated: 2024-01-01\n---\n"), 0o600))
		// #nosec G204 - tmpDir from t.TempDir(), command is fixed
		require.NoError(t, exec.Command("git", "init").Run())
		// #nosec G204 - command is fixed
		require.NoError(t, exec.Command("git", "config", "user.email", "test@example.com").Run())
		// #nosec G204 - command is fixed
		require.NoError(t, exec.Command("git", "config", "user.name", "Test User").Run())
		// #nosec G204 - command is fixed
		require.NoError(t, exec.Command("git", "remote", "add", "origin", "https://gitlab.com/owner/repo.git").Run())
		require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "f"), []byte("x"), 0o600))
		// #nosec G204 - command is fixed
		require.NoError(t, exec.Command("git", "add", "f").Run())
		// #nosec G204 - commit message is fixed
		require.NoError(t, exec.Command("git", "commit", "-m", "init").Run())
		// #nosec G204 - branch name is fixed
		_ = exec.Command("git", "branch", "-m", "main").Run()
		require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "kira.yml"), []byte("version: \"1.0\"\ngit:\n  trunk_branch: main\n"), 0o600))

		rootCmd.SetArgs([]string{"done", "014", "--dry-run"})
		err := rootCmd.Execute()
		require.NoError(t, err)
	})
}

const (
	testDoneStatusPath   = "/api/v3/repos/owner/repo/commits/abc123/status"
	testDoneCommentsPath = "/api/v3/repos/owner/repo/pulls/42/comments"
)

func TestRunPRChecks(t *testing.T) {
	ctx := context.Background()
	owner, repo := "owner", "repo"
	headSHA := "abc123"
	prNum := 42
	pr := &github.PullRequest{
		Number: github.Int(prNum),
		Head: &github.PullRequestBranch{
			SHA: github.String(headSHA),
		},
	}

	t.Run("checks passing and no comments returns nil", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			switch r.URL.Path {
			case testDoneStatusPath:
				_, _ = w.Write([]byte(`{"state":"success","total_count":1}`))
			case testDoneCommentsPath:
				_, _ = w.Write([]byte(`[]`))
			default:
				w.WriteHeader(http.StatusNotFound)
			}
		}))
		defer server.Close()

		baseURL, _ := url.Parse(server.URL + "/api/v3/")
		client := github.NewClient(server.Client())
		client.BaseURL = baseURL

		err := runPRChecks(ctx, client, owner, repo, pr, true, true, false)
		require.NoError(t, err)
	})

	t.Run("checks failing returns error unless force", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			switch r.URL.Path {
			case testDoneStatusPath:
				_, _ = w.Write([]byte(`{"state":"failure","total_count":2}`))
			case testDoneCommentsPath:
				_, _ = w.Write([]byte(`[]`))
			default:
				w.WriteHeader(http.StatusNotFound)
			}
		}))
		defer server.Close()

		baseURL, _ := url.Parse(server.URL + "/api/v3/")
		client := github.NewClient(server.Client())
		client.BaseURL = baseURL

		err := runPRChecks(ctx, client, owner, repo, pr, true, true, false)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "required status checks have not passed")
		assert.Contains(t, err.Error(), "state: failure")
		assert.Contains(t, err.Error(), "--force")

		errForce := runPRChecks(ctx, client, owner, repo, pr, true, true, true)
		require.NoError(t, errForce)
	})

	t.Run("unresolved comments returns error unless force", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			switch r.URL.Path {
			case testDoneStatusPath:
				_, _ = w.Write([]byte(`{"state":"success","total_count":0}`))
			case testDoneCommentsPath:
				_, _ = w.Write([]byte(`[{"id":1,"body":"comment"}]`))
			default:
				w.WriteHeader(http.StatusNotFound)
			}
		}))
		defer server.Close()

		baseURL, _ := url.Parse(server.URL + "/api/v3/")
		client := github.NewClient(server.Client())
		client.BaseURL = baseURL

		err := runPRChecks(ctx, client, owner, repo, pr, true, true, false)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "review comment")
		assert.Contains(t, err.Error(), "42")
		assert.Contains(t, err.Error(), "--force")

		errForce := runPRChecks(ctx, client, owner, repo, pr, true, true, true)
		require.NoError(t, errForce)
	})

	t.Run("nil pr head SHA returns error", func(t *testing.T) {
		badPR := &github.PullRequest{Number: github.Int(1)}
		err := runPRChecks(ctx, nil, owner, repo, badPR, true, true, false)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "no head SHA")
	})
}
