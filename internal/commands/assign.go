// Package commands implements the CLI commands for the kira tool.
package commands

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"
	yaml "gopkg.in/yaml.v3"

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

// Operation name for "no change, already assigned to same user".
const opAlreadyAssigned = "already_assigned"

// WorkItemUpdateResult tracks the result of updating a single work item.
type WorkItemUpdateResult struct {
	WorkItemPath string
	WorkItemID   string // Display identifier (ID or path)
	Success      bool
	Error        error
	Operation    string // "assign", "unassign", "append", or opAlreadyAssigned
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
	cfg, err := config.LoadConfig()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	if err := checkWorkDir(cfg); err != nil {
		return err
	}

	flags, err := parseAssignFlags(cmd)
	if err != nil {
		return err
	}

	workItems, userIdentifier := parseAssignArgs(args, flags)

	if err := validateAssignInput(workItems, userIdentifier, flags, cfg); err != nil {
		return err
	}

	// Phase 2: Resolve and validate work items exist.
	workItemPaths, err := resolveWorkItems(workItems, cfg)
	if err != nil {
		return err
	}

	// Phase 3: Collect users and resolve user identifier if provided.
	users, err := collectUsersForAssignment(cfg)
	if err != nil {
		return fmt.Errorf("failed to collect users: %w", err)
	}

	var resolvedUser *UserInfo
	if userIdentifier != "" {
		resolvedUser, err = resolveUserIdentifier(userIdentifier, users)
		if err != nil {
			return err
		}
	}

	// Phase 8: Process work item updates with batch processing and progress
	results := processWorkItemUpdates(workItemPaths, resolvedUser, flags, users, cfg)
	return handleAssignResults(results, workItemPaths, flags, resolvedUser)
}

// handleAssignResults displays batch or single-item output and returns an error if any update failed.
func handleAssignResults(results []WorkItemUpdateResult, workItemPaths []string, flags AssignFlags, resolvedUser *UserInfo) error {
	if len(workItemPaths) > 1 || flags.DryRun {
		displayBatchSummary(results)
	} else if len(results) > 0 && !results[0].Success {
		displayBatchSummary(results)
	} else if len(results) == 1 && results[0].Success && !flags.DryRun {
		displaySingleSuccessMessage(results[0], resolvedUser, flags)
	}
	for _, result := range results {
		if !result.Success {
			return fmt.Errorf("one or more work items failed to update")
		}
	}
	return nil
}

// getWorkItemDisplayID extracts a display identifier from a work item file path.
// Returns the work item ID if available, otherwise returns a shortened path.
func getWorkItemDisplayID(workItemPath string, cfg *config.Config) string {
	// Try to extract ID from front matter
	frontMatter, _, err := parseWorkItemFrontMatter(workItemPath, cfg)
	if err == nil {
		if idValue, exists := frontMatter["id"]; exists {
			if idStr, ok := idValue.(string); ok && idStr != "" {
				return idStr
			}
			// Handle numeric IDs
			if idNum, ok := idValue.(int); ok {
				return fmt.Sprintf("%03d", idNum)
			}
		}
	}

	// Fallback to shortened path
	base := filepath.Base(workItemPath)
	if strings.HasSuffix(base, ".md") {
		return base[:len(base)-3] // Remove .md extension
	}
	return base
}

// processWorkItemInDryRun validates a work item in dry-run mode.
func processWorkItemInDryRun(path string, cfg *config.Config) WorkItemUpdateResult {
	displayID := getWorkItemDisplayID(path, cfg)
	_, _, err := parseWorkItemFrontMatter(path, cfg)
	if err != nil {
		return WorkItemUpdateResult{
			WorkItemPath: path,
			WorkItemID:   displayID,
			Success:      false,
			Error:        fmt.Errorf("dry-run: failed to parse work item: %w", err),
			Operation:    "validate",
		}
	}
	return WorkItemUpdateResult{
		WorkItemPath: path,
		WorkItemID:   displayID,
		Success:      true,
		Operation:    "validate",
	}
}

// processUnassignWorkItem handles unassign operation for a work item.
func processUnassignWorkItem(
	workItemPath string,
	displayID string,
	field string,
	showProgress bool,
	cfg *config.Config,
) WorkItemUpdateResult {
	result := WorkItemUpdateResult{
		WorkItemPath: workItemPath,
		WorkItemID:   displayID,
		Success:      false,
		Operation:    "unassign",
	}

	if err := updateWorkItemFieldUnassign(workItemPath, field, cfg); err != nil {
		result.Error = fmt.Errorf("failed to update work item %s: %w", displayID, err)
		if showProgress {
			displayWorkItemProgress(result)
		}
		return result
	}
	result.Success = true
	if showProgress {
		displayWorkItemProgress(result)
	}
	return result
}

