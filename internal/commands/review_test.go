package commands

import (
	"bytes"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"kira/internal/config"
	gh "kira/internal/github"
	"kira/internal/validation"
)

const (
	testWorkItemPath = ".work/3_review/012-submit-for-review.prd.md"
)

func TestReviewCommandRegistration(t *testing.T) {
	t.Run("command is registered in root", func(t *testing.T) {
		// Verify reviewCmd is in the root command's list of commands
		found := false
		for _, cmd := range rootCmd.Commands() {
			if cmd.Name() == "review" {
				found = true
				break
			}
		}
		assert.True(t, found, "review command should be registered in root command")
	})

	t.Run("command appears in help output", func(t *testing.T) {
		// Capture help output
		buf := new(bytes.Buffer)
		rootCmd.SetOut(buf)
		rootCmd.SetErr(buf)

		err := rootCmd.Help()
		require.NoError(t, err)

		output := buf.String()
		assert.Contains(t, output, "review", "help output should contain 'review' command")
	})

	t.Run("review command shows help", func(t *testing.T) {
		// Capture help output for review command
		buf := new(bytes.Buffer)
		reviewCmd.SetOut(buf)
		reviewCmd.SetErr(buf)

		err := reviewCmd.Help()
		require.NoError(t, err)

		output := buf.String()
		// Just verify the help output is not empty and contains "review"
		assert.NotEmpty(t, output, "help output should not be empty")
		assert.Contains(t, output, "review", "review command help should contain command name")
		// Verify it contains some expected content
		assert.True(t,
			strings.Contains(output, "Submit work item for review") ||
				strings.Contains(output, "Automatically derives work item ID"),
			"help should contain command description")
	})
}

func TestReviewCommandFlagDefaults(t *testing.T) {
	t.Run("draft flag defaults to true", func(t *testing.T) {
		cmd := createTestReviewCmd()
		cmd.SetArgs([]string{})

		err := cmd.ParseFlags([]string{})
		require.NoError(t, err)

		draft, err := cmd.Flags().GetBool("draft")
		require.NoError(t, err)
		assert.True(t, draft, "--draft should default to true")
	})

	t.Run("no-trunk-update flag defaults to false", func(t *testing.T) {
		cmd := createTestReviewCmd()
		cmd.SetArgs([]string{})

		err := cmd.ParseFlags([]string{})
		require.NoError(t, err)

		noTrunkUpdate, err := cmd.Flags().GetBool("no-trunk-update")
		require.NoError(t, err)
		assert.False(t, noTrunkUpdate, "--no-trunk-update should default to false")
	})

	t.Run("no-rebase flag defaults to false", func(t *testing.T) {
		cmd := createTestReviewCmd()
		cmd.SetArgs([]string{})

		err := cmd.ParseFlags([]string{})
		require.NoError(t, err)

		noRebase, err := cmd.Flags().GetBool("no-rebase")
		require.NoError(t, err)
		assert.False(t, noRebase, "--no-rebase should default to false")
	})

	t.Run("reviewer flag defaults to empty slice", func(t *testing.T) {
		cmd := createTestReviewCmd()
		cmd.SetArgs([]string{})

		err := cmd.ParseFlags([]string{})
		require.NoError(t, err)

		reviewers, err := cmd.Flags().GetStringArray("reviewer")
		require.NoError(t, err)
		assert.Empty(t, reviewers, "--reviewer should default to empty slice")
	})

	t.Run("title flag defaults to empty string", func(t *testing.T) {
		cmd := createTestReviewCmd()
		cmd.SetArgs([]string{})

		err := cmd.ParseFlags([]string{})
		require.NoError(t, err)

		title, err := cmd.Flags().GetString("title")
		require.NoError(t, err)
		assert.Empty(t, title, "--title should default to empty string")
	})

	t.Run("description flag defaults to empty string", func(t *testing.T) {
		cmd := createTestReviewCmd()
		cmd.SetArgs([]string{})

		err := cmd.ParseFlags([]string{})
		require.NoError(t, err)

		description, err := cmd.Flags().GetString("description")
		require.NoError(t, err)
		assert.Empty(t, description, "--description should default to empty string")
	})
}

func TestReviewCommandFlagParsing(t *testing.T) {
	t.Run("draft flag can be set to false", func(t *testing.T) {
		cmd := createTestReviewCmd()
		cmd.SetArgs([]string{"--draft=false"})

		err := cmd.ParseFlags([]string{"--draft=false"})
		require.NoError(t, err)

		draft, err := cmd.Flags().GetBool("draft")
		require.NoError(t, err)
		assert.False(t, draft, "--draft=false should set draft to false")
	})

	t.Run("draft flag can be set to true explicitly", func(t *testing.T) {
		cmd := createTestReviewCmd()
		cmd.SetArgs([]string{"--draft=true"})

		err := cmd.ParseFlags([]string{"--draft=true"})
		require.NoError(t, err)

		draft, err := cmd.Flags().GetBool("draft")
		require.NoError(t, err)
		assert.True(t, draft, "--draft=true should set draft to true")
	})

	t.Run("no-trunk-update flag can be set to true", func(t *testing.T) {
		cmd := createTestReviewCmd()
		cmd.SetArgs([]string{"--no-trunk-update"})

		err := cmd.ParseFlags([]string{"--no-trunk-update"})
		require.NoError(t, err)

		noTrunkUpdate, err := cmd.Flags().GetBool("no-trunk-update")
		require.NoError(t, err)
		assert.True(t, noTrunkUpdate, "--no-trunk-update should set flag to true")
	})

	t.Run("no-rebase flag can be set to true", func(t *testing.T) {
		cmd := createTestReviewCmd()
		cmd.SetArgs([]string{"--no-rebase"})

		err := cmd.ParseFlags([]string{"--no-rebase"})
		require.NoError(t, err)

		noRebase, err := cmd.Flags().GetBool("no-rebase")
		require.NoError(t, err)
		assert.True(t, noRebase, "--no-rebase should set flag to true")
	})

	t.Run("title flag accepts string value", func(t *testing.T) {
		cmd := createTestReviewCmd()
		testTitle := "Custom PR Title"
		cmd.SetArgs([]string{"--title", testTitle})

		err := cmd.ParseFlags([]string{"--title", testTitle})
		require.NoError(t, err)

		title, err := cmd.Flags().GetString("title")
		require.NoError(t, err)
		assert.Equal(t, testTitle, title, "--title should accept string value")
	})

	t.Run("description flag accepts string value", func(t *testing.T) {
		cmd := createTestReviewCmd()
		testDescription := "Custom PR Description"
		cmd.SetArgs([]string{"--description", testDescription})

		err := cmd.ParseFlags([]string{"--description", testDescription})
		require.NoError(t, err)

		description, err := cmd.Flags().GetString("description")
		require.NoError(t, err)
		assert.Equal(t, testDescription, description, "--description should accept string value")
	})

	t.Run("reviewer flag accepts single value", func(t *testing.T) {
		cmd := createTestReviewCmd()
		cmd.SetArgs([]string{"--reviewer", "user1"})

		err := cmd.ParseFlags([]string{"--reviewer", "user1"})
		require.NoError(t, err)

		reviewers, err := cmd.Flags().GetStringArray("reviewer")
		require.NoError(t, err)
		assert.Len(t, reviewers, 1, "should have one reviewer")
		assert.Equal(t, "user1", reviewers[0], "reviewer should be 'user1'")
	})

	t.Run("reviewer flag accepts multiple values", func(t *testing.T) {
		cmd := createTestReviewCmd()
		cmd.SetArgs([]string{"--reviewer", "user1", "--reviewer", "user2"})

		err := cmd.ParseFlags([]string{"--reviewer", "user1", "--reviewer", "user2"})
		require.NoError(t, err)

		reviewers, err := cmd.Flags().GetStringArray("reviewer")
		require.NoError(t, err)
		assert.Len(t, reviewers, 2, "should have two reviewers")
		assert.Contains(t, reviewers, "user1", "should contain user1")
		assert.Contains(t, reviewers, "user2", "should contain user2")
	})

	t.Run("reviewer flag accepts email addresses", func(t *testing.T) {
		cmd := createTestReviewCmd()
		cmd.SetArgs([]string{"--reviewer", "user@example.com"})

		err := cmd.ParseFlags([]string{"--reviewer", "user@example.com"})
		require.NoError(t, err)

		reviewers, err := cmd.Flags().GetStringArray("reviewer")
		require.NoError(t, err)
		assert.Len(t, reviewers, 1, "should have one reviewer")
		assert.Equal(t, "user@example.com", reviewers[0], "reviewer should be email address")
	})

	t.Run("all flags can be used together", func(t *testing.T) {
		cmd := createTestReviewCmd()
		args := []string{
			"--reviewer", "user1",
			"--reviewer", "user2",
			"--draft=false",
			"--no-trunk-update",
			"--no-rebase",
			"--title", "Test Title",
			"--description", "Test Description",
		}
		cmd.SetArgs(args)

		err := cmd.ParseFlags(args)
		require.NoError(t, err)

		reviewers, _ := cmd.Flags().GetStringArray("reviewer")
		draft, _ := cmd.Flags().GetBool("draft")
		noTrunkUpdate, _ := cmd.Flags().GetBool("no-trunk-update")
		noRebase, _ := cmd.Flags().GetBool("no-rebase")
		title, _ := cmd.Flags().GetString("title")
		description, _ := cmd.Flags().GetString("description")

		assert.Len(t, reviewers, 2)
		assert.False(t, draft)
		assert.True(t, noTrunkUpdate)
		assert.True(t, noRebase)
		assert.Equal(t, "Test Title", title)
		assert.Equal(t, "Test Description", description)
	})
}

func TestReviewCommandNoArgs(t *testing.T) {
	t.Run("command accepts no positional arguments", func(t *testing.T) {
		// Create a command with NoArgs validation
		cmd := &cobra.Command{
			Use:  "review",
			Args: cobra.NoArgs,
			RunE: func(cmd *cobra.Command, _ []string) error {
				return cmd.Help()
			},
		}
		cmd.SetArgs([]string{"invalid-arg"})

		err := cmd.Execute()
		require.Error(t, err, "should error when positional arguments provided")
		assert.Contains(t, err.Error(), "unknown", "error should indicate unknown command or argument")
	})
}

// TestReviewCommandIntegration tests the command in a more realistic scenario
func TestReviewCommandIntegration(t *testing.T) {
	t.Run("command can be executed via root", func(t *testing.T) {
		// Verify review command is accessible from root
		// This test verifies the command is properly registered
		found := false
		for _, cmd := range rootCmd.Commands() {
			if cmd.Name() == "review" {
				found = true
				// Verify it has the expected flags
				assert.True(t, cmd.Flags().Lookup("draft") != nil, "draft flag should exist")
				assert.True(t, cmd.Flags().Lookup("reviewer") != nil, "reviewer flag should exist")
				break
			}
		}
		assert.True(t, found, "review command should be accessible from root")
	})
}

// Helper function to create a fresh command instance for testing
func createTestReviewCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use: "review",
		RunE: func(cmd *cobra.Command, _ []string) error {
			return cmd.Help()
		},
	}
	cmd.Flags().StringArray("reviewer", []string{}, "Specify reviewer")
	cmd.Flags().Bool("draft", true, "Create as draft PR")
	cmd.Flags().Bool("no-trunk-update", false, "Skip updating trunk branch status")
	cmd.Flags().Bool("no-rebase", false, "Skip rebasing current branch")
	cmd.Flags().String("title", "", "Custom PR title")
	cmd.Flags().String("description", "", "Custom PR description")
	return cmd
}

func TestReviewCommandWithFreshInstance(t *testing.T) {
	t.Run("fresh command instance has correct defaults", func(t *testing.T) {
		cmd := createTestReviewCmd()
		cmd.SetArgs([]string{})

		// Parse flags
		err := cmd.ParseFlags([]string{})
		require.NoError(t, err)

		draft, _ := cmd.Flags().GetBool("draft")
		reviewers, _ := cmd.Flags().GetStringArray("reviewer")

		assert.True(t, draft, "draft should default to true")
		assert.Empty(t, reviewers, "reviewers should default to empty")
	})
}

// TestDeriveWorkItemFromBranch tests the deriveWorkItemFromBranch function
func TestDeriveWorkItemFromBranch(t *testing.T) {
	t.Run("extracts ID from valid branch names", func(t *testing.T) {
		testCases := []struct {
			branchName string
			expectedID string
		}{
			{"012-submit-for-review", "012"},
			{"001-feature-name", "001"},
			{"999-long-branch-name-with-many-dashes", "999"},
			{"123-simple", "123"},
			{"000-test", "000"},
		}

		for _, tc := range testCases {
			t.Run(tc.branchName, func(t *testing.T) {
				id, err := deriveWorkItemFromBranch(tc.branchName)
				require.NoError(t, err, "should extract ID from valid branch name")
				assert.Equal(t, tc.expectedID, id, "extracted ID should match expected")
			})
		}
	})

	t.Run("returns error for branch without dash", func(t *testing.T) {
		testCases := []string{
			"012submit",
			"012",
			"no-dash-here-but-wait-there-is-one",
		}

		for _, branchName := range testCases {
			// Only test cases that actually don't have a dash
			if !strings.Contains(branchName, "-") {
				t.Run(branchName, func(t *testing.T) {
					_, err := deriveWorkItemFromBranch(branchName)
					require.Error(t, err, "should return error for branch without dash")
					assert.Contains(t, err.Error(), "does not follow kira naming convention", "error should mention naming convention")
				})
			}
		}
	})

	t.Run("returns error for invalid ID formats", func(t *testing.T) {
		testCases := []struct {
			branchName  string
			description string
		}{
			{"12-feature", "ID not 3 digits (2 digits)"},
			{"1234-feature", "ID not 3 digits (4 digits)"},
			{"abc-feature", "ID contains letters"},
			{"01a-feature", "ID contains letters"},
			{"1-feature", "ID is single digit"},
			{"a12-feature", "ID starts with letter"},
			{"12a-feature", "ID ends with letter"},
		}

		for _, tc := range testCases {
			t.Run(tc.description, func(t *testing.T) {
				_, err := deriveWorkItemFromBranch(tc.branchName)
				require.Error(t, err, "should return error for invalid ID format")
				assert.Contains(t, err.Error(), "invalid work item ID", "error should mention invalid ID")
			})
		}
	})

	t.Run("returns error for empty branch name", func(t *testing.T) {
		_, err := deriveWorkItemFromBranch("")
		require.Error(t, err, "should return error for empty branch name")
		assert.Contains(t, err.Error(), "cannot be empty", "error should mention empty")
	})

	t.Run("returns error when ID is missing before dash", func(t *testing.T) {
		_, err := deriveWorkItemFromBranch("-feature-name")
		require.Error(t, err, "should return error when ID is missing")
		assert.Contains(t, err.Error(), "work item ID is missing", "error should mention missing ID")
	})
}

