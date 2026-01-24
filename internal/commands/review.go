// Package commands implements the CLI commands for the kira tool.
package commands

import (
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/spf13/cobra"
	yaml "gopkg.in/yaml.v3"

	"kira/internal/config"
	"kira/internal/validation"
)

var reviewCmd = &cobra.Command{
	Use:   "review",
	Short: "Submit work item for review and create GitHub pull request",
	Long: `Automatically derives work item ID from current branch name and creates a draft PR.

The command will:
1. Derive work item ID from current branch name
2. Move work item to review status
3. Create or update GitHub pull request
4. Optionally update trunk branch status and rebase`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, _ []string) error {
		// Load configuration
		cfg, err := config.LoadConfig()
		if err != nil {
			return fmt.Errorf("failed to load configuration: %w", err)
		}

		// Validate branch context (Phase 2)
		if err := validateBranchContext(cfg); err != nil {
			return err
		}

		// Get current branch and derive work item ID (Phase 2)
		currentBranch, err := getCurrentBranch("")
		if err != nil {
			return fmt.Errorf("failed to determine current branch: %w", err)
		}

		workItemID, err := deriveWorkItemFromBranch(currentBranch)
		if err != nil {
			return err
		}

		// Load and validate work item (Phase 4)
		workItem, _, err := loadWorkItem(cfg, workItemID)
		if err != nil {
			return err
		}

		if err := validateWorkItemStatusForReview(workItem, cfg); err != nil {
			var alreadyErr *alreadyInReviewError
			if errors.As(err, &alreadyErr) {
				// Already in review is a successful no-op
				_, _ = fmt.Fprintln(cmd.OutOrStdout(), alreadyErr.Error())
				return nil
			}
			return err
		}

		if err := validateRequiredFieldsForReview(workItem, cfg); err != nil {
			return err
		}

		// Ensure working directory is clean before proceeding with review operations (Phase 5)
		if err := checkUncommittedChangesForReview(); err != nil {
			return err
		}

		// Extract flags to validate parsing (actual logic will be implemented in later phases)
		_, _ = cmd.Flags().GetStringArray("reviewer")
		_, _ = cmd.Flags().GetBool("draft")
		_, _ = cmd.Flags().GetBool("no-trunk-update")
		_, _ = cmd.Flags().GetBool("no-rebase")
		_, _ = cmd.Flags().GetString("title")
		_, _ = cmd.Flags().GetString("description")

		// Placeholder: return help for now (will be replaced in later phases)
		_ = workItemID // Suppress unused variable warning
		return cmd.Help()
	},
}

func init() {
	reviewCmd.Flags().StringArray("reviewer", []string{}, "Specify reviewer (can be used multiple times). Can be user number from 'kira user' command or email address")
	reviewCmd.Flags().Bool("draft", true, "Create as draft PR (default: true)")
	reviewCmd.Flags().Bool("no-trunk-update", false, "Skip updating trunk branch status (overrides config)")
	reviewCmd.Flags().Bool("no-rebase", false, "Skip rebasing current branch after trunk update (overrides config)")
	reviewCmd.Flags().String("title", "", "Custom PR title (derived from work item if not provided)")
	reviewCmd.Flags().String("description", "", "Custom PR description (uses work item content if not provided)")
}

// deriveWorkItemFromBranch extracts the work item ID from a branch name.
// Branch format: {id}-{title} (e.g., "012-submit-for-review")
// Returns the 3-digit work item ID or an error if the format is invalid.
func deriveWorkItemFromBranch(branchName string) (string, error) {
	if branchName == "" {
		return "", fmt.Errorf("branch name cannot be empty")
	}

	// Find first dash in branch name
	dashIndex := strings.Index(branchName, "-")
	if dashIndex == -1 {
		return "", fmt.Errorf("branch name '%s' does not follow kira naming convention: expected format '{id}-{title}'", branchName)
	}

	// Extract ID from beginning of branch name
	workItemID := branchName[:dashIndex]
	if workItemID == "" {
		return "", fmt.Errorf("branch name '%s' does not follow kira naming convention: work item ID is missing", branchName)
	}

	// Validate ID format (must be exactly 3 digits)
	idRegex := regexp.MustCompile(`^\d{3}$`)
	if !idRegex.MatchString(workItemID) {
		return "", fmt.Errorf("invalid work item ID '%s' in branch name '%s': ID must be exactly 3 digits", workItemID, branchName)
	}

	return workItemID, nil
}

// validateBranchContext ensures the command is not run on the trunk branch.
// Returns an error if the current branch is the trunk branch, nil otherwise.
func validateBranchContext(cfg *config.Config) error {
	// Get current branch
	currentBranch, err := getCurrentBranch("")
	if err != nil {
		return fmt.Errorf("failed to determine current branch: %w", err)
	}

	// Determine trunk branch (using same logic as start command)
	trunkBranch, err := determineTrunkBranch(cfg, "", "", false)
	if err != nil {
		return fmt.Errorf("failed to determine trunk branch: %w", err)
	}

	// Check if current branch is trunk branch
	if currentBranch == trunkBranch {
		return fmt.Errorf("cannot run 'kira review' on trunk branch '%s'", trunkBranch)
	}

	return nil
}

const (
	yamlDelimiter = "---"
	statusReview  = "review"
)

