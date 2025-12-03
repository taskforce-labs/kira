// Package commands implements the CLI commands for the kira tool.
package commands

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"
	yaml "gopkg.in/yaml.v3"

	"kira/internal/config"
)

var moveCmd = &cobra.Command{
	Use:   "move <work-item-id> [target-status]",
	Short: "Move a work item to a different status folder",
	Long:  `Moves the work item to the target status folder. Will display options if target status not provided.`,
	Args:  cobra.RangeArgs(1, 2),
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := checkWorkDir(); err != nil {
			return err
		}

		cfg, err := config.LoadConfig()
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}

		workItemID := args[0]
		var targetStatus string
		if len(args) > 1 {
			targetStatus = args[1]
		}

		commitFlag, _ := cmd.Flags().GetBool("commit")
		return moveWorkItem(cfg, workItemID, targetStatus, commitFlag)
	},
}

func init() {
	moveCmd.Flags().BoolP("commit", "c", false, "Commit the move to git")
}

const unknownValue = "unknown"

// extractWorkItemMetadata extracts work item metadata from front matter
func extractWorkItemMetadata(filePath string) (workItemType, id, title, currentStatus string, err error) {
	content, err := safeReadFile(filePath)
	if err != nil {
		return unknownValue, "", "", unknownValue, fmt.Errorf("failed to read work item file: %w", err)
	}

	// Extract YAML front matter between the first pair of --- lines
	lines := strings.Split(string(content), "\n")
	var yamlLines []string
	inYAML := false
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if i == 0 && trimmed == "---" {
			inYAML = true
			continue
		}
		if inYAML {
			if trimmed == "---" {
				break
			}
			yamlLines = append(yamlLines, line)
		}
	}

	// Parse YAML to extract fields
	type workItemFields struct {
		Kind   string `yaml:"kind"`
		ID     string `yaml:"id"`
		Title  string `yaml:"title"`
		Status string `yaml:"status"`
	}

	fields := &workItemFields{}
	if len(yamlLines) > 0 {
		if err := yaml.Unmarshal([]byte(strings.Join(yamlLines, "\n")), fields); err != nil {
			return unknownValue, "", "", unknownValue, fmt.Errorf("failed to parse front matter: %w", err)
		}
	}

	// Use defaults for missing fields
	workItemType = fields.Kind
	if workItemType == "" {
		workItemType = unknownValue
	}
	id = fields.ID
	if id == "" {
		id = unknownValue
	}
	title = fields.Title
	if title == "" {
		title = unknownValue
	}
	currentStatus = fields.Status
	if currentStatus == "" {
		currentStatus = unknownValue
	}

	return workItemType, id, title, currentStatus, nil
}

// buildCommitMessage constructs commit message from templates with variable replacement
func buildCommitMessage(cfg *config.Config, workItemType, id, title, currentStatus, targetStatus string) (subject, body string, err error) {
	// Get templates from config, use defaults if empty
	subjectTemplate := cfg.Commit.MoveSubjectTemplate
	if subjectTemplate == "" {
		subjectTemplate = "Move {type} {id} to {target_status}"
	}

	bodyTemplate := cfg.Commit.MoveBodyTemplate
	if bodyTemplate == "" {
		bodyTemplate = "{title} ({current_status} -> {target_status})"
	}

	// Replace template variables in subject
	subject = subjectTemplate
	subject = strings.ReplaceAll(subject, "{type}", workItemType)
	subject = strings.ReplaceAll(subject, "{id}", id)
	subject = strings.ReplaceAll(subject, "{title}", title)
	subject = strings.ReplaceAll(subject, "{current_status}", currentStatus)
	subject = strings.ReplaceAll(subject, "{target_status}", targetStatus)

	// Replace template variables in body
	body = bodyTemplate
	body = strings.ReplaceAll(body, "{type}", workItemType)
	body = strings.ReplaceAll(body, "{id}", id)
	body = strings.ReplaceAll(body, "{title}", title)
	body = strings.ReplaceAll(body, "{current_status}", currentStatus)
	body = strings.ReplaceAll(body, "{target_status}", targetStatus)

	// Trim whitespace
	subject = strings.TrimSpace(subject)
	body = strings.TrimSpace(body)

	return subject, body, nil
}

// checkStagedChanges checks if there are any staged changes excluding specified paths
func checkStagedChanges(excludePaths []string) (bool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "git", "diff", "--cached", "--name-only")
	output, err := cmd.Output()
	if err != nil {
		// If git is not available or not a git repo, return error
		return false, fmt.Errorf("git is not available or not a git repository")
	}

	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Check if this file is in the exclude list
		excluded := false
		for _, excludePath := range excludePaths {
			if line == excludePath || strings.HasPrefix(line, excludePath+"/") {
				excluded = true
				break
			}
		}

		if !excluded {
			// Found a staged file that's not excluded
			return true, nil
		}
	}

	return false, nil
}