// setupTestGitRepo creates a temporary git repository for testing
func setupTestGitRepo(t *testing.T, branchName string) string {
	tmpDir := t.TempDir()
	originalDir, err := os.Getwd()
	require.NoError(t, err)
	require.NoError(t, os.Chdir(tmpDir))
	t.Cleanup(func() {
		_ = os.Chdir(originalDir)
	})

	// Initialize git repository
	require.NoError(t, exec.Command("git", "init").Run())
	require.NoError(t, exec.Command("git", "config", "user.email", "test@example.com").Run())
	require.NoError(t, exec.Command("git", "config", "user.name", "Test User").Run())

	// Create initial commit on default branch (main or master)
	require.NoError(t, os.WriteFile("test.txt", []byte("test"), 0o600))
	require.NoError(t, exec.Command("git", "add", "test.txt").Run())
	require.NoError(t, exec.Command("git", "commit", "-m", "Initial commit").Run())

	// Create and checkout specified branch if provided and different from current branch
	if branchName != "" {
		// Get current branch name
		currentBranchCmd := exec.Command("git", "rev-parse", "--abbrev-ref", "HEAD")
		currentBranchOutput, err := currentBranchCmd.Output()
		require.NoError(t, err)
		currentBranch := strings.TrimSpace(string(currentBranchOutput))

		// Only create new branch if it's different from current branch
		if branchName != currentBranch {
			require.NoError(t, exec.Command("git", "checkout", "-b", branchName).Run())
		}
	}

	return tmpDir
}

// TestValidateBranchContext tests the validateBranchContext function
func TestValidateBranchContext(t *testing.T) {
	t.Run("rejects trunk branch (main)", func(t *testing.T) {
		tmpDir := setupTestGitRepo(t, "main")
		_ = tmpDir // Use tmpDir to avoid unused variable

		cfg := &config.Config{
			Git: &config.GitConfig{
				TrunkBranch: "main",
			},
		}

		err := validateBranchContext(cfg)
		require.Error(t, err, "should reject trunk branch")
		assert.Contains(t, err.Error(), "cannot run 'kira review' on trunk branch", "error should mention trunk branch")
		assert.Contains(t, err.Error(), "main", "error should mention branch name")
	})

	t.Run("rejects trunk branch (master)", func(t *testing.T) {
		// Create repo and switch to master
		tmpDir := t.TempDir()
		originalDir, err := os.Getwd()
		require.NoError(t, err)
		require.NoError(t, os.Chdir(tmpDir))
		defer func() { _ = os.Chdir(originalDir) }()

		require.NoError(t, exec.Command("git", "init").Run())
		require.NoError(t, exec.Command("git", "config", "user.email", "test@example.com").Run())
		require.NoError(t, exec.Command("git", "config", "user.name", "Test User").Run())
		require.NoError(t, exec.Command("git", "checkout", "-b", "master").Run())
		require.NoError(t, os.WriteFile("test.txt", []byte("test"), 0o600))
		require.NoError(t, exec.Command("git", "add", "test.txt").Run())
		require.NoError(t, exec.Command("git", "commit", "-m", "Initial commit").Run())

		cfg := &config.Config{
			Git: &config.GitConfig{
				TrunkBranch: "master",
			},
		}

		err = validateBranchContext(cfg)
		require.Error(t, err, "should reject trunk branch")
		assert.Contains(t, err.Error(), "cannot run 'kira review' on trunk branch", "error should mention trunk branch")
		assert.Contains(t, err.Error(), "master", "error should mention branch name")
	})

	t.Run("accepts feature branch", func(t *testing.T) {
		tmpDir := setupTestGitRepo(t, "012-test-feature")
		_ = tmpDir // Use tmpDir to avoid unused variable

		cfg := &config.Config{
			Git: &config.GitConfig{
				TrunkBranch: "main",
			},
		}

		err := validateBranchContext(cfg)
		assert.NoError(t, err, "should accept feature branch")
	})

	t.Run("handles auto-detected trunk (main)", func(t *testing.T) {
		tmpDir := setupTestGitRepo(t, "main")
		_ = tmpDir // Use tmpDir to avoid unused variable

		// Config with empty TrunkBranch should auto-detect main
		cfg := &config.Config{
			Git: &config.GitConfig{
				TrunkBranch: "", // Empty means auto-detect
			},
		}

		err := validateBranchContext(cfg)
		require.Error(t, err, "should reject trunk branch when auto-detected")
		assert.Contains(t, err.Error(), "cannot run 'kira review' on trunk branch", "error should mention trunk branch")
	})

	t.Run("handles auto-detected trunk (master)", func(t *testing.T) {
		// Create repo with master branch (no main)
		tmpDir := t.TempDir()
		originalDir, err := os.Getwd()
		require.NoError(t, err)
		require.NoError(t, os.Chdir(tmpDir))
		defer func() { _ = os.Chdir(originalDir) }()

		require.NoError(t, exec.Command("git", "init").Run())
		require.NoError(t, exec.Command("git", "config", "user.email", "test@example.com").Run())
		require.NoError(t, exec.Command("git", "config", "user.name", "Test User").Run())
		require.NoError(t, exec.Command("git", "checkout", "-b", "master").Run())
		require.NoError(t, os.WriteFile("test.txt", []byte("test"), 0o600))
		require.NoError(t, exec.Command("git", "add", "test.txt").Run())
		require.NoError(t, exec.Command("git", "commit", "-m", "Initial commit").Run())

		// Config with empty TrunkBranch should auto-detect master
		cfg := &config.Config{
			Git: &config.GitConfig{
				TrunkBranch: "", // Empty means auto-detect
			},
		}

		err = validateBranchContext(cfg)
		require.Error(t, err, "should reject trunk branch when auto-detected")
		assert.Contains(t, err.Error(), "cannot run 'kira review' on trunk branch", "error should mention trunk branch")
	})

	t.Run("accepts feature branch with auto-detected trunk", func(t *testing.T) {
		tmpDir := setupTestGitRepo(t, "012-test-feature")
		_ = tmpDir // Use tmpDir to avoid unused variable

		// Config with empty TrunkBranch should auto-detect main
		cfg := &config.Config{
			Git: &config.GitConfig{
				TrunkBranch: "", // Empty means auto-detect
			},
		}

		err := validateBranchContext(cfg)
		assert.NoError(t, err, "should accept feature branch when trunk is auto-detected")
	})
}

func TestLoadWorkItem(t *testing.T) {
	t.Run("loads work item successfully", func(t *testing.T) {
		tmpDir := t.TempDir()
		require.NoError(t, os.Chdir(tmpDir))
		defer func() { _ = os.Chdir("/") }()

		cfg := &config.DefaultConfig

		// Create .work directory structure and a test work item
		require.NoError(t, os.MkdirAll(".work/1_todo", 0o700))
		require.NoError(t, os.WriteFile(testFilePath, []byte(testWorkItemContent), 0o600))

		workItem, path, err := loadWorkItem(cfg, "001")
		require.NoError(t, err)
		require.NotNil(t, workItem)
		assert.Equal(t, testFilePath, path)
		assert.Equal(t, "001", workItem.ID)
		assert.Equal(t, "Test Feature", workItem.Title)
		assert.Equal(t, "todo", workItem.Status)
		assert.Equal(t, "prd", workItem.Kind)
		assert.Equal(t, "2024-01-01", workItem.Created)
	})

	t.Run("returns not found error when work item does not exist", func(t *testing.T) {
		tmpDir := t.TempDir()
		require.NoError(t, os.Chdir(tmpDir))
		defer func() { _ = os.Chdir("/") }()

		cfg := &config.DefaultConfig

		// Create empty .work tree without any matching work items
		require.NoError(t, os.MkdirAll(".work/1_todo", 0o700))

		_, _, err := loadWorkItem(cfg, "001")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "work item 001 not found")
	})
}

func TestValidateWorkItemStatusForReview(t *testing.T) {
	cfg := &config.DefaultConfig

	t.Run("accepts todo and doing statuses", func(t *testing.T) {
		for _, status := range []string{"todo", "doing"} {
			workItem := &validation.WorkItem{Status: status}
			err := validateWorkItemStatusForReview(workItem, cfg)
			assert.NoError(t, err, "status %s should be accepted", status)
		}
	})

	t.Run("treats review status as already in review", func(t *testing.T) {
		workItem := &validation.WorkItem{Status: "review"}
		err := validateWorkItemStatusForReview(workItem, cfg)
		require.Error(t, err)

		var alreadyErr *alreadyInReviewError
		assert.True(t, errors.As(err, &alreadyErr), "error should be of type alreadyInReviewError")
		assert.Equal(t, "Work item is already in review status.", alreadyErr.Error())
	})

	t.Run("rejects non-reviewable statuses", func(t *testing.T) {
		testCases := []string{"backlog", "done", "archived"}
		for _, status := range testCases {
			workItem := &validation.WorkItem{Status: status}
			err := validateWorkItemStatusForReview(workItem, cfg)
			require.Error(t, err, "status %s should be rejected", status)
			assert.Contains(t, err.Error(), "cannot submit for review")
			assert.Contains(t, err.Error(), status)
		}
	})

	t.Run("rejects empty status", func(t *testing.T) {
		workItem := &validation.WorkItem{Status: ""}
		err := validateWorkItemStatusForReview(workItem, cfg)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "work item has empty status")
	})
}

func TestValidateRequiredFieldsForReview(t *testing.T) {
	t.Run("passes when all required fields are present", func(t *testing.T) {
		cfg := config.DefaultConfig
		workItem := &validation.WorkItem{
			ID:      "001",
			Title:   "Test Feature",
			Status:  "doing",
			Kind:    "prd",
			Created: "2024-01-01",
			Fields:  map[string]interface{}{},
		}

		err := validateRequiredFieldsForReview(workItem, &cfg)
		assert.NoError(t, err)
	})

	t.Run("reports missing core fields", func(t *testing.T) {
		cfg := config.DefaultConfig
		workItem := &validation.WorkItem{
			ID:      "001",
			Title:   "",
			Status:  "",
			Kind:    "prd",
			Created: "2024-01-01",
			Fields:  map[string]interface{}{},
		}

		err := validateRequiredFieldsForReview(workItem, &cfg)
		require.Error(t, err)
		msg := err.Error()
		assert.Contains(t, msg, "work item missing required fields:")
		assert.Contains(t, msg, "title")
		assert.Contains(t, msg, "status")
		assert.Contains(t, msg, "Update work item and try again")
	})

	t.Run("honors additional required custom fields", func(t *testing.T) {
		cfg := config.DefaultConfig
		cfg.Validation.RequiredFields = append(cfg.Validation.RequiredFields, "review_pr_url")

		workItem := &validation.WorkItem{
			ID:      "001",
			Title:   "Test Feature",
			Status:  "doing",
			Kind:    "prd",
			Created: "2024-01-01",
			Fields: map[string]interface{}{
				"review_pr_url": "https://example.com/pr/1",
			},
		}

		err := validateRequiredFieldsForReview(workItem, &cfg)
		assert.NoError(t, err)

		// Now test missing custom field
		workItemMissing := &validation.WorkItem{
			ID:      "001",
			Title:   "Test Feature",
			Status:  "doing",
			Kind:    "prd",
			Created: "2024-01-01",
			Fields:  map[string]interface{}{},
		}

		err = validateRequiredFieldsForReview(workItemMissing, &cfg)
		require.Error(t, err)
		msg := err.Error()
		assert.Contains(t, msg, "review_pr_url")
	})
}

func TestCheckUncommittedChangesForReview(t *testing.T) {
	t.Run("returns nil when repository is clean", func(t *testing.T) {
		tmpDir := setupTestGitRepo(t, "main")
		_ = tmpDir // avoid unused variable warning

		err := checkUncommittedChangesForReview()
		assert.NoError(t, err, "expected no error for clean repository")
	})

	t.Run("returns error when repository has uncommitted changes", func(t *testing.T) {
		tmpDir := setupTestGitRepo(t, "main")
		_ = tmpDir // avoid unused variable warning

		// Introduce an uncommitted change
		require.NoError(t, os.WriteFile("test.txt", []byte("modified"), 0o600))

		err := checkUncommittedChangesForReview()
		require.Error(t, err, "expected error when uncommitted changes are present")
		assert.Contains(t, err.Error(), "uncommitted changes detected. Commit or stash changes before submitting for review")
	})
}

