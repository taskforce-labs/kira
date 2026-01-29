// Package commands implements the CLI commands for the kira tool.
package commands

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"kira/internal/config"
	"kira/internal/validation"
)

var saveCmd = &cobra.Command{
	Use:   "save [commit-message]",
	Short: "Update work items and commit changes to git",
	Long: `Updates the updated field in work items and commits changes to git.
Validates all non-archived work items before staging.`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.LoadConfig()
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}
		if err := checkWorkDir(cfg); err != nil {
			return err
		}

		var commitMessage string
		if len(args) > 0 {
			commitMessage = args[0]
		}

		dryRunFlag, _ := cmd.Flags().GetBool("dry-run")
		return saveWorkItems(cfg, commitMessage, dryRunFlag)
	},
}

func init() {
	saveCmd.Flags().Bool("dry-run", false, "Show what would be done without making changes")
}

func saveWorkItems(cfg *config.Config, commitMessage string, dryRun bool) error {
	// Validate all work items first
	result, err := validation.ValidateWorkItems(cfg)
	if err != nil {
		return fmt.Errorf("failed to validate work items: %w", err)
	}

	if result.HasErrors() {
		fmt.Println("Validation errors found:")
		for _, err := range result.Errors {
			fmt.Printf("  %s\n", err.Error())
		}
		return fmt.Errorf("validation failed - fix errors before saving")
	}

	if dryRun {
		fmt.Println("[DRY RUN] Would perform the following operations:")
		fmt.Println("[DRY RUN] Update timestamps for modified work items")
	} else {
		// Update timestamps for modified work items
		if err := updateWorkItemTimestamps(cfg); err != nil {
			return fmt.Errorf("failed to update timestamps: %w", err)
		}
	}

	// Check for external changes (always runs even in dry-run for validation)
	hasExternalChanges, err := checkExternalChanges(cfg)
	if err != nil {
		return fmt.Errorf("failed to check for external changes: %w", err)
	}

	if hasExternalChanges {
		workFolder := config.GetWorkFolderPath(cfg)
		fmt.Printf("Warning: External changes detected outside %s/ directory.\n", workFolder)
		fmt.Println("Skipping commit to avoid mixing work item changes with other changes.")
		return nil
	}

	// Stage only work folder changes
	if err := stageWorkChanges(cfg, dryRun); err != nil {
		return fmt.Errorf("failed to stage work changes: %w", err)
	}

	// Commit changes
	if commitMessage == "" {
		commitMessage = cfg.Commit.DefaultMessage
	}

	if err := commitChanges(commitMessage, dryRun); err != nil {
		return fmt.Errorf("failed to commit changes: %w", err)
	}

	if dryRun {
		fmt.Println("[DRY RUN] Would save and commit work items successfully.")
	} else {
		fmt.Println("Work items saved and committed successfully.")
	}
	return nil
}

func updateWorkItemTimestamps(cfg *config.Config) error {
	currentTime := time.Now().Format("2006-01-02T15:04:05Z")
	workFolder := config.GetWorkFolderPath(cfg)

	return filepath.Walk(workFolder, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		// Skip template files, IDEAS.md, and archived items
		if strings.Contains(path, "template") ||
			strings.HasSuffix(path, "IDEAS.md") ||
			strings.Contains(path, "z_archive") {
			return nil
		}

		// Only process markdown files
		if !strings.HasSuffix(path, ".md") {
			return nil
		}

		// Update the updated timestamp
		return updateFileTimestamp(path, currentTime, cfg)
	})
}

func updateFileTimestamp(filePath, timestamp string, cfg *config.Config) error {
	content, err := safeReadFile(filePath, cfg)
	if err != nil {
		return err
	}

	lines := strings.Split(string(content), "\n")
	updated := false

	for i, line := range lines {
		if strings.HasPrefix(strings.TrimSpace(line), "updated:") {
			lines[i] = fmt.Sprintf("updated: %s", timestamp)
			updated = true
			break
		}
	}

	// If no updated field found, add it after the created field
	if !updated {
		for i, line := range lines {
			if strings.HasPrefix(strings.TrimSpace(line), "created:") {
				// Insert updated field after created field
				newLines := make([]string, len(lines)+1)
				copy(newLines, lines[:i+1])
				newLines[i+1] = fmt.Sprintf("updated: %s", timestamp)
				copy(newLines[i+2:], lines[i+1:])
				lines = newLines
				break
			}
		}
	}

	return os.WriteFile(filePath, []byte(strings.Join(lines, "\n")), 0o600)
}

// sanitizeCommitMessage validates and sanitizes a commit message
func sanitizeCommitMessage(msg string) (string, error) {
	// Remove newlines and other dangerous characters
	msg = strings.ReplaceAll(msg, "\n", " ")
	msg = strings.ReplaceAll(msg, "\r", "")
	msg = strings.TrimSpace(msg)

	// Validate length
	if len(msg) == 0 {
		return "", fmt.Errorf("commit message cannot be empty")
	}
	if len(msg) > 1000 {
		return "", fmt.Errorf("commit message too long (max 1000 characters)")
	}

	// Check for shell metacharacters that could be dangerous
	// Note: parentheses () and angle brackets <> are allowed as they're safe in commit messages
	// and needed for status transitions (e.g., "todo -> doing")
	dangerous := []string{"`", "$", "{", "}", "[", "]", "|", "&", ";"}
	for _, char := range dangerous {
		if strings.Contains(msg, char) {
			return "", fmt.Errorf("commit message contains invalid character: %s", char)
		}
	}

	return msg, nil
}

func checkExternalChanges(cfg *config.Config) (bool, error) {
	// Check git status for changes outside work folder
	// Note: This function always executes git commands even in dry-run mode because it's a read-only check
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	output, err := executeCommand(ctx, "git", []string{"status", "--porcelain"}, "", false)
	if err != nil {
		// If git is not available or not a git repo, assume no external changes
		return false, nil
	}

	workFolder := config.GetWorkFolderPath(cfg)
	workPrefix := workFolder + "/"
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		if line != "" && !strings.HasPrefix(line, " ") && !strings.HasPrefix(line, "??") {
			parts := strings.Fields(line)
			if len(parts) > 1 {
				filePath := parts[1]
				if !strings.HasPrefix(filePath, workPrefix) && filePath != workFolder {
					return true, nil
				}
			}
		}
	}

	return false, nil
}

func stageWorkChanges(cfg *config.Config, dryRun bool) error {
	// Stage all changes in work folder
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	workFolder := config.GetWorkFolderPath(cfg)
	addPath := workFolder + "/"
	_, err := executeCommand(ctx, "git", []string{"add", addPath}, "", dryRun)
	return err
}

func commitChanges(message string, dryRun bool) error {
	// Sanitize commit message
	sanitized, err := sanitizeCommitMessage(message)
	if err != nil {
		return fmt.Errorf("invalid commit message: %w", err)
	}

	if dryRun {
		fmt.Printf("[DRY RUN] git commit -m %q\n", sanitized)
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	_, err = executeCommand(ctx, "git", []string{"commit", "-m", sanitized}, "", false)
	return err
}