// commitMove stages the moved files and commits with sanitized message
func commitMove(oldPath, newPath, subject, body string) error {
	// Check for staged changes excluding moved file paths
	hasOtherStaged, err := checkStagedChanges([]string{oldPath, newPath})
	if err != nil {
		// Check if error is about git not being available
		if strings.Contains(err.Error(), "git is not available") {
			return fmt.Errorf("git is not available. Install git to use --commit flag")
		}
		if strings.Contains(err.Error(), "not a git repository") {
			return fmt.Errorf("not a git repository. Initialize git to use --commit flag")
		}
		return fmt.Errorf("failed to check staged changes: %w", err)
	}

	if hasOtherStaged {
		return fmt.Errorf("other files are already staged. Commit or unstage them before using --commit flag, or use 'kira save' to commit all changes together")
	}

	// Stage both old file deletion and new file addition
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Stage the old file deletion first using git rm --cached
	// This properly stages the deletion even if the file no longer exists
	cmd := exec.CommandContext(ctx, "git", "rm", "--cached", oldPath)
	if err := cmd.Run(); err != nil {
		// If git rm fails (file wasn't tracked), try git add -u to stage deletions
		oldDir := filepath.Dir(oldPath)
		// #nosec G204 - oldDir is derived from validated file path, safe to use
		cmd = exec.CommandContext(ctx, "git", "add", "-u", oldDir)
		_ = cmd.Run() // If that also fails, the file might not have been tracked - proceed anyway
	}

	// Stage the new file addition
	cmd = exec.CommandContext(ctx, "git", "add", newPath)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to stage changes: %w", err)
	}

	// Sanitize subject and body separately
	sanitizedSubject, err := sanitizeCommitMessage(subject)
	if err != nil {
		return fmt.Errorf("invalid commit message subject: %w", err)
	}

	var sanitizedBody string
	if body != "" {
		sanitizedBody, err = sanitizeCommitMessage(body)
		if err != nil {
			return fmt.Errorf("invalid commit message body: %w", err)
		}
	}

	// Commit with multi-line message
	commitCtx, commitCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer commitCancel()

	var commitCmd *exec.Cmd
	if sanitizedBody != "" {
		// #nosec G204 - sanitized messages have been validated and sanitized by sanitizeCommitMessage
		commitCmd = exec.CommandContext(commitCtx, "git", "commit", "-m", sanitizedSubject, "-m", sanitizedBody)
	} else {
		// #nosec G204 - sanitized message has been validated and sanitized by sanitizeCommitMessage
		commitCmd = exec.CommandContext(commitCtx, "git", "commit", "-m", sanitizedSubject)
	}

	output, err := commitCmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to commit: %s: %w", string(output), err)
	}

	return nil
}

func moveWorkItem(cfg *config.Config, workItemID, targetStatus string, commitFlag bool) error {
	// Find the work item file
	workItemPath, err := findWorkItemFile(workItemID)
	if err != nil {
		return err
	}

	// Extract metadata BEFORE moving (to get current status)
	var workItemType, id, title, currentStatus string
	if commitFlag {
		workItemType, id, title, currentStatus, err = extractWorkItemMetadata(workItemPath)
		if err != nil {
			return fmt.Errorf("failed to extract work item metadata: %w", err)
		}
	}

	// Get target status if not provided
	if targetStatus == "" {
		var err error
		targetStatus, err = selectTargetStatus(cfg)
		if err != nil {
			return err
		}
	}

	// Validate target status
	if _, exists := cfg.StatusFolders[targetStatus]; !exists {
		return fmt.Errorf("invalid target status: %s", targetStatus)
	}

	// Get target folder path
	targetFolder := filepath.Join(".work", cfg.StatusFolders[targetStatus])

	// Move the file
	filename := filepath.Base(workItemPath)
	targetPath := filepath.Join(targetFolder, filename)

	if err := os.Rename(workItemPath, targetPath); err != nil {
		return fmt.Errorf("failed to move work item: %w", err)
	}

	// Update the status in the file
	if err := updateWorkItemStatus(targetPath, targetStatus); err != nil {
		return fmt.Errorf("failed to update work item status: %w", err)
	}

	// Commit if flag is set
	if commitFlag {
		// Build commit message
		subject, body, err := buildCommitMessage(cfg, workItemType, id, title, currentStatus, targetStatus)
		if err != nil {
			fmt.Printf("Moved work item %s to %s\n", workItemID, targetStatus)
			return fmt.Errorf("failed to build commit message: %w", err)
		}

		// Commit the move
		// Note: workItemPath is the old path (before move), targetPath is the new path (after move)
		if err := commitMove(workItemPath, targetPath, subject, body); err != nil {
			fmt.Printf("Moved work item %s to %s\n", workItemID, targetStatus)
			return fmt.Errorf("failed to commit move: %w", err)
		}

		fmt.Printf("Moved work item %s to %s and committed\n", workItemID, targetStatus)
		return nil
	}

	fmt.Printf("Moved work item %s to %s\n", workItemID, targetStatus)
	return nil
}

func selectTargetStatus(cfg *config.Config) (string, error) {
	fmt.Println("Available statuses:")
	var statuses []string
	for status := range cfg.StatusFolders {
		statuses = append(statuses, status)
	}

	for i, status := range statuses {
		fmt.Printf("%d. %s\n", i+1, status)
	}

	fmt.Print("Select target status (number): ")
	reader := bufio.NewReader(os.Stdin)
	input, err := reader.ReadString('\n')
	if err != nil {
		return "", err
	}

	choice, err := strconv.Atoi(strings.TrimSpace(input))
	if err != nil || choice < 1 || choice > len(statuses) {
		return "", fmt.Errorf("invalid status selection")
	}

	return statuses[choice-1], nil
}
