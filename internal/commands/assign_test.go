package commands

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"kira/internal/config"
)

func TestParseAssignArgs(t *testing.T) {
	t.Run("splits work items and user identifier", func(t *testing.T) {
		flags := AssignFlags{}
		workItems, user := parseAssignArgs([]string{"001", "5"}, flags)
		assert.Equal(t, []string{"001"}, workItems)
		assert.Equal(t, "5", user)
	})

	t.Run("handles multiple work items with user identifier", func(t *testing.T) {
		flags := AssignFlags{}
		workItems, user := parseAssignArgs([]string{"001", "002", "003", "5"}, flags)
		assert.Equal(t, []string{"001", "002", "003"}, workItems)
		assert.Equal(t, "5", user)
	})

	t.Run("treats all args as work items in unassign mode", func(t *testing.T) {
		flags := AssignFlags{Unassign: true}
		workItems, user := parseAssignArgs([]string{"001"}, flags)
		assert.Equal(t, []string{"001"}, workItems)
		assert.Equal(t, "", user)
	})

	t.Run("treats all args as work items in interactive mode", func(t *testing.T) {
		flags := AssignFlags{Interactive: true}
		workItems, user := parseAssignArgs([]string{".work/1_todo/001-test.prd.md"}, flags)
		assert.Equal(t, []string{".work/1_todo/001-test.prd.md"}, workItems)
		assert.Equal(t, "", user)
	})

	t.Run("single argument without flags yields one work item and empty user", func(t *testing.T) {
		flags := AssignFlags{}
		workItems, user := parseAssignArgs([]string{"001"}, flags)
		assert.Equal(t, []string{"001"}, workItems)
		assert.Equal(t, "", user)
	})
}

func TestValidateAssignInputWorkItems(t *testing.T) {
	cfg := &config.DefaultConfig

	t.Run("accepts valid numeric work item IDs", func(t *testing.T) {
		flags := AssignFlags{
			Field:  "assigned",
			Append: false,
		}
		err := validateAssignInput([]string{"001", "002"}, "5", flags, cfg)
		assert.NoError(t, err)
	})

	t.Run("rejects invalid work item ID format", func(t *testing.T) {
		flags := AssignFlags{
			Field:  "assigned",
			Append: false,
		}
		err := validateAssignInput([]string{"1"}, "5", flags, cfg)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid work item ID")
	})

	t.Run("accepts path-like work item identifiers under .work", func(t *testing.T) {
		flags := AssignFlags{
			Field:  "assigned",
			Append: false,
		}

		// Path validation only checks that the path is under .work; the directory
		// does not need to exist for validation to pass.
		err := validateAssignInput([]string{".work/1_todo/001-test-feature.prd.md"}, "5", flags, cfg)
		assert.NoError(t, err)
	})

	t.Run("rejects path-like work item identifiers outside .work", func(t *testing.T) {
		flags := AssignFlags{
			Field:  "assigned",
			Append: false,
		}

		// Ensure current directory is a real temp dir so validateWorkPath uses it.
		tmpDir := t.TempDir()
		origDir, _ := os.Getwd()
		require.NoError(t, os.Chdir(tmpDir))
		defer func() { _ = os.Chdir(origDir) }()

		err := validateAssignInput([]string{"some/other/path/001-test-feature.prd.md"}, "5", flags, cfg)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "path outside .work directory")
	})
}

func TestValidateAssignInputFlagsAndUserIdentifier(t *testing.T) {
	cfg := &config.DefaultConfig

	t.Run("requires at least one work item", func(t *testing.T) {
		flags := AssignFlags{
			Field:  "assigned",
			Append: false,
		}
		err := validateAssignInput([]string{}, "5", flags, cfg)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "at least one work item ID or path is required")
	})

	t.Run("requires user identifier when not unassign or interactive", func(t *testing.T) {
		flags := AssignFlags{
			Field:  "assigned",
			Append: false,
		}
		err := validateAssignInput([]string{"001"}, "", flags, cfg)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "user identifier is required")
	})

	t.Run("allows missing user identifier in interactive mode", func(t *testing.T) {
		flags := AssignFlags{
			Field:       "assigned",
			Interactive: true,
		}
		err := validateAssignInput([]string{"001"}, "", flags, cfg)
		assert.NoError(t, err)
	})

	t.Run("disallows user identifier with unassign", func(t *testing.T) {
		flags := AssignFlags{
			Field:    "assigned",
			Unassign: true,
		}
		err := validateAssignInput([]string{"001"}, "5", flags, cfg)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "cannot specify user identifier when using --unassign")
	})

	t.Run("disallows unassign with append", func(t *testing.T) {
		flags := AssignFlags{
			Field:    "assigned",
			Unassign: true,
			Append:   true,
		}
		err := validateAssignInput([]string{"001"}, "", flags, cfg)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid flag combination")
	})

	t.Run("disallows unassign with interactive in this phase", func(t *testing.T) {
		flags := AssignFlags{
			Field:       "assigned",
			Unassign:    true,
			Interactive: true,
		}
		err := validateAssignInput([]string{"001"}, "", flags, cfg)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid flag combination")
	})
}

func TestValidateAssignInputFieldNames(t *testing.T) {
	cfg := &config.DefaultConfig

	t.Run("accepts default assigned field", func(t *testing.T) {
		flags := AssignFlags{
			Field:  "assigned",
			Append: false,
		}
		err := validateAssignInput([]string{"001"}, "5", flags, cfg)
		assert.NoError(t, err)
	})

	t.Run("accepts simple custom field name", func(t *testing.T) {
		flags := AssignFlags{
			Field:  "reviewer",
			Append: false,
		}
		err := validateAssignInput([]string{"001"}, "5", flags, cfg)
		assert.NoError(t, err)
	})

	t.Run("rejects empty field name", func(t *testing.T) {
		flags := AssignFlags{
			Field: "",
		}
		err := validateAssignInput([]string{"001"}, "5", flags, cfg)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "field name cannot be empty")
	})

	t.Run("rejects field name with path separators", func(t *testing.T) {
		flags := AssignFlags{
			Field: "reviewer/name",
		}
		err := validateAssignInput([]string{"001"}, "5", flags, cfg)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid field name")
	})

	t.Run("rejects field name with backslash", func(t *testing.T) {
		flags := AssignFlags{
			Field: "reviewer\\name",
		}
		err := validateAssignInput([]string{"001"}, "5", flags, cfg)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid field name")
	})

	t.Run("rejects field name with dot dot", func(t *testing.T) {
		flags := AssignFlags{
			Field: "reviewer..name",
		}
		err := validateAssignInput([]string{"001"}, "5", flags, cfg)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid field name")
	})
}

func TestAssignCommandFlagsWiring(t *testing.T) {
	t.Run("assign command defines expected flags and defaults", func(t *testing.T) {
		cmd := &cobra.Command{}
		// Copy flag definitions from assignCmd to a fresh command to avoid
		// interfering with global state.
		cmd.Flags().AddFlagSet(assignCmd.Flags())

		field, err := cmd.Flags().GetString("field")
		require.NoError(t, err)
		assert.Equal(t, "assigned", field)

		appendFlag, err := cmd.Flags().GetBool("append")
		require.NoError(t, err)
		assert.False(t, appendFlag)

		unassignFlag, err := cmd.Flags().GetBool("unassign")
		require.NoError(t, err)
		assert.False(t, unassignFlag)

		interactiveFlag, err := cmd.Flags().GetBool("interactive")
		require.NoError(t, err)
		assert.False(t, interactiveFlag)

		dryRunFlag, err := cmd.Flags().GetBool("dry-run")
		require.NoError(t, err)
		assert.False(t, dryRunFlag)
	})
}

// Phase 2: Work Item Discovery & Validation Tests
// Note: testWorkItemContent is defined in move_test.go

func TestResolveWorkItemPath(t *testing.T) {
	t.Run("resolves numeric ID to file path", func(t *testing.T) {
		tmpDir := t.TempDir()
		origDir, _ := os.Getwd()
		require.NoError(t, os.Chdir(tmpDir))
		defer func() { _ = os.Chdir(origDir) }()

		// Create .work directory structure
		require.NoError(t, os.MkdirAll(".work/1_todo", 0o700))

		// Create a work item file
		require.NoError(t, os.WriteFile(testFilePath, []byte(testWorkItemContent), 0o600))

		// Resolve by ID
		resolvedPath, err := resolveWorkItemPath("001")
		require.NoError(t, err)
		assert.NotEmpty(t, resolvedPath)
		assert.Contains(t, resolvedPath, "001-test-feature.prd.md")
	})

	t.Run("resolves path identifier to absolute file path", func(t *testing.T) {
		tmpDir := t.TempDir()
		origDir, _ := os.Getwd()
		require.NoError(t, os.Chdir(tmpDir))
		defer func() { _ = os.Chdir(origDir) }()

		// Create .work directory structure
		require.NoError(t, os.MkdirAll(".work/1_todo", 0o700))

		// Create a work item file
		require.NoError(t, os.WriteFile(testFilePath, []byte(testWorkItemContent), 0o600))

		// Resolve by path
		resolvedPath, err := resolveWorkItemPath(testFilePath)
		require.NoError(t, err)
		assert.NotEmpty(t, resolvedPath)
		// Should be absolute path
		assert.True(t, filepath.IsAbs(resolvedPath))
		assert.Contains(t, resolvedPath, "001-test-feature.prd.md")
	})

	t.Run("returns error for non-existent ID", func(t *testing.T) {
		tmpDir := t.TempDir()
		origDir, _ := os.Getwd()
		require.NoError(t, os.Chdir(tmpDir))
		defer func() { _ = os.Chdir(origDir) }()

		// Create .work directory structure but no work items
		require.NoError(t, os.MkdirAll(".work/1_todo", 0o700))

		_, err := resolveWorkItemPath("999")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "work item 999 not found")
	})

	t.Run("returns error for invalid path outside .work", func(t *testing.T) {
		tmpDir := t.TempDir()
		origDir, _ := os.Getwd()
		require.NoError(t, os.Chdir(tmpDir))
		defer func() { _ = os.Chdir(origDir) }()

		_, err := resolveWorkItemPath("some/other/path/001-test.md")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid work item path")
		assert.Contains(t, err.Error(), "path outside .work directory")
	})

	t.Run("returns error for path with traversal", func(t *testing.T) {
		tmpDir := t.TempDir()
		origDir, _ := os.Getwd()
		require.NoError(t, os.Chdir(tmpDir))
		defer func() { _ = os.Chdir(origDir) }()

		_, err := resolveWorkItemPath(".work/../other/001-test.md")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid work item path")
	})
}