func TestUpdateWorkItemStatusOnCurrentBranch(t *testing.T) {
	t.Run("moves work item from todo to review and updates status", func(t *testing.T) {
		tmpDir := t.TempDir()
		require.NoError(t, os.Chdir(tmpDir))
		defer func() { _ = os.Chdir("/") }()

		cfg := &config.DefaultConfig

		// Create .work directory structure
		require.NoError(t, os.MkdirAll(".work/1_todo", 0o700))
		require.NoError(t, os.MkdirAll(".work/3_review", 0o700))

		// Seed a todo work item
		const workItemID = "001"
		const fileName = "001-test-feature.prd.md"
		sourcePath := ".work/1_todo/" + fileName
		require.NoError(t, os.WriteFile(sourcePath, []byte(testWorkItemContent), 0o600))

		// Act
		err := updateWorkItemStatusOnCurrentBranch(cfg, workItemID, statusReview)
		require.NoError(t, err)

		// Assert: file moved
		targetPath := ".work/3_review/" + fileName
		_, err = os.Stat(targetPath)
		require.NoError(t, err)

		// Old path no longer exists
		_, err = os.Stat(sourcePath)
		require.Error(t, err)
		assert.True(t, os.IsNotExist(err))

		// Status updated in front matter
		content, err := os.ReadFile(targetPath)
		require.NoError(t, err)
		text := string(content)
		assert.Contains(t, text, "status: review")
		assert.NotContains(t, text, "status: todo")
	})

	t.Run("moves work item from doing to review and preserves other front matter", func(t *testing.T) {
		tmpDir := t.TempDir()
		require.NoError(t, os.Chdir(tmpDir))
		defer func() { _ = os.Chdir("/") }()

		cfg := &config.DefaultConfig

		// Create .work directory structure
		require.NoError(t, os.MkdirAll(".work/2_doing", 0o700))
		require.NoError(t, os.MkdirAll(".work/3_review", 0o700))

		const workItemID = "002"
		const fileName = "002-another-feature.prd.md"
		sourcePath := ".work/2_doing/" + fileName

		// Work item with richer front matter to verify preservation
		workItemContent := `---
id: 002
title: Another Feature
status: doing
kind: prd
created: 2024-01-02
tags: [github, notifications]
custom_field: custom-value
---

# Another Feature
`
		require.NoError(t, os.WriteFile(sourcePath, []byte(workItemContent), 0o600))

		// Act
		err := updateWorkItemStatusOnCurrentBranch(cfg, workItemID, statusReview)
		require.NoError(t, err)

		// Assert: file moved
		targetPath := ".work/3_review/" + fileName
		_, err = os.Stat(targetPath)
		require.NoError(t, err)

		// Front matter fields preserved (except status value)
		content, err := os.ReadFile(targetPath)
		require.NoError(t, err)
		text := string(content)

		assert.Contains(t, text, "id: 002")
		assert.Contains(t, text, "title: Another Feature")
		assert.Contains(t, text, "kind: prd")
		assert.Contains(t, text, "created: 2024-01-02")
		assert.Contains(t, text, "tags: [github, notifications]")
		assert.Contains(t, text, "custom_field: custom-value")
		assert.Contains(t, text, "status: review")
		assert.NotContains(t, text, "status: doing")
	})

	t.Run("returns error when target status is not configured", func(t *testing.T) {
		tmpDir := t.TempDir()
		require.NoError(t, os.Chdir(tmpDir))
		defer func() { _ = os.Chdir("/") }()

		// Start from default config but clear StatusFolders to simulate misconfiguration
		cfg := config.DefaultConfig
		cfg.StatusFolders = map[string]string{
			"todo":  "1_todo",
			"doing": "2_doing",
			// intentionally omit "review"
		}

		// Minimal .work tree with a matching work item to satisfy path validation
		require.NoError(t, os.MkdirAll(".work/1_todo", 0o700))
		const workItemID = "003"
		const fileName = "003-missing-review-status.prd.md"
		sourcePath := ".work/1_todo/" + fileName
		workItemContent := `---
id: 003
title: Missing Review Status
status: todo
kind: prd
created: 2024-01-03
---

# Missing Review Status
`
		require.NoError(t, os.WriteFile(sourcePath, []byte(workItemContent), 0o600))

		err := updateWorkItemStatusOnCurrentBranch(&cfg, workItemID, statusReview)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid target status")
	})

	t.Run("returns error when work item does not exist", func(t *testing.T) {
		tmpDir := t.TempDir()
		require.NoError(t, os.Chdir(tmpDir))
		defer func() { _ = os.Chdir("/") }()

		cfg := &config.DefaultConfig

		// Empty .work tree without matching work item
		require.NoError(t, os.MkdirAll(".work/1_todo", 0o700))
		require.NoError(t, os.MkdirAll(".work/3_review", 0o700))

		err := updateWorkItemStatusOnCurrentBranch(cfg, "999", statusReview)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "work item with ID 999 not found")
	})
}

func TestValidateRemoteExists(t *testing.T) {
	t.Run("validates existing remote", func(t *testing.T) {
		tmpDir := setupTestGitRepo(t, "main")
		_ = tmpDir // avoid unused variable warning

		// Add a remote
		require.NoError(t, exec.Command("git", "remote", "add", "origin", "https://github.com/test/repo.git").Run())

		cfg := &config.Config{
			Git: &config.GitConfig{
				Remote: "origin",
			},
		}

		err := validateRemoteExists(cfg)
		assert.NoError(t, err, "should validate existing remote")
	})

	t.Run("validates default origin remote", func(t *testing.T) {
		tmpDir := setupTestGitRepo(t, "main")
		_ = tmpDir // avoid unused variable warning

		// Add origin remote
		require.NoError(t, exec.Command("git", "remote", "add", "origin", "https://github.com/test/repo.git").Run())

		cfg := &config.Config{
			Git: &config.GitConfig{
				Remote: "", // Empty should default to "origin"
			},
		}

		err := validateRemoteExists(cfg)
		assert.NoError(t, err, "should validate default origin remote")
	})

	t.Run("validates custom remote from config", func(t *testing.T) {
		tmpDir := setupTestGitRepo(t, "main")
		_ = tmpDir // avoid unused variable warning

		// Add a custom remote
		require.NoError(t, exec.Command("git", "remote", "add", "upstream", "https://github.com/test/repo.git").Run())

		cfg := &config.Config{
			Git: &config.GitConfig{
				Remote: "upstream",
			},
		}

		err := validateRemoteExists(cfg)
		assert.NoError(t, err, "should validate custom remote")
	})

	t.Run("returns error for non-existent remote", func(t *testing.T) {
		tmpDir := setupTestGitRepo(t, "main")
		_ = tmpDir // avoid unused variable warning

		cfg := &config.Config{
			Git: &config.GitConfig{
				Remote: "nonexistent",
			},
		}

		err := validateRemoteExists(cfg)
		require.Error(t, err, "should return error for non-existent remote")
		assert.Contains(t, err.Error(), "GitHub remote 'nonexistent' not configured")
	})

	t.Run("returns error for nil config", func(t *testing.T) {
		err := validateRemoteExists(nil)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "configuration cannot be nil")
	})
}

func TestCheckBranchOnRemote(t *testing.T) {
	t.Run("returns true when branch exists on remote", func(t *testing.T) {
		tmpDir := setupTestGitRepo(t, "012-test-branch")
		_ = tmpDir // avoid unused variable warning

		// Create a bare repository to simulate remote
		remoteDir := t.TempDir()
		// #nosec G204 - remoteDir is from t.TempDir() which is safe
		require.NoError(t, exec.Command("git", "init", "--bare", remoteDir).Run())

		// Add remote
		// #nosec G204 - remoteDir is from t.TempDir() which is safe
		require.NoError(t, exec.Command("git", "remote", "add", "origin", remoteDir).Run())

		// Push branch to remote
		require.NoError(t, exec.Command("git", "push", "-u", "origin", "012-test-branch").Run())

		cfg := &config.Config{
			Git: &config.GitConfig{
				Remote: "origin",
			},
		}

		exists, err := checkBranchOnRemote("012-test-branch", cfg)
		require.NoError(t, err)
		assert.True(t, exists, "branch should exist on remote")
	})

	t.Run("returns false when branch does not exist on remote", func(t *testing.T) {
		tmpDir := setupTestGitRepo(t, "012-test-branch")
		_ = tmpDir // avoid unused variable warning

		// Create a bare repository to simulate remote
		remoteDir := t.TempDir()
		// #nosec G204 - remoteDir is from t.TempDir() which is safe
		require.NoError(t, exec.Command("git", "init", "--bare", remoteDir).Run())

		// Add remote
		// #nosec G204 - remoteDir is from t.TempDir() which is safe
		require.NoError(t, exec.Command("git", "remote", "add", "origin", remoteDir).Run())

		// Don't push branch - it won't exist on remote

		cfg := &config.Config{
			Git: &config.GitConfig{
				Remote: "origin",
			},
		}

		exists, err := checkBranchOnRemote("012-test-branch", cfg)
		require.NoError(t, err)
		assert.False(t, exists, "branch should not exist on remote")
	})

	t.Run("returns error for invalid remote", func(t *testing.T) {
		tmpDir := setupTestGitRepo(t, "012-test-branch")
		_ = tmpDir // avoid unused variable warning

		cfg := &config.Config{
			Git: &config.GitConfig{
				Remote: "nonexistent",
			},
		}

		_, err := checkBranchOnRemote("012-test-branch", cfg)
		require.Error(t, err)
		// Error message may vary - check for key indicators
		assert.True(t,
			strings.Contains(err.Error(), "does not exist") ||
				strings.Contains(err.Error(), "does not appear to be a git repository") ||
				strings.Contains(err.Error(), "Could not read from remote"),
			"should indicate remote doesn't exist or is invalid: %s", err.Error())
	})

	t.Run("returns error for empty branch name", func(t *testing.T) {
		cfg := &config.Config{
			Git: &config.GitConfig{
				Remote: "origin",
			},
		}

		_, err := checkBranchOnRemote("", cfg)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "branch name cannot be empty")
	})

	t.Run("returns error for nil config", func(t *testing.T) {
		_, err := checkBranchOnRemote("012-test-branch", nil)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "configuration cannot be nil")
	})
}