// alreadyInReviewError is returned when a work item is already in review status.
// Callers should treat this as a successful outcome and not as a fatal error.
type alreadyInReviewError struct{}

func (e *alreadyInReviewError) Error() string {
	return "Work item is already in review status."
}

// loadWorkItem loads a single work item by ID using existing work item utilities.
// It returns the parsed work item and its file path.
func loadWorkItem(cfg *config.Config, workItemID string) (*validation.WorkItem, string, error) {
	if cfg == nil {
		return nil, "", fmt.Errorf("configuration cannot be nil")
	}
	if strings.TrimSpace(workItemID) == "" {
		return nil, "", fmt.Errorf("work item ID cannot be empty")
	}

	workItemPath, err := findWorkItemFile(workItemID)
	if err != nil {
		// Normalize "not found" error to match PRD messaging
		if strings.Contains(err.Error(), "work item with ID") {
			return nil, "", fmt.Errorf("work item %s not found", workItemID)
		}
		return nil, "", fmt.Errorf("failed to load work item %s: %w", workItemID, err)
	}

	content, err := safeReadFile(workItemPath)
	if err != nil {
		return nil, "", fmt.Errorf("failed to read work item file %s: %w", workItemPath, err)
	}

	// Extract YAML front matter between the first pair of --- lines
	lines := strings.Split(string(content), "\n")
	var yamlLines []string
	inYAML := false
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if i == 0 && trimmed == yamlDelimiter {
			inYAML = true
			continue
		}
		if inYAML {
			if trimmed == yamlDelimiter {
				break
			}
			yamlLines = append(yamlLines, line)
		}
	}

	workItem := &validation.WorkItem{Fields: make(map[string]interface{})}
	if len(yamlLines) > 0 {
		if err := yaml.Unmarshal([]byte(strings.Join(yamlLines, "\n")), workItem); err != nil {
			return nil, "", fmt.Errorf("failed to parse work item front matter: %w", err)
		}
	}

	// Validate that the work item ID in the file matches the derived ID when present.
	if workItem.ID != "" && workItem.ID != workItemID {
		return nil, "", fmt.Errorf("work item ID mismatch: expected %s but found %s", workItemID, workItem.ID)
	}

	return workItem, workItemPath, nil
}

// validateWorkItemStatusForReview ensures the work item is in a status that
// can be moved to review. It allows "todo" and "doing", treats "review" as a
// successful no-op, and rejects all other statuses.
func validateWorkItemStatusForReview(workItem *validation.WorkItem, _ *config.Config) error {
	if workItem == nil {
		return fmt.Errorf("work item cannot be nil")
	}

	status := strings.TrimSpace(strings.ToLower(workItem.Status))
	switch status {
	case "todo", "doing":
		return nil
	case statusReview:
		return &alreadyInReviewError{}
	case "":
		return fmt.Errorf("cannot submit for review: work item has empty status. Only 'todo' or 'doing' status can be moved to review")
	default:
		return fmt.Errorf("cannot submit for review: work item is in %s status. Only 'todo' or 'doing' status can be moved to review", workItem.Status)
	}
}

// validateRequiredFieldsForReview validates that the work item contains all
// required fields configured in cfg.Validation.RequiredFields. It checks both
// core fields and any additional custom fields present in WorkItem.Fields.
func validateRequiredFieldsForReview(workItem *validation.WorkItem, cfg *config.Config) error {
	if workItem == nil {
		return fmt.Errorf("work item cannot be nil")
	}
	if cfg == nil {
		return fmt.Errorf("configuration cannot be nil")
	}

	var missing []string
	for _, field := range cfg.Validation.RequiredFields {
		if isRequiredFieldMissing(workItem, field) {
			missing = append(missing, field)
		}
	}

	if len(missing) == 0 {
		return nil
	}

	return fmt.Errorf("work item missing required fields: %s. Update work item and try again", strings.Join(missing, ", "))
}

// isRequiredFieldMissing encapsulates the logic for determining whether a single
// configured field is missing from the provided work item.
func isRequiredFieldMissing(workItem *validation.WorkItem, field string) bool {
	switch field {
	case "id":
		return strings.TrimSpace(workItem.ID) == ""
	case "title":
		return strings.TrimSpace(workItem.Title) == ""
	case "status":
		return strings.TrimSpace(workItem.Status) == ""
	case "kind":
		return strings.TrimSpace(workItem.Kind) == ""
	case "created":
		return strings.TrimSpace(workItem.Created) == ""
	default:
		// Custom required fields are stored in the Fields map.
		value, exists := workItem.Fields[field]
		return !exists || isEmptyWorkItemField(value)
	}
}

// isEmptyWorkItemField determines whether a dynamic work item field should be
// considered "missing" for required field validation.
func isEmptyWorkItemField(value interface{}) bool {
	if value == nil {
		return true
	}

	switch v := value.(type) {
	case string:
		return strings.TrimSpace(v) == ""
	default:
		return false
	}
}

// checkUncommittedChangesForReview ensures there are no uncommitted changes
// in the current repository before running review operations.
// It returns a clear, PRD-aligned error message when changes are detected.
func checkUncommittedChangesForReview() error {
	hasUncommitted, err := checkUncommittedChanges("", false)
	if err != nil {
		return err
	}

	if hasUncommitted {
		return fmt.Errorf("uncommitted changes detected. Commit or stash changes before submitting for review")
	}

	return nil
}