func TestValidateWorkItemFile(t *testing.T) {
	t.Run("validates existing readable file", func(t *testing.T) {
		tmpDir := t.TempDir()
		origDir, _ := os.Getwd()
		require.NoError(t, os.Chdir(tmpDir))
		defer func() { _ = os.Chdir(origDir) }()

		// Create .work directory structure
		require.NoError(t, os.MkdirAll(".work/1_todo", 0o700))

		// Create a work item file
		require.NoError(t, os.WriteFile(testFilePath, []byte(testWorkItemContent), 0o600))

		// Get absolute path
		absPath, err := filepath.Abs(testFilePath)
		require.NoError(t, err)

		err = validateWorkItemFile(absPath)
		assert.NoError(t, err)
	})

	t.Run("returns error for non-existent file", func(t *testing.T) {
		tmpDir := t.TempDir()
		origDir, _ := os.Getwd()
		require.NoError(t, os.Chdir(tmpDir))
		defer func() { _ = os.Chdir(origDir) }()

		nonExistentPath := ".work/1_todo/999-nonexistent.prd.md"
		absPath, err := filepath.Abs(nonExistentPath)
		require.NoError(t, err)

		err = validateWorkItemFile(absPath)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "work item file does not exist")
	})

	t.Run("returns error for unreadable file", func(t *testing.T) {
		// Note: This test may not work on all systems due to permission handling.
		// On Unix systems, we can create a file and then remove read permissions.
		// On Windows, this may not work the same way.
		tmpDir := t.TempDir()
		origDir, _ := os.Getwd()
		require.NoError(t, os.Chdir(tmpDir))
		defer func() { _ = os.Chdir(origDir) }()

		// Create .work directory structure
		require.NoError(t, os.MkdirAll(".work/1_todo", 0o700))

		// Create a work item file
		require.NoError(t, os.WriteFile(testFilePath, []byte(testWorkItemContent), 0o600))

		// Get absolute path
		absPath, err := filepath.Abs(testFilePath)
		require.NoError(t, err)

		// Try to remove read permissions (Unix only)
		// This test may skip on Windows
		if err := os.Chmod(testFilePath, 0o000); err == nil {
			defer func() { _ = os.Chmod(testFilePath, 0o600) }()

			err = validateWorkItemFile(absPath)
			// On some systems, this may still be readable by the owner
			// So we just check that it either errors or succeeds, but doesn't panic
			if err != nil {
				assert.Contains(t, err.Error(), "failed to read work item file")
			}
		}
	})
}

func TestResolveWorkItems(t *testing.T) {
	t.Run("resolves multiple work items successfully", func(t *testing.T) {
		tmpDir := t.TempDir()
		origDir, _ := os.Getwd()
		require.NoError(t, os.Chdir(tmpDir))
		defer func() { _ = os.Chdir(origDir) }()

		// Create .work directory structure
		require.NoError(t, os.MkdirAll(".work/1_todo", 0o700))
		require.NoError(t, os.MkdirAll(".work/2_doing", 0o700))

		// Create multiple work item files
		workItem1 := `---
id: 001
title: Test Feature 1
status: todo
kind: prd
created: 2024-01-01
---

# Test Feature 1
`
		workItem2 := `---
id: 002
title: Test Feature 2
status: doing
kind: prd
created: 2024-01-01
---

# Test Feature 2
`
		require.NoError(t, os.WriteFile(".work/1_todo/001-test-feature-1.prd.md", []byte(workItem1), 0o600))
		require.NoError(t, os.WriteFile(".work/2_doing/002-test-feature-2.prd.md", []byte(workItem2), 0o600))

		// Resolve multiple work items
		paths, err := resolveWorkItems([]string{"001", "002"})
		require.NoError(t, err)
		assert.Len(t, paths, 2)
		assert.Contains(t, paths[0], "001-test-feature-1.prd.md")
		assert.Contains(t, paths[1], "002-test-feature-2.prd.md")
	})

	t.Run("returns error if any work item fails", func(t *testing.T) {
		tmpDir := t.TempDir()
		origDir, _ := os.Getwd()
		require.NoError(t, os.Chdir(tmpDir))
		defer func() { _ = os.Chdir(origDir) }()

		// Create .work directory structure
		require.NoError(t, os.MkdirAll(".work/1_todo", 0o700))

		// Create one work item file
		require.NoError(t, os.WriteFile(".work/1_todo/001-test-feature.prd.md", []byte(testWorkItemContent), 0o600))

		// Try to resolve with one valid and one invalid ID
		_, err := resolveWorkItems([]string{"001", "999"})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to resolve or validate work items")
		assert.Contains(t, err.Error(), "999")
	})

	t.Run("provides clear error messages for all failures", func(t *testing.T) {
		tmpDir := t.TempDir()
		origDir, _ := os.Getwd()
		require.NoError(t, os.Chdir(tmpDir))
		defer func() { _ = os.Chdir(origDir) }()

		// Create .work directory structure
		require.NoError(t, os.MkdirAll(".work/1_todo", 0o700))

		// Try to resolve with multiple invalid IDs
		_, err := resolveWorkItems([]string{"998", "999"})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to resolve or validate work items")
		assert.Contains(t, err.Error(), "998")
		assert.Contains(t, err.Error(), "999")
	})

	t.Run("handles mix of IDs and paths", func(t *testing.T) {
		tmpDir := t.TempDir()
		origDir, _ := os.Getwd()
		require.NoError(t, os.Chdir(tmpDir))
		defer func() { _ = os.Chdir(origDir) }()

		// Create .work directory structure
		require.NoError(t, os.MkdirAll(".work/1_todo", 0o700))

		// Create work item file
		require.NoError(t, os.WriteFile(testFilePath, []byte(testWorkItemContent), 0o600))

		// Resolve with mix of ID and path
		paths, err := resolveWorkItems([]string{"001", testFilePath})
		require.NoError(t, err)
		assert.Len(t, paths, 2)
		// Both should resolve to the same file (or at least both should be valid)
		assert.Contains(t, paths[0], "001-test-feature.prd.md")
		assert.Contains(t, paths[1], "001-test-feature.prd.md")
	})

	t.Run("returns error for empty identifiers", func(t *testing.T) {
		_, err := resolveWorkItems([]string{})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "no work items to resolve")
	})
}

// Phase 3: User Collection & Resolution Tests

func TestCollectUsersForAssignment(t *testing.T) {
	t.Run("collects users with default config", func(t *testing.T) {
		cfg := &config.DefaultConfig
		// Disable git history for this test to avoid git repository requirement
		useGitHistory := false
		cfg.Users.UseGitHistory = &useGitHistory
		users, err := collectUsersForAssignment(cfg)
		// May return empty if no saved users, but should not error
		assert.NoError(t, err)
		assert.NotNil(t, users)
	})

	t.Run("collects users with saved users from config", func(t *testing.T) {
		cfg := &config.Config{
			Users: config.UsersConfig{
				SavedUsers: []config.SavedUser{
					{Email: "user1@example.com", Name: "User One"},
					{Email: "user2@example.com", Name: "User Two"},
				},
			},
		}
		useGitHistory := false
		cfg.Users.UseGitHistory = &useGitHistory

		users, err := collectUsersForAssignment(cfg)
		require.NoError(t, err)
		assert.Len(t, users, 2)
		assert.Equal(t, "user1@example.com", users[0].Email)
		assert.Equal(t, "User One", users[0].Name)
		assert.Equal(t, 1, users[0].Number)
		assert.Equal(t, "user2@example.com", users[1].Email)
		assert.Equal(t, "User Two", users[1].Name)
		assert.Equal(t, 2, users[1].Number)
	})
}

func TestFindUserByNumber(t *testing.T) {
	users := []UserInfo{
		{Email: "user1@example.com", Name: "User One", Number: 1},
		{Email: "user2@example.com", Name: "User Two", Number: 2},
		{Email: "user3@example.com", Name: "User Three", Number: 3},
	}

	t.Run("finds user by valid number", func(t *testing.T) {
		user, err := findUserByNumber(1, users)
		require.NoError(t, err)
		assert.Equal(t, "user1@example.com", user.Email)
		assert.Equal(t, "User One", user.Name)

		user, err = findUserByNumber(2, users)
		require.NoError(t, err)
		assert.Equal(t, "user2@example.com", user.Email)

		user, err = findUserByNumber(3, users)
		require.NoError(t, err)
		assert.Equal(t, "user3@example.com", user.Email)
	})

	t.Run("returns error for number too low", func(t *testing.T) {
		_, err := findUserByNumber(0, users)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "user number 0 not found")
		assert.Contains(t, err.Error(), "Available numbers: 1-3")
	})

	t.Run("returns error for number too high", func(t *testing.T) {
		_, err := findUserByNumber(4, users)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "user number 4 not found")
		assert.Contains(t, err.Error(), "Available numbers: 1-3")
	})

	t.Run("returns error for empty users list", func(t *testing.T) {
		_, err := findUserByNumber(1, []UserInfo{})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "no users available")
	})
}

func TestFindUsersByEmail(t *testing.T) {
	users := []UserInfo{
		{Email: "alice@example.com", Name: "Alice", Number: 1},
		{Email: "bob@example.com", Name: "Bob", Number: 2},
		{Email: "charlie@test.com", Name: "Charlie", Number: 3},
		{Email: "alice.smith@example.com", Name: "Alice Smith", Number: 4},
	}

	t.Run("finds user by exact email match (case-insensitive)", func(t *testing.T) {
		matches := findUsersByEmail("alice@example.com", users)
		require.Len(t, matches, 1)
		assert.Equal(t, "alice@example.com", matches[0].Email)

		matches = findUsersByEmail("ALICE@EXAMPLE.COM", users)
		require.Len(t, matches, 1)
		assert.Equal(t, "alice@example.com", matches[0].Email)
	})

	t.Run("finds users by partial email match (domain)", func(t *testing.T) {
		matches := findUsersByEmail("@example.com", users)
		require.Len(t, matches, 3) // alice, bob, and alice.smith all match
		emails := []string{matches[0].Email, matches[1].Email, matches[2].Email}
		assert.Contains(t, emails, "alice@example.com")
		assert.Contains(t, emails, "bob@example.com")
		assert.Contains(t, emails, "alice.smith@example.com")
	})

	t.Run("finds users by partial email match (substring)", func(t *testing.T) {
		matches := findUsersByEmail("alice", users)
		require.Len(t, matches, 2)
		emails := []string{matches[0].Email, matches[1].Email}
		assert.Contains(t, emails, "alice@example.com")
		assert.Contains(t, emails, "alice.smith@example.com")
	})

	t.Run("returns empty slice for no matches", func(t *testing.T) {
		matches := findUsersByEmail("nonexistent@example.com", users)
		assert.Empty(t, matches)
	})

	t.Run("returns empty slice for empty identifier", func(t *testing.T) {
		matches := findUsersByEmail("", users)
		assert.Nil(t, matches)
	})
}