func TestCheckBranchDiverged(t *testing.T) {
	t.Run("returns false when branch not on remote", func(t *testing.T) {
		tmpDir := setupTestGitRepo(t, "012-test-branch")
		_ = tmpDir // avoid unused variable warning

		// Create a bare repository to simulate remote
		remoteDir := t.TempDir()
		// #nosec G204 - remoteDir is from t.TempDir() which is safe
		require.NoError(t, exec.Command("git", "init", "--bare", remoteDir).Run())

		// Add remote but don't push branch
		// #nosec G204 - remoteDir is from t.TempDir() which is safe
		require.NoError(t, exec.Command("git", "remote", "add", "origin", remoteDir).Run())

		cfg := &config.Config{
			Git: &config.GitConfig{
				Remote: "origin",
			},
		}

		diverged, err := checkBranchDiverged("012-test-branch", cfg)
		require.NoError(t, err)
		assert.False(t, diverged, "branch not on remote should not be considered diverged")
	})

	t.Run("returns false when branch up-to-date with remote", func(t *testing.T) {
		tmpDir := setupTestGitRepo(t, "012-test-branch")
		_ = tmpDir // avoid unused variable warning

		// Create a bare repository to simulate remote
		remoteDir := t.TempDir()
		// #nosec G204 - remoteDir is from t.TempDir() which is safe
		require.NoError(t, exec.Command("git", "init", "--bare", remoteDir).Run())

		// Add remote and push branch
		// #nosec G204 - remoteDir is from t.TempDir() which is safe
		require.NoError(t, exec.Command("git", "remote", "add", "origin", remoteDir).Run())
		require.NoError(t, exec.Command("git", "push", "-u", "origin", "012-test-branch").Run())

		cfg := &config.Config{
			Git: &config.GitConfig{
				Remote: "origin",
			},
		}

		diverged, err := checkBranchDiverged("012-test-branch", cfg)
		require.NoError(t, err)
		assert.False(t, diverged, "branch up-to-date should not be considered diverged")
	})

	t.Run("returns false when branch ahead of remote", func(t *testing.T) {
		tmpDir := setupTestGitRepo(t, "012-test-branch")
		_ = tmpDir // avoid unused variable warning

		// Create a bare repository to simulate remote
		remoteDir := t.TempDir()
		// #nosec G204 - remoteDir is from t.TempDir() which is safe
		require.NoError(t, exec.Command("git", "init", "--bare", remoteDir).Run())

		// Add remote and push initial commit
		// #nosec G204 - remoteDir is from t.TempDir() which is safe
		require.NoError(t, exec.Command("git", "remote", "add", "origin", remoteDir).Run())
		require.NoError(t, exec.Command("git", "push", "-u", "origin", "012-test-branch").Run())

		// Make a new commit locally (ahead of remote)
		require.NoError(t, os.WriteFile("newfile.txt", []byte("new content"), 0o600))
		require.NoError(t, exec.Command("git", "add", "newfile.txt").Run())
		require.NoError(t, exec.Command("git", "commit", "-m", "New commit").Run())

		cfg := &config.Config{
			Git: &config.GitConfig{
				Remote: "origin",
			},
		}

		diverged, err := checkBranchDiverged("012-test-branch", cfg)
		require.NoError(t, err)
		assert.False(t, diverged, "branch ahead of remote should not be considered diverged")
	})

	t.Run("returns false when branch behind remote", func(t *testing.T) {
		tmpDir := setupTestGitRepo(t, "012-test-branch")
		_ = tmpDir // avoid unused variable warning

		// Create a bare repository to simulate remote
		remoteDir := t.TempDir()
		// #nosec G204 - remoteDir is from t.TempDir() which is safe
		require.NoError(t, exec.Command("git", "init", "--bare", remoteDir).Run())

		// Add remote and push initial commit
		// #nosec G204 - remoteDir is from t.TempDir() which is safe
		require.NoError(t, exec.Command("git", "remote", "add", "origin", remoteDir).Run())
		require.NoError(t, exec.Command("git", "push", "-u", "origin", "012-test-branch").Run())

		// Make a commit on remote (simulate by cloning, committing, pushing)
		cloneDir := t.TempDir()
		// #nosec G204 - cloneDir and remoteDir are from t.TempDir() which is safe
		require.NoError(t, exec.Command("git", "clone", remoteDir, cloneDir).Run())
		// #nosec G204 - cloneDir is from t.TempDir() which is safe
		require.NoError(t, exec.Command("git", "-C", cloneDir, "config", "user.email", "test@example.com").Run())
		// #nosec G204 - cloneDir is from t.TempDir() which is safe
		require.NoError(t, exec.Command("git", "-C", cloneDir, "config", "user.name", "Test User").Run())
		// #nosec G204 - cloneDir is from t.TempDir() which is safe
		require.NoError(t, exec.Command("git", "-C", cloneDir, "checkout", "012-test-branch").Run())
		require.NoError(t, os.WriteFile(filepath.Join(cloneDir, "remotefile.txt"), []byte("remote content"), 0o600))
		// #nosec G204 - cloneDir is from t.TempDir() which is safe
		require.NoError(t, exec.Command("git", "-C", cloneDir, "add", "remotefile.txt").Run())
		// #nosec G204 - cloneDir is from t.TempDir() which is safe
		require.NoError(t, exec.Command("git", "-C", cloneDir, "commit", "-m", "Remote commit").Run())
		// #nosec G204 - cloneDir is from t.TempDir() which is safe
		require.NoError(t, exec.Command("git", "-C", cloneDir, "push", "origin", "012-test-branch").Run())

		cfg := &config.Config{
			Git: &config.GitConfig{
				Remote: "origin",
			},
		}

		diverged, err := checkBranchDiverged("012-test-branch", cfg)
		require.NoError(t, err)
		assert.False(t, diverged, "branch behind remote should not be considered diverged")
	})

	t.Run("returns error when branch has diverged", func(t *testing.T) {
		tmpDir := setupTestGitRepo(t, "012-test-branch")
		_ = tmpDir // avoid unused variable warning

		// Create a bare repository to simulate remote
		remoteDir := t.TempDir()
		// #nosec G204 - remoteDir is from t.TempDir() which is safe
		require.NoError(t, exec.Command("git", "init", "--bare", remoteDir).Run())

		// Add remote and push initial commit
		// #nosec G204 - remoteDir is from t.TempDir() which is safe
		require.NoError(t, exec.Command("git", "remote", "add", "origin", remoteDir).Run())
		require.NoError(t, exec.Command("git", "push", "-u", "origin", "012-test-branch").Run())

		// Make a commit locally
		require.NoError(t, os.WriteFile("localfile.txt", []byte("local content"), 0o600))
		require.NoError(t, exec.Command("git", "add", "localfile.txt").Run())
		require.NoError(t, exec.Command("git", "commit", "-m", "Local commit").Run())

		// Make a different commit on remote (simulate by cloning, committing, pushing)
		cloneDir := t.TempDir()
		// #nosec G204 - cloneDir and remoteDir are from t.TempDir() which is safe
		require.NoError(t, exec.Command("git", "clone", remoteDir, cloneDir).Run())
		// #nosec G204 - cloneDir is from t.TempDir() which is safe
		require.NoError(t, exec.Command("git", "-C", cloneDir, "config", "user.email", "test@example.com").Run())
		// #nosec G204 - cloneDir is from t.TempDir() which is safe
		require.NoError(t, exec.Command("git", "-C", cloneDir, "config", "user.name", "Test User").Run())
		// #nosec G204 - cloneDir is from t.TempDir() which is safe
		require.NoError(t, exec.Command("git", "-C", cloneDir, "checkout", "012-test-branch").Run())
		require.NoError(t, os.WriteFile(filepath.Join(cloneDir, "remotefile.txt"), []byte("remote content"), 0o600))
		// #nosec G204 - cloneDir is from t.TempDir() which is safe
		require.NoError(t, exec.Command("git", "-C", cloneDir, "add", "remotefile.txt").Run())
		// #nosec G204 - cloneDir is from t.TempDir() which is safe
		require.NoError(t, exec.Command("git", "-C", cloneDir, "commit", "-m", "Remote commit").Run())
		// #nosec G204 - cloneDir is from t.TempDir() which is safe
		require.NoError(t, exec.Command("git", "-C", cloneDir, "push", "origin", "012-test-branch").Run())

		cfg := &config.Config{
			Git: &config.GitConfig{
				Remote: "origin",
			},
		}

		diverged, err := checkBranchDiverged("012-test-branch", cfg)
		require.Error(t, err, "should return error when branch has diverged")
		assert.True(t, diverged, "diverged should be true")
		assert.Contains(t, err.Error(), "branch has diverged from remote")
		assert.Contains(t, err.Error(), "Pull latest changes or resolve conflicts")
	})

	t.Run("returns error for empty branch name", func(t *testing.T) {
		cfg := &config.Config{
			Git: &config.GitConfig{
				Remote: "origin",
			},
		}

		_, err := checkBranchDiverged("", cfg)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "branch name cannot be empty")
	})

	t.Run("returns error for nil config", func(t *testing.T) {
		_, err := checkBranchDiverged("012-test-branch", nil)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "configuration cannot be nil")
	})
}

func TestPushBranchIfNeeded(t *testing.T) {
	t.Run("pushes branch when not on remote", func(t *testing.T) {
		tmpDir := setupTestGitRepo(t, "012-test-branch")
		_ = tmpDir // avoid unused variable warning

		remoteDir := t.TempDir()
		// #nosec G204 - remoteDir is from t.TempDir() which is safe
		require.NoError(t, exec.Command("git", "init", "--bare", remoteDir).Run())
		// #nosec G204 - remoteDir is from t.TempDir() which is safe
		require.NoError(t, exec.Command("git", "remote", "add", "origin", remoteDir).Run())

		cfg := &config.Config{
			Git: &config.GitConfig{Remote: "origin"},
		}

		err := pushBranchIfNeeded("012-test-branch", cfg)
		require.NoError(t, err)

		exists, err := checkBranchOnRemote("012-test-branch", cfg)
		require.NoError(t, err)
		assert.True(t, exists, "branch should exist on remote after push")
	})

	t.Run("skips push when already on remote and up-to-date", func(t *testing.T) {
		tmpDir := setupTestGitRepo(t, "012-test-branch")
		_ = tmpDir // avoid unused variable warning

		remoteDir := t.TempDir()
		// #nosec G204 - remoteDir is from t.TempDir() which is safe
		require.NoError(t, exec.Command("git", "init", "--bare", remoteDir).Run())
		// #nosec G204 - remoteDir is from t.TempDir() which is safe
		require.NoError(t, exec.Command("git", "remote", "add", "origin", remoteDir).Run())
		require.NoError(t, exec.Command("git", "push", "-u", "origin", "012-test-branch").Run())

		cfg := &config.Config{
			Git: &config.GitConfig{Remote: "origin"},
		}

		err := pushBranchIfNeeded("012-test-branch", cfg)
		require.NoError(t, err)
	})

	t.Run("errors when branch diverged", func(t *testing.T) {
		tmpDir := setupTestGitRepo(t, "012-test-branch")
		_ = tmpDir // avoid unused variable warning

		remoteDir := t.TempDir()
		// #nosec G204 - remoteDir is from t.TempDir() which is safe
		require.NoError(t, exec.Command("git", "init", "--bare", remoteDir).Run())
		// #nosec G204 - remoteDir is from t.TempDir() which is safe
		require.NoError(t, exec.Command("git", "remote", "add", "origin", remoteDir).Run())
		require.NoError(t, exec.Command("git", "push", "-u", "origin", "012-test-branch").Run())

		require.NoError(t, os.WriteFile("localfile.txt", []byte("local content"), 0o600))
		require.NoError(t, exec.Command("git", "add", "localfile.txt").Run())
		require.NoError(t, exec.Command("git", "commit", "-m", "Local commit").Run())

		cloneDir := t.TempDir()
		// #nosec G204 - cloneDir and remoteDir are from t.TempDir() which is safe
		require.NoError(t, exec.Command("git", "clone", remoteDir, cloneDir).Run())
		// #nosec G204 - cloneDir is from t.TempDir() which is safe
		require.NoError(t, exec.Command("git", "-C", cloneDir, "config", "user.email", "test@example.com").Run())
		// #nosec G204 - cloneDir is from t.TempDir() which is safe
		require.NoError(t, exec.Command("git", "-C", cloneDir, "config", "user.name", "Test User").Run())
		// #nosec G204 - cloneDir is from t.TempDir() which is safe
		require.NoError(t, exec.Command("git", "-C", cloneDir, "checkout", "012-test-branch").Run())
		require.NoError(t, os.WriteFile(filepath.Join(cloneDir, "remotefile.txt"), []byte("remote content"), 0o600))
		// #nosec G204 - cloneDir is from t.TempDir() which is safe
		require.NoError(t, exec.Command("git", "-C", cloneDir, "add", "remotefile.txt").Run())
		// #nosec G204 - cloneDir is from t.TempDir() which is safe
		require.NoError(t, exec.Command("git", "-C", cloneDir, "commit", "-m", "Remote commit").Run())
		// #nosec G204 - cloneDir is from t.TempDir() which is safe
		require.NoError(t, exec.Command("git", "-C", cloneDir, "push", "origin", "012-test-branch").Run())

		cfg := &config.Config{
			Git: &config.GitConfig{Remote: "origin"},
		}

		err := pushBranchIfNeeded("012-test-branch", cfg)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "branch has diverged from remote")
		assert.Contains(t, err.Error(), "Pull latest changes or resolve conflicts")
	})

	t.Run("returns error for empty branch name", func(t *testing.T) {
		cfg := &config.Config{Git: &config.GitConfig{Remote: "origin"}}
		err := pushBranchIfNeeded("", cfg)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "branch name cannot be empty")
	})

	t.Run("returns error for nil config", func(t *testing.T) {
		err := pushBranchIfNeeded("012-test-branch", nil)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "configuration cannot be nil")
	})
}

// TestGetGitHubToken tests the getGitHubToken function
func TestGetGitHubToken(t *testing.T) {
	const envKey = "KIRA_GITHUB_TOKEN"

	t.Run("reads token from KIRA_GITHUB_TOKEN when set", func(t *testing.T) {
		prev := os.Getenv(envKey)
		t.Cleanup(func() {
			if prev == "" {
				_ = os.Unsetenv(envKey)
			} else {
				_ = os.Setenv(envKey, prev)
			}
		})
		require.NoError(t, os.Setenv(envKey, "test-token-123"))

		token, err := getGitHubToken(&config.Config{})
		require.NoError(t, err)
		assert.Equal(t, "test-token-123", token)
	})

	t.Run("trims whitespace from token", func(t *testing.T) {
		prev := os.Getenv(envKey)
		t.Cleanup(func() {
			if prev == "" {
				_ = os.Unsetenv(envKey)
			} else {
				_ = os.Setenv(envKey, prev)
			}
		})
		require.NoError(t, os.Setenv(envKey, "  test-token-123  "))

		token, err := getGitHubToken(&config.Config{})
		require.NoError(t, err)
		assert.Equal(t, "test-token-123", token)
	})

	t.Run("returns error when KIRA_GITHUB_TOKEN is unset", func(t *testing.T) {
		prev := os.Getenv(envKey)
		t.Cleanup(func() {
			if prev == "" {
				_ = os.Unsetenv(envKey)
			} else {
				_ = os.Setenv(envKey, prev)
			}
		})
		require.NoError(t, os.Unsetenv(envKey))

		_, err := getGitHubToken(&config.Config{})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "GitHub token required for PR creation")
		assert.Contains(t, err.Error(), "KIRA_GITHUB_TOKEN")
	})

	t.Run("returns error when KIRA_GITHUB_TOKEN is empty", func(t *testing.T) {
		prev := os.Getenv(envKey)
		t.Cleanup(func() {
			if prev == "" {
				_ = os.Unsetenv(envKey)
			} else {
				_ = os.Setenv(envKey, prev)
			}
		})
		require.NoError(t, os.Setenv(envKey, ""))

		_, err := getGitHubToken(&config.Config{})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "GitHub token required for PR creation")
		assert.Contains(t, err.Error(), "KIRA_GITHUB_TOKEN")
	})

	t.Run("returns error when KIRA_GITHUB_TOKEN is only whitespace", func(t *testing.T) {
		prev := os.Getenv(envKey)
		t.Cleanup(func() {
			if prev == "" {
				_ = os.Unsetenv(envKey)
			} else {
				_ = os.Setenv(envKey, prev)
			}
		})
		require.NoError(t, os.Setenv(envKey, "   \n\t  "))

		_, err := getGitHubToken(&config.Config{})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "GitHub token required for PR creation")
		assert.Contains(t, err.Error(), "KIRA_GITHUB_TOKEN")
	})
}

// TestValidateGitHubToken tests the validateGitHubToken function
// Note: These tests use real GitHub API calls, so they may fail if:
// - Network is unavailable
// - GitHub API is down
// - Rate limits are exceeded
// For production, we'd use mocks, but for now we test with real API
func TestValidateGitHubToken(t *testing.T) {
	t.Run("returns error for empty token", func(t *testing.T) {
		err := validateGitHubToken("")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "GitHub token validation failed")
		assert.Contains(t, err.Error(), "repo")
	})

	t.Run("returns error for whitespace-only token", func(t *testing.T) {
		err := validateGitHubToken("   \n\t  ")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "GitHub token validation failed")
		assert.Contains(t, err.Error(), "repo")
	})

	t.Run("returns error for invalid token", func(t *testing.T) {
		// Use a clearly invalid token
		err := validateGitHubToken("invalid-token-12345")
		require.Error(t, err)
		// Should get authentication error
		assert.Contains(t, err.Error(), "GitHub token validation failed")
	})

	// Note: We don't test with a valid token here because:
	// 1. We don't want to require a real GitHub token in tests
	// 2. We don't want to expose tokens in test code
	// 3. The actual validation will be tested in integration tests
	// The unit tests verify error handling for invalid/missing tokens
}

