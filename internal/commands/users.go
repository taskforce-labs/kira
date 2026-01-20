// Package commands implements the CLI commands for the kira tool.
package commands

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"kira/internal/config"
)

var usersCmd = &cobra.Command{
	Use:   "users",
	Short: "List all users from git history and configuration",
	Long: `List email addresses of all users from the git history and associate them with
a number that can be used to assign work items to the correct user. These numbers
will be in order of first commit to the repo, so new users will have a higher
number than existing users.`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, _ []string) error {
		cfg, err := config.LoadConfig()
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}

		format, _ := cmd.Flags().GetString("format")
		limit, _ := cmd.Flags().GetInt("limit")
		limitChanged := cmd.Flags().Changed("limit")

		return listUsers(cfg, format, limit, limitChanged)
	},
}

func init() {
	usersCmd.Flags().StringP("format", "f", "table", "Output format: table, list, or json")
	usersCmd.Flags().IntP("limit", "l", 0, "Limit number of commits to process (0 = no limit)")
}

// UserInfo represents a user with their information.
type UserInfo struct {
	Email       string
	Name        string
	FirstCommit *time.Time // nil for saved users without git history
	Source      string     // "git" or "config"
	Order       int        // Original order in config for saved users (0-based)
	Number      int        // Assigned sequential number
}

func listUsers(cfg *config.Config, format string, limit int, limitChanged bool) error {
	if err := validateUsersArgs(format, limit); err != nil {
		return err
	}

	useGitHistory := getUseGitHistorySetting(cfg)
	commitLimit := getCommitLimit(limit, limitChanged, cfg)

	userMap, err := collectUsers(useGitHistory, commitLimit, cfg)
	if err != nil {
		return err
	}

	users := processAndSortUsers(userMap, useGitHistory)

	return displayUsers(users, format)
}

func extractGitUsers(limit int) ([]UserInfo, error) {
	if err := checkGitRepository(); err != nil {
		return nil, err
	}

	args := buildGitLogArgs(limit)
	output, err := runGitLogCommand(args)
	if err != nil {
		return nil, err
	}

	userMap, err := parseGitLogOutput(output)
	if err != nil {
		return nil, err
	}

	return convertUserMapToSlice(userMap), nil
}

func checkGitRepository() error {
	if _, err := os.Stat(".git"); os.IsNotExist(err) {
		return fmt.Errorf("not a git repository")
	}
	return nil
}

func buildGitLogArgs(limit int) []string {
	args := []string{"log", "--all", "--format=%ae|%an|%ai", "--reverse"}
	if limit > 0 {
		args = append(args, fmt.Sprintf("-%d", limit))
	}
	return args
}

func runGitLogCommand(args []string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	output, err := executeCommand(ctx, "git", args, "", false)
	if err != nil {
		errStr := err.Error()
		// Check if git is not installed
		if strings.Contains(errStr, "executable file not found") ||
			strings.Contains(errStr, "No such file or directory") ||
			strings.Contains(errStr, "command not found") {
			return "", fmt.Errorf("git is not installed or not in PATH")
		}
		// Check if repository is empty or has no commits
		if strings.Contains(errStr, "does not have any commits yet") ||
			strings.Contains(errStr, "fatal: your current branch") ||
			strings.Contains(errStr, "fatal: bad default revision") {
			return "", nil // Empty repository, not an error
		}
		return "", fmt.Errorf("failed to execute git log: %w", err)
	}
	return output, nil
}

func parseGitLogOutput(output string) (map[string]*UserInfo, error) {
	userMap := make(map[string]*UserInfo) // key: email (lowercase)

	lines := strings.Split(strings.TrimSpace(output), "\n")
	for _, line := range lines {
		user, err := parseGitLogLine(line)
		if err != nil {
			continue // Skip malformed lines
		}
		if user == nil {
			continue // Skip invalid entries
		}

		emailLower := strings.ToLower(user.Email)

		// Check if we've seen this email before
		if existing, exists := userMap[emailLower]; exists {
			// Update if this commit is earlier
			if user.FirstCommit != nil && user.FirstCommit.Before(*existing.FirstCommit) {
				existing.FirstCommit = user.FirstCommit
				if user.Name != "" {
					existing.Name = user.Name
				}
			}
		} else {
			// New user
			userMap[emailLower] = user
		}
	}

	return userMap, nil
}