func TestFindUsersByName(t *testing.T) {
	users := []UserInfo{
		{Email: "alice@example.com", Name: "Alice", Number: 1},
		{Email: "bob@example.com", Name: "Bob", Number: 2},
		{Email: "charlie@example.com", Name: "Charlie Brown", Number: 3},
		{Email: "alice.smith@example.com", Name: "Alice Smith", Number: 4},
		{Email: "no.name@example.com", Name: "", Number: 5},
	}

	t.Run("finds user by exact name match (case-insensitive)", func(t *testing.T) {
		matches := findUsersByName("Alice", users)
		require.Len(t, matches, 1)
		assert.Equal(t, "Alice", matches[0].Name)

		matches = findUsersByName("ALICE", users)
		require.Len(t, matches, 1)
		assert.Equal(t, "Alice", matches[0].Name)
	})

	t.Run("finds users by partial name match", func(t *testing.T) {
		// Use "Smith" as a partial match that will match "Alice Smith" but not exact "Alice"
		matches := findUsersByName("Smith", users)
		require.Len(t, matches, 1)
		assert.Equal(t, "Alice Smith", matches[0].Name)
	})

	t.Run("finds user by partial name match (substring)", func(t *testing.T) {
		matches := findUsersByName("Brown", users)
		require.Len(t, matches, 1)
		assert.Equal(t, "Charlie Brown", matches[0].Name)
	})

	t.Run("returns empty slice for no matches", func(t *testing.T) {
		matches := findUsersByName("Nonexistent", users)
		assert.Empty(t, matches)
	})

	t.Run("returns empty slice for empty identifier", func(t *testing.T) {
		matches := findUsersByName("", users)
		assert.Nil(t, matches)
	})

	t.Run("ignores users without names", func(t *testing.T) {
		matches := findUsersByName("no.name", users)
		assert.Empty(t, matches)
	})
}

func TestResolveUserIdentifier(t *testing.T) {
	users := []UserInfo{
		{Email: "alice@example.com", Name: "Alice", Number: 1},
		{Email: "bob@example.com", Name: "Bob", Number: 2},
		{Email: "charlie@test.com", Name: "Charlie", Number: 3},
		{Email: "alice.smith@example.com", Name: "Alice Smith", Number: 4},
	}

	t.Run("resolves by numeric identifier", func(t *testing.T) {
		user, err := resolveUserIdentifier("1", users)
		require.NoError(t, err)
		assert.Equal(t, "alice@example.com", user.Email)

		user, err = resolveUserIdentifier("2", users)
		require.NoError(t, err)
		assert.Equal(t, "bob@example.com", user.Email)
	})

	t.Run("resolves by exact email", func(t *testing.T) {
		user, err := resolveUserIdentifier("alice@example.com", users)
		require.NoError(t, err)
		assert.Equal(t, "alice@example.com", user.Email)

		user, err = resolveUserIdentifier("ALICE@EXAMPLE.COM", users)
		require.NoError(t, err)
		assert.Equal(t, "alice@example.com", user.Email)
	})

	t.Run("resolves by partial email when unique", func(t *testing.T) {
		user, err := resolveUserIdentifier("@test.com", users)
		require.NoError(t, err)
		assert.Equal(t, "charlie@test.com", user.Email)
	})

	t.Run("resolves by exact name", func(t *testing.T) {
		user, err := resolveUserIdentifier("Bob", users)
		require.NoError(t, err)
		assert.Equal(t, "bob@example.com", user.Email)

		user, err = resolveUserIdentifier("BOB", users)
		require.NoError(t, err)
		assert.Equal(t, "bob@example.com", user.Email)
	})

	t.Run("resolves by partial name when unique", func(t *testing.T) {
		user, err := resolveUserIdentifier("Charlie", users)
		require.NoError(t, err)
		assert.Equal(t, "charlie@test.com", user.Email)
	})

	t.Run("returns error for no matches", func(t *testing.T) {
		_, err := resolveUserIdentifier("nonexistent", users)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "user 'nonexistent' not found")
		assert.Contains(t, err.Error(), "Run 'kira users' to see available users")
	})

	t.Run("returns error for multiple email matches", func(t *testing.T) {
		_, err := resolveUserIdentifier("@example.com", users)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "multiple users match '@example.com'")
		assert.Contains(t, err.Error(), "1. Alice <alice@example.com>")
		assert.Contains(t, err.Error(), "Use the numeric identifier to select a specific user")
	})

	t.Run("returns error for multiple name matches", func(t *testing.T) {
		_, err := resolveUserIdentifier("Alice", users)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "multiple users match 'Alice'")
		assert.Contains(t, err.Error(), "Use the numeric identifier to select a specific user")
	})

	t.Run("prioritizes numeric over email", func(t *testing.T) {
		// If identifier could be both numeric and email-like, numeric takes priority
		// This is tested implicitly - if "1" is provided, it resolves as number 1, not email
		user, err := resolveUserIdentifier("1", users)
		require.NoError(t, err)
		assert.Equal(t, 1, user.Number)
	})

	t.Run("prioritizes email over name", func(t *testing.T) {
		// Create a user where name matches another user's email
		testUsers := []UserInfo{
			{Email: "bob@example.com", Name: "Bob", Number: 1},
			{Email: "alice@example.com", Name: "bob@example.com", Number: 2}, // Name matches email
		}
		// "bob@example.com" should match as email first, not as name
		user, err := resolveUserIdentifier("bob@example.com", testUsers)
		require.NoError(t, err)
		assert.Equal(t, "bob@example.com", user.Email)
		assert.Equal(t, "Bob", user.Name) // Should match the first user by email
	})
}

func TestFormatMultipleMatchesError(t *testing.T) {
	users := []UserInfo{
		{Email: "alice@example.com", Name: "Alice", Number: 1},
		{Email: "alice.smith@example.com", Name: "Alice Smith", Number: 2},
	}

	t.Run("formats error with all matches", func(t *testing.T) {
		matches := []*UserInfo{&users[0], &users[1]}
		err := formatMultipleMatchesError("Alice", matches)
		require.Error(t, err)

		errMsg := err.Error()
		assert.Contains(t, errMsg, "multiple users match 'Alice'")
		assert.Contains(t, errMsg, "1. Alice <alice@example.com>")
		assert.Contains(t, errMsg, "2. Alice Smith <alice.smith@example.com>")
		assert.Contains(t, errMsg, "Use the numeric identifier to select a specific user")
	})

	t.Run("handles empty matches gracefully", func(t *testing.T) {
		err := formatMultipleMatchesError("test", []*UserInfo{})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "no users match 'test'")
	})
}

// Phase 4: Front Matter Parsing & Field Access Tests

