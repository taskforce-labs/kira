// Package commands implements the CLI commands for the kira tool.
package commands

import (
	"bufio"
	"context"
	"fmt"
	"os"
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
		cfg, err := config.LoadConfig()
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}
		if err := checkWorkDir(cfg); err != nil {
			return err
		}

		workItemID := args[0]
		var targetStatus string
		if len(args) > 1 {
			targetStatus = args[1]
		}

		commitFlag, _ := cmd.Flags().GetBool("commit")
		dryRunFlag, _ := cmd.Flags().GetBool("dry-run")
		return moveWorkItem(cfg, workItemID, targetStatus, commitFlag, dryRunFlag)
	},
}

func init() {
	moveCmd.Flags().BoolP("commit", "c", false, "Commit the move to git")
	moveCmd.Flags().Bool("dry-run", false, "Show what would be done without making changes")
}

const unknownValue = "unknown"

// extractWorkItemMetadata extracts work item metadata from front matter
func extractWorkItemMetadata(filePath string, cfg *config.Config) (workItemType, id, title, currentStatus string, repos []string, err error) {
	content, err := safeReadFile(filePath, cfg)
	if err != nil {
		return unknownValue, "", "", unknownValue, nil, fmt.Errorf("failed to read work item file: %w", err)
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
		Kind   string   `yaml:"kind"`
		ID     string   `yaml:"id"`
		Title  string   `yaml:"title"`
		Status string   `yaml:"status"`
		Repos  []string `yaml:"repos"`
	}

	fields := &workItemFields{}
	if len(yamlLines) > 0 {
		if err := yaml.Unmarshal([]byte(strings.Join(yamlLines, "\n")), fields); err != nil {
			return unknownValue, "", "", unknownValue, nil, fmt.Errorf("failed to parse front matter: %w", err)
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
	repos = fields.Repos // nil if absent or empty

	return workItemType, id, title, currentStatus, repos, nil
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
// Note: This function always executes git commands even in dry-run mode because it's a read-only check
func checkStagedChanges(excludePaths []string) (bool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	output, err := executeCommand(ctx, "git", []string{"diff", "--cached", "--name-only"}, "", false)
	if err != nil {
		// If git is not available or not a git repo, return error
		return false, fmt.Errorf("git is not available or not a git repository")
	}

	lines := strings.Split(output, "\n")
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

// validateStagedChanges checks for staged changes and returns an error if conditions aren't met
func validateStagedChanges(excludePaths []string) error {
	hasOtherStaged, err := checkStagedChanges(excludePaths)
	if err != nil {
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
	return nil
}

// verifyDeletionStaged checks if the oldPath deletion is staged
// Git may show this as a deletion (D) or as a rename (R) when both old and new paths are staged
func verifyDeletionStaged(ctx context.Context, oldPath, dir string, dryRun bool) (bool, error) {
	if dryRun {
		return true, nil // Skip verification in dry-run mode
	}
	output, err := executeCommand(ctx, "git", []string{"diff", "--cached", "--name-status"}, dir, false)
	if err != nil {
		return false, fmt.Errorf("failed to check staged changes: %w", err)
	}
	// Convert oldPath to relative path if dir is provided (Git outputs relative paths)
	checkPath := oldPath
	if dir != "" {
		relPath, err := filepath.Rel(dir, oldPath)
		if err == nil {
			checkPath = relPath
		}
		// Also try with forward slashes (Git uses forward slashes even on Windows)
		checkPath = filepath.ToSlash(checkPath)
	}
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		// Check if this line contains the oldPath
		// Format: "D\tpath/to/file" (deletion) or "R100\told/path\tnew/path" (rename)
		parts := strings.Fields(line)
		if len(parts) >= 2 {
			status := parts[0]
			// Normalize path in output to use forward slashes
			pathInOutput := filepath.ToSlash(parts[1])
			// Check for deletion
			if status == "D" && pathInOutput == checkPath {
				return true, nil
			}
			// Check for rename (R followed by similarity percentage)
			if strings.HasPrefix(status, "R") && len(parts) >= 3 && pathInOutput == checkPath {
				return true, nil
			}
		}
	}
	return false, nil
}

// stageFileChanges stages the old file deletion and new file addition for a move
func stageFileChanges(ctx context.Context, oldPath, newPath string, dryRun bool) error {
	// Stage the old file deletion first using git rm --cached
	_, cmdErr := executeCommand(ctx, "git", []string{"rm", "--cached", oldPath}, "", dryRun)
	if cmdErr != nil && !dryRun {
		// If git rm fails (file wasn't tracked), try git add -u to stage deletions
		oldDir := filepath.Dir(oldPath)
		_, fallbackErr := executeCommand(ctx, "git", []string{"add", "-u", oldDir}, "", dryRun)
		if fallbackErr != nil {
			return fmt.Errorf("failed to stage deletion: git rm --cached failed (%v) and git add -u fallback also failed (%w)", cmdErr, fallbackErr)
		}
	}

	// Verify deletion was staged (skip in dry-run mode)
	if !dryRun {
		deletionStaged, err := verifyDeletionStaged(ctx, oldPath, "", dryRun)
		if err != nil {
			return fmt.Errorf("failed to verify deletion was staged: %w", err)
		}
		if !deletionStaged {
			return fmt.Errorf("deletion was not staged: file %s deletion not found in staged changes", oldPath)
		}
	}

	// Stage the new file addition
	_, cmdErr = executeCommand(ctx, "git", []string{"add", newPath}, "", dryRun)
	if cmdErr != nil && !dryRun {
		return fmt.Errorf("failed to stage changes: %w", cmdErr)
	}
	return nil
}

// commitMove stages the moved files and commits with sanitized message
func commitMove(oldPath, newPath, subject, body string, dryRun bool) error {
	// Skip staged changes check in dry-run mode (it requires a git repo)
	if !dryRun {
		if err := validateStagedChanges([]string{oldPath, newPath}); err != nil {
			return err
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := stageFileChanges(ctx, oldPath, newPath, dryRun); err != nil {
		return err
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

	var commitArgs []string
	if sanitizedBody != "" {
		commitArgs = []string{"commit", "-m", sanitizedSubject, "-m", sanitizedBody}
	} else {
		commitArgs = []string{"commit", "-m", sanitizedSubject}
	}

	_, err = executeCommandCombinedOutput(commitCtx, "git", commitArgs, "", dryRun)
	if err != nil && !dryRun {
		return fmt.Errorf("failed to commit: %w", err)
	}

	return nil
}

// moveWorkItemDryRun shows what would happen without making changes
func moveWorkItemDryRun(cfg *config.Config, workItemPath, targetPath, targetStatus string, commitFlag bool, metadata workItemMetadata) error {
	fmt.Println("[DRY RUN] Would perform the following operations:")
	fmt.Printf("[DRY RUN] Move file: %s -> %s\n", workItemPath, targetPath)
	fmt.Printf("[DRY RUN] Update status field: %s -> %s\n", metadata.currentStatus, targetStatus)

	if !commitFlag {
		return nil
	}

	subject, body, err := buildCommitMessage(cfg, metadata.workItemType, metadata.id, metadata.title, metadata.currentStatus, targetStatus)
	if err != nil {
		return fmt.Errorf("failed to build commit message: %w", err)
	}
	fmt.Printf("[DRY RUN] Commit message subject: %s\n", subject)
	if body != "" {
		fmt.Printf("[DRY RUN] Commit message body: %s\n", body)
	}
	// Show git commands that would be executed
	return commitMove(workItemPath, targetPath, subject, body, true)
}

// workItemMetadata holds extracted metadata from a work item file
type workItemMetadata struct {
	workItemType  string
	id            string
	title         string
	currentStatus string
	repos         []string // optional: work item repos override for polyrepo
}

func moveWorkItem(cfg *config.Config, workItemID, targetStatus string, commitFlag, dryRun bool) error {
	// Find the work item file
	workItemPath, err := findWorkItemFile(workItemID, cfg)
	if err != nil {
		return err
	}

	// Extract metadata BEFORE moving (to get current status)
	var metadata workItemMetadata
	if commitFlag || dryRun {
		metadata.workItemType, metadata.id, metadata.title, metadata.currentStatus, metadata.repos, err = extractWorkItemMetadata(workItemPath, cfg)
		if err != nil {
			return fmt.Errorf("failed to extract work item metadata: %w", err)
		}
	}

	// Get target status if not provided
	if targetStatus == "" {
		if dryRun {
			return fmt.Errorf("target status must be provided when using --dry-run")
		}
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
	targetFolder := filepath.Join(config.GetWorkFolderPath(cfg), cfg.StatusFolders[targetStatus])
	filename := filepath.Base(workItemPath)
	targetPath := filepath.Join(targetFolder, filename)

	if dryRun {
		return moveWorkItemDryRun(cfg, workItemPath, targetPath, targetStatus, commitFlag, metadata)
	}

	return executeMoveWorkItem(cfg, workItemID, workItemPath, targetPath, targetStatus, commitFlag, metadata)
}

// executeMoveWorkItem performs the actual move operation
func executeMoveWorkItem(cfg *config.Config, workItemID, workItemPath, targetPath, targetStatus string, commitFlag bool, metadata workItemMetadata) error {
	if err := os.Rename(workItemPath, targetPath); err != nil {
		return fmt.Errorf("failed to move work item: %w", err)
	}

	// Update the status in the file
	if err := updateWorkItemStatus(targetPath, targetStatus, cfg); err != nil {
		return fmt.Errorf("failed to update work item status: %w", err)
	}

	if !commitFlag {
		fmt.Printf("Moved work item %s to %s\n", workItemID, targetStatus)
		return nil
	}

	// Build and execute commit
	subject, body, err := buildCommitMessage(cfg, metadata.workItemType, metadata.id, metadata.title, metadata.currentStatus, targetStatus)
	if err != nil {
		fmt.Printf("Moved work item %s to %s\n", workItemID, targetStatus)
		return fmt.Errorf("failed to build commit message: %w", err)
	}

	if err := commitMove(workItemPath, targetPath, subject, body, false); err != nil {
		fmt.Printf("Moved work item %s to %s\n", workItemID, targetStatus)
		return fmt.Errorf("failed to commit move: %w", err)
	}

	fmt.Printf("Moved work item %s to %s and committed\n", workItemID, targetStatus)
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