func TestGeneratePRTitle(t *testing.T) {
	t.Run("generates title with default template and all variables", func(t *testing.T) {
		workItem := &validation.WorkItem{
			ID:    "012",
			Title: "Submit for review",
			Kind:  "prd",
		}
		cfg := &config.Config{
			Review: &config.ReviewConfig{},
		}

		title, err := generatePRTitle(workItem, cfg)
		require.NoError(t, err)
		assert.Equal(t, "[012] Submit for review", title)
	})

	t.Run("generates title with custom template", func(t *testing.T) {
		workItem := &validation.WorkItem{
			ID:    "012",
			Title: "Submit for review",
			Kind:  "prd",
		}
		cfg := &config.Config{
			Review: &config.ReviewConfig{
				PRTitle: "{kind}: {title} ({id})",
			},
		}

		title, err := generatePRTitle(workItem, cfg)
		require.NoError(t, err)
		assert.Equal(t, "prd: Submit for review (012)", title)
	})

	t.Run("handles missing variables with empty strings", func(t *testing.T) {
		workItem := &validation.WorkItem{
			ID:    "",
			Title: "",
			Kind:  "",
		}
		cfg := &config.Config{
			Review: &config.ReviewConfig{
				PRTitle: "[{id}] {title} ({kind})",
			},
		}

		title, err := generatePRTitle(workItem, cfg)
		require.NoError(t, err)
		assert.Equal(t, "[]  ()", title)
	})

	t.Run("truncates long titles to 200 chars", func(t *testing.T) {
		longTitle := strings.Repeat("a", 300)
		workItem := &validation.WorkItem{
			ID:    "012",
			Title: longTitle,
			Kind:  "prd",
		}
		cfg := &config.Config{
			Review: &config.ReviewConfig{
				PRTitle: "[{id}] {title}",
			},
		}

		title, err := generatePRTitle(workItem, cfg)
		require.NoError(t, err)
		assert.LessOrEqual(t, len(title), 200)
		assert.Equal(t, 200, len(title))
		assert.True(t, strings.HasPrefix(title, "[012]"))
	})

	t.Run("sanitizes newlines and carriage returns", func(t *testing.T) {
		workItem := &validation.WorkItem{
			ID:    "012",
			Title: "Title\nwith\rnewlines",
			Kind:  "prd",
		}
		cfg := &config.Config{
			Review: &config.ReviewConfig{
				PRTitle: "[{id}] {title}",
			},
		}

		title, err := generatePRTitle(workItem, cfg)
		require.NoError(t, err)
		assert.Equal(t, "[012] Title with newlines", title)
		assert.NotContains(t, title, "\n")
		assert.NotContains(t, title, "\r")
	})

	t.Run("handles empty template by using default", func(t *testing.T) {
		workItem := &validation.WorkItem{
			ID:    "012",
			Title: "Submit for review",
			Kind:  "prd",
		}
		cfg := &config.Config{
			Review: &config.ReviewConfig{
				PRTitle: "",
			},
		}

		title, err := generatePRTitle(workItem, cfg)
		require.NoError(t, err)
		assert.Equal(t, "[012] Submit for review", title)
	})

	t.Run("trims whitespace", func(t *testing.T) {
		workItem := &validation.WorkItem{
			ID:    "012",
			Title: "  Title with spaces  ",
			Kind:  "prd",
		}
		cfg := &config.Config{
			Review: &config.ReviewConfig{
				PRTitle: "[{id}] {title}",
			},
		}

		title, err := generatePRTitle(workItem, cfg)
		require.NoError(t, err)
		// Final result is trimmed, so trailing spaces are removed
		assert.Equal(t, "[012]   Title with spaces", title)
	})

	t.Run("returns error for nil work item", func(t *testing.T) {
		cfg := &config.Config{
			Review: &config.ReviewConfig{},
		}

		_, err := generatePRTitle(nil, cfg)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "work item cannot be nil")
	})

	t.Run("returns error for nil config", func(t *testing.T) {
		workItem := &validation.WorkItem{
			ID:    "012",
			Title: "Test",
			Kind:  "prd",
		}

		_, err := generatePRTitle(workItem, nil)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "configuration cannot be nil")
	})
}

func TestGeneratePRDescription(t *testing.T) {
	t.Run("generates description with default template and all variables", func(t *testing.T) {
		tmpDir := setupTestGitRepo(t, "012-test-feature")
		_ = tmpDir // Use tmpDir to avoid unused variable

		workItem := &validation.WorkItem{
			ID:    "012",
			Title: "Submit for review",
			Kind:  "prd",
		}
		workItemPath := testWorkItemPath

		// Add a GitHub remote
		// #nosec G204 - tmpDir is from t.TempDir() which is safe
		require.NoError(t, exec.Command("git", "-C", tmpDir, "remote", "add", "origin", "https://github.com/test-owner/test-repo.git").Run())

		cfg := &config.Config{
			Git: &config.GitConfig{
				TrunkBranch: "main",
				Remote:      "origin",
			},
			Review: &config.ReviewConfig{},
		}

		description, err := generatePRDescription(workItem, workItemPath, cfg)
		require.NoError(t, err)
		// Default template uses [{id}-{title}] which becomes [012-Submit for review] (spaces, not hyphens)
		assert.Contains(t, description, "[012-Submit for review]")
		assert.Contains(t, description, "https://github.com/test-owner/test-repo/blob/main/.work/3_review/012-submit-for-review.prd.md")
	})

	t.Run("generates description with custom template", func(t *testing.T) {
		tmpDir := setupTestGitRepo(t, "012-test-feature")
		_ = tmpDir // Use tmpDir to avoid unused variable

		workItem := &validation.WorkItem{
			ID:    "012",
			Title: "Submit for review",
			Kind:  "prd",
		}
		workItemPath := testWorkItemPath

		// Add a GitHub remote
		// #nosec G204 - tmpDir is from t.TempDir() which is safe
		require.NoError(t, exec.Command("git", "-C", tmpDir, "remote", "add", "origin", "https://github.com/test-owner/test-repo.git").Run())

		cfg := &config.Config{
			Git: &config.GitConfig{
				TrunkBranch: "main",
				Remote:      "origin",
			},
			Review: &config.ReviewConfig{
				PRDescription: "Work item {id}: {title}\n\nDetails: {work_item_url}",
			},
		}

		description, err := generatePRDescription(workItem, workItemPath, cfg)
		require.NoError(t, err)
		assert.Contains(t, description, "Work item 012: Submit for review")
		assert.Contains(t, description, "https://github.com/test-owner/test-repo/blob/main/.work/3_review/012-submit-for-review.prd.md")
	})

	t.Run("handles missing GitHub repo info gracefully", func(t *testing.T) {
		tmpDir := setupTestGitRepo(t, "012-test-feature")
		_ = tmpDir // Use tmpDir to avoid unused variable

		workItem := &validation.WorkItem{
			ID:    "012",
			Title: "Submit for review",
			Kind:  "prd",
		}
		workItemPath := ".work/3_review/012-submit-for-review.prd.md"

		// No remote configured
		cfg := &config.Config{
			Git: &config.GitConfig{
				TrunkBranch: "main",
			},
			Review: &config.ReviewConfig{
				PRDescription: "View: [{id}-{title}]({work_item_url})",
			},
		}

		description, err := generatePRDescription(workItem, workItemPath, cfg)
		require.NoError(t, err) // Should not fail
		// Template uses [{id}-{title}] which becomes [012-Submit for review]
		assert.Contains(t, description, "[012-Submit for review]")
		// URL variable should be replaced with empty string
		assert.Contains(t, description, "]()") // Empty URL in markdown link
		// The URL should be empty since we can't get repo info
		assert.NotContains(t, description, "https://github.com")
	})

	t.Run("handles missing trunk branch gracefully", func(t *testing.T) {
		tmpDir := setupTestGitRepo(t, "012-test-feature")
		_ = tmpDir // Use tmpDir to avoid unused variable

		workItem := &validation.WorkItem{
			ID:    "012",
			Title: "Submit for review",
			Kind:  "prd",
		}
		workItemPath := testWorkItemPath

		// Add a GitHub remote
		// #nosec G204 - tmpDir is from t.TempDir() which is safe
		require.NoError(t, exec.Command("git", "-C", tmpDir, "remote", "add", "origin", "https://github.com/test-owner/test-repo.git").Run())

		// Configure with non-existent trunk branch
		cfg := &config.Config{
			Git: &config.GitConfig{
				TrunkBranch: "nonexistent-branch",
				Remote:      "origin",
			},
			Review: &config.ReviewConfig{
				PRDescription: "View: [{id}-{title}]({work_item_url})",
			},
		}

		description, err := generatePRDescription(workItem, workItemPath, cfg)
		require.NoError(t, err) // Should not fail
		// Template uses [{id}-{title}] which becomes [012-Submit for review]
		assert.Contains(t, description, "[012-Submit for review]")
		// URL should be empty since trunk branch doesn't exist
		assert.NotContains(t, description, "https://github.com")
	})

	t.Run("handles missing variables with empty strings", func(t *testing.T) {
		workItem := &validation.WorkItem{
			ID:    "",
			Title: "",
			Kind:  "prd",
		}
		workItemPath := testWorkItemPath

		cfg := &config.Config{
			Review: &config.ReviewConfig{
				PRDescription: "[{id}-{title}]({work_item_url})",
			},
		}

		description, err := generatePRDescription(workItem, workItemPath, cfg)
		require.NoError(t, err)
		assert.Contains(t, description, "[-]")
	})

	t.Run("normalizes work item path correctly", func(t *testing.T) {
		tmpDir := setupTestGitRepo(t, "012-test-feature")
		_ = tmpDir // Use tmpDir to avoid unused variable

		workItem := &validation.WorkItem{
			ID:    "012",
			Title: "Submit for review",
			Kind:  "prd",
		}
		// Test with Windows-style path separators and leading ./
		workItemPath := ".work\\3_review\\012-submit-for-review.prd.md"

		// Add a GitHub remote
		// #nosec G204 - tmpDir is from t.TempDir() which is safe
		require.NoError(t, exec.Command("git", "-C", tmpDir, "remote", "add", "origin", "https://github.com/test-owner/test-repo.git").Run())

		cfg := &config.Config{
			Git: &config.GitConfig{
				TrunkBranch: "main",
				Remote:      "origin",
			},
			Review: &config.ReviewConfig{
				PRDescription: "View: {work_item_url}",
			},
		}

		description, err := generatePRDescription(workItem, workItemPath, cfg)
		require.NoError(t, err)
		// Path should be normalized to forward slashes in the URL
		assert.Contains(t, description, ".work/3_review/012-submit-for-review.prd.md")
		assert.NotContains(t, description, "\\")
	})

	t.Run("handles empty template by using default", func(t *testing.T) {
		tmpDir := setupTestGitRepo(t, "012-test-feature")
		_ = tmpDir // Use tmpDir to avoid unused variable

		workItem := &validation.WorkItem{
			ID:    "012",
			Title: "Submit for review",
			Kind:  "prd",
		}
		workItemPath := testWorkItemPath

		// Add a GitHub remote
		// #nosec G204 - tmpDir is from t.TempDir() which is safe
		require.NoError(t, exec.Command("git", "-C", tmpDir, "remote", "add", "origin", "https://github.com/test-owner/test-repo.git").Run())

		cfg := &config.Config{
			Git: &config.GitConfig{
				TrunkBranch: "main",
				Remote:      "origin",
			},
			Review: &config.ReviewConfig{
				PRDescription: "",
			},
		}

		description, err := generatePRDescription(workItem, workItemPath, cfg)
		require.NoError(t, err)
		// Template uses [{id}-{title}] which becomes [012-Submit for review]
		assert.Contains(t, description, "[012-Submit for review]")
	})

	t.Run("returns error for nil work item", func(t *testing.T) {
		cfg := &config.Config{
			Review: &config.ReviewConfig{},
		}

		_, err := generatePRDescription(nil, ".work/test.md", cfg)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "work item cannot be nil")
	})

	t.Run("returns error for nil config", func(t *testing.T) {
		workItem := &validation.WorkItem{
			ID:    "012",
			Title: "Test",
			Kind:  "prd",
		}

		_, err := generatePRDescription(workItem, ".work/test.md", nil)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "configuration cannot be nil")
	})
}

// TestFindExistingPR tests the findExistingPR function
// Note: These tests verify function structure and error handling.
// Actual API calls with real GitHub tokens should be tested in integration tests.
func TestFindExistingPR(t *testing.T) {
	t.Run("returns error for nil client", func(t *testing.T) {
		_, err := findExistingPR(nil, "owner", "repo", "branch")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "GitHub client cannot be nil")
	})

	t.Run("returns error for empty owner", func(t *testing.T) {
		client, _ := gh.CreateGitHubClient("test-token")
		_, err := findExistingPR(client, "", "repo", "branch")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "owner cannot be empty")
	})

	t.Run("returns error for empty repo", func(t *testing.T) {
		client, _ := gh.CreateGitHubClient("test-token")
		_, err := findExistingPR(client, "owner", "", "branch")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "repo cannot be empty")
	})

	t.Run("returns error for empty branch name", func(t *testing.T) {
		client, _ := gh.CreateGitHubClient("test-token")
		_, err := findExistingPR(client, "owner", "repo", "")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "branch name cannot be empty")
	})

	t.Run("returns error for whitespace-only owner", func(t *testing.T) {
		client, _ := gh.CreateGitHubClient("test-token")
		_, err := findExistingPR(client, "   ", "repo", "branch")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "owner cannot be empty")
	})

	t.Run("returns error for whitespace-only repo", func(t *testing.T) {
		client, _ := gh.CreateGitHubClient("test-token")
		_, err := findExistingPR(client, "owner", "   ", "branch")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "repo cannot be empty")
	})

	t.Run("returns error for whitespace-only branch name", func(t *testing.T) {
		client, _ := gh.CreateGitHubClient("test-token")
		_, err := findExistingPR(client, "owner", "repo", "   ")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "branch name cannot be empty")
	})

	// Note: Tests for actual API calls (finding PRs, handling API errors, etc.)
	// should be done in integration tests with a real GitHub token and test repository.
	// The unit tests above verify:
	// 1. Input validation (nil client, empty parameters)
	// 2. Error handling structure
	// 3. Function signature and return types
	//
	// Integration tests should verify:
	// - Finding existing PR by branch name
	// - Returning nil when no PR exists (not an error)
	// - Handling API errors (authentication, network, rate limiting)
	// - Handling draft PRs correctly
	// - Filtering by owner to handle forks correctly
}