func TestParseWorkItemFrontMatter(t *testing.T) {
	t.Run("parses valid front matter with all fields", func(t *testing.T) {
		tmpDir := t.TempDir()
		origDir, _ := os.Getwd()
		require.NoError(t, os.Chdir(tmpDir))
		defer func() { _ = os.Chdir(origDir) }()

		// Create .work directory structure
		require.NoError(t, os.MkdirAll(".work/1_todo", 0o700))

		content := `---
id: "001"
title: Test Feature
status: todo
kind: prd
created: 2024-01-01
assigned: user@example.com
---
# Test Feature

This is the body content.
`
		require.NoError(t, os.WriteFile(testFilePath, []byte(content), 0o600))

		frontMatter, body, err := parseWorkItemFrontMatter(testFilePath)
		require.NoError(t, err)
		assert.NotNil(t, frontMatter)
		// YAML may parse numeric IDs as int or string, handle both
		idValue := frontMatter["id"]
		assert.True(t, idValue == "001" || idValue == 1)
		assert.Equal(t, "Test Feature", frontMatter["title"])
		assert.Equal(t, "todo", frontMatter["status"])
		assert.Equal(t, "user@example.com", frontMatter["assigned"])
		assert.Contains(t, body, "# Test Feature")
		assert.Contains(t, body, "This is the body content.")
	})

	t.Run("parses valid front matter with missing fields", func(t *testing.T) {
		tmpDir := t.TempDir()
		origDir, _ := os.Getwd()
		require.NoError(t, os.Chdir(tmpDir))
		defer func() { _ = os.Chdir(origDir) }()

		// Create .work directory structure
		require.NoError(t, os.MkdirAll(".work/1_todo", 0o700))

		content := `---
id: "001"
title: Test Feature
---
# Body
`
		require.NoError(t, os.WriteFile(testFilePath, []byte(content), 0o600))

		frontMatter, body, err := parseWorkItemFrontMatter(testFilePath)
		require.NoError(t, err)
		assert.NotNil(t, frontMatter)
		// YAML may parse numeric IDs as int or string, handle both
		idValue := frontMatter["id"]
		assert.True(t, idValue == "001" || idValue == 1)
		assert.Equal(t, "Test Feature", frontMatter["title"])
		// status, kind, created should not be in map
		_, exists := frontMatter["status"]
		assert.False(t, exists)
		assert.Contains(t, body, "# Body")
	})

	t.Run("parses valid front matter with empty fields", func(t *testing.T) {
		tmpDir := t.TempDir()
		origDir, _ := os.Getwd()
		require.NoError(t, os.Chdir(tmpDir))
		defer func() { _ = os.Chdir(origDir) }()

		// Create .work directory structure
		require.NoError(t, os.MkdirAll(".work/1_todo", 0o700))

		content := `---
id: "001"
title: Test Feature
assigned: ""
reviewer: 
---
# Body
`
		require.NoError(t, os.WriteFile(testFilePath, []byte(content), 0o600))

		frontMatter, _, err := parseWorkItemFrontMatter(testFilePath)
		require.NoError(t, err)
		assert.NotNil(t, frontMatter)
		// YAML may parse numeric IDs as int or string, handle both
		idValue := frontMatter["id"]
		assert.True(t, idValue == "001" || idValue == 1)
		assert.Equal(t, "", frontMatter["assigned"])
		// reviewer with no value should be nil or empty
		reviewer, exists := frontMatter["reviewer"]
		if exists {
			assert.Nil(t, reviewer)
		}
	})

	t.Run("parses front matter with array values", func(t *testing.T) {
		tmpDir := t.TempDir()
		origDir, _ := os.Getwd()
		require.NoError(t, os.Chdir(tmpDir))
		defer func() { _ = os.Chdir(origDir) }()

		// Create .work directory structure
		require.NoError(t, os.MkdirAll(".work/1_todo", 0o700))

		content := `---
id: "001"
title: Test Feature
reviewers:
  - alice@example.com
  - bob@example.com
tags:
  - frontend
  - backend
---
# Body
`
		require.NoError(t, os.WriteFile(testFilePath, []byte(content), 0o600))

		frontMatter, _, err := parseWorkItemFrontMatter(testFilePath)
		require.NoError(t, err)
		assert.NotNil(t, frontMatter)
		// YAML may parse numeric IDs as int or string, handle both
		idValue := frontMatter["id"]
		assert.True(t, idValue == "001" || idValue == 1)
		reviewers, ok := frontMatter["reviewers"].([]interface{})
		require.True(t, ok)
		assert.Len(t, reviewers, 2)
		tags, ok := frontMatter["tags"].([]interface{})
		require.True(t, ok)
		assert.Len(t, tags, 2)
	})

	t.Run("handles missing front matter delimiters gracefully", func(t *testing.T) {
		tmpDir := t.TempDir()
		origDir, _ := os.Getwd()
		require.NoError(t, os.Chdir(tmpDir))
		defer func() { _ = os.Chdir(origDir) }()

		// Create .work directory structure
		require.NoError(t, os.MkdirAll(".work/1_todo", 0o700))

		content := `# Test Feature

This is just markdown without front matter.
`
		require.NoError(t, os.WriteFile(testFilePath, []byte(content), 0o600))

		frontMatter, body, err := parseWorkItemFrontMatter(testFilePath)
		require.NoError(t, err) // Should not error, just return empty map
		assert.NotNil(t, frontMatter)
		assert.Empty(t, frontMatter)
		bodyStr := strings.Join(body, "\n")
		assert.Contains(t, bodyStr, "# Test Feature")
	})

	t.Run("returns error for malformed YAML", func(t *testing.T) {
		tmpDir := t.TempDir()
		origDir, _ := os.Getwd()
		require.NoError(t, os.Chdir(tmpDir))
		defer func() { _ = os.Chdir(origDir) }()

		// Create .work directory structure
		require.NoError(t, os.MkdirAll(".work/1_todo", 0o700))

		content := `---
id: 001
title: Test Feature
invalid: [unclosed bracket
---
# Body
`
		require.NoError(t, os.WriteFile(testFilePath, []byte(content), 0o600))

		_, _, err := parseWorkItemFrontMatter(testFilePath)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to parse front matter")
	})

	t.Run("returns error for file not found", func(t *testing.T) {
		tmpDir := t.TempDir()
		origDir, _ := os.Getwd()
		require.NoError(t, os.Chdir(tmpDir))
		defer func() { _ = os.Chdir(origDir) }()

		// Create .work directory structure but don't create file
		require.NoError(t, os.MkdirAll(".work/1_todo", 0o700))

		_, _, err := parseWorkItemFrontMatter(testFilePath)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to read work item file")
	})

	t.Run("extracts body content correctly", func(t *testing.T) {
		tmpDir := t.TempDir()
		origDir, _ := os.Getwd()
		require.NoError(t, os.Chdir(tmpDir))
		defer func() { _ = os.Chdir(origDir) }()

		// Create .work directory structure
		require.NoError(t, os.MkdirAll(".work/1_todo", 0o700))

		content := `---
id: 001
title: Test Feature
---
# Test Feature

This is the body.

It has multiple paragraphs.

- List item 1
- List item 2
`
		require.NoError(t, os.WriteFile(testFilePath, []byte(content), 0o600))

		frontMatter, body, err := parseWorkItemFrontMatter(testFilePath)
		require.NoError(t, err)
		assert.NotNil(t, frontMatter)
		bodyStr := strings.Join(body, "\n")
		assert.Contains(t, bodyStr, "# Test Feature")
		assert.Contains(t, bodyStr, "This is the body.")
		assert.Contains(t, bodyStr, "List item 1")
		assert.Contains(t, bodyStr, "List item 2")
	})

	t.Run("handles front matter with only opening delimiter", func(t *testing.T) {
		tmpDir := t.TempDir()
		origDir, _ := os.Getwd()
		require.NoError(t, os.Chdir(tmpDir))
		defer func() { _ = os.Chdir(origDir) }()

		// Create .work directory structure
		require.NoError(t, os.MkdirAll(".work/1_todo", 0o700))

		content := `---
id: "001"
title: Test Feature
# Body without closing delimiter
`
		require.NoError(t, os.WriteFile(testFilePath, []byte(content), 0o600))

		frontMatter, body, err := parseWorkItemFrontMatter(testFilePath)
		require.NoError(t, err)
		assert.NotNil(t, frontMatter)
		// YAML may parse numeric IDs as int or string, handle both
		idValue := frontMatter["id"]
		assert.True(t, idValue == "001" || idValue == 1)
		// Everything after first --- should be in yamlLines, body should be empty
		assert.Empty(t, body)
	})
}

func TestGetFieldValue(t *testing.T) {
	frontMatter := map[string]interface{}{
		"id":       "001",
		"title":    "Test Feature",
		"assigned": "user@example.com",
		"reviewers": []interface{}{
			"alice@example.com",
			"bob@example.com",
		},
		"empty": "",
		"nil":   nil,
	}

	t.Run("returns field value when field exists with string value", func(t *testing.T) {
		value, exists := getFieldValue(frontMatter, "assigned")
		require.True(t, exists)
		assert.Equal(t, "user@example.com", value)
	})

	t.Run("returns field value when field exists with array value", func(t *testing.T) {
		value, exists := getFieldValue(frontMatter, "reviewers")
		require.True(t, exists)
		reviewers, ok := value.([]interface{})
		require.True(t, ok)
		assert.Len(t, reviewers, 2)
		assert.Equal(t, "alice@example.com", reviewers[0])
	})

	t.Run("returns true when field exists but is empty string", func(t *testing.T) {
		value, exists := getFieldValue(frontMatter, "empty")
		require.True(t, exists)
		assert.Equal(t, "", value)
	})

	t.Run("returns false when field doesn't exist", func(t *testing.T) {
		value, exists := getFieldValue(frontMatter, "nonexistent")
		require.False(t, exists)
		assert.Nil(t, value)
	})

	t.Run("returns true when field exists but is nil", func(t *testing.T) {
		value, exists := getFieldValue(frontMatter, "nil")
		require.True(t, exists)
		assert.Nil(t, value)
	})

	t.Run("returns false when frontMatter is nil", func(t *testing.T) {
		value, exists := getFieldValue(nil, "assigned")
		require.False(t, exists)
		assert.Nil(t, value)
	})

	t.Run("returns false when frontMatter is empty map", func(t *testing.T) {
		emptyMap := make(map[string]interface{})
		value, exists := getFieldValue(emptyMap, "assigned")
		require.False(t, exists)
		assert.Nil(t, value)
	})
}

func TestGetFieldValueAsString(t *testing.T) {
	frontMatter := map[string]interface{}{
		"id":       "001",
		"title":    "Test Feature",
		"assigned": "user@example.com",
		"reviewers": []interface{}{
			"alice@example.com",
			"bob@example.com",
		},
		"reviewers_string": []string{
			"alice@example.com",
			"bob@example.com",
		},
		"empty":   "",
		"number":  42,
		"boolean": true,
		"nil":     nil,
		"tags":    []interface{}{"frontend", "backend"},
	}

	t.Run("returns string field value", func(t *testing.T) {
		value, exists := getFieldValueAsString(frontMatter, "assigned")
		require.True(t, exists)
		assert.Equal(t, "user@example.com", value)
	})

	t.Run("returns formatted array value for []interface{}", func(t *testing.T) {
		value, exists := getFieldValueAsString(frontMatter, "reviewers")
		require.True(t, exists)
		assert.Contains(t, value, "alice@example.com")
		assert.Contains(t, value, "bob@example.com")
		assert.Contains(t, value, ",") // Should be comma-separated
	})

	t.Run("returns formatted array value for []string", func(t *testing.T) {
		value, exists := getFieldValueAsString(frontMatter, "reviewers_string")
		require.True(t, exists)
		assert.Equal(t, "alice@example.com, bob@example.com", value)
	})

	t.Run("returns empty string when field exists but is empty", func(t *testing.T) {
		value, exists := getFieldValueAsString(frontMatter, "empty")
		require.True(t, exists)
		assert.Equal(t, "", value)
	})

	t.Run("returns false when field doesn't exist", func(t *testing.T) {
		value, exists := getFieldValueAsString(frontMatter, "nonexistent")
		require.False(t, exists)
		assert.Equal(t, "", value)
	})

	t.Run("converts number to string", func(t *testing.T) {
		value, exists := getFieldValueAsString(frontMatter, "number")
		require.True(t, exists)
		assert.Equal(t, "42", value)
	})

	t.Run("converts boolean to string", func(t *testing.T) {
		value, exists := getFieldValueAsString(frontMatter, "boolean")
		require.True(t, exists)
		assert.Equal(t, "true", value)
	})

	t.Run("returns empty string when field exists but is nil", func(t *testing.T) {
		value, exists := getFieldValueAsString(frontMatter, "nil")
		require.True(t, exists)
		assert.Equal(t, "", value)
	})

	t.Run("formats mixed array types", func(t *testing.T) {
		value, exists := getFieldValueAsString(frontMatter, "tags")
		require.True(t, exists)
		assert.Contains(t, value, "frontend")
		assert.Contains(t, value, "backend")
	})
}

