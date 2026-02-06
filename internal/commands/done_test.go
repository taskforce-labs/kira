package commands

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

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

func TestDoneNotOnTrunk(t *testing.T) {
	origDir, err := os.Getwd()
	require.NoError(t, err)
	defer func() { _ = os.Chdir(origDir) }()

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
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "f"), []byte("x"), 0o600))
	// #nosec G204 - command is fixed
	require.NoError(t, exec.Command("git", "add", "f").Run())
	// #nosec G204 - commit message is fixed
	require.NoError(t, exec.Command("git", "commit", "-m", "init").Run())
	// #nosec G204 - branch name is fixed
	_ = exec.Command("git", "branch", "-m", "main").Run()
	// #nosec G204 - branch name is fixed
	require.NoError(t, exec.Command("git", "checkout", "-b", "014-feature").Run())
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "kira.yml"), []byte("version: \"1.0\"\ngit:\n  trunk_branch: main\n"), 0o600))

	rootCmd.SetArgs([]string{"done", "014"})
	resetHelpFlag(rootCmd)
	execErr := rootCmd.Execute()
	require.Error(t, execErr)
	assert.Contains(t, execErr.Error(), "trunk")
}

const testDonePullsPath = "/api/v3/repos/owner/repo/pulls"

func TestDonePRNotFound(t *testing.T) {
	origDir, err := os.Getwd()
	require.NoError(t, err)
	defer func() { _ = os.Chdir(origDir) }()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == testDonePullsPath {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte("[]"))
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

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
	require.NoError(t, exec.Command("git", "remote", "add", "origin", "https://github.com/owner/repo.git").Run())
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "f"), []byte("x"), 0o600))
	// #nosec G204 - command is fixed
	require.NoError(t, exec.Command("git", "add", "f").Run())
	// #nosec G204 - commit message is fixed
	require.NoError(t, exec.Command("git", "commit", "-m", "init").Run())
	// #nosec G204 - branch name is fixed
	_ = exec.Command("git", "branch", "-m", "main").Run()
	kiraYml := fmt.Sprintf("version: \"1.0\"\ngit:\n  trunk_branch: main\nworkspace:\n  git_base_url: %s\n", server.URL)
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "kira.yml"), []byte(kiraYml), 0o600))

	restore := setenv("KIRA_GITHUB_TOKEN", "test-token")
	defer restore()

	rootCmd.SetArgs([]string{"done", "014"})
	resetHelpFlag(rootCmd)
	execErr := rootCmd.Execute()
	require.Error(t, execErr)
	assert.Contains(t, execErr.Error(), "no pull request found")
}

func setenv(key, value string) func() {
	old := os.Getenv(key)
	_ = os.Setenv(key, value)
	return func() {
		if old == "" {
			_ = os.Unsetenv(key)
		} else {
			_ = os.Setenv(key, old)
		}
	}
}

func TestDoneDryRunAlreadyMergedPR(t *testing.T) {
	origDir, err := os.Getwd()
	require.NoError(t, err)
	defer func() { _ = os.Chdir(origDir) }()

	mergedAt := "2024-06-01T12:00:00Z"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == testDonePullsPath {
			prs := []*github.PullRequest{
				{
					Number:         github.Int(42),
					Head:           &github.PullRequestBranch{Ref: github.String("014-feature")},
					MergedAt:       &github.Timestamp{Time: mustParseTime(mergedAt)},
					MergeCommitSHA: github.String("abc123"),
				},
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(prs)
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

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
	require.NoError(t, exec.Command("git", "remote", "add", "origin", "https://github.com/owner/repo.git").Run())
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "f"), []byte("x"), 0o600))
	// #nosec G204 - command is fixed
	require.NoError(t, exec.Command("git", "add", "f").Run())
	// #nosec G204 - commit message is fixed
	require.NoError(t, exec.Command("git", "commit", "-m", "init").Run())
	// #nosec G204 - branch name is fixed
	_ = exec.Command("git", "branch", "-m", "main").Run()
	kiraYml := fmt.Sprintf("version: \"1.0\"\ngit:\n  trunk_branch: main\nworkspace:\n  git_base_url: %s\n", server.URL)
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "kira.yml"), []byte(kiraYml), 0o600))

	restore := setenv("KIRA_GITHUB_TOKEN", "test-token")
	defer restore()

	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetArgs([]string{"done", "014", "--dry-run"})
	resetHelpFlag(rootCmd)
	execErr := rootCmd.Execute()
	require.NoError(t, execErr)
	assert.Contains(t, buf.String(), "idempotent")
}

