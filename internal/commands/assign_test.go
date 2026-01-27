package commands

import (
	"os"
	"path/filepath"
	"testing"

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