// processAppendWorkItem handles append operation for a work item.
func processAppendWorkItem(
	workItemPath string,
	displayID string,
	field string,
	resolvedUser *UserInfo,
	showProgress bool,
	cfg *config.Config,
) WorkItemUpdateResult {
	result := WorkItemUpdateResult{
		WorkItemPath: workItemPath,
		WorkItemID:   displayID,
		Success:      false,
		Operation:    "append",
	}

	if resolvedUser == nil {
		result.Error = fmt.Errorf("user identifier is required for assignment")
		if showProgress {
			displayWorkItemProgress(result)
		}
		return result
	}

	if err := updateWorkItemFieldAppend(workItemPath, field, resolvedUser.Email, cfg); err != nil {
		result.Error = fmt.Errorf("failed to update work item %s: %w", displayID, err)
		if showProgress {
			displayWorkItemProgress(result)
		}
		return result
	}
	result.Success = true
	if showProgress {
		displayWorkItemProgress(result)
	}
	return result
}

// processAssignWorkItem handles assign operation for a work item.
func processAssignWorkItem(
	workItemPath string,
	displayID string,
	field string,
	resolvedUser *UserInfo,
	showProgress bool,
	cfg *config.Config,
) WorkItemUpdateResult {
	result := WorkItemUpdateResult{
		WorkItemPath: workItemPath,
		WorkItemID:   displayID,
		Success:      false,
		Operation:    "assign",
	}

	if resolvedUser == nil {
		result.Error = fmt.Errorf("user identifier is required for assignment")
		if showProgress {
			displayWorkItemProgress(result)
		}
		return result
	}

	current, err := getCurrentAssignment(workItemPath, field, cfg)
	if err == nil && current != "" {
		// Same user: exact email match or display format match
		if current == resolvedUser.Email || current == formatUserDisplay(*resolvedUser) {
			result.Success = true
			result.Operation = opAlreadyAssigned
			if showProgress {
				displayWorkItemProgress(result)
			}
			return result
		}
		// Current is comma-separated list (array display); check if email is in it
		if strings.Contains(current, resolvedUser.Email) {
			parts := strings.Split(current, ", ")
			for _, p := range parts {
				if strings.TrimSpace(p) == resolvedUser.Email {
					result.Success = true
					result.Operation = opAlreadyAssigned
					if showProgress {
						displayWorkItemProgress(result)
					}
					return result
				}
			}
		}
	}

	if err := updateWorkItemField(workItemPath, field, resolvedUser.Email, cfg); err != nil {
		result.Error = fmt.Errorf("failed to update work item %s: %w", displayID, err)
		if showProgress {
			displayWorkItemProgress(result)
		}
		return result
	}
	result.Success = true
	if showProgress {
		displayWorkItemProgress(result)
	}
	return result
}

// processSingleWorkItem processes a single work item update.
func processSingleWorkItem(
	workItemPath string,
	displayID string,
	resolvedUser *UserInfo,
	flags AssignFlags,
	showProgress bool,
	users []UserInfo,
	cfg *config.Config,
) WorkItemUpdateResult {
	if showProgress {
		fmt.Printf("Processing work item %s...\n", displayID)
	}

	// For unassign mode, remove the field
	if flags.Unassign {
		return processUnassignWorkItem(workItemPath, displayID, flags.Field, showProgress, cfg)
	}

	// For interactive mode, show selection and process
	if flags.Interactive {
		// Get current assignment for this work item
		currentAssignment, err := getCurrentAssignment(workItemPath, flags.Field, cfg)
		if err != nil {
			result := WorkItemUpdateResult{
				WorkItemPath: workItemPath,
				WorkItemID:   displayID,
				Success:      false,
				Operation:    "interactive",
				Error:        fmt.Errorf("failed to get current assignment: %w", err),
			}
			if showProgress {
				displayWorkItemProgress(result)
			}
			return result
		}

		// Show interactive selection
		selection, err := showInteractiveSelection(users, currentAssignment, flags.Field, os.Stdin)
		if err != nil {
			result := WorkItemUpdateResult{
				WorkItemPath: workItemPath,
				WorkItemID:   displayID,
				Success:      false,
				Operation:    "interactive",
				Error:        fmt.Errorf("interactive selection failed: %w", err),
			}
			if showProgress {
				displayWorkItemProgress(result)
			}
			return result
		}

		// Handle selection: 0 = unassign, 1+ = assign to user
		if selection == 0 {
			return processUnassignWorkItem(workItemPath, displayID, flags.Field, showProgress, cfg)
		}

		// Resolve selected user
		selectedUser, err := findUserByNumber(selection, users)
		if err != nil {
			result := WorkItemUpdateResult{
				WorkItemPath: workItemPath,
				WorkItemID:   displayID,
				Success:      false,
				Operation:    "interactive",
				Error:        fmt.Errorf("failed to resolve selected user: %w", err),
			}
			if showProgress {
				displayWorkItemProgress(result)
			}
			return result
		}

		// Process assignment based on append flag
		if flags.Append {
			return processAppendWorkItem(workItemPath, displayID, flags.Field, selectedUser, showProgress, cfg)
		}

		// Switch mode: update field with user email
		return processAssignWorkItem(workItemPath, displayID, flags.Field, selectedUser, showProgress, cfg)
	}

	// For append mode, handle in Phase 6
	if flags.Append {
		return processAppendWorkItem(workItemPath, displayID, flags.Field, resolvedUser, showProgress, cfg)
	}

	// Switch mode: update field with user email
	return processAssignWorkItem(workItemPath, displayID, flags.Field, resolvedUser, showProgress, cfg)
}

