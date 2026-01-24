package commands

import (
	"bytes"
	"errors"
	"os"
	"os/exec"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"kira/internal/config"
	"kira/internal/validation"
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
