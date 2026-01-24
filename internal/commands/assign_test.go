package commands

import (
	"os"
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