// processWorkItemUpdates processes work item updates based on flags.
// Returns a slice of results for each work item processed.
func processWorkItemUpdates(workItemPaths []string, resolvedUser *UserInfo, flags AssignFlags, users []UserInfo, cfg *config.Config) []WorkItemUpdateResult {
	var results []WorkItemUpdateResult
	showProgress := len(workItemPaths) > 1

	// Skip if dry-run mode
	if flags.DryRun {
		for _, path := range workItemPaths {
			res := processWorkItemInDryRun(path, cfg)
			if res.Success {
				displayID := res.WorkItemID
				if flags.Unassign {
					fmt.Printf("Would unassign work item %s\n", displayID)
				} else if resolvedUser != nil {
					fmt.Printf("Would assign work item %s to %s\n", displayID, formatUserDisplay(*resolvedUser))
				}
			}
			results = append(results, res)
		}
		return results
	}

	// Process each work item
	for _, workItemPath := range workItemPaths {
		displayID := getWorkItemDisplayID(workItemPath, cfg)
		result := processSingleWorkItem(workItemPath, displayID, resolvedUser, flags, showProgress, users, cfg)
		results = append(results, result)
	}

	return results
}

// displaySingleSuccessMessage prints the PRD success message for a single work item.
func displaySingleSuccessMessage(result WorkItemUpdateResult, resolvedUser *UserInfo, flags AssignFlags) {
	id := result.WorkItemID
	switch result.Operation {
	case "unassign":
		fmt.Printf("Unassigned work item %s\n", id)
	case "append":
		if resolvedUser != nil {
			fmt.Printf("Added %s to %s for work item %s\n", formatUserDisplay(*resolvedUser), flags.Field, id)
		}
	case opAlreadyAssigned:
		if resolvedUser != nil {
			fmt.Printf("Work item %s is already assigned to %s. Use --unassign to clear or specify a different user.\n", id, formatUserDisplay(*resolvedUser))
		}
	case "assign":
		if resolvedUser != nil {
			if flags.Field != "assigned" {
				fmt.Printf("Assigned %s for work item %s to %s\n", flags.Field, id, formatUserDisplay(*resolvedUser))
			} else {
				fmt.Printf("Assigned work item %s to %s\n", id, formatUserDisplay(*resolvedUser))
			}
		}
	default:
		// Fallback for validate or other operations
		if result.Success {
			op := result.Operation
			if op == "validate" {
				op = "validated"
			}
			fmt.Printf("  ✓ Work item %s: %s successfully\n", id, op)
		}
	}
}

// displayWorkItemProgress shows progress for processing a single work item.
func displayWorkItemProgress(result WorkItemUpdateResult) {
	if result.Success {
		operation := result.Operation
		if operation == "validate" {
			operation = "validated"
		}
		fmt.Printf("  ✓ Work item %s: %s successfully\n", result.WorkItemID, operation)
	} else {
		fmt.Printf("  ✗ Work item %s: failed - %v\n", result.WorkItemID, result.Error)
	}
}