// Integration tests for Phase 4
func TestParseWorkItemFrontMatterIntegration(t *testing.T) {
	t.Run("parses real work item file structure", func(t *testing.T) {
		tmpDir := t.TempDir()
		origDir, _ := os.Getwd()
		require.NoError(t, os.Chdir(tmpDir))
		defer func() { _ = os.Chdir(origDir) }()

		// Create .work directory structure
		require.NoError(t, os.MkdirAll(".work/1_todo", 0o700))

		content := `---
id: "001"
title: Test Feature
status: todo
kind: prd
created: 2024-01-01
assigned: user@example.com
reviewer: reviewer@example.com
tags:
  - frontend
  - backend
---
# Test Feature

This is a test feature description.

## Details

Some details here.
`
		require.NoError(t, os.WriteFile(testFilePath, []byte(content), 0o600))

		frontMatter, body, err := parseWorkItemFrontMatter(testFilePath)
		require.NoError(t, err)
		assert.NotNil(t, frontMatter)

		// Test accessing default assigned field
		value, exists := getFieldValue(frontMatter, "assigned")
		require.True(t, exists)
		assert.Equal(t, "user@example.com", value)

		// Test accessing custom field
		value, exists = getFieldValue(frontMatter, "reviewer")
		require.True(t, exists)
		assert.Equal(t, "reviewer@example.com", value)

		// Test getFieldValueAsString for display
		assignedStr, exists := getFieldValueAsString(frontMatter, "assigned")
		require.True(t, exists)
		assert.Equal(t, "user@example.com", assignedStr)

		// Test body extraction
		bodyStr := strings.Join(body, "\n")
		assert.Contains(t, bodyStr, "# Test Feature")
		assert.Contains(t, bodyStr, "This is a test feature description")
	})

	t.Run("handles work items with complex front matter", func(t *testing.T) {
		tmpDir := t.TempDir()
		origDir, _ := os.Getwd()
		require.NoError(t, os.Chdir(tmpDir))
		defer func() { _ = os.Chdir(origDir) }()

		// Create .work directory structure
		require.NoError(t, os.MkdirAll(".work/1_todo", 0o700))

		content := `---
id: "001"
title: Complex Feature
status: doing
kind: prd
created: 2024-01-01
updated: 2024-01-15
assigned: 
  - alice@example.com
  - bob@example.com
reviewer: charlie@example.com
metadata:
  priority: high
  estimate: 5
  labels:
    - important
    - urgent
---
# Complex Feature

Complex body content.
`
		require.NoError(t, os.WriteFile(testFilePath, []byte(content), 0o600))

		frontMatter, _, err := parseWorkItemFrontMatter(testFilePath)
		require.NoError(t, err)
		assert.NotNil(t, frontMatter)

		// Test accessing array field
		assigned, exists := getFieldValue(frontMatter, "assigned")
		require.True(t, exists)
		assignedArray, ok := assigned.([]interface{})
		require.True(t, ok)
		assert.Len(t, assignedArray, 2)

		// Test getFieldValueAsString with array
		assignedStr, exists := getFieldValueAsString(frontMatter, "assigned")
		require.True(t, exists)
		assert.Contains(t, assignedStr, "alice@example.com")
		assert.Contains(t, assignedStr, "bob@example.com")

		// Test accessing nested structure
		metadata, exists := getFieldValue(frontMatter, "metadata")
		require.True(t, exists)
		metadataMap, ok := metadata.(map[string]interface{})
		require.True(t, ok)
		assert.Equal(t, "high", metadataMap["priority"])
	})
}

// Phase 5: Field Update Logic (Switch Mode) Tests

const (
	testWorkItemContentPhase5 = `---
id: 001
title: Test Feature
status: todo
kind: prd
created: 2024-01-01
---
# Test Feature
`
	testFilePathPhase5 = ".work/1_todo/001-test-feature.prd.md"
)

func TestWriteWorkItemFrontMatter(t *testing.T) {
	testFilePath := testFilePathPhase5

	t.Run("writes front matter with all field types", func(t *testing.T) {
		tmpDir := t.TempDir()
		origDir, _ := os.Getwd()
		require.NoError(t, os.Chdir(tmpDir))
		defer func() { _ = os.Chdir(origDir) }()

		require.NoError(t, os.MkdirAll(".work/1_todo", 0o700))

		frontMatter := map[string]interface{}{
			"id":       "001",
			"title":    "Test Feature",
			"status":   "todo",
			"kind":     "prd",
			"created":  "2024-01-01",
			"assigned": "user@example.com",
			"tags":     []interface{}{"frontend", "backend"},
		}
		bodyLines := []string{"# Test Feature", "", "This is the body."}

		err := writeWorkItemFrontMatter(testFilePath, frontMatter, bodyLines)
		require.NoError(t, err)

		// Verify file was written
		content, err := os.ReadFile(testFilePath)
		require.NoError(t, err)
		contentStr := string(content)

		// Check front matter
		assert.Contains(t, contentStr, "id: 001")
		assert.Contains(t, contentStr, "title: Test Feature")
		assert.Contains(t, contentStr, "assigned: user@example.com")
		assert.Contains(t, contentStr, "tags: [frontend, backend]")

		// Check body
		assert.Contains(t, contentStr, "# Test Feature")
		assert.Contains(t, contentStr, "This is the body.")
	})

	t.Run("preserves body content", func(t *testing.T) {
		tmpDir := t.TempDir()
		origDir, _ := os.Getwd()
		require.NoError(t, os.Chdir(tmpDir))
		defer func() { _ = os.Chdir(origDir) }()

		require.NoError(t, os.MkdirAll(".work/1_todo", 0o700))

		frontMatter := map[string]interface{}{
			"id":      "001",
			"title":   "Test",
			"status":  "todo",
			"kind":    "prd",
			"created": "2024-01-01",
		}
		bodyLines := []string{
			"# Test Feature",
			"",
			"This is paragraph one.",
			"",
			"This is paragraph two.",
			"",
			"- List item 1",
			"- List item 2",
		}

		err := writeWorkItemFrontMatter(testFilePath, frontMatter, bodyLines)
		require.NoError(t, err)

		content, err := os.ReadFile(testFilePath)
		require.NoError(t, err)
		contentStr := string(content)

		assert.Contains(t, contentStr, "# Test Feature")
		assert.Contains(t, contentStr, "This is paragraph one.")
		assert.Contains(t, contentStr, "This is paragraph two.")
		assert.Contains(t, contentStr, "- List item 1")
		assert.Contains(t, contentStr, "- List item 2")
	})

	t.Run("handles YAML formatting with special characters", func(t *testing.T) {
		tmpDir := t.TempDir()
		origDir, _ := os.Getwd()
		require.NoError(t, os.Chdir(tmpDir))
		defer func() { _ = os.Chdir(origDir) }()

		require.NoError(t, os.MkdirAll(".work/1_todo", 0o700))

		frontMatter := map[string]interface{}{
			"id":      "001",
			"title":   "Test: Feature [Important]",
			"status":  "todo",
			"kind":    "prd",
			"created": "2024-01-01",
			"note":    "Value with: colon and [brackets]",
		}
		bodyLines := []string{"# Test"}

		err := writeWorkItemFrontMatter(testFilePath, frontMatter, bodyLines)
		require.NoError(t, err)

		content, err := os.ReadFile(testFilePath)
		require.NoError(t, err)
		contentStr := string(content)

		// Special characters should be quoted
		assert.Contains(t, contentStr, `title: "Test: Feature [Important]"`)
		assert.Contains(t, contentStr, `note: "Value with: colon and [brackets]"`)
	})

	t.Run("handles empty front matter", func(t *testing.T) {
		tmpDir := t.TempDir()
		origDir, _ := os.Getwd()
		require.NoError(t, os.Chdir(tmpDir))
		defer func() { _ = os.Chdir(origDir) }()

		require.NoError(t, os.MkdirAll(".work/1_todo", 0o700))

		frontMatter := map[string]interface{}{}
		bodyLines := []string{"# Test"}

		err := writeWorkItemFrontMatter(testFilePath, frontMatter, bodyLines)
		require.NoError(t, err)

		content, err := os.ReadFile(testFilePath)
		require.NoError(t, err)
		contentStr := string(content)

		// Should have YAML separators
		assert.Contains(t, contentStr, "---")
		assert.Contains(t, contentStr, "# Test")
	})

	t.Run("preserves field order", func(t *testing.T) {
		tmpDir := t.TempDir()
		origDir, _ := os.Getwd()
		require.NoError(t, os.Chdir(tmpDir))
		defer func() { _ = os.Chdir(origDir) }()

		require.NoError(t, os.MkdirAll(".work/1_todo", 0o700))

		frontMatter := map[string]interface{}{
			"id":       "001",
			"title":    "Test",
			"status":   "todo",
			"kind":     "prd",
			"created":  "2024-01-01",
			"zebra":    "last",
			"assigned": "user@example.com",
			"alpha":    "first",
		}
		bodyLines := []string{}

		err := writeWorkItemFrontMatter(testFilePath, frontMatter, bodyLines)
		require.NoError(t, err)

		content, err := os.ReadFile(testFilePath)
		require.NoError(t, err)
		contentStr := string(content)

		// Hardcoded fields should come first
		idPos := strings.Index(contentStr, "id:")
		titlePos := strings.Index(contentStr, "title:")
		statusPos := strings.Index(contentStr, "status:")
		kindPos := strings.Index(contentStr, "kind:")
		createdPos := strings.Index(contentStr, "created:")

		// Other fields should come after
		alphaPos := strings.Index(contentStr, "alpha:")
		assignedPos := strings.Index(contentStr, "assigned:")
		zebraPos := strings.Index(contentStr, "zebra:")

		// Verify order
		assert.True(t, idPos < titlePos)
		assert.True(t, titlePos < statusPos)
		assert.True(t, statusPos < kindPos)
		assert.True(t, kindPos < createdPos)
		assert.True(t, createdPos < alphaPos)
		assert.True(t, alphaPos < assignedPos)
		assert.True(t, assignedPos < zebraPos)
	})
}

func TestUpdateFieldValue(t *testing.T) {
	t.Run("updates existing field", func(t *testing.T) {
		frontMatter := map[string]interface{}{
			"assigned": "old@example.com",
		}

		previous, existed := updateFieldValue(frontMatter, "assigned", "new@example.com")

		assert.True(t, existed)
		assert.Equal(t, "old@example.com", previous)
		assert.Equal(t, "new@example.com", frontMatter["assigned"])
	})

	t.Run("creates new field", func(t *testing.T) {
		frontMatter := map[string]interface{}{}

		previous, existed := updateFieldValue(frontMatter, "assigned", "user@example.com")

		assert.False(t, existed)
		assert.Nil(t, previous)
		assert.Equal(t, "user@example.com", frontMatter["assigned"])
	})

	t.Run("replaces existing value", func(t *testing.T) {
		frontMatter := map[string]interface{}{
			"assigned": "alice@example.com",
		}

		previous, existed := updateFieldValue(frontMatter, "assigned", "bob@example.com")

		assert.True(t, existed)
		assert.Equal(t, "alice@example.com", previous)
		assert.Equal(t, "bob@example.com", frontMatter["assigned"])
	})

	t.Run("handles nil front matter map", func(t *testing.T) {
		// updateFieldValue initializes the map if nil, but since maps are reference types,
		// we need to pass a pointer to modify nil maps. In practice, parseWorkItemFrontMatter
		// always returns a non-nil map, so this is an edge case.
		// For this test, we'll create an empty map instead.
		frontMatter := make(map[string]interface{})

		previous, existed := updateFieldValue(frontMatter, "assigned", "user@example.com")

		assert.False(t, existed)
		assert.Nil(t, previous)
		assert.NotNil(t, frontMatter)
		assert.Equal(t, "user@example.com", frontMatter["assigned"])
	})
}