func parseGitLogLine(line string) (*UserInfo, error) {
	line = strings.TrimSpace(line)
	if line == "" {
		return nil, nil
	}

	parts := strings.Split(line, "|")
	if len(parts) != 3 {
		return nil, fmt.Errorf("malformed line: %s", line)
	}

	email := strings.TrimSpace(parts[0])
	name := strings.TrimSpace(parts[1])
	dateStr := strings.TrimSpace(parts[2])

	// Skip entries without email (missing author info)
	if email == "" {
		return nil, nil
	}

	// Parse commit date
	commitDate, err := parseGitDate(dateStr)
	if err != nil {
		return nil, err
	}

	return &UserInfo{
		Email:       email,
		Name:        name,
		FirstCommit: commitDate,
		Source:      "git",
	}, nil
}

func parseGitDate(dateStr string) (*time.Time, error) {
	// Git %ai format: "2006-01-02 15:04:05 -0700" or "2006-01-02 15:04:05 +0700"
	// Try with space before timezone first
	commitDate, err := time.Parse("2006-01-02 15:04:05 -0700", dateStr)
	if err != nil {
		// Try with + sign
		commitDate, err = time.Parse("2006-01-02 15:04:05 +0700", dateStr)
		if err != nil {
			// Try RFC3339 format
			commitDate, err = time.Parse(time.RFC3339, dateStr)
			if err != nil {
				return nil, err
			}
		}
	}

	return &commitDate, nil
}

func convertUserMapToSlice(userMap map[string]*UserInfo) []UserInfo {
	users := make([]UserInfo, 0, len(userMap))
	for _, user := range userMap {
		users = append(users, *user)
	}
	return users
}

func displayUsers(users []UserInfo, format string) error {
	if len(users) == 0 {
		fmt.Println("No users found.")
		return nil
	}

	switch format {
	case "list":
		return displayUsersList(users)
	case "json":
		return displayUsersJSON(users)
	case "table":
		fallthrough
	default:
		return displayUsersTable(users)
	}
}

func displayUsersList(users []UserInfo) error {
	for _, user := range users {
		display := formatUserDisplay(user)
		fmt.Printf("%d. %s\n", user.Number, display)
	}
	return nil
}

func displayUsersTable(users []UserInfo) error {
	if len(users) == 0 {
		return nil
	}

	// Calculate column widths
	maxNumberLen := len(fmt.Sprintf("%d", users[len(users)-1].Number))
	maxUserLen := 0
	maxDateLen := 10  // "YYYY-MM-DD"
	maxSourceLen := 6 // "config"

	for _, user := range users {
		display := formatUserDisplay(user)
		if len(display) > maxUserLen {
			maxUserLen = len(display)
		}
	}

	// Set minimum widths
	if maxNumberLen < 6 {
		maxNumberLen = 6
	}
	if maxUserLen < 4 {
		maxUserLen = 4
	}
	if maxUserLen > 60 {
		maxUserLen = 60 // Cap at 60 for readability
	}

	// Print header
	headerFormat := fmt.Sprintf("%%-%ds %%-%ds %%-%ds %%-%ds\n", maxNumberLen, maxUserLen, maxDateLen, maxSourceLen)
	fmt.Printf(headerFormat, "Number", "User", "First Commit", "Source")
	fmt.Println(strings.Repeat("-", maxNumberLen+maxUserLen+maxDateLen+maxSourceLen+3))

	// Print rows
	rowFormat := fmt.Sprintf("%%-%dd %%-%ds %%-%ds %%-%ds\n", maxNumberLen, maxUserLen, maxDateLen, maxSourceLen)
	for _, user := range users {
		display := formatUserDisplay(user)
		dateStr := ""
		if user.FirstCommit != nil {
			dateStr = user.FirstCommit.Format("2006-01-02")
		}
		if len(display) > maxUserLen {
			display = display[:maxUserLen-3] + "..."
		}
		fmt.Printf(rowFormat, user.Number, display, dateStr, user.Source)
	}
	return nil
}

func displayUsersJSON(users []UserInfo) error {
	type jsonUser struct {
		Number      int     `json:"number"`
		Email       string  `json:"email"`
		Name        string  `json:"name"`
		FirstCommit *string `json:"first_commit"` // ISO 8601 format or null
		Source      string  `json:"source"`
		Display     string  `json:"display"` // Formatted "Name <email>" or email
	}

	jsonUsers := make([]jsonUser, len(users))
	for i, user := range users {
		var firstCommitStr *string
		if user.FirstCommit != nil {
			formatted := user.FirstCommit.Format(time.RFC3339)
			firstCommitStr = &formatted
		}

		jsonUsers[i] = jsonUser{
			Number:      user.Number,
			Email:       user.Email,
			Name:        user.Name,
			FirstCommit: firstCommitStr,
			Source:      user.Source,
			Display:     formatUserDisplay(user),
		}
	}

	output := map[string]interface{}{
		"users": jsonUsers,
	}

	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(output)
}