// displayBatchSummary displays a summary of batch operation results.
func displayBatchSummary(results []WorkItemUpdateResult) {
	if len(results) == 0 {
		return
	}

	fmt.Println("\nOperation Results:")
	fmt.Println("───────────────────────────────────────────────────────────────")

	successCount := 0
	failureCount := 0
	var failedItems []WorkItemUpdateResult

	for _, result := range results {
		if result.Success {
			successCount++
			displayWorkItemProgress(result)
		} else {
			failureCount++
			failedItems = append(failedItems, result)
			displayWorkItemProgress(result)
		}
	}

	fmt.Println("───────────────────────────────────────────────────────────────")
	fmt.Printf("Summary: %d succeeded, %d failed\n", successCount, failureCount)

	if len(failedItems) > 0 {
		fmt.Println("\nFailed work items:")
		for _, result := range failedItems {
			fmt.Printf("  - %s: %v\n", result.WorkItemID, result.Error)
		}
	}
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
			if err := validateWorkPath(token, cfg); err != nil {
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
		return fmt.Errorf("invalid flag combination: --unassign cannot be used together with --interactive (use --interactive and select 0 to unassign)")
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

// resolveWorkItemPath resolves a work item identifier (ID or path) to an absolute file path.
// If identifier is a path, it validates and returns the absolute path.
// If identifier is an ID, it uses findWorkItemFile to locate the file.
func resolveWorkItemPath(identifier string, cfg *config.Config) (string, error) {
	// If identifier is a path, validate and return it.
	if isWorkItemPath(identifier) {
		if err := validateWorkPath(identifier, cfg); err != nil {
			return "", fmt.Errorf("invalid work item path '%s': %w", identifier, err)
		}

		// Get absolute path for consistency.
		absPath, err := filepath.Abs(identifier)
		if err != nil {
			return "", fmt.Errorf("failed to resolve work item path '%s': %w", identifier, err)
		}

		return absPath, nil
	}

	// Treat as work item ID and find the file.
	workItemPath, err := findWorkItemFile(identifier, cfg)
	if err != nil {
		return "", fmt.Errorf("work item %s not found", identifier)
	}

	// Get absolute path for consistency.
	absPath, err := filepath.Abs(workItemPath)
	if err != nil {
		return "", fmt.Errorf("failed to resolve work item path for ID '%s': %w", identifier, err)
	}

	return absPath, nil
}

// validateWorkItemFile validates that a work item file exists and is readable.
func validateWorkItemFile(path string, cfg *config.Config) error {
	// Check if file exists.
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return fmt.Errorf("work item file does not exist: %s", path)
	} else if err != nil {
		return fmt.Errorf("failed to access work item file: %w", err)
	}

	// Try to read the file to ensure it's readable.
	// Use safeReadFile which validates the path is within .work.
	if _, err := safeReadFile(path, cfg); err != nil {
		return fmt.Errorf("failed to read work item file: %w", err)
	}

	return nil
}

// resolveWorkItems resolves multiple work item identifiers to file paths and validates them.
// Returns an error if any work item cannot be resolved or validated.
func resolveWorkItems(identifiers []string, cfg *config.Config) ([]string, error) {
	if len(identifiers) == 0 {
		return nil, fmt.Errorf("no work items to resolve")
	}

	var resolvedPaths []string
	var errors []string

	for _, identifier := range identifiers {
		path, err := resolveWorkItemPath(identifier, cfg)
		if err != nil {
			errors = append(errors, fmt.Sprintf("  %s: %v", identifier, err))
			continue
		}

		if err := validateWorkItemFile(path, cfg); err != nil {
			errors = append(errors, fmt.Sprintf("  %s: %v", identifier, err))
			continue
		}

		resolvedPaths = append(resolvedPaths, path)
	}

	if len(errors) > 0 {
		errorMsg := "failed to resolve or validate work items:\n" + strings.Join(errors, "\n")
		return nil, fmt.Errorf("%s", errorMsg)
	}

	return resolvedPaths, nil
}

// collectUsersForAssignment collects users using the same logic as the kira users command.
// This ensures consistency between the two commands.
func collectUsersForAssignment(cfg *config.Config) ([]UserInfo, error) {
	useGitHistory := getUseGitHistorySetting(cfg)
	commitLimit := getCommitLimit(0, false, cfg)

	userMap, err := collectUsers(useGitHistory, commitLimit, cfg)
	if err != nil {
		return nil, err
	}

	users := processAndSortUsers(userMap, useGitHistory)
	return users, nil
}

// resolveUserIdentifier resolves a user identifier to a UserInfo.
// It tries different resolution methods in priority order:
// 1. Numeric identifier (via findUserByNumber)
// 2. Exact email match (case-insensitive)
// 3. Partial email match (if unique)
// 4. Exact name match (case-insensitive)
// 5. Partial name match (if unique)
// Returns an error if no matches or multiple matches (with list of matches).
func resolveUserIdentifier(identifier string, users []UserInfo) (*UserInfo, error) {
	// Try numeric identifier first
	if num, err := strconv.Atoi(identifier); err == nil {
		return findUserByNumber(num, users)
	}

	// Try email matching (exact, then partial)
	emailMatches := findUsersByEmail(identifier, users)
	if len(emailMatches) == 1 {
		return emailMatches[0], nil
	} else if len(emailMatches) > 1 {
		return nil, formatMultipleMatchesError(identifier, emailMatches)
	}

	// Try name matching (exact, then partial)
	nameMatches := findUsersByName(identifier, users)
	if len(nameMatches) == 1 {
		return nameMatches[0], nil
	} else if len(nameMatches) > 1 {
		return nil, formatMultipleMatchesError(identifier, nameMatches)
	}

	// No matches found
	return nil, fmt.Errorf("user '%s' not found. Run 'kira users' to see available users", identifier)
}

// findUserByNumber looks up a user by their numeric identifier (1-based).
// Returns an error with available range if not found.
func findUserByNumber(number int, users []UserInfo) (*UserInfo, error) {
	if len(users) == 0 {
		return nil, fmt.Errorf("no users available")
	}

	if number < 1 || number > len(users) {
		return nil, fmt.Errorf("user number %d not found. Available numbers: 1-%d", number, len(users))
	}

	// Users are numbered 1-based, so index is number-1
	return &users[number-1], nil
}

// findUsersByEmail finds users matching an email identifier (exact or partial).
// Case-insensitive matching is used.
// Returns all matches (empty slice if none).
func findUsersByEmail(email string, users []UserInfo) []*UserInfo {
	if email == "" {
		return nil
	}

	emailLower := strings.ToLower(email)
	var exactMatches []*UserInfo
	var partialMatches []*UserInfo

	for i := range users {
		userEmailLower := strings.ToLower(users[i].Email)

		// Try exact match first
		if userEmailLower == emailLower {
			exactMatches = append(exactMatches, &users[i])
			continue
		}

		// Try partial match
		// If identifier starts with "@", match domain
		if strings.HasPrefix(email, "@") {
			if strings.HasSuffix(userEmailLower, emailLower) {
				partialMatches = append(partialMatches, &users[i])
			}
		} else if strings.Contains(email, "@") {
			// If identifier contains "@" but doesn't start with it, match substring
			if strings.Contains(userEmailLower, emailLower) {
				partialMatches = append(partialMatches, &users[i])
			}
		} else {
			// If identifier doesn't contain "@", match as substring (e.g., "alice" matches "alice@example.com")
			if strings.Contains(userEmailLower, emailLower) {
				partialMatches = append(partialMatches, &users[i])
			}
		}
	}

	// Return exact matches if any, otherwise return partial matches
	if len(exactMatches) > 0 {
		return exactMatches
	}
	return partialMatches
}

// findUsersByName finds users matching a name identifier (exact or partial).
// Case-insensitive matching is used.
// Returns all matches (empty slice if none).
// Exact matches are returned separately from partial matches.
func findUsersByName(name string, users []UserInfo) []*UserInfo {
	if name == "" {
		return nil
	}

	nameLower := strings.ToLower(name)
	var exactMatches []*UserInfo
	var partialMatches []*UserInfo

	for i := range users {
		userNameLower := strings.ToLower(users[i].Name)

		// Skip users without names
		if userNameLower == "" {
			continue
		}

		// Try exact match first
		if userNameLower == nameLower {
			exactMatches = append(exactMatches, &users[i])
			continue
		}

		// Try partial match (substring) - but only if not an exact match
		if strings.Contains(userNameLower, nameLower) {
			partialMatches = append(partialMatches, &users[i])
		}
	}

	// Return exact matches if any, otherwise return partial matches
	if len(exactMatches) > 0 {
		return exactMatches
	}
	return partialMatches
}

// formatMultipleMatchesError formats an error message showing all matching users with their numbers.
// Used when multiple users match an identifier.
func formatMultipleMatchesError(identifier string, matches []*UserInfo) error {
	if len(matches) == 0 {
		return fmt.Errorf("no users match '%s'", identifier)
	}

	var lines []string
	lines = append(lines, fmt.Sprintf("multiple users match '%s':", identifier))
	for _, match := range matches {
		display := formatUserDisplay(*match)
		lines = append(lines, fmt.Sprintf("  %d. %s", match.Number, display))
	}
	lines = append(lines, "Use the numeric identifier to select a specific user.")

	// Use %s format to avoid linter warning about non-constant format string
	return fmt.Errorf("%s", strings.Join(lines, "\n"))
}

// Phase 4: Front Matter Parsing & Field Access

const yamlSeparator = "---"

// parseWorkItemFrontMatter reads a work item file and parses its YAML front matter.
// Returns the parsed front matter as a map, the body content as lines, and any error.
// The front matter is expected to be between the first pair of --- lines.
func parseWorkItemFrontMatter(filePath string, cfg *config.Config) (map[string]interface{}, []string, error) {
	content, err := safeReadFile(filePath, cfg)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to read work item file: %w", err)
	}

	// Extract YAML front matter between the first pair of --- lines
	lines := strings.Split(string(content), "\n")
	var yamlLines []string
	var bodyLines []string
	inYAML := false
	yamlEndFound := false

	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if i == 0 && trimmed == yamlSeparator {
			inYAML = true
			continue
		}
		if inYAML {
			if trimmed == yamlSeparator {
				yamlEndFound = true
				inYAML = false
				continue
			}
			yamlLines = append(yamlLines, line)
		} else if yamlEndFound {
			// After YAML ends, collect all remaining lines as body
			bodyLines = append(bodyLines, line)
		} else {
			// If no front matter delimiters found, treat all content as body
			bodyLines = append(bodyLines, line)
		}
	}

	// Parse YAML front matter
	frontMatter := make(map[string]interface{})
	if len(yamlLines) > 0 {
		// Preserve id as string from the raw line so YAML never interprets 017 as octal 15.
		idRaw := extractIDFromYAMLLines(yamlLines)
		if err := yaml.Unmarshal([]byte(strings.Join(yamlLines, "\n")), frontMatter); err != nil {
			return nil, nil, fmt.Errorf("failed to parse front matter: %w", err)
		}
		if idRaw != "" {
			frontMatter["id"] = idRaw
		}
	}

	return frontMatter, bodyLines, nil
}