// TestCreateGitHubPR tests the createGitHubPR function
// Note: These tests verify function structure and error handling.
// Actual API calls with real GitHub tokens should be tested in integration tests.
func TestCreateGitHubPR(t *testing.T) {
	t.Run("returns error for nil client", func(t *testing.T) {
		_, err := createGitHubPR(nil, "owner", "repo", "branch", "main", "Title", "Description", true)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "GitHub client cannot be nil")
	})

	t.Run("returns error for empty owner", func(t *testing.T) {
		client, _ := gh.CreateGitHubClient("test-token")
		_, err := createGitHubPR(client, "", "repo", "branch", "main", "Title", "Description", true)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "owner cannot be empty")
	})

	t.Run("returns error for empty repo", func(t *testing.T) {
		client, _ := gh.CreateGitHubClient("test-token")
		_, err := createGitHubPR(client, "owner", "", "branch", "main", "Title", "Description", true)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "repo cannot be empty")
	})

	t.Run("returns error for empty branch name", func(t *testing.T) {
		client, _ := gh.CreateGitHubClient("test-token")
		_, err := createGitHubPR(client, "owner", "repo", "", "main", "Title", "Description", true)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "branch name cannot be empty")
	})

	t.Run("returns error for empty base branch", func(t *testing.T) {
		client, _ := gh.CreateGitHubClient("test-token")
		_, err := createGitHubPR(client, "owner", "repo", "branch", "", "Title", "Description", true)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "base branch cannot be empty")
	})

	t.Run("returns error for empty title", func(t *testing.T) {
		client, _ := gh.CreateGitHubClient("test-token")
		_, err := createGitHubPR(client, "owner", "repo", "branch", "main", "", "Description", true)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "PR title cannot be empty")
	})

	t.Run("returns error for whitespace-only owner", func(t *testing.T) {
		client, _ := gh.CreateGitHubClient("test-token")
		_, err := createGitHubPR(client, "   ", "repo", "branch", "main", "Title", "Description", true)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "owner cannot be empty")
	})

	t.Run("returns error for whitespace-only repo", func(t *testing.T) {
		client, _ := gh.CreateGitHubClient("test-token")
		_, err := createGitHubPR(client, "owner", "   ", "branch", "main", "Title", "Description", true)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "repo cannot be empty")
	})

	t.Run("returns error for whitespace-only branch name", func(t *testing.T) {
		client, _ := gh.CreateGitHubClient("test-token")
		_, err := createGitHubPR(client, "owner", "repo", "   ", "main", "Title", "Description", true)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "branch name cannot be empty")
	})

	t.Run("returns error for whitespace-only base branch", func(t *testing.T) {
		client, _ := gh.CreateGitHubClient("test-token")
		_, err := createGitHubPR(client, "owner", "repo", "branch", "   ", "Title", "Description", true)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "base branch cannot be empty")
	})

	t.Run("returns error for whitespace-only title", func(t *testing.T) {
		client, _ := gh.CreateGitHubClient("test-token")
		_, err := createGitHubPR(client, "owner", "repo", "branch", "main", "   ", "Description", true)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "PR title cannot be empty")
	})

	t.Run("accepts empty description", func(t *testing.T) {
		// Empty description is allowed by GitHub, so validation should pass
		// The actual API call will fail without a real token, but validation should succeed
		client, _ := gh.CreateGitHubClient("test-token")
		// This will fail at API call, but not at validation
		_, err := createGitHubPR(client, "owner", "repo", "branch", "main", "Title", "", true)
		// Error should be from API call, not validation
		require.Error(t, err)
		// Should not be a validation error
		assert.NotContains(t, err.Error(), "cannot be empty")
	})

	// Note: Tests for actual API calls (creating PRs, handling API errors, etc.)
	// should be done in integration tests with a real GitHub token and test repository.
	// The unit tests above verify:
	// 1. Input validation (nil client, empty parameters, whitespace-only parameters)
	// 2. Error handling structure
	// 3. Function signature and return types
	// 4. Empty description is allowed (validation passes)
	//
	// Integration tests should verify:
	// - Creating draft PR successfully
	// - Creating ready PR successfully (isDraft=false)
	// - Returning PR with correct fields (title, body, head branch, base branch, draft status, HTML URL)
	// - Handling API errors (401 Unauthorized, 403 Forbidden, 404 Not Found, 422 Unprocessable Entity)
	// - Handling network errors
	// - Handling validation errors (branch doesn't exist, PR already exists)
}

// TestExtractTagsFromWorkItem tests the extractTagsFromWorkItem function
func TestExtractTagsFromWorkItem(t *testing.T) {
	t.Run("extracts tags from []string format", func(t *testing.T) {
		workItem := &validation.WorkItem{
			Fields: map[string]interface{}{
				"tags": []string{"github", "notifications", "review"},
			},
		}

		tags := extractTagsFromWorkItem(workItem)
		require.Len(t, tags, 3)
		assert.Contains(t, tags, "github")
		assert.Contains(t, tags, "notifications")
		assert.Contains(t, tags, "review")
	})

	t.Run("extracts tags from []interface{} format", func(t *testing.T) {
		workItem := &validation.WorkItem{
			Fields: map[string]interface{}{
				"tags": []interface{}{"github", "notifications", "review"},
			},
		}

		tags := extractTagsFromWorkItem(workItem)
		require.Len(t, tags, 3)
		assert.Contains(t, tags, "github")
		assert.Contains(t, tags, "notifications")
		assert.Contains(t, tags, "review")
	})

	t.Run("returns empty slice when tags field doesn't exist", func(t *testing.T) {
		workItem := &validation.WorkItem{
			Fields: map[string]interface{}{
				"other_field": "value",
			},
		}

		tags := extractTagsFromWorkItem(workItem)
		assert.Empty(t, tags)
	})

	t.Run("returns empty slice when Fields is nil", func(t *testing.T) {
		workItem := &validation.WorkItem{
			Fields: nil,
		}

		tags := extractTagsFromWorkItem(workItem)
		assert.Empty(t, tags)
	})

	t.Run("returns empty slice when workItem is nil", func(t *testing.T) {
		tags := extractTagsFromWorkItem(nil)
		assert.Empty(t, tags)
	})

	t.Run("filters out empty strings and trims whitespace", func(t *testing.T) {
		workItem := &validation.WorkItem{
			Fields: map[string]interface{}{
				"tags": []string{"  github  ", "", "notifications", "  ", "review"},
			},
		}

		tags := extractTagsFromWorkItem(workItem)
		require.Len(t, tags, 3)
		assert.Contains(t, tags, "github")
		assert.Contains(t, tags, "notifications")
		assert.Contains(t, tags, "review")
		// Verify whitespace was trimmed
		for _, tag := range tags {
			assert.Equal(t, strings.TrimSpace(tag), tag, "tag should be trimmed: %q", tag)
		}
	})

	t.Run("handles mixed string and non-string types in []interface{}", func(t *testing.T) {
		workItem := &validation.WorkItem{
			Fields: map[string]interface{}{
				"tags": []interface{}{"github", 123, "notifications", true, "review"},
			},
		}

		tags := extractTagsFromWorkItem(workItem)
		require.Len(t, tags, 3)
		assert.Contains(t, tags, "github")
		assert.Contains(t, tags, "notifications")
		assert.Contains(t, tags, "review")
		// Non-string types should be skipped
		assert.NotContains(t, tags, "123")
		assert.NotContains(t, tags, "true")
	})

	t.Run("returns empty slice for unknown tag type", func(t *testing.T) {
		workItem := &validation.WorkItem{
			Fields: map[string]interface{}{
				"tags": "not-an-array",
			},
		}

		tags := extractTagsFromWorkItem(workItem)
		assert.Empty(t, tags)
	})
}

// TestAddPRLabels tests the addPRLabels function
// Note: These tests verify function structure and error handling.
// Actual API calls with real GitHub tokens should be tested in integration tests.
func TestAddPRLabels(t *testing.T) {
	t.Run("returns error for nil client", func(t *testing.T) {
		err := addPRLabels(nil, "owner", "repo", 1, []string{"label1"})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "GitHub client cannot be nil")
	})

	t.Run("returns error for empty owner", func(t *testing.T) {
		client, _ := gh.CreateGitHubClient("test-token")
		err := addPRLabels(client, "", "repo", 1, []string{"label1"})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "owner cannot be empty")
	})

	t.Run("returns error for empty repo", func(t *testing.T) {
		client, _ := gh.CreateGitHubClient("test-token")
		err := addPRLabels(client, "owner", "", 1, []string{"label1"})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "repo cannot be empty")
	})

	t.Run("returns error for invalid PR number", func(t *testing.T) {
		client, _ := gh.CreateGitHubClient("test-token")
		err := addPRLabels(client, "owner", "repo", 0, []string{"label1"})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "PR number must be greater than 0")
	})

	t.Run("returns error for negative PR number", func(t *testing.T) {
		client, _ := gh.CreateGitHubClient("test-token")
		err := addPRLabels(client, "owner", "repo", -1, []string{"label1"})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "PR number must be greater than 0")
	})

	t.Run("returns error for nil tags", func(t *testing.T) {
		client, _ := gh.CreateGitHubClient("test-token")
		err := addPRLabels(client, "owner", "repo", 1, nil)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "tags cannot be nil")
	})

	t.Run("returns no error for empty tags", func(t *testing.T) {
		client, _ := gh.CreateGitHubClient("test-token")
		err := addPRLabels(client, "owner", "repo", 1, []string{})
		assert.NoError(t, err, "empty tags should not cause an error")
	})

	t.Run("returns error for whitespace-only owner", func(t *testing.T) {
		client, _ := gh.CreateGitHubClient("test-token")
		err := addPRLabels(client, "   ", "repo", 1, []string{"label1"})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "owner cannot be empty")
	})

	t.Run("returns error for whitespace-only repo", func(t *testing.T) {
		client, _ := gh.CreateGitHubClient("test-token")
		err := addPRLabels(client, "owner", "   ", 1, []string{"label1"})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "repo cannot be empty")
	})

	// Note: Tests for actual API calls (adding labels, handling 404 errors, etc.)
	// should be done in integration tests with a real GitHub token and test repository.
	// The unit tests above verify:
	// 1. Input validation (nil client, empty parameters, invalid PR number, nil tags)
	// 2. Error handling structure
	// 3. Function signature and return types
	// 4. Empty tags array is handled gracefully
	//
	// Integration tests should verify:
	// - Adding existing labels successfully
	// - Skipping non-existent labels with warning (404 handling)
	// - Handling other API errors gracefully (network errors, authentication errors)
	// - Adding multiple labels in sequence
	// - Labels are added with 1:1 mapping (tag → label)
}

// TestGetNumberedUsers tests the getNumberedUsers function
func TestGetNumberedUsers(t *testing.T) {
	t.Run("returns users with correct numbering from saved users", func(t *testing.T) {
		cfg := &config.Config{
			Users: config.UsersConfig{
				SavedUsers: []config.SavedUser{
					{Email: "user1@example.com", Name: "User One"},
					{Email: "user2@example.com", Name: "User Two"},
					{Email: "user3@example.com", Name: "User Three"},
				},
				UseGitHistory: func() *bool { b := false; return &b }(),
			},
		}

		users, err := getNumberedUsers(cfg)
		require.NoError(t, err)
		require.Len(t, users, 3)

		// Verify numbers are assigned correctly (1-based)
		assert.Equal(t, 1, users[0].Number)
		assert.Equal(t, "user1@example.com", users[0].Email)
		assert.Equal(t, 2, users[1].Number)
		assert.Equal(t, "user2@example.com", users[1].Email)
		assert.Equal(t, 3, users[2].Number)
		assert.Equal(t, "user3@example.com", users[2].Email)
	})

	t.Run("returns error for nil config", func(t *testing.T) {
		_, err := getNumberedUsers(nil)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "configuration cannot be nil")
	})

	t.Run("handles empty user list", func(t *testing.T) {
		cfg := &config.Config{
			Users: config.UsersConfig{
				SavedUsers:    []config.SavedUser{},
				UseGitHistory: func() *bool { b := false; return &b }(),
			},
		}

		users, err := getNumberedUsers(cfg)
		require.NoError(t, err)
		assert.Empty(t, users)
	})
}