func formatUserDisplay(user UserInfo) string {
	if user.Name != "" {
		return fmt.Sprintf("%s <%s>", user.Name, user.Email)
	}
	return user.Email
}

func validateUsersArgs(format string, limit int) error {
	// Validate format
	if format != "table" && format != "list" && format != "json" {
		return fmt.Errorf("invalid format: %s (must be table, list, or json)", format)
	}

	// Validate limit
	if limit < 0 {
		return fmt.Errorf("invalid limit: %d (must be >= 0)", limit)
	}

	return nil
}

func getUseGitHistorySetting(cfg *config.Config) bool {
	useGitHistory := true
	if cfg.Users.UseGitHistory != nil {
		useGitHistory = *cfg.Users.UseGitHistory
	}
	return useGitHistory
}

func getCommitLimit(limit int, limitChanged bool, cfg *config.Config) int {
	// If limit was explicitly set to 0, it means "no limit" - override config
	if limitChanged && limit == 0 {
		return 0
	}
	// If limit was not changed (default 0) and config has a limit, use config
	if !limitChanged && cfg.Users.CommitLimit > 0 {
		return cfg.Users.CommitLimit
	}
	// Otherwise use the provided limit value
	return limit
}

func collectUsers(useGitHistory bool, commitLimit int, cfg *config.Config) (map[string]*UserInfo, error) {
	userMap := make(map[string]*UserInfo) // key: email (lowercase) for deduplication

	if useGitHistory {
		// Extract users from git history
		gitUsers, err := extractGitUsers(commitLimit)
		if err != nil {
			return nil, err
		}

		// Add git users to map (after filtering)
		for i := range gitUsers {
			emailLower := strings.ToLower(gitUsers[i].Email)
			// Check if should be ignored
			if shouldIgnoreEmail(gitUsers[i].Email, cfg) {
				continue
			}
			userMap[emailLower] = &gitUsers[i]
		}
	}

	// Add saved users (with duplicate detection)
	for i, savedUser := range cfg.Users.SavedUsers {
		// Skip saved users with empty email (invalid config)
		if savedUser.Email == "" {
			continue
		}

		emailLower := strings.ToLower(savedUser.Email)
		if existing, exists := userMap[emailLower]; exists {
			// Duplicate: merge information
			// Saved user name takes precedence if provided
			if savedUser.Name != "" {
				existing.Name = savedUser.Name
			}
			// Keep git history commit date and source
			// For duplicates, we keep the existing order (from git history if present)
		} else {
			// New user from config
			userMap[emailLower] = &UserInfo{
				Email:       savedUser.Email,
				Name:        savedUser.Name,
				FirstCommit: nil, // No git history
				Source:      "config",
				Order:       i, // Track original config order
			}
		}
	}

	return userMap, nil
}

func processAndSortUsers(userMap map[string]*UserInfo, useGitHistory bool) []UserInfo {
	// Convert map to slice
	users := make([]UserInfo, 0, len(userMap))
	for _, user := range userMap {
		users = append(users, *user)
	}

	// Sort users
	sort.Slice(users, func(i, j int) bool {
		// When git history is disabled, sort by config order for saved users
		if !useGitHistory {
			return users[i].Order < users[j].Order
		}

		// When git history is enabled, sort by commit date, then by email
		if users[i].FirstCommit == nil && users[j].FirstCommit == nil {
			return users[i].Email < users[j].Email
		}
		if users[i].FirstCommit == nil {
			return false // nil dates go last
		}
		if users[j].FirstCommit == nil {
			return true
		}
		if users[i].FirstCommit.Equal(*users[j].FirstCommit) {
			return users[i].Email < users[j].Email
		}
		return users[i].FirstCommit.Before(*users[j].FirstCommit)
	})

	// Assign numbers
	for i := range users {
		users[i].Number = i + 1
	}

	return users
}

// shouldIgnoreEmail checks if an email should be ignored based on config.
// Only applies when use_git_history is true.
func shouldIgnoreEmail(email string, cfg *config.Config) bool {
	emailLower := strings.ToLower(email)

	// Check exact email matches
	for _, ignored := range cfg.Users.IgnoredEmails {
		if strings.ToLower(ignored) == emailLower {
			return true
		}
	}

	// Check pattern matches
	for _, pattern := range cfg.Users.IgnoredPatterns {
		patternLower := strings.ToLower(pattern)
		matched, err := filepath.Match(patternLower, emailLower)
		if err == nil && matched {
			return true
		}
	}

	return false
}