// extractIDFromYAMLLines finds the "id:" line in raw YAML and returns the value as string (unchanged).
func extractIDFromYAMLLines(yamlLines []string) string {
	const idKey = "id:"
	for _, line := range yamlLines {
		trimmed := strings.TrimSpace(line)
		if !strings.HasPrefix(trimmed, idKey) {
			continue
		}
		val := strings.TrimSpace(trimmed[len(idKey):])
		if len(val) >= 2 && (val[0] == '"' && val[len(val)-1] == '"' || val[0] == '\'' && val[len(val)-1] == '\'') {
			val = val[1 : len(val)-1]
		}
		return val
	}
	return ""
}

// getFieldValue retrieves a field value from the front matter map.
// Returns the value and true if the field exists (even if empty), or nil and false if it doesn't exist.
func getFieldValue(frontMatter map[string]interface{}, fieldName string) (interface{}, bool) {
	if frontMatter == nil {
		return nil, false
	}

	value, exists := frontMatter[fieldName]
	if !exists {
		return nil, false
	}

	return value, true
}

// getFieldValueAsString retrieves a field value from the front matter and converts it to a string.
// Returns the string representation and true if the field exists, or empty string and false if it doesn't.
// Array values are joined with commas. Other types are converted using fmt.Sprintf.
func getFieldValueAsString(frontMatter map[string]interface{}, fieldName string) (string, bool) {
	value, exists := getFieldValue(frontMatter, fieldName)
	if !exists {
		return "", false
	}

	// Handle different value types
	switch v := value.(type) {
	case string:
		return v, true
	case []string:
		return strings.Join(v, ", "), true
	case []interface{}:
		// Convert []interface{} to []string for display
		var strValues []string
		for _, item := range v {
			strValues = append(strValues, fmt.Sprintf("%v", item))
		}
		return strings.Join(strValues, ", "), true
	case nil:
		return "", true // Field exists but is nil, return empty string
	default:
		// For other types (int, bool, etc.), convert to string
		return fmt.Sprintf("%v", v), true
	}
}