// TestResolveUserByNumber tests the resolveUserByNumber function
func TestResolveUserByNumber(t *testing.T) {
	t.Run("resolves valid user numbers to emails", func(t *testing.T) {
		cfg := &config.Config{
			Users: config.UsersConfig{
				SavedUsers: []config.SavedUser{
					{Email: "user1@example.com", Name: "User One"},
					{Email: "user2@example.com", Name: "User Two"},
					{Email: "user3@example.com", Name: "User Three"},
				},
				UseGitHistory: func() *bool { b := false; return &b }(),
			},
		}

		email, err := resolveUserByNumber("1", cfg)
		require.NoError(t, err)
		assert.Equal(t, "user1@example.com", email)

		email, err = resolveUserByNumber("2", cfg)
		require.NoError(t, err)
		assert.Equal(t, "user2@example.com", email)

		email, err = resolveUserByNumber("3", cfg)
		require.NoError(t, err)
		assert.Equal(t, "user3@example.com", email)
	})

	t.Run("returns error for user number too high", func(t *testing.T) {
		cfg := &config.Config{
			Users: config.UsersConfig{
				SavedUsers: []config.SavedUser{
					{Email: "user1@example.com", Name: "User One"},
				},
				UseGitHistory: func() *bool { b := false; return &b }(),
			},
		}

		_, err := resolveUserByNumber("2", cfg)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "user number '2' not found")
		assert.Contains(t, err.Error(), "Run 'kira users' to see available users")
	})

	t.Run("returns error for user number zero", func(t *testing.T) {
		cfg := &config.Config{
			Users: config.UsersConfig{
				SavedUsers: []config.SavedUser{
					{Email: "user1@example.com", Name: "User One"},
				},
				UseGitHistory: func() *bool { b := false; return &b }(),
			},
		}

		_, err := resolveUserByNumber("0", cfg)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "must be a positive integer")
	})

	t.Run("returns error for negative user number", func(t *testing.T) {
		cfg := &config.Config{
			Users: config.UsersConfig{
				SavedUsers: []config.SavedUser{
					{Email: "user1@example.com", Name: "User One"},
				},
				UseGitHistory: func() *bool { b := false; return &b }(),
			},
		}

		_, err := resolveUserByNumber("-1", cfg)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "must be a positive integer")
	})

	t.Run("returns error for non-numeric user number", func(t *testing.T) {
		cfg := &config.Config{
			Users: config.UsersConfig{
				SavedUsers: []config.SavedUser{
					{Email: "user1@example.com", Name: "User One"},
				},
				UseGitHistory: func() *bool { b := false; return &b }(),
			},
		}

		_, err := resolveUserByNumber("abc", cfg)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "must be a positive integer")
	})

	t.Run("returns error for empty user list", func(t *testing.T) {
		cfg := &config.Config{
			Users: config.UsersConfig{
				SavedUsers:    []config.SavedUser{},
				UseGitHistory: func() *bool { b := false; return &b }(),
			},
		}

		_, err := resolveUserByNumber("1", cfg)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "no users available")
		assert.Contains(t, err.Error(), "Run 'kira users' to see available users")
	})

	t.Run("returns error for nil config", func(t *testing.T) {
		_, err := resolveUserByNumber("1", nil)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "configuration cannot be nil")
	})
}

// TestResolveReviewers tests the resolveReviewers function
func TestResolveReviewers(t *testing.T) {
	t.Run("resolves user numbers to emails", func(t *testing.T) {
		cfg := &config.Config{
			Users: config.UsersConfig{
				SavedUsers: []config.SavedUser{
					{Email: "user1@example.com", Name: "User One"},
					{Email: "user2@example.com", Name: "User Two"},
				},
				UseGitHistory: func() *bool { b := false; return &b }(),
			},
		}

		reviewers, err := resolveReviewers([]string{"1", "2"}, cfg)
		require.NoError(t, err)
		require.Len(t, reviewers, 2)
		assert.Equal(t, "user1@example.com", reviewers[0])
		assert.Equal(t, "user2@example.com", reviewers[1])
	})

	t.Run("handles email addresses", func(t *testing.T) {
		cfg := &config.Config{
			Users: config.UsersConfig{
				SavedUsers:    []config.SavedUser{},
				UseGitHistory: func() *bool { b := false; return &b }(),
			},
		}

		reviewers, err := resolveReviewers([]string{"user@example.com", "reviewer@test.com"}, cfg)
		require.NoError(t, err)
		require.Len(t, reviewers, 2)
		assert.Equal(t, "user@example.com", reviewers[0])
		assert.Equal(t, "reviewer@test.com", reviewers[1])
	})

	t.Run("handles GitHub usernames", func(t *testing.T) {
		cfg := &config.Config{
			Users: config.UsersConfig{
				SavedUsers:    []config.SavedUser{},
				UseGitHistory: func() *bool { b := false; return &b }(),
			},
		}

		reviewers, err := resolveReviewers([]string{"octocat", "github-user"}, cfg)
		require.NoError(t, err)
		require.Len(t, reviewers, 2)
		assert.Equal(t, "octocat", reviewers[0])
		assert.Equal(t, "github-user", reviewers[1])
	})

	t.Run("handles mixed types", func(t *testing.T) {
		cfg := &config.Config{
			Users: config.UsersConfig{
				SavedUsers: []config.SavedUser{
					{Email: "user1@example.com", Name: "User One"},
				},
				UseGitHistory: func() *bool { b := false; return &b }(),
			},
		}

		reviewers, err := resolveReviewers([]string{"1", "user@example.com", "octocat"}, cfg)
		require.NoError(t, err)
		require.Len(t, reviewers, 3)
		assert.Equal(t, "user1@example.com", reviewers[0])
		assert.Equal(t, "user@example.com", reviewers[1])
		assert.Equal(t, "octocat", reviewers[2])
	})

	t.Run("returns error for invalid user numbers", func(t *testing.T) {
		cfg := &config.Config{
			Users: config.UsersConfig{
				SavedUsers: []config.SavedUser{
					{Email: "user1@example.com", Name: "User One"},
				},
				UseGitHistory: func() *bool { b := false; return &b }(),
			},
		}

		_, err := resolveReviewers([]string{"1", "999"}, cfg)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "user number '999' not found")
	})

	t.Run("handles empty slice", func(t *testing.T) {
		cfg := &config.Config{
			Users: config.UsersConfig{
				SavedUsers:    []config.SavedUser{},
				UseGitHistory: func() *bool { b := false; return &b }(),
			},
		}

		reviewers, err := resolveReviewers([]string{}, cfg)
		require.NoError(t, err)
		assert.Empty(t, reviewers)
	})

	t.Run("skips empty strings in specs", func(t *testing.T) {
		cfg := &config.Config{
			Users: config.UsersConfig{
				SavedUsers: []config.SavedUser{
					{Email: "user1@example.com", Name: "User One"},
				},
				UseGitHistory: func() *bool { b := false; return &b }(),
			},
		}

		reviewers, err := resolveReviewers([]string{"1", "", "  ", "user@example.com"}, cfg)
		require.NoError(t, err)
		require.Len(t, reviewers, 2)
		assert.Equal(t, "user1@example.com", reviewers[0])
		assert.Equal(t, "user@example.com", reviewers[1])
	})

	t.Run("trims whitespace from specs", func(t *testing.T) {
		cfg := &config.Config{
			Users: config.UsersConfig{
				SavedUsers: []config.SavedUser{
					{Email: "user1@example.com", Name: "User One"},
				},
				UseGitHistory: func() *bool { b := false; return &b }(),
			},
		}

		reviewers, err := resolveReviewers([]string{" 1 ", " user@example.com ", " octocat "}, cfg)
		require.NoError(t, err)
		require.Len(t, reviewers, 3)
		assert.Equal(t, "user1@example.com", reviewers[0])
		assert.Equal(t, "user@example.com", reviewers[1])
		assert.Equal(t, "octocat", reviewers[2])
	})

	t.Run("returns error for nil config", func(t *testing.T) {
		_, err := resolveReviewers([]string{"1"}, nil)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "configuration cannot be nil")
	})

	t.Run("handles usernames that look like numbers but contain non-digits", func(t *testing.T) {
		cfg := &config.Config{
			Users: config.UsersConfig{
				SavedUsers:    []config.SavedUser{},
				UseGitHistory: func() *bool { b := false; return &b }(),
			},
		}

		// "123abc" should be treated as a GitHub username, not a user number
		reviewers, err := resolveReviewers([]string{"123abc"}, cfg)
		require.NoError(t, err)
		require.Len(t, reviewers, 1)
		assert.Equal(t, "123abc", reviewers[0])
	})

	t.Run("handles email addresses with numbers", func(t *testing.T) {
		cfg := &config.Config{
			Users: config.UsersConfig{
				SavedUsers:    []config.SavedUser{},
				UseGitHistory: func() *bool { b := false; return &b }(),
			},
		}

		// "user123@example.com" should be treated as an email, not a user number
		reviewers, err := resolveReviewers([]string{"user123@example.com"}, cfg)
		require.NoError(t, err)
		require.Len(t, reviewers, 1)
		assert.Equal(t, "user123@example.com", reviewers[0])
	})
}

// TestRequestPRReviews tests the requestPRReviews function
// Note: These tests verify function structure, parameter validation, and config respect.
// Actual API calls with real GitHub tokens should be tested in integration tests.
func TestRequestPRReviews(t *testing.T) {
	t.Run("returns error for nil client", func(t *testing.T) {
		cfg := &config.Config{
			Review: &config.ReviewConfig{
				AutoRequestReviews: func() *bool { b := true; return &b }(),
			},
		}
		err := requestPRReviews(nil, "owner", "repo", 1, []string{"reviewer1"}, cfg)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "GitHub client cannot be nil")
	})

	t.Run("returns error for empty owner", func(t *testing.T) {
		client, _ := gh.CreateGitHubClient("test-token")
		cfg := &config.Config{
			Review: &config.ReviewConfig{
				AutoRequestReviews: func() *bool { b := true; return &b }(),
			},
		}
		err := requestPRReviews(client, "", "repo", 1, []string{"reviewer1"}, cfg)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "owner cannot be empty")
	})

	t.Run("returns error for empty repo", func(t *testing.T) {
		client, _ := gh.CreateGitHubClient("test-token")
		cfg := &config.Config{
			Review: &config.ReviewConfig{
				AutoRequestReviews: func() *bool { b := true; return &b }(),
			},
		}
		err := requestPRReviews(client, "owner", "", 1, []string{"reviewer1"}, cfg)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "repo cannot be empty")
	})

	t.Run("returns error for invalid prNumber", func(t *testing.T) {
		client, _ := gh.CreateGitHubClient("test-token")
		cfg := &config.Config{
			Review: &config.ReviewConfig{
				AutoRequestReviews: func() *bool { b := true; return &b }(),
			},
		}
		err := requestPRReviews(client, "owner", "repo", 0, []string{"reviewer1"}, cfg)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "PR number must be greater than 0")
	})

	t.Run("returns error for negative prNumber", func(t *testing.T) {
		client, _ := gh.CreateGitHubClient("test-token")
		cfg := &config.Config{
			Review: &config.ReviewConfig{
				AutoRequestReviews: func() *bool { b := true; return &b }(),
			},
		}
		err := requestPRReviews(client, "owner", "repo", -1, []string{"reviewer1"}, cfg)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "PR number must be greater than 0")
	})

	t.Run("returns error for nil reviewers", func(t *testing.T) {
		client, _ := gh.CreateGitHubClient("test-token")
		cfg := &config.Config{
			Review: &config.ReviewConfig{
				AutoRequestReviews: func() *bool { b := true; return &b }(),
			},
		}
		err := requestPRReviews(client, "owner", "repo", 1, nil, cfg)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "reviewers cannot be nil")
	})

	t.Run("returns error for nil config", func(t *testing.T) {
		client, _ := gh.CreateGitHubClient("test-token")
		err := requestPRReviews(client, "owner", "repo", 1, []string{"reviewer1"}, nil)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "configuration cannot be nil")
	})

	t.Run("returns error for whitespace-only owner", func(t *testing.T) {
		client, _ := gh.CreateGitHubClient("test-token")
		cfg := &config.Config{
			Review: &config.ReviewConfig{
				AutoRequestReviews: func() *bool { b := true; return &b }(),
			},
		}
		err := requestPRReviews(client, "   ", "repo", 1, []string{"reviewer1"}, cfg)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "owner cannot be empty")
	})

	t.Run("returns error for whitespace-only repo", func(t *testing.T) {
		client, _ := gh.CreateGitHubClient("test-token")
		cfg := &config.Config{
			Review: &config.ReviewConfig{
				AutoRequestReviews: func() *bool { b := true; return &b }(),
			},
		}
		err := requestPRReviews(client, "owner", "   ", 1, []string{"reviewer1"}, cfg)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "repo cannot be empty")
	})

	t.Run("returns early for empty reviewers list", func(t *testing.T) {
		client, _ := gh.CreateGitHubClient("test-token")
		cfg := &config.Config{
			Review: &config.ReviewConfig{
				AutoRequestReviews: func() *bool { b := true; return &b }(),
			},
		}
		// Should return nil (no error) when reviewers list is empty
		err := requestPRReviews(client, "owner", "repo", 1, []string{}, cfg)
		require.NoError(t, err)
	})

	t.Run("skips requesting reviews when AutoRequestReviews is false", func(t *testing.T) {
		client, _ := gh.CreateGitHubClient("test-token")
		cfg := &config.Config{
			Review: &config.ReviewConfig{
				AutoRequestReviews: func() *bool { b := false; return &b }(),
			},
		}
		// Should return nil (no error) when auto-request is disabled
		err := requestPRReviews(client, "owner", "repo", 1, []string{"reviewer1"}, cfg)
		require.NoError(t, err)
	})

	t.Run("requests reviews when AutoRequestReviews is true", func(t *testing.T) {
		client, _ := gh.CreateGitHubClient("test-token")
		cfg := &config.Config{
			Review: &config.ReviewConfig{
				AutoRequestReviews: func() *bool { b := true; return &b }(),
			},
		}
		// This will fail at API call (no real token), but validation should pass
		err := requestPRReviews(client, "owner", "repo", 1, []string{"reviewer1"}, cfg)
		// Error should be from API call, not validation
		require.Error(t, err)
		// Should not be a validation error
		assert.NotContains(t, err.Error(), "cannot be nil")
		assert.NotContains(t, err.Error(), "cannot be empty")
		assert.NotContains(t, err.Error(), "must be greater than 0")
	})

	t.Run("defaults to true when config.Review is nil", func(t *testing.T) {
		client, _ := gh.CreateGitHubClient("test-token")
		cfg := &config.Config{
			Review: nil, // No review config
		}
		// Should default to true and attempt to request reviews
		// This will fail at API call (no real token), but validation should pass
		err := requestPRReviews(client, "owner", "repo", 1, []string{"reviewer1"}, cfg)
		// Error should be from API call, not validation
		require.Error(t, err)
		// Should not be a validation error
		assert.NotContains(t, err.Error(), "cannot be nil")
		assert.NotContains(t, err.Error(), "cannot be empty")
	})

	t.Run("defaults to true when AutoRequestReviews is nil", func(t *testing.T) {
		client, _ := gh.CreateGitHubClient("test-token")
		cfg := &config.Config{
			Review: &config.ReviewConfig{
				AutoRequestReviews: nil, // Not set
			},
		}
		// Should default to true and attempt to request reviews
		// This will fail at API call (no real token), but validation should pass
		err := requestPRReviews(client, "owner", "repo", 1, []string{"reviewer1"}, cfg)
		// Error should be from API call, not validation
		require.Error(t, err)
		// Should not be a validation error
		assert.NotContains(t, err.Error(), "cannot be nil")
		assert.NotContains(t, err.Error(), "cannot be empty")
	})

	// Note: Tests for actual API calls (requesting reviews, handling API errors, etc.)
	// should be done in integration tests with a real GitHub token and test repository.
	// The unit tests above verify:
	// 1. Input validation (nil client, empty parameters, invalid prNumber, nil reviewers, nil config)
	// 2. Config respect (AutoRequestReviews true/false/nil/default)
	// 3. Early return for empty reviewers list
	// 4. Error handling structure
	// 5. Function signature and return types
	//
	// Integration tests should verify:
	// - Requesting reviews successfully
	// - Handling API errors (401 Unauthorized, 403 Forbidden, 404 Not Found, 422 Unprocessable Entity)
	// - Handling invalid reviewers (422 errors)
	// - Handling network errors
	// - Handling context timeout
}

