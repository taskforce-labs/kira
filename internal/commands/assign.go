// Package commands implements the CLI commands for the kira tool.
package commands

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

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

	// Phase 2: Resolve and validate work items exist.
	workItemPaths, err := resolveWorkItems(workItems)
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

	// Store resolved paths and user for use in later phases (not used yet in Phase 3).
	_ = workItemPaths
	_ = resolvedUser

	// Phase 3: command stops after user collection and resolution.
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

// resolveWorkItemPath resolves a work item identifier (ID or path) to an absolute file path.
// If identifier is a path, it validates and returns the absolute path.
// If identifier is an ID, it uses findWorkItemFile to locate the file.
func resolveWorkItemPath(identifier string) (string, error) {
	// If identifier is a path, validate and return it.
	if isWorkItemPath(identifier) {
		if err := validateWorkPath(identifier); err != nil {
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
	workItemPath, err := findWorkItemFile(identifier)
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
func validateWorkItemFile(path string) error {
	// Check if file exists.
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return fmt.Errorf("work item file does not exist: %s", path)
	} else if err != nil {
		return fmt.Errorf("failed to access work item file: %w", err)
	}

	// Try to read the file to ensure it's readable.
	// Use safeReadFile which validates the path is within .work.
	if _, err := safeReadFile(path); err != nil {
		return fmt.Errorf("failed to read work item file: %w", err)
	}

	return nil
}

// resolveWorkItems resolves multiple work item identifiers to file paths and validates them.
// Returns an error if any work item cannot be resolved or validated.
func resolveWorkItems(identifiers []string) ([]string, error) {
	if len(identifiers) == 0 {
		return nil, fmt.Errorf("no work items to resolve")
	}

	var resolvedPaths []string
	var errors []string

	for _, identifier := range identifiers {
		path, err := resolveWorkItemPath(identifier)
		if err != nil {
			errors = append(errors, fmt.Sprintf("  %s: %v", identifier, err))
			continue
		}

		if err := validateWorkItemFile(path); err != nil {
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
func parseWorkItemFrontMatter(filePath string) (map[string]interface{}, []string, error) {
	content, err := safeReadFile(filePath)
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
		if err := yaml.Unmarshal([]byte(strings.Join(yamlLines, "\n")), frontMatter); err != nil {
			return nil, nil, fmt.Errorf("failed to parse front matter: %w", err)
		}
	}

	return frontMatter, bodyLines, nil
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