// Phase 5: Field Update Logic (Switch Mode)

// writeWorkItemFrontMatter writes the front matter and body back to a work item file.
// It preserves field order by writing hardcoded fields first, then sorted other fields.
func writeWorkItemFrontMatter(filePath string, frontMatter map[string]interface{}, bodyLines []string) error {
	var sb strings.Builder

	// Write YAML separator
	sb.WriteString(yamlSeparator)
	sb.WriteString("\n")

	// Define hardcoded fields in order
	hardcodedFields := []string{"id", "title", "status", "kind", "created"}
	hardcodedSet := make(map[string]bool)
	for _, field := range hardcodedFields {
		hardcodedSet[field] = true
	}

	// Write hardcoded fields first
	for _, field := range hardcodedFields {
		if value, exists := frontMatter[field]; exists {
			if err := writeYAMLFieldValue(&sb, field, value); err != nil {
				return fmt.Errorf("failed to write field '%s': %w", field, err)
			}
		}
	}

	// Collect and sort other fields
	var otherFields []string
	for key := range frontMatter {
		if !hardcodedSet[key] {
			otherFields = append(otherFields, key)
		}
	}
	sort.Strings(otherFields)

	// Write other fields in sorted order
	for _, key := range otherFields {
		value := frontMatter[key]
		if err := writeYAMLFieldValue(&sb, key, value); err != nil {
			return fmt.Errorf("failed to write field '%s': %w", key, err)
		}
	}

	// Write closing YAML separator
	sb.WriteString(yamlSeparator)
	sb.WriteString("\n")

	// Write body content
	if len(bodyLines) > 0 {
		bodyContent := strings.Join(bodyLines, "\n")
		sb.WriteString(bodyContent)
		// Ensure file ends with newline if body has content
		if !strings.HasSuffix(bodyContent, "\n") {
			sb.WriteString("\n")
		}
	}

	// Write to file with permissions 0o600
	if err := os.WriteFile(filePath, []byte(sb.String()), 0o600); err != nil {
		return fmt.Errorf("failed to write work item file: %w", err)
	}

	return nil
}

// writeYAMLFieldValue writes a single YAML field to a string builder.
func writeYAMLFieldValue(sb *strings.Builder, key string, value interface{}) error {
	switch v := value.(type) {
	case string:
		formatted := yamlFormatStringValue(v)
		fmt.Fprintf(sb, "%s: %s\n", key, formatted)
	case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64, float32, float64:
		fmt.Fprintf(sb, "%s: %v\n", key, v)
	case bool:
		fmt.Fprintf(sb, "%s: %v\n", key, v)
	case []interface{}:
		fmt.Fprintf(sb, "%s: [", key)
		for i, item := range v {
			if i > 0 {
				sb.WriteString(", ")
			}
			sb.WriteString(yamlFormatArrayItem(item))
		}
		sb.WriteString("]\n")
	case []string:
		fmt.Fprintf(sb, "%s: [", key)
		for i, item := range v {
			if i > 0 {
				sb.WriteString(", ")
			}
			sb.WriteString(yamlFormatStringValue(item))
		}
		sb.WriteString("]\n")
	case nil:
		fmt.Fprintf(sb, "%s: null\n", key)
	default:
		// For complex types, use YAML marshaling
		yamlData, err := yaml.Marshal(map[string]interface{}{key: value})
		if err != nil {
			return fmt.Errorf("failed to marshal field '%s': %w", key, err)
		}
		// Extract the line(s) for this field from the marshaled output
		yamlStr := strings.TrimSpace(string(yamlData))
		lines := strings.Split(yamlStr, "\n")
		for _, line := range lines {
			sb.WriteString(line)
			sb.WriteString("\n")
		}
	}
	return nil
}