func mustParseTime(s string) time.Time {
	t, err := time.Parse("2006-01-02T15:04:05Z", s)
	if err != nil {
		panic(err)
	}
	return t
}

const (
	testDoneStatusPath    = "/api/v3/repos/owner/repo/commits/abc123/status"
	testDoneCommentsPath  = "/api/v3/repos/owner/repo/pulls/42/comments"
	testDoneDeleteRefPath = "/api/v3/repos/o/r/git/refs/heads/014-feature"
	testOwner             = "owner"
	testRepo              = "repo"
	testHeadSHA           = "abc123"
)

func TestRunPRChecks(t *testing.T) {
	ctx := context.Background()
	owner, repo := testOwner, testRepo
	headSHA := testHeadSHA
	prNum := 42
	pr := &github.PullRequest{
		Number: github.Int(prNum),
		Head: &github.PullRequestBranch{
			SHA: github.String(headSHA),
		},
	}

	// Basic checks
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

func TestRunPRChecksMergeable(t *testing.T) {
	ctx := context.Background()
	owner, repo := testOwner, testRepo
	headSHA := testHeadSHA
	prNum := 42

	t.Run("PR with Mergeable=true passes without API call", func(t *testing.T) {
		mergeablePR := &github.PullRequest{
			Number:    github.Int(prNum),
			Mergeable: github.Bool(true),
			Head: &github.PullRequestBranch{
				SHA: github.String(headSHA),
			},
		}
		// No server needed - should use Mergeable field directly
		client := github.NewClient(nil)
		err := runPRChecks(ctx, client, owner, repo, mergeablePR, true, false, false)
		require.NoError(t, err)
	})

	t.Run("PR with Mergeable=false fails", func(t *testing.T) {
		mergeablePR := &github.PullRequest{
			Number:    github.Int(prNum),
			Mergeable: github.Bool(false),
			Head: &github.PullRequestBranch{
				SHA: github.String(headSHA),
			},
		}
		client := github.NewClient(nil)
		err := runPRChecks(ctx, client, owner, repo, mergeablePR, true, false, false)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "not mergeable")
		assert.Contains(t, err.Error(), "--force")
	})
}

func TestRunPRChecksPending(t *testing.T) {
	ctx := context.Background()
	owner, repo := testOwner, testRepo
	headSHA := testHeadSHA
	prNum := 42

	t.Run("pending status with no failing checks passes", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			switch r.URL.Path {
			case testDoneStatusPath:
				// Pending state with all checks pending or success
				_, _ = w.Write([]byte(`{"state":"pending","total_count":2,"statuses":[{"state":"pending","context":"check1"},{"state":"success","context":"check2"}]}`))
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

		// PR with Mergeable=nil (not set) to trigger combined status check
		prNoMergeable := &github.PullRequest{
			Number:    github.Int(prNum),
			Mergeable: nil,
			Head: &github.PullRequestBranch{
				SHA: github.String(headSHA),
			},
		}
		err := runPRChecks(ctx, client, owner, repo, prNoMergeable, true, false, false)
		require.NoError(t, err, "pending status with no failing checks should pass")
	})

	t.Run("pending status with failing checks fails", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			switch r.URL.Path {
			case testDoneStatusPath:
				// Pending state but one check is failing
				_, _ = w.Write([]byte(`{"state":"pending","total_count":2,"statuses":[{"state":"failure","context":"check1"},{"state":"pending","context":"check2"}]}`))
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

		prNoMergeable := &github.PullRequest{
			Number:    github.Int(prNum),
			Mergeable: nil,
			Head: &github.PullRequestBranch{
				SHA: github.String(headSHA),
			},
		}
		err := runPRChecks(ctx, client, owner, repo, prNoMergeable, true, false, false)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "required status checks have not passed")
		assert.Contains(t, err.Error(), "check1")
		assert.Contains(t, err.Error(), "failure")
	})
}

func TestBuildDoneCommitMessage(t *testing.T) {
	assert.Equal(t, "014: My feature", buildDoneCommitMessage("{id}: {title}", "014", "My feature"))
	assert.Equal(t, "014 merge: My feature", buildDoneCommitMessage("{id} merge: {title}", "014", "My feature"))
	assert.Equal(t, "no placeholders", buildDoneCommitMessage("no placeholders", "014", "Title"))
}

