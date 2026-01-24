// Package commands implements the CLI commands for the kira tool.
package commands

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"kira/internal/config"
)

// AssignFlags holds all flags for the assign command.
type AssignFlags struct {
	Field       string
	Append      bool
	Unassign    bool
	Interactive bool
	DryRun      bool
}

var assignCmd = &cobra.Command{
	Use:   "assign <work-item-id...> [user-identifier]",
	Short: "Assign work items to users",
	Long: `Assign one or more work items to a user by updating user-related fields
in the work item's front matter.

Work items can be specified by numeric ID (e.g. 001) or by full path to the
work item file under the .work/ directory. User identifiers can be numeric
user numbers from ` + "`kira users`" + `, email addresses, or names.

Examples:
  kira assign 001 5
  kira assign 001 002 003 5
  kira assign .work/1_todo/001-test.prd.md user@example.com
  kira assign 001 --interactive
  kira assign 001 --unassign
  kira assign 001 5 --field reviewer
  kira assign 001 5 --append`,
	Args: cobra.MinimumNArgs(1),
	RunE: runAssign,
}

func init() {
	assignCmd.Flags().StringP("field", "f", "assigned", "Target field name to update (default: assigned)")
	assignCmd.Flags().BoolP("append", "a", false, "Append user to existing field value instead of replacing")
	assignCmd.Flags().BoolP("unassign", "u", false, "Clear the target field (remove assignment)")
	assignCmd.Flags().BoolP("interactive", "I", false, "Select user interactively from available users")
	assignCmd.Flags().Bool("dry-run", false, "Preview what would be done without making changes")
}

// runAssign is the entrypoint for the assign command.
// Phase 1 only performs input parsing and validation.
func runAssign(cmd *cobra.Command, args []string) error {
	if err := checkWorkDir(); err != nil {
		return err
	}

	cfg, err := config.LoadConfig()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	flags, err := parseAssignFlags(cmd)
	if err != nil {
		return err
	}

	workItems, userIdentifier := parseAssignArgs(args, flags)

	if err := validateAssignInput(workItems, userIdentifier, flags, cfg); err != nil {
		return err
	}

	// Phase 1: command stops after validation.
	return nil
}

func parseAssignFlags(cmd *cobra.Command) (AssignFlags, error) {
	field, err := cmd.Flags().GetString("field")
	if err != nil {
		return AssignFlags{}, err
	}
	appendFlag, err := cmd.Flags().GetBool("append")
	if err != nil {
		return AssignFlags{}, err
	}
	unassignFlag, err := cmd.Flags().GetBool("unassign")
	if err != nil {
		return AssignFlags{}, err
	}
	interactiveFlag, err := cmd.Flags().GetBool("interactive")
	if err != nil {
		return AssignFlags{}, err
	}
	dryRunFlag, err := cmd.Flags().GetBool("dry-run")
	if err != nil {
		return AssignFlags{}, err
	}

	return AssignFlags{
		Field:       field,
		Append:      appendFlag,
		Unassign:    unassignFlag,
		Interactive: interactiveFlag,
		DryRun:      dryRunFlag,
	}, nil
}

// parseAssignArgs splits positional arguments into work item identifiers and an optional user identifier.
func parseAssignArgs(args []string, flags AssignFlags) (workItems []string, userIdentifier string) {
	if len(args) == 0 {
		return nil, ""
	}

	// In unassign mode, all arguments are work items; user identifier is not allowed.
	if flags.Unassign {
		return append([]string{}, args...), ""
	}

	// In interactive mode, user identifier is optional; treat all args as work items.
	if flags.Interactive {
		return append([]string{}, args...), ""
	}

	if len(args) == 1 {
		// Single work item; user identifier must be provided separately or via flags (not supported in phase 1).
		return []string{args[0]}, ""
	}

	// Default: last argument is user identifier, preceding are work items.
	workItems = append([]string{}, args[:len(args)-1]...)
	userIdentifier = args[len(args)-1]
	return workItems, userIdentifier
}

// validateAssignInput validates work item identifiers, user identifier, and flag combinations.
func validateAssignInput(workItems []string, userIdentifier string, flags AssignFlags, cfg *config.Config) error {
	if err := validateWorkItemsPresent(workItems); err != nil {
		return err
	}

	if err := validateAssignFieldName(flags.Field); err != nil {
		return err
	}

	if err := validateAssignFlagCombinations(userIdentifier, flags); err != nil {
		return err
	}

	if err := validateAssignUserIdentifierRequired(userIdentifier, flags); err != nil {
		return err
	}

	// Validate work item tokens as IDs or paths.
	for _, token := range workItems {
		if isWorkItemPath(token) {
			if err := validateWorkPath(token); err != nil {
				return err
			}
			continue
		}

		// Treat as work item ID and validate format.
		if err := validateWorkItemID(token, cfg); err != nil {
			return err
		}
	}

	return nil
}

func validateWorkItemsPresent(workItems []string) error {
	if len(workItems) == 0 {
		return fmt.Errorf("at least one work item ID or path is required")
	}
	return nil
}

func validateAssignFieldName(field string) error {
	if strings.TrimSpace(field) == "" {
		return fmt.Errorf("field name cannot be empty")
	}
	if strings.Contains(field, "/") || strings.Contains(field, "\\") || strings.Contains(field, "..") {
		return fmt.Errorf("invalid field name '%s': field name must not contain path separators or '..'", field)
	}
	return nil
}

func validateAssignFlagCombinations(userIdentifier string, flags AssignFlags) error {
	if !flags.Unassign {
		return nil
	}

	if userIdentifier != "" {
		return fmt.Errorf("cannot specify user identifier when using --unassign")
	}
	if flags.Append {
		return fmt.Errorf("invalid flag combination: --unassign cannot be used together with --append")
	}
	if flags.Interactive {
		return fmt.Errorf("invalid flag combination: --unassign cannot be used together with --interactive in this phase")
	}

	return nil
}

func validateAssignUserIdentifierRequired(userIdentifier string, flags AssignFlags) error {
	if flags.Unassign || flags.Interactive {
		return nil
	}

	if strings.TrimSpace(userIdentifier) == "" {
		return fmt.Errorf("user identifier is required unless --unassign or --interactive is used")
	}

	return nil
}

func isWorkItemPath(token string) bool {
	return strings.Contains(token, "/") || strings.Contains(token, "\\") || strings.HasSuffix(token, ".md")
}