// yamlFormatStringValue formats a string value for YAML output, adding quotes when necessary.
func yamlFormatStringValue(s string) string {
	if needsYAMLQuoting(s) {
		return yamlQuotedString(s)
	}
	return s
}

// needsYAMLQuoting returns true if the string must be double-quoted for valid YAML output.
func needsYAMLQuoting(s string) bool {
	if s == "" || strings.TrimSpace(s) != s {
		return true
	}
	const yamlSpecialChars = ":#[]{},\"'\\\n\r\t&*!|>%"
	return strings.ContainsAny(s, yamlSpecialChars)
}

// yamlQuotedString returns a double-quoted YAML scalar with proper escaping.
func yamlQuotedString(s string) string {
	var b strings.Builder
	b.WriteByte('"')
	for i := 0; i < len(s); i++ {
		switch c := s[i]; c {
		case '\\':
			b.WriteString(`\\`)
		case '"':
			b.WriteString(`\"`)
		case '\n':
			b.WriteString(`\n`)
		case '\r':
			b.WriteString(`\r`)
		case '\t':
			b.WriteString(`\t`)
		default:
			b.WriteByte(c)
		}
	}
	b.WriteByte('"')
	return b.String()
}

// yamlFormatArrayItem formats an array item for YAML output.
func yamlFormatArrayItem(item interface{}) string {
	switch v := item.(type) {
	case string:
		return yamlFormatStringValue(v)
	case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64, float32, float64, bool:
		return fmt.Sprintf("%v", v)
	default:
		return yamlFormatStringValue(fmt.Sprintf("%v", v))
	}
}

// updateFieldValue updates or creates a field in the front matter map (switch mode).
// Returns the previous value (if any) and whether the field existed.
func updateFieldValue(
	frontMatter map[string]interface{},
	fieldName string,
	value string,
) (previousValue interface{}, existed bool) {
	if frontMatter == nil {
		frontMatter = make(map[string]interface{})
	}

	previousValue, existed = frontMatter[fieldName]
	frontMatter[fieldName] = value
	return previousValue, existed
}

// updateTimestamp updates the 'updated' field in the front matter with the current timestamp.
// Creates the field if it doesn't exist.
func updateTimestamp(frontMatter map[string]interface{}) {
	if frontMatter == nil {
		frontMatter = make(map[string]interface{})
	}
	// Use ISO 8601 format with time in UTC, consistent with save.go
	frontMatter["updated"] = time.Now().UTC().Format("2006-01-02T15:04:05Z")
}

// updateWorkItemField updates a field in a work item's front matter (switch mode).
// It reads the file, updates the field, updates the timestamp, and writes the file back.
func updateWorkItemField(
	filePath string,
	fieldName string,
	userEmail string,
	cfg *config.Config,
) error {
	// Parse front matter and body
	frontMatter, bodyLines, err := parseWorkItemFrontMatter(filePath, cfg)
	if err != nil {
		return fmt.Errorf("failed to parse work item: %w", err)
	}

	// Update field value (switch mode - replaces existing)
	updateFieldValue(frontMatter, fieldName, userEmail)

	// Update timestamp
	updateTimestamp(frontMatter)

	// Write back to file
	if err := writeWorkItemFrontMatter(filePath, frontMatter, bodyLines); err != nil {
		return fmt.Errorf("failed to write work item: %w", err)
	}

	return nil
}

// Phase 6: Append Mode Logic

// appendToField appends a user email to a field in the front matter (append mode).
// It handles:
// - Missing fields: creates field with the new user
// - Empty string fields: sets to the new user
// - Single string values: converts to array and appends
// - Array values ([]string or []interface{}): appends if not duplicate
func appendToField(
	frontMatter map[string]interface{},
	fieldName string,
	userEmail string,
) {
	if frontMatter == nil {
		frontMatter = make(map[string]interface{})
	}

	currentValue, exists := frontMatter[fieldName]

	// If field doesn't exist, create it with the new user
	if !exists {
		frontMatter[fieldName] = userEmail
		return
	}

	// If field is empty string, set to the new user
	if str, ok := currentValue.(string); ok {
		if str == "" {
			frontMatter[fieldName] = userEmail
			return
		}
		// Convert single string to array
		frontMatter[fieldName] = []string{str, userEmail}
		return
	}

	// If field is []string, append if not duplicate
	if arr, ok := currentValue.([]string); ok {
		// Check for duplicates
		for _, existing := range arr {
			if existing == userEmail {
				return // Already exists, don't add duplicate
			}
		}
		frontMatter[fieldName] = append(arr, userEmail)
		return
	}

	// If field is []interface{}, convert to []string and append if not duplicate
	if arr, ok := currentValue.([]interface{}); ok {
		// Convert []interface{} to []string
		strArr := make([]string, 0, len(arr))
		for _, item := range arr {
			// Convert each item to string
			strItem := fmt.Sprintf("%v", item)
			strArr = append(strArr, strItem)
		}
		// Check for duplicates
		for _, existing := range strArr {
			if existing == userEmail {
				// Already exists, convert to []string but don't add duplicate
				frontMatter[fieldName] = strArr
				return
			}
		}
		// Append new user (no duplicate found)
		frontMatter[fieldName] = append(strArr, userEmail)
		return
	}

	// For other types, convert to string and create array
	// This handles edge cases like numeric or boolean values
	strValue := fmt.Sprintf("%v", currentValue)
	if strValue == userEmail {
		return // Already matches, don't create duplicate
	}
	frontMatter[fieldName] = []string{strValue, userEmail}
}