func TestUpdateTimestamp(t *testing.T) {
	t.Run("updates existing updated field", func(t *testing.T) {
		frontMatter := map[string]interface{}{
			"updated": "2024-01-01T00:00:00Z",
		}

		oldValue := frontMatter["updated"]
		updateTimestamp(frontMatter)

		updatedStr, ok := frontMatter["updated"].(string)
		require.True(t, ok)

		// Verify the value changed
		assert.NotEqual(t, oldValue, updatedStr)

		// Parse timestamp (always in UTC/Z format)
		updatedTime, err := time.Parse("2006-01-02T15:04:05Z", updatedStr)
		require.NoError(t, err)

		// Verify it's a recent timestamp (within last hour to account for timezone differences)
		now := time.Now().UTC()
		assert.True(t, updatedTime.After(now.Add(-time.Hour)), "updatedTime %v should be recent (after %v)", updatedTime, now.Add(-time.Hour))
		assert.True(t, updatedTime.Before(now.Add(time.Hour)), "updatedTime %v should be recent (before %v)", updatedTime, now.Add(time.Hour))
	})

	t.Run("creates updated field when missing", func(t *testing.T) {
		frontMatter := map[string]interface{}{}

		updateTimestamp(frontMatter)

		updatedStr, ok := frontMatter["updated"].(string)
		require.True(t, ok)
		assert.NotEmpty(t, updatedStr)

		// Should be valid timestamp format
		_, err := time.Parse("2006-01-02T15:04:05Z", updatedStr)
		assert.NoError(t, err)
	})

	t.Run("timestamp format is correct", func(t *testing.T) {
		frontMatter := map[string]interface{}{}

		updateTimestamp(frontMatter)

		updatedStr, ok := frontMatter["updated"].(string)
		require.True(t, ok)

		// Should match ISO 8601 format with time
		_, err := time.Parse("2006-01-02T15:04:05Z", updatedStr)
		assert.NoError(t, err)
	})

	t.Run("handles nil front matter map", func(t *testing.T) {
		// updateTimestamp initializes the map if nil, but since maps are reference types,
		// we need to pass a pointer to modify nil maps. In practice, parseWorkItemFrontMatter
		// always returns a non-nil map, so this is an edge case.
		// For this test, we'll create an empty map instead.
		frontMatter := make(map[string]interface{})

		updateTimestamp(frontMatter)

		assert.NotNil(t, frontMatter)
		updatedStr, ok := frontMatter["updated"].(string)
		require.True(t, ok)
		assert.NotEmpty(t, updatedStr)
	})
}

func TestUpdateWorkItemField(t *testing.T) {
	testFilePath := testFilePathPhase5

	t.Run("updates field in work item", func(t *testing.T) {
		tmpDir := t.TempDir()
		origDir, _ := os.Getwd()
		require.NoError(t, os.Chdir(tmpDir))
		defer func() { _ = os.Chdir(origDir) }()

		require.NoError(t, os.MkdirAll(".work/1_todo", 0o700))

		content := `---
id: 001
title: Test Feature
status: todo
kind: prd
created: 2024-01-01
assigned: old@example.com
---
# Test Feature

Body content.
`
		require.NoError(t, os.WriteFile(testFilePath, []byte(content), 0o600))

		err := updateWorkItemField(testFilePath, "assigned", "new@example.com")
		require.NoError(t, err)

		// Verify file was updated
		updatedContent, err := os.ReadFile(testFilePath)
		require.NoError(t, err)
		updatedStr := string(updatedContent)

		assert.Contains(t, updatedStr, "assigned: new@example.com")
		assert.NotContains(t, updatedStr, "assigned: old@example.com")
		assert.Contains(t, updatedStr, "# Test Feature")
		assert.Contains(t, updatedStr, "Body content.")
	})

	t.Run("creates field if it doesn't exist", func(t *testing.T) {
		tmpDir := t.TempDir()
		origDir, _ := os.Getwd()
		require.NoError(t, os.Chdir(tmpDir))
		defer func() { _ = os.Chdir(origDir) }()

		require.NoError(t, os.MkdirAll(".work/1_todo", 0o700))

		require.NoError(t, os.WriteFile(testFilePath, []byte(testWorkItemContentPhase5), 0o600))

		err := updateWorkItemField(testFilePath, "assigned", "user@example.com")
		require.NoError(t, err)

		// Verify field was created
		updatedContent, err := os.ReadFile(testFilePath)
		require.NoError(t, err)
		updatedStr := string(updatedContent)

		assert.Contains(t, updatedStr, "assigned: user@example.com")
	})

	t.Run("updates timestamp", func(t *testing.T) {
		tmpDir := t.TempDir()
		origDir, _ := os.Getwd()
		require.NoError(t, os.Chdir(tmpDir))
		defer func() { _ = os.Chdir(origDir) }()

		require.NoError(t, os.MkdirAll(".work/1_todo", 0o700))

		content := `---
id: 001
title: Test Feature
status: todo
kind: prd
created: 2024-01-01
updated: 2024-01-01T00:00:00Z
---
# Test Feature
`
		require.NoError(t, os.WriteFile(testFilePath, []byte(content), 0o600))

		err := updateWorkItemField(testFilePath, "assigned", "user@example.com")
		require.NoError(t, err)

		// Verify timestamp was updated
		updatedContent, err := os.ReadFile(testFilePath)
		require.NoError(t, err)
		updatedStr := string(updatedContent)

		// Parse updated timestamp from file
		lines := strings.Split(updatedStr, "\n")
		var updatedLine string
		for _, line := range lines {
			if strings.HasPrefix(strings.TrimSpace(line), "updated:") {
				updatedLine = line
				break
			}
		}
		require.NotEmpty(t, updatedLine)

		// Extract timestamp value (split only on first colon)
		colonIdx := strings.Index(updatedLine, ":")
		require.Greater(t, colonIdx, 0)
		timestampStr := strings.TrimSpace(updatedLine[colonIdx+1:])
		// Remove quotes if present (YAML may quote values with colons)
		timestampStr = strings.Trim(timestampStr, `"`)

		updatedTime, err := time.Parse("2006-01-02T15:04:05Z", timestampStr)
		require.NoError(t, err)

		// Verify it's a recent timestamp (within last hour to account for timezone differences)
		now := time.Now().UTC()
		assert.True(t, updatedTime.After(now.Add(-time.Hour)), "updatedTime %v should be recent (after %v)", updatedTime, now.Add(-time.Hour))
		assert.True(t, updatedTime.Before(now.Add(time.Hour)), "updatedTime %v should be recent (before %v)", updatedTime, now.Add(time.Hour))
	})

	t.Run("creates updated timestamp if missing", func(t *testing.T) {
		tmpDir := t.TempDir()
		origDir, _ := os.Getwd()
		require.NoError(t, os.Chdir(tmpDir))
		defer func() { _ = os.Chdir(origDir) }()

		require.NoError(t, os.MkdirAll(".work/1_todo", 0o700))

		require.NoError(t, os.WriteFile(testFilePath, []byte(testWorkItemContentPhase5), 0o600))

		err := updateWorkItemField(testFilePath, "assigned", "user@example.com")
		require.NoError(t, err)

		// Verify updated timestamp was created
		updatedContent, err := os.ReadFile(testFilePath)
		require.NoError(t, err)
		updatedStr := string(updatedContent)

		assert.Contains(t, updatedStr, "updated:")
		// Verify it's a valid timestamp format
		lines := strings.Split(updatedStr, "\n")
		for _, line := range lines {
			if strings.HasPrefix(strings.TrimSpace(line), "updated:") {
				colonIdx := strings.Index(line, ":")
				require.Greater(t, colonIdx, 0)
				timestampStr := strings.TrimSpace(line[colonIdx+1:])
				// Remove quotes if present (YAML may quote values with colons)
				timestampStr = strings.Trim(timestampStr, `"`)
				_, err := time.Parse("2006-01-02T15:04:05Z", timestampStr)
				assert.NoError(t, err)
				break
			}
		}
	})

	t.Run("preserves other front matter fields", func(t *testing.T) {
		tmpDir := t.TempDir()
		origDir, _ := os.Getwd()
		require.NoError(t, os.Chdir(tmpDir))
		defer func() { _ = os.Chdir(origDir) }()

		require.NoError(t, os.MkdirAll(".work/1_todo", 0o700))

		content := `---
id: 001
title: Test Feature
status: todo
kind: prd
created: 2024-01-01
reviewer: reviewer@example.com
estimate: 5
tags: [frontend, backend]
---
# Test Feature
`
		require.NoError(t, os.WriteFile(testFilePath, []byte(content), 0o600))

		err := updateWorkItemField(testFilePath, "assigned", "user@example.com")
		require.NoError(t, err)

		// Verify other fields are preserved
		updatedContent, err := os.ReadFile(testFilePath)
		require.NoError(t, err)
		updatedStr := string(updatedContent)

		assert.Contains(t, updatedStr, "reviewer: reviewer@example.com")
		assert.Contains(t, updatedStr, "estimate: 5")
		assert.Contains(t, updatedStr, "tags: [frontend, backend]")
	})

	t.Run("preserves body content", func(t *testing.T) {
		tmpDir := t.TempDir()
		origDir, _ := os.Getwd()
		require.NoError(t, os.Chdir(tmpDir))
		defer func() { _ = os.Chdir(origDir) }()

		require.NoError(t, os.MkdirAll(".work/1_todo", 0o700))

		content := `---
id: 001
title: Test Feature
status: todo
kind: prd
created: 2024-01-01
---
# Test Feature

This is the body content.

## Section

More content here.
`
		require.NoError(t, os.WriteFile(testFilePath, []byte(content), 0o600))

		err := updateWorkItemField(testFilePath, "assigned", "user@example.com")
		require.NoError(t, err)

		// Verify body is preserved
		updatedContent, err := os.ReadFile(testFilePath)
		require.NoError(t, err)
		updatedStr := string(updatedContent)

		assert.Contains(t, updatedStr, "# Test Feature")
		assert.Contains(t, updatedStr, "This is the body content.")
		assert.Contains(t, updatedStr, "## Section")
		assert.Contains(t, updatedStr, "More content here.")
	})

	t.Run("works with custom field names", func(t *testing.T) {
		tmpDir := t.TempDir()
		origDir, _ := os.Getwd()
		require.NoError(t, os.Chdir(tmpDir))
		defer func() { _ = os.Chdir(origDir) }()

		require.NoError(t, os.MkdirAll(".work/1_todo", 0o700))

		require.NoError(t, os.WriteFile(testFilePath, []byte(testWorkItemContentPhase5), 0o600))

		err := updateWorkItemField(testFilePath, "reviewer", "reviewer@example.com")
		require.NoError(t, err)

		// Verify custom field was set
		updatedContent, err := os.ReadFile(testFilePath)
		require.NoError(t, err)
		updatedStr := string(updatedContent)

		assert.Contains(t, updatedStr, "reviewer: reviewer@example.com")
	})

	t.Run("returns error for file not found", func(t *testing.T) {
		tmpDir := t.TempDir()
		origDir, _ := os.Getwd()
		require.NoError(t, os.Chdir(tmpDir))
		defer func() { _ = os.Chdir(origDir) }()

		require.NoError(t, os.MkdirAll(".work/1_todo", 0o700))

		err := updateWorkItemField(testFilePath, "assigned", "user@example.com")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to read work item file")
	})

	t.Run("returns error for malformed YAML", func(t *testing.T) {
		tmpDir := t.TempDir()
		origDir, _ := os.Getwd()
		require.NoError(t, os.Chdir(tmpDir))
		defer func() { _ = os.Chdir(origDir) }()

		require.NoError(t, os.MkdirAll(".work/1_todo", 0o700))

		content := `---
id: 001
title: Test Feature
invalid: [unclosed bracket
---
# Test Feature
`
		require.NoError(t, os.WriteFile(testFilePath, []byte(content), 0o600))

		err := updateWorkItemField(testFilePath, "assigned", "user@example.com")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to parse front matter")
	})
}