func TestMergePullRequest(t *testing.T) {
	ctx := context.Background()
	owner, repo := "o", "r"
	pr := &github.PullRequest{Number: github.Int(10)}

	t.Run("calls merge API with strategy and message", func(t *testing.T) {
		var mergeMethod, commitMsg string
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodPut || r.URL.Path != "/api/v3/repos/o/r/pulls/10/merge" {
				w.WriteHeader(http.StatusNotFound)
				return
			}
			var body struct {
				CommitMessage string `json:"commit_message"`
				MergeMethod   string `json:"merge_method"`
			}
			_ = json.NewDecoder(r.Body).Decode(&body)
			mergeMethod = body.MergeMethod
			commitMsg = body.CommitMessage
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"sha":"merged-sha","merged":true,"message":"Merged"}`))
		}))
		defer server.Close()

		baseURL, _ := url.Parse(server.URL + "/api/v3/")
		client := github.NewClient(server.Client())
		client.BaseURL = baseURL

		err := mergePullRequest(ctx, client, owner, repo, pr, "squash", "014: My PR")
		require.NoError(t, err)
		assert.Equal(t, "squash", mergeMethod)
		assert.Equal(t, "014: My PR", commitMsg)
	})

	t.Run("nil PR returns error", func(t *testing.T) {
		client := github.NewClient(nil)
		err := mergePullRequest(ctx, client, owner, repo, nil, "merge", "msg")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid pull request")
	})
}

func TestPullTrunk(t *testing.T) {
	ctx := context.Background()
	t.Run("invalid repo root returns error", func(t *testing.T) {
		err := pullTrunk(ctx, "/nonexistent-dir-kira-done-test", "origin", "main")
		require.Error(t, err)
	})
}

func TestUpdateWorkItemDoneMetadata(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := &config.Config{ConfigDir: tmpDir, Workspace: &config.WorkspaceConfig{WorkFolder: "."}}
	require.NoError(t, os.MkdirAll(filepath.Join(tmpDir, "4_done"), 0o700))
	workPath := filepath.Join(tmpDir, "4_done", "014-done.prd.md")
	content := `---
id: "014"
title: Done item
status: done
kind: prd
created: "2024-01-01"
---

# Content
`
	require.NoError(t, os.WriteFile(workPath, []byte(content), 0o600))

	err := updateWorkItemDoneMetadata(workPath, "2024-06-01T12:00:00Z", "abc123", 42, "squash", cfg)
	require.NoError(t, err)

	frontMatter, _, parseErr := parseWorkItemFrontMatter(workPath, cfg)
	require.NoError(t, parseErr)
	assert.Equal(t, "2024-06-01T12:00:00Z", frontMatter["merged_at"])
	assert.Equal(t, "abc123", frontMatter["merge_commit_sha"])
	assert.Equal(t, 42, frontMatter["pr_number"])
	assert.Equal(t, "squash", frontMatter["merge_strategy"])
}

func TestDeleteBranch(t *testing.T) {
	ctx := context.Background()
	owner, repo := "o", "r"

	t.Run("204 success returns nil", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodDelete || r.URL.Path != testDoneDeleteRefPath {
				w.WriteHeader(http.StatusNotFound)
				return
			}
			w.WriteHeader(http.StatusNoContent)
		}))
		defer server.Close()
		baseURL, _ := url.Parse(server.URL + "/api/v3/")
		client := github.NewClient(server.Client())
		client.BaseURL = baseURL
		err := deleteBranch(ctx, client, owner, repo, "014-feature")
		require.NoError(t, err)
	})

	t.Run("404 already deleted returns nil", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == testDoneDeleteRefPath {
				w.WriteHeader(http.StatusNotFound)
				_, _ = w.Write([]byte(`{"message":"Reference does not exist"}`))
				return
			}
			w.WriteHeader(http.StatusNotFound)
		}))
		defer server.Close()
		baseURL, _ := url.Parse(server.URL + "/api/v3/")
		client := github.NewClient(server.Client())
		client.BaseURL = baseURL
		err := deleteBranch(ctx, client, owner, repo, "014-feature")
		require.NoError(t, err)
	})

	t.Run("422 already deleted returns nil", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == testDoneDeleteRefPath {
				w.WriteHeader(http.StatusUnprocessableEntity)
				_, _ = w.Write([]byte(`{"message":"Reference does not exist"}`))
				return
			}
			w.WriteHeader(http.StatusNotFound)
		}))
		defer server.Close()
		baseURL, _ := url.Parse(server.URL + "/api/v3/")
		client := github.NewClient(server.Client())
		client.BaseURL = baseURL
		err := deleteBranch(ctx, client, owner, repo, "014-feature")
		require.NoError(t, err)
	})

	t.Run("500 returns error", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == testDoneDeleteRefPath {
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			w.WriteHeader(http.StatusNotFound)
		}))
		defer server.Close()
		baseURL, _ := url.Parse(server.URL + "/api/v3/")
		client := github.NewClient(server.Client())
		client.BaseURL = baseURL
		err := deleteBranch(ctx, client, owner, repo, "014-feature")
		require.Error(t, err)
	})
}

func TestDeleteLocalBranch(t *testing.T) {
	ctx := context.Background()

	t.Run("deletes existing branch", func(t *testing.T) {
		tmpDir := t.TempDir()
		require.NoError(t, os.Chdir(tmpDir))
		defer func() { _ = os.Chdir("/") }()

		// Create git repo
		// #nosec G204 - tmpDir is from t.TempDir(), command is fixed
		require.NoError(t, exec.Command("git", "init").Run())
		// #nosec G204 - command is fixed
		require.NoError(t, exec.Command("git", "config", "user.email", "test@example.com").Run())
		// #nosec G204 - command is fixed
		require.NoError(t, exec.Command("git", "config", "user.name", "Test User").Run())
		require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "f"), []byte("x"), 0o600))
		// #nosec G204 - command is fixed
		require.NoError(t, exec.Command("git", "add", "f").Run())
		// #nosec G204 - command is fixed
		require.NoError(t, exec.Command("git", "commit", "-m", "initial").Run())

		// Get the current branch name (default branch may be main, master, etc.)
		// #nosec G204 - command is fixed
		currentBranchOutput, err := exec.Command("git", "rev-parse", "--abbrev-ref", "HEAD").Output()
		require.NoError(t, err)
		defaultBranch := strings.TrimSpace(string(currentBranchOutput))

		// Create and checkout a branch
		// #nosec G204 - command is fixed
		require.NoError(t, exec.Command("git", "checkout", "-b", "test-branch").Run())
		// #nosec G204 - command is fixed
		require.NoError(t, exec.Command("git", "checkout", defaultBranch).Run())

		// Verify branch exists
		// #nosec G204 - command is fixed
		checkCmd := exec.Command("git", "show-ref", "--verify", "--quiet", "refs/heads/test-branch")
		require.NoError(t, checkCmd.Run(), "branch should exist before deletion")

		// Delete branch
		err = deleteLocalBranch(ctx, tmpDir, "test-branch")
		require.NoError(t, err)

		// Verify branch is gone
		// #nosec G204 - command is fixed
		checkCmd = exec.Command("git", "show-ref", "--verify", "--quiet", "refs/heads/test-branch")
		require.Error(t, checkCmd.Run(), "branch should not exist after deletion")
	})

	t.Run("idempotent: returns nil if branch doesn't exist", func(t *testing.T) {
		tmpDir := t.TempDir()
		require.NoError(t, os.Chdir(tmpDir))
		defer func() { _ = os.Chdir("/") }()

		// Create git repo
		// #nosec G204 - tmpDir is from t.TempDir(), command is fixed
		require.NoError(t, exec.Command("git", "init").Run())
		// #nosec G204 - command is fixed
		require.NoError(t, exec.Command("git", "config", "user.email", "test@example.com").Run())
		// #nosec G204 - command is fixed
		require.NoError(t, exec.Command("git", "config", "user.name", "Test User").Run())
		require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "f"), []byte("x"), 0o600))
		// #nosec G204 - command is fixed
		require.NoError(t, exec.Command("git", "add", "f").Run())
		// #nosec G204 - command is fixed
		require.NoError(t, exec.Command("git", "commit", "-m", "initial").Run())

		// Try to delete non-existent branch
		err := deleteLocalBranch(ctx, tmpDir, "non-existent-branch")
		require.NoError(t, err, "should return nil for non-existent branch (idempotent)")
	})
}