// Phase 7: Unassign Logic

// clearField removes a field from the front matter map.
// Returns true if the field existed before deletion, false otherwise.
func clearField(frontMatter map[string]interface{}, fieldName string) (existed bool) {
	if frontMatter == nil {
		return false
	}

	_, existed = frontMatter[fieldName]
	if existed {
		delete(frontMatter, fieldName)
	}
	return existed
}

// updateWorkItemFieldUnassign removes a field from a work item's front matter.
// It reads the file, removes the field, updates the timestamp, and writes the file back.
func updateWorkItemFieldUnassign(
	filePath string,
	fieldName string,
	cfg *config.Config,
) error {
	// Parse front matter and body
	frontMatter, bodyLines, err := parseWorkItemFrontMatter(filePath, cfg)
	if err != nil {
		return fmt.Errorf("failed to parse work item: %w", err)
	}

	// Remove field (unassign mode - deletes the field)
	clearField(frontMatter, fieldName)

	// Update timestamp (always update, even if field didn't exist)
	updateTimestamp(frontMatter)

	// Write back to file
	if err := writeWorkItemFrontMatter(filePath, frontMatter, bodyLines); err != nil {
		return fmt.Errorf("failed to write work item: %w", err)
	}

	return nil
}

// updateWorkItemFieldAppend updates a field in a work item's front matter (append mode).
// It reads the file, appends to the field, updates the timestamp, and writes the file back.
func updateWorkItemFieldAppend(
	filePath string,
	fieldName string,
	userEmail string,
	cfg *config.Config,
) error {
	// Parse front matter and body
	frontMatter, bodyLines, err := parseWorkItemFrontMatter(filePath, cfg)
	if err != nil {
		return fmt.Errorf("failed to parse work item: %w", err)
	}

	// Append to field value (append mode - adds to existing)
	appendToField(frontMatter, fieldName, userEmail)

	// Update timestamp
	updateTimestamp(frontMatter)

	// Write back to file
	if err := writeWorkItemFrontMatter(filePath, frontMatter, bodyLines); err != nil {
		return fmt.Errorf("failed to write work item: %w", err)
	}

	return nil
}

// Phase 9: Interactive Mode

// getCurrentAssignment retrieves the current assignment value for a work item field.
// Returns the formatted string for display (or empty string if not assigned).
func getCurrentAssignment(workItemPath, fieldName string, cfg *config.Config) (string, error) {
	frontMatter, _, err := parseWorkItemFrontMatter(workItemPath, cfg)
	if err != nil {
		return "", fmt.Errorf("failed to parse work item: %w", err)
	}

	value, exists := getFieldValueAsString(frontMatter, fieldName)
	if !exists || value == "" {
		return "", nil
	}

	return value, nil
}

// showInteractiveSelection displays users in a numbered list and prompts for selection.
// Returns the selected user number (0 for unassign, 1+ for users) or an error.
// The inputReader parameter allows for testing by providing a mock input source.
func showInteractiveSelection(users []UserInfo, currentAssignment, fieldName string, inputReader io.Reader) (int, error) {
	if len(users) == 0 {
		return 0, fmt.Errorf("no users available for selection")
	}

	// Display header
	fmt.Println("Available users:")
	fmt.Println(strings.Repeat("-", 50))

	// Show current assignment if exists
	if currentAssignment != "" {
		fmt.Printf("Current assignment (%s): %s\n\n", fieldName, currentAssignment)
	}

	// Display users (same format as kira users list format)
	for _, user := range users {
		display := formatUserDisplay(user)
		fmt.Printf("%d. %s\n", user.Number, display)
	}

	// Add unassign option
	fmt.Println("0. Unassign")
	fmt.Println()

	// Get selection with retry logic
	const maxRetries = 3
	reader := bufio.NewReader(inputReader)

	for attempt := 0; attempt < maxRetries; attempt++ {
		fmt.Print("Select user (number): ")
		input, err := reader.ReadString('\n')
		if err != nil {
			return 0, fmt.Errorf("failed to read input: %w", err)
		}

		input = strings.TrimSpace(input)
		selection, err := strconv.Atoi(input)
		if err != nil {
			fmt.Printf("Invalid input: please enter a number (0-%d)\n", len(users))
			continue
		}

		// Validate selection is within valid range
		if selection < 0 || selection > len(users) {
			fmt.Printf("Invalid selection: please enter a number between 0 and %d\n", len(users))
			continue
		}

		return selection, nil
	}

	return 0, fmt.Errorf("too many invalid input attempts")
}