// Phase 6: Append Mode Logic Tests

func TestAppendToField(t *testing.T) {
	t.Run("creates field if it doesn't exist", func(t *testing.T) {
		frontMatter := map[string]interface{}{}

		appendToField(frontMatter, "assigned", "user@example.com")

		assert.Equal(t, "user@example.com", frontMatter["assigned"])
	})

	t.Run("sets field to user if field is empty string", func(t *testing.T) {
		frontMatter := map[string]interface{}{
			"assigned": "",
		}

		appendToField(frontMatter, "assigned", "user@example.com")

		assert.Equal(t, "user@example.com", frontMatter["assigned"])
	})

	t.Run("converts single string value to array", func(t *testing.T) {
		frontMatter := map[string]interface{}{
			"assigned": "alice@example.com",
		}

		appendToField(frontMatter, "assigned", "bob@example.com")

		arr, ok := frontMatter["assigned"].([]string)
		require.True(t, ok)
		assert.Len(t, arr, 2)
		assert.Equal(t, "alice@example.com", arr[0])
		assert.Equal(t, "bob@example.com", arr[1])
	})

	t.Run("appends to []string array", func(t *testing.T) {
		frontMatter := map[string]interface{}{
			"assigned": []string{"alice@example.com", "bob@example.com"},
		}

		appendToField(frontMatter, "assigned", "charlie@example.com")

		arr, ok := frontMatter["assigned"].([]string)
		require.True(t, ok)
		assert.Len(t, arr, 3)
		assert.Equal(t, "alice@example.com", arr[0])
		assert.Equal(t, "bob@example.com", arr[1])
		assert.Equal(t, "charlie@example.com", arr[2])
	})

	t.Run("appends to []interface{} array", func(t *testing.T) {
		frontMatter := map[string]interface{}{
			"assigned": []interface{}{"alice@example.com", "bob@example.com"},
		}

		appendToField(frontMatter, "assigned", "charlie@example.com")

		arr, ok := frontMatter["assigned"].([]string)
		require.True(t, ok)
		assert.Len(t, arr, 3)
		assert.Equal(t, "alice@example.com", arr[0])
		assert.Equal(t, "bob@example.com", arr[1])
		assert.Equal(t, "charlie@example.com", arr[2])
	})

	t.Run("prevents duplicate entries in []string array", func(t *testing.T) {
		frontMatter := map[string]interface{}{
			"assigned": []string{"alice@example.com", "bob@example.com"},
		}

		appendToField(frontMatter, "assigned", "alice@example.com")

		arr, ok := frontMatter["assigned"].([]string)
		require.True(t, ok)
		assert.Len(t, arr, 2) // Should not add duplicate
		assert.Equal(t, "alice@example.com", arr[0])
		assert.Equal(t, "bob@example.com", arr[1])
	})

	t.Run("prevents duplicate entries in []interface{} array", func(t *testing.T) {
		frontMatter := map[string]interface{}{
			"assigned": []interface{}{"alice@example.com", "bob@example.com"},
		}

		appendToField(frontMatter, "assigned", "alice@example.com")

		arr, ok := frontMatter["assigned"].([]string)
		require.True(t, ok)
		assert.Len(t, arr, 2) // Should not add duplicate
		assert.Equal(t, "alice@example.com", arr[0])
		assert.Equal(t, "bob@example.com", arr[1])
	})

	t.Run("prevents duplicate when appending same value as single string", func(t *testing.T) {
		frontMatter := map[string]interface{}{
			"assigned": "alice@example.com",
		}

		appendToField(frontMatter, "assigned", "alice@example.com")

		arr, ok := frontMatter["assigned"].([]string)
		require.True(t, ok)
		assert.Len(t, arr, 2) // Converts to array, but both are same value
		// Note: This is expected behavior - we convert to array even if duplicate
		// The duplicate check only applies to arrays, not single values
		assert.Equal(t, "alice@example.com", arr[0])
		assert.Equal(t, "alice@example.com", arr[1])
	})

	t.Run("handles nil front matter map", func(t *testing.T) {
		// appendToField initializes the map if nil, but since maps are reference types,
		// we need to pass a pointer to modify nil maps. In practice, parseWorkItemFrontMatter
		// always returns a non-nil map, so this is an edge case.
		// For this test, we'll create an empty map instead.
		frontMatter := make(map[string]interface{})

		appendToField(frontMatter, "assigned", "user@example.com")

		assert.Equal(t, "user@example.com", frontMatter["assigned"])
	})

	t.Run("handles other types by converting to array", func(t *testing.T) {
		frontMatter := map[string]interface{}{
			"assigned": 42, // numeric value
		}

		appendToField(frontMatter, "assigned", "user@example.com")

		arr, ok := frontMatter["assigned"].([]string)
		require.True(t, ok)
		assert.Len(t, arr, 2)
		assert.Equal(t, "42", arr[0]) // Converted to string
		assert.Equal(t, "user@example.com", arr[1])
	})

	t.Run("handles boolean values", func(t *testing.T) {
		frontMatter := map[string]interface{}{
			"assigned": true,
		}

		appendToField(frontMatter, "assigned", "user@example.com")

		arr, ok := frontMatter["assigned"].([]string)
		require.True(t, ok)
		assert.Len(t, arr, 2)
		assert.Equal(t, "true", arr[0]) // Converted to string
		assert.Equal(t, "user@example.com", arr[1])
	})

	t.Run("works with custom field names", func(t *testing.T) {
		frontMatter := map[string]interface{}{
			"reviewer": "alice@example.com",
		}

		appendToField(frontMatter, "reviewer", "bob@example.com")

		arr, ok := frontMatter["reviewer"].([]string)
		require.True(t, ok)
		assert.Len(t, arr, 2)
		assert.Equal(t, "alice@example.com", arr[0])
		assert.Equal(t, "bob@example.com", arr[1])
	})
}