// TestUpdateTrunkStatus tests the updateTrunkStatus function
const testWorkItemContentForTrunkStatus = `---
id: 012
title: Test Feature
status: doing
kind: prd
created: 2024-01-01
---
# Test Feature
`

func TestUpdateTrunkStatus(t *testing.T) {
	t.Run("updates trunk status successfully when work item exists on trunk", func(t *testing.T) {
		tmpDir := setupTestGitRepo(t, "main")
		_ = tmpDir

		// We're already on main branch from setupTestGitRepo

		// Create .work directory structure on trunk
		require.NoError(t, os.MkdirAll(".work/2_doing", 0o700))
		require.NoError(t, os.MkdirAll(".work/3_review", 0o700))

		// Create work item on trunk in doing status
		require.NoError(t, os.WriteFile(".work/2_doing/012-test-feature.prd.md", []byte(testWorkItemContentForTrunkStatus), 0o600))
		require.NoError(t, exec.Command("git", "add", ".work").Run())
		require.NoError(t, exec.Command("git", "commit", "-m", "Add work item").Run())

		// Create feature branch
		require.NoError(t, exec.Command("git", "checkout", "-b", "012-test-feature").Run())

		cfg := &config.Config{
			Git: &config.GitConfig{
				TrunkBranch: "main",
				Remote:      "origin",
			},
			StatusFolders: map[string]string{
				"doing":  "2_doing",
				"review": "3_review",
			},
		}

		// Update trunk status
		err := updateTrunkStatus("012", cfg)
		require.NoError(t, err)

		// Verify we're back on feature branch
		currentBranch, err := getCurrentBranch("")
		require.NoError(t, err)
		assert.Equal(t, "012-test-feature", currentBranch)

		// Verify work item was moved to review on trunk
		require.NoError(t, exec.Command("git", "checkout", "main").Run())
		content, err := os.ReadFile(".work/3_review/012-test-feature.prd.md")
		require.NoError(t, err)
		assert.Contains(t, string(content), "status: review")
	})

	t.Run("copies work item from feature branch when not on trunk", func(t *testing.T) {
		tmpDir := setupTestGitRepo(t, "main")
		_ = tmpDir

		// We're already on main branch from setupTestGitRepo
		require.NoError(t, os.MkdirAll(".work/3_review", 0o700))
		// Create a placeholder file so git can track the directory
		require.NoError(t, os.WriteFile(".work/3_review/.gitkeep", []byte(""), 0o600))
		require.NoError(t, exec.Command("git", "add", ".work").Run())
		require.NoError(t, exec.Command("git", "commit", "-m", "Initial commit").Run())

		// Create feature branch with work item
		require.NoError(t, exec.Command("git", "checkout", "-b", "012-test-feature").Run())
		require.NoError(t, os.MkdirAll(".work/2_doing", 0o700))

		require.NoError(t, os.WriteFile(".work/2_doing/012-test-feature.prd.md", []byte(testWorkItemContentForTrunkStatus), 0o600))
		require.NoError(t, exec.Command("git", "add", ".work").Run())
		require.NoError(t, exec.Command("git", "commit", "-m", "Add work item").Run())

		cfg := &config.Config{
			Git: &config.GitConfig{
				TrunkBranch: "main",
				Remote:      "origin",
			},
			StatusFolders: map[string]string{
				"doing":  "2_doing",
				"review": "3_review",
			},
		}

		// Update trunk status (should copy from feature branch)
		err := updateTrunkStatus("012", cfg)
		require.NoError(t, err)

		// Verify we're back on feature branch
		currentBranch, err := getCurrentBranch("")
		require.NoError(t, err)
		assert.Equal(t, "012-test-feature", currentBranch)

		// Verify work item was copied and moved to review on trunk
		require.NoError(t, exec.Command("git", "checkout", "main").Run())
		content, err := os.ReadFile(".work/3_review/012-test-feature.prd.md")
		require.NoError(t, err)
		assert.Contains(t, string(content), "status: review")
		assert.Contains(t, string(content), "id: 012")
		assert.Contains(t, string(content), "title: Test Feature")
	})

	t.Run("handles stash failures gracefully when no changes to stash", func(t *testing.T) {
		tmpDir := setupTestGitRepo(t, "main")
		_ = tmpDir

		// We're already on main branch from setupTestGitRepo
		require.NoError(t, os.MkdirAll(".work/2_doing", 0o700))
		require.NoError(t, os.MkdirAll(".work/3_review", 0o700))

		require.NoError(t, os.WriteFile(".work/2_doing/012-test-feature.prd.md", []byte(testWorkItemContentForTrunkStatus), 0o600))
		require.NoError(t, exec.Command("git", "add", ".work").Run())
		require.NoError(t, exec.Command("git", "commit", "-m", "Add work item").Run())

		// Create feature branch (clean working directory - no changes to stash)
		require.NoError(t, exec.Command("git", "checkout", "-b", "012-test-feature").Run())

		cfg := &config.Config{
			Git: &config.GitConfig{
				TrunkBranch: "main",
				Remote:      "origin",
			},
			StatusFolders: map[string]string{
				"doing":  "2_doing",
				"review": "3_review",
			},
		}

		// Should succeed even though there's nothing to stash
		err := updateTrunkStatus("012", cfg)
		require.NoError(t, err)
	})

	t.Run("restores stashed uncommitted changes after trunk update", func(t *testing.T) {
		tmpDir := setupTestGitRepo(t, "main")
		_ = tmpDir

		require.NoError(t, os.MkdirAll(".work/2_doing", 0o700))
		require.NoError(t, os.MkdirAll(".work/3_review", 0o700))
		require.NoError(t, os.WriteFile(".work/2_doing/012-test-feature.prd.md", []byte(testWorkItemContentForTrunkStatus), 0o600))
		require.NoError(t, exec.Command("git", "add", ".work").Run())
		require.NoError(t, exec.Command("git", "commit", "-m", "Add work item").Run())

		require.NoError(t, exec.Command("git", "checkout", "-b", "012-test-feature").Run())

		// Leave uncommitted change on feature branch
		require.NoError(t, os.WriteFile("uncommitted.txt", []byte("stash me"), 0o600))

		cfg := &config.Config{
			Git: &config.GitConfig{
				TrunkBranch: "main",
				Remote:      "origin",
			},
			StatusFolders: map[string]string{
				"doing":  "2_doing",
				"review": "3_review",
			},
		}

		err := updateTrunkStatus("012", cfg)
		require.NoError(t, err)

		currentBranch, err := getCurrentBranch("")
		require.NoError(t, err)
		assert.Equal(t, "012-test-feature", currentBranch)

		// Stashed changes should be restored
		content, err := os.ReadFile("uncommitted.txt")
		require.NoError(t, err)
		assert.Equal(t, "stash me", string(content))
	})

	t.Run("switches branches correctly", func(t *testing.T) {
		tmpDir := setupTestGitRepo(t, "main")
		_ = tmpDir

		// We're already on main branch from setupTestGitRepo
		require.NoError(t, os.MkdirAll(".work/2_doing", 0o700))
		require.NoError(t, os.MkdirAll(".work/3_review", 0o700))

		require.NoError(t, os.WriteFile(".work/2_doing/012-test-feature.prd.md", []byte(testWorkItemContentForTrunkStatus), 0o600))
		require.NoError(t, exec.Command("git", "add", ".work").Run())
		require.NoError(t, exec.Command("git", "commit", "-m", "Add work item").Run())

		// Create feature branch
		require.NoError(t, exec.Command("git", "checkout", "-b", "012-test-feature").Run())

		cfg := &config.Config{
			Git: &config.GitConfig{
				TrunkBranch: "main",
				Remote:      "origin",
			},
			StatusFolders: map[string]string{
				"doing":  "2_doing",
				"review": "3_review",
			},
		}

		// Verify we start on feature branch
		currentBranch, err := getCurrentBranch("")
		require.NoError(t, err)
		assert.Equal(t, "012-test-feature", currentBranch)

		// Update trunk status
		err = updateTrunkStatus("012", cfg)
		require.NoError(t, err)

		// Verify we're back on feature branch
		currentBranch, err = getCurrentBranch("")
		require.NoError(t, err)
		assert.Equal(t, "012-test-feature", currentBranch)
	})

	t.Run("returns error for nil config", func(t *testing.T) {
		err := updateTrunkStatus("012", nil)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "configuration cannot be nil")
	})

	t.Run("returns error for empty work item ID", func(t *testing.T) {
		cfg := &config.Config{
			Git: &config.GitConfig{
				TrunkBranch: "main",
			},
		}
		err := updateTrunkStatus("", cfg)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "work item ID cannot be empty")
	})

	t.Run("returns error when work item not found on feature branch", func(t *testing.T) {
		tmpDir := setupTestGitRepo(t, "main")
		_ = tmpDir

		// We're already on main branch from setupTestGitRepo
		require.NoError(t, os.MkdirAll(".work/3_review", 0o700))
		// Create a placeholder file so git can track the directory
		require.NoError(t, os.WriteFile(".work/3_review/.gitkeep", []byte(""), 0o600))
		require.NoError(t, exec.Command("git", "add", ".work").Run())
		require.NoError(t, exec.Command("git", "commit", "-m", "Initial commit").Run())

		// Create feature branch without work item (but with a commit so branch exists)
		require.NoError(t, exec.Command("git", "checkout", "-b", "012-test-feature").Run())
		require.NoError(t, os.WriteFile("test2.txt", []byte("test"), 0o600))
		require.NoError(t, exec.Command("git", "add", "test2.txt").Run())
		require.NoError(t, exec.Command("git", "commit", "-m", "Feature branch commit").Run())

		cfg := &config.Config{
			Git: &config.GitConfig{
				TrunkBranch: "main",
				Remote:      "origin",
			},
			StatusFolders: map[string]string{
				"doing":  "2_doing",
				"review": "3_review",
			},
		}

		// Should fail because work item doesn't exist
		err := updateTrunkStatus("012", cfg)
		require.Error(t, err)
		assert.True(t,
			strings.Contains(err.Error(), "not found on feature branch") ||
				strings.Contains(err.Error(), "work item 012 not found"),
			"error should mention work item not found: %s", err.Error())
	})
}

// TestPerformRebase tests the performRebase function.
func TestPerformRebase(t *testing.T) {
	t.Run("rebases successfully when no conflicts", func(t *testing.T) {
		_ = setupTestGitRepo(t, "main")

		require.NoError(t, exec.Command("git", "checkout", "-b", "012-test-feature").Run())
		require.NoError(t, os.WriteFile("feature.txt", []byte("feature"), 0o600))
		require.NoError(t, exec.Command("git", "add", "feature.txt").Run())
		require.NoError(t, exec.Command("git", "commit", "-m", "Feature commit").Run())

		cfg := &config.Config{
			Git: &config.GitConfig{
				TrunkBranch: "main",
				Remote:      "origin",
			},
			StatusFolders: map[string]string{
				"doing":  "2_doing",
				"review": "3_review",
			},
		}

		err := performRebase(cfg)
		require.NoError(t, err)

		currentBranch, err := getCurrentBranch("")
		require.NoError(t, err)
		assert.Equal(t, "012-test-feature", currentBranch)
	})

	t.Run("detects and reports conflicts with clear error message", func(t *testing.T) {
		_ = setupTestGitRepo(t, "main")

		require.NoError(t, os.WriteFile("conflict.txt", []byte("line1\n"), 0o600))
		require.NoError(t, exec.Command("git", "add", "conflict.txt").Run())
		require.NoError(t, exec.Command("git", "commit", "-m", "Add conflict file").Run())

		require.NoError(t, exec.Command("git", "checkout", "-b", "012-test-feature").Run())
		require.NoError(t, os.WriteFile("conflict.txt", []byte("line1\nfeature\n"), 0o600))
		require.NoError(t, exec.Command("git", "add", "conflict.txt").Run())
		require.NoError(t, exec.Command("git", "commit", "-m", "Feature change").Run())

		require.NoError(t, exec.Command("git", "checkout", "main").Run())
		require.NoError(t, os.WriteFile("conflict.txt", []byte("line1\nmain\n"), 0o600))
		require.NoError(t, exec.Command("git", "add", "conflict.txt").Run())
		require.NoError(t, exec.Command("git", "commit", "-m", "Main change").Run())

		require.NoError(t, exec.Command("git", "checkout", "012-test-feature").Run())

		cfg := &config.Config{
			Git: &config.GitConfig{
				TrunkBranch: "main",
				Remote:      "origin",
			},
			StatusFolders: map[string]string{
				"doing":  "2_doing",
				"review": "3_review",
			},
		}

		err := performRebase(cfg)
		require.Error(t, err)
		errStr := err.Error()
		assert.Contains(t, errStr, "rebase conflicts detected", "error should mention conflicts")
		assert.Contains(t, errStr, "git rebase --continue", "error should mention resolution step")
		assert.Contains(t, errStr, "kira review", "error should mention re-run command")
	})
}