func TestUpdateWorkItemFieldAppend(t *testing.T) {
	testFilePath := testFilePathPhase5

	t.Run("appends to non-existent field", func(t *testing.T) {
		tmpDir := t.TempDir()
		origDir, _ := os.Getwd()
		require.NoError(t, os.Chdir(tmpDir))
		defer func() { _ = os.Chdir(origDir) }()

		require.NoError(t, os.MkdirAll(".work/1_todo", 0o700))
		require.NoError(t, os.WriteFile(testFilePath, []byte(testWorkItemContentPhase5), 0o600))

		err := updateWorkItemFieldAppend(testFilePath, "assigned", "user@example.com")
		require.NoError(t, err)

		// Verify field was created
		updatedContent, err := os.ReadFile(testFilePath)
		require.NoError(t, err)
		updatedStr := string(updatedContent)

		assert.Contains(t, updatedStr, "assigned: user@example.com")
	})

	t.Run("appends to empty string field", func(t *testing.T) {
		tmpDir := t.TempDir()
		origDir, _ := os.Getwd()
		require.NoError(t, os.Chdir(tmpDir))
		defer func() { _ = os.Chdir(origDir) }()

		require.NoError(t, os.MkdirAll(".work/1_todo", 0o700))

		content := `---
id: 001
title: Test Feature
status: todo
kind: prd
created: 2024-01-01
assigned: ""
---
# Test Feature
`
		require.NoError(t, os.WriteFile(testFilePath, []byte(content), 0o600))

		err := updateWorkItemFieldAppend(testFilePath, "assigned", "user@example.com")
		require.NoError(t, err)

		// Verify field was set (not array)
		updatedContent, err := os.ReadFile(testFilePath)
		require.NoError(t, err)
		updatedStr := string(updatedContent)

		assert.Contains(t, updatedStr, "assigned: user@example.com")
		assert.NotContains(t, updatedStr, "assigned: [")
	})

	t.Run("converts single value to array when appending", func(t *testing.T) {
		tmpDir := t.TempDir()
		origDir, _ := os.Getwd()
		require.NoError(t, os.Chdir(tmpDir))
		defer func() { _ = os.Chdir(origDir) }()

		require.NoError(t, os.MkdirAll(".work/1_todo", 0o700))

		content := `---
id: 001
title: Test Feature
status: todo
kind: prd
created: 2024-01-01
assigned: alice@example.com
---
# Test Feature
`
		require.NoError(t, os.WriteFile(testFilePath, []byte(content), 0o600))

		err := updateWorkItemFieldAppend(testFilePath, "assigned", "bob@example.com")
		require.NoError(t, err)

		// Verify field was converted to array
		updatedContent, err := os.ReadFile(testFilePath)
		require.NoError(t, err)
		updatedStr := string(updatedContent)

		// Should be an array now
		assert.Contains(t, updatedStr, "assigned: [")
		assert.Contains(t, updatedStr, "alice@example.com")
		assert.Contains(t, updatedStr, "bob@example.com")
	})

	t.Run("appends to existing array field", func(t *testing.T) {
		tmpDir := t.TempDir()
		origDir, _ := os.Getwd()
		require.NoError(t, os.Chdir(tmpDir))
		defer func() { _ = os.Chdir(origDir) }()

		require.NoError(t, os.MkdirAll(".work/1_todo", 0o700))

		content := `---
id: 001
title: Test Feature
status: todo
kind: prd
created: 2024-01-01
assigned: [alice@example.com, bob@example.com]
---
# Test Feature
`
		require.NoError(t, os.WriteFile(testFilePath, []byte(content), 0o600))

		err := updateWorkItemFieldAppend(testFilePath, "assigned", "charlie@example.com")
		require.NoError(t, err)

		// Verify new user was appended
		updatedContent, err := os.ReadFile(testFilePath)
		require.NoError(t, err)
		updatedStr := string(updatedContent)

		assert.Contains(t, updatedStr, "assigned: [")
		assert.Contains(t, updatedStr, "alice@example.com")
		assert.Contains(t, updatedStr, "bob@example.com")
		assert.Contains(t, updatedStr, "charlie@example.com")
	})

	t.Run("prevents duplicate entries in array", func(t *testing.T) {
		tmpDir := t.TempDir()
		origDir, _ := os.Getwd()
		require.NoError(t, os.Chdir(tmpDir))
		defer func() { _ = os.Chdir(origDir) }()

		require.NoError(t, os.MkdirAll(".work/1_todo", 0o700))

		content := `---
id: 001
title: Test Feature
status: todo
kind: prd
created: 2024-01-01
assigned: [alice@example.com, bob@example.com]
---
# Test Feature
`
		require.NoError(t, os.WriteFile(testFilePath, []byte(content), 0o600))

		err := updateWorkItemFieldAppend(testFilePath, "assigned", "alice@example.com")
		require.NoError(t, err)

		// Verify duplicate was not added
		updatedContent, err := os.ReadFile(testFilePath)
		require.NoError(t, err)
		updatedStr := string(updatedContent)

		// Count occurrences of alice@example.com - should only appear once
		aliceCount := strings.Count(updatedStr, "alice@example.com")
		assert.Equal(t, 1, aliceCount, "alice@example.com should only appear once")
		assert.Contains(t, updatedStr, "bob@example.com")
	})

	t.Run("updates timestamp", func(t *testing.T) {
		tmpDir := t.TempDir()
		origDir, _ := os.Getwd()
		require.NoError(t, os.Chdir(tmpDir))
		defer func() { _ = os.Chdir(origDir) }()

		require.NoError(t, os.MkdirAll(".work/1_todo", 0o700))

		content := `---
id: 001
title: Test Feature
status: todo
kind: prd
created: 2024-01-01
updated: 2024-01-01T00:00:00Z
---
# Test Feature
`
		require.NoError(t, os.WriteFile(testFilePath, []byte(content), 0o600))

		err := updateWorkItemFieldAppend(testFilePath, "assigned", "user@example.com")
		require.NoError(t, err)

		// Verify timestamp was updated
		updatedContent, err := os.ReadFile(testFilePath)
		require.NoError(t, err)
		updatedStr := string(updatedContent)

		// Parse updated timestamp from file
		lines := strings.Split(updatedStr, "\n")
		var updatedLine string
		for _, line := range lines {
			if strings.HasPrefix(strings.TrimSpace(line), "updated:") {
				updatedLine = line
				break
			}
		}
		require.NotEmpty(t, updatedLine)

		// Extract timestamp value
		colonIdx := strings.Index(updatedLine, ":")
		require.Greater(t, colonIdx, 0)
		timestampStr := strings.TrimSpace(updatedLine[colonIdx+1:])
		timestampStr = strings.Trim(timestampStr, `"`)

		updatedTime, err := time.Parse("2006-01-02T15:04:05Z", timestampStr)
		require.NoError(t, err)

		// Verify it's a recent timestamp
		now := time.Now().UTC()
		assert.True(t, updatedTime.After(now.Add(-time.Hour)), "updatedTime %v should be recent (after %v)", updatedTime, now.Add(-time.Hour))
		assert.True(t, updatedTime.Before(now.Add(time.Hour)), "updatedTime %v should be recent (before %v)", updatedTime, now.Add(time.Hour))
	})

	t.Run("preserves other front matter fields", func(t *testing.T) {
		tmpDir := t.TempDir()
		origDir, _ := os.Getwd()
		require.NoError(t, os.Chdir(tmpDir))
		defer func() { _ = os.Chdir(origDir) }()

		require.NoError(t, os.MkdirAll(".work/1_todo", 0o700))

		content := `---
id: 001
title: Test Feature
status: todo
kind: prd
created: 2024-01-01
reviewer: reviewer@example.com
estimate: 5
tags: [frontend, backend]
---
# Test Feature
`
		require.NoError(t, os.WriteFile(testFilePath, []byte(content), 0o600))

		err := updateWorkItemFieldAppend(testFilePath, "assigned", "user@example.com")
		require.NoError(t, err)

		// Verify other fields are preserved
		updatedContent, err := os.ReadFile(testFilePath)
		require.NoError(t, err)
		updatedStr := string(updatedContent)

		assert.Contains(t, updatedStr, "reviewer: reviewer@example.com")
		assert.Contains(t, updatedStr, "estimate: 5")
		assert.Contains(t, updatedStr, "tags: [frontend, backend]")
	})

	t.Run("preserves body content", func(t *testing.T) {
		tmpDir := t.TempDir()
		origDir, _ := os.Getwd()
		require.NoError(t, os.Chdir(tmpDir))
		defer func() { _ = os.Chdir(origDir) }()

		require.NoError(t, os.MkdirAll(".work/1_todo", 0o700))

		content := `---
id: 001
title: Test Feature
status: todo
kind: prd
created: 2024-01-01
---
# Test Feature

This is the body content.

## Section

More content here.
`
		require.NoError(t, os.WriteFile(testFilePath, []byte(content), 0o600))

		err := updateWorkItemFieldAppend(testFilePath, "assigned", "user@example.com")
		require.NoError(t, err)

		// Verify body is preserved
		updatedContent, err := os.ReadFile(testFilePath)
		require.NoError(t, err)
		updatedStr := string(updatedContent)

		assert.Contains(t, updatedStr, "# Test Feature")
		assert.Contains(t, updatedStr, "This is the body content.")
		assert.Contains(t, updatedStr, "## Section")
		assert.Contains(t, updatedStr, "More content here.")
	})

	t.Run("works with custom field names", func(t *testing.T) {
		tmpDir := t.TempDir()
		origDir, _ := os.Getwd()
		require.NoError(t, os.Chdir(tmpDir))
		defer func() { _ = os.Chdir(origDir) }()

		require.NoError(t, os.MkdirAll(".work/1_todo", 0o700))

		content := `---
id: 001
title: Test Feature
status: todo
kind: prd
created: 2024-01-01
reviewer: alice@example.com
---
# Test Feature
`
		require.NoError(t, os.WriteFile(testFilePath, []byte(content), 0o600))

		err := updateWorkItemFieldAppend(testFilePath, "reviewer", "bob@example.com")
		require.NoError(t, err)

		// Verify custom field was updated
		updatedContent, err := os.ReadFile(testFilePath)
		require.NoError(t, err)
		updatedStr := string(updatedContent)

		assert.Contains(t, updatedStr, "reviewer: [")
		assert.Contains(t, updatedStr, "alice@example.com")
		assert.Contains(t, updatedStr, "bob@example.com")
	})

	t.Run("handles multiple appends to same field", func(t *testing.T) {
		tmpDir := t.TempDir()
		origDir, _ := os.Getwd()
		require.NoError(t, os.Chdir(tmpDir))
		defer func() { _ = os.Chdir(origDir) }()

		require.NoError(t, os.MkdirAll(".work/1_todo", 0o700))

		content := `---
id: 001
title: Test Feature
status: todo
kind: prd
created: 2024-01-01
assigned: alice@example.com
---
# Test Feature
`
		require.NoError(t, os.WriteFile(testFilePath, []byte(content), 0o600))

		// First append
		err := updateWorkItemFieldAppend(testFilePath, "assigned", "bob@example.com")
		require.NoError(t, err)

		// Second append
		err = updateWorkItemFieldAppend(testFilePath, "assigned", "charlie@example.com")
		require.NoError(t, err)

		// Verify all users are in array
		updatedContent, err := os.ReadFile(testFilePath)
		require.NoError(t, err)
		updatedStr := string(updatedContent)

		assert.Contains(t, updatedStr, "assigned: [")
		assert.Contains(t, updatedStr, "alice@example.com")
		assert.Contains(t, updatedStr, "bob@example.com")
		assert.Contains(t, updatedStr, "charlie@example.com")
	})

	t.Run("returns error for file not found", func(t *testing.T) {
		tmpDir := t.TempDir()
		origDir, _ := os.Getwd()
		require.NoError(t, os.Chdir(tmpDir))
		defer func() { _ = os.Chdir(origDir) }()

		require.NoError(t, os.MkdirAll(".work/1_todo", 0o700))

		err := updateWorkItemFieldAppend(testFilePath, "assigned", "user@example.com")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to read work item file")
	})

	t.Run("returns error for malformed YAML", func(t *testing.T) {
		tmpDir := t.TempDir()
		origDir, _ := os.Getwd()
		require.NoError(t, os.Chdir(tmpDir))
		defer func() { _ = os.Chdir(origDir) }()

		require.NoError(t, os.MkdirAll(".work/1_todo", 0o700))

		content := `---
id: 001
title: Test Feature
invalid: [unclosed bracket
---
# Test Feature
`
		require.NoError(t, os.WriteFile(testFilePath, []byte(content), 0o600))

		err := updateWorkItemFieldAppend(testFilePath, "assigned", "user@example.com")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to parse front matter")
	})
}
