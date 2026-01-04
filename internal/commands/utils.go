package commands

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// validateWorkPath ensures a path is safe and within the .work directory
func validateWorkPath(path string) error {
	// Clean the path to remove .. and other traversal attempts
	cleanPath := filepath.Clean(path)

	// Resolve to absolute path
	absPath, err := filepath.Abs(cleanPath)
	if err != nil {
		return fmt.Errorf("invalid path: %w", err)
	}

	// Get absolute path of .work directory
	workDir, err := filepath.Abs(".work")
	if err != nil {
		return fmt.Errorf("failed to resolve .work directory: %w", err)
	}

	// Ensure the path is within .work directory
	// Use separator to prevent partial matches (e.g., .work-backup)
	workDirWithSep := workDir + string(filepath.Separator)
	if !strings.HasPrefix(absPath+string(filepath.Separator), workDirWithSep) && absPath != workDir {
		return fmt.Errorf("path outside .work directory: %s", path)
	}

	return nil
}

// safeReadFile reads a file after validating the path is within .work/
func safeReadFile(filePath string) ([]byte, error) {
	if err := validateWorkPath(filePath); err != nil {
		return nil, err
	}
	// #nosec G304 - path has been validated by validateWorkPath above
	return os.ReadFile(filePath)
}

// safeReadProjectFile reads a file from project root (like RELEASES.md, kira.yml)
// It validates the file is in the current directory and doesn't contain path traversal
func safeReadProjectFile(filePath string) ([]byte, error) {
	// Clean the path to remove .. and other traversal attempts
	cleanPath := filepath.Clean(filePath)

	// Ensure it's a simple filename or relative path without traversal
	if strings.Contains(cleanPath, "..") {
		return nil, fmt.Errorf("path contains traversal: %s", filePath)
	}

	// Resolve to absolute path
	absPath, err := filepath.Abs(cleanPath)
	if err != nil {
		return nil, fmt.Errorf("invalid path: %w", err)
	}

	// Get absolute path of current directory
	currentDir, err := filepath.Abs(".")
	if err != nil {
		return nil, fmt.Errorf("failed to resolve current directory: %w", err)
	}

	// Ensure the path is within current directory
	currentDirWithSep := currentDir + string(filepath.Separator)
	if !strings.HasPrefix(absPath+string(filepath.Separator), currentDirWithSep) && absPath != currentDir {
		return nil, fmt.Errorf("path outside project directory: %s", filePath)
	}

	// #nosec G304 - path has been validated above
	return os.ReadFile(filePath)
}

// findWorkItemFile searches for a work item file by ID
func findWorkItemFile(workItemID string) (string, error) {
	var foundPath string

	err := filepath.Walk(".work", func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		// Check if this is a work item file with the matching ID
		if strings.HasSuffix(path, ".md") && !strings.Contains(path, "template") && !strings.HasSuffix(path, "IDEAS.md") {
			// Read the file to check the ID
			content, err := safeReadFile(path)
			if err != nil {
				return err
			}

			// Simple check for ID in front matter
			if strings.Contains(string(content), fmt.Sprintf("id: %s", workItemID)) {
				foundPath = path
				return filepath.SkipDir
			}
		}

		return nil
	})
	if err != nil {
		return "", fmt.Errorf("failed to search for work item: %w", err)
	}

	if foundPath == "" {
		return "", fmt.Errorf("work item with ID %s not found", workItemID)
	}

	return foundPath, nil
}

// updateWorkItemStatus updates the status field in a work item file
func updateWorkItemStatus(filePath, newStatus string) error {
	content, err := safeReadFile(filePath)
	if err != nil {
		return err
	}

	lines := strings.Split(string(content), "\n")
	for i, line := range lines {
		if strings.HasPrefix(strings.TrimSpace(line), "status:") {
			lines[i] = fmt.Sprintf("status: %s", newStatus)
			break
		}
	}

	return os.WriteFile(filePath, []byte(strings.Join(lines, "\n")), 0o600)
}

// getWorkItemFiles returns all work item files in a directory
func getWorkItemFiles(sourcePath string) ([]string, error) {
	var files []string

	err := filepath.Walk(sourcePath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		if strings.HasSuffix(path, ".md") && !strings.Contains(path, "template") {
			files = append(files, path)
		}

		return nil
	})

	return files, err
}

// archiveWorkItems archives work items to the archive directory
func archiveWorkItems(workItems []string, sourcePath string) (string, error) {
	// Create archive directory
	date := time.Now().Format("2006-01-02")
	archiveDir := filepath.Join(".work", "z_archive", date, filepath.Base(sourcePath))

	if err := os.MkdirAll(archiveDir, 0o700); err != nil {
		return "", fmt.Errorf("failed to create archive directory: %w", err)
	}

	// Copy work items to archive
	for _, workItem := range workItems {
		filename := filepath.Base(workItem)
		archivePath := filepath.Join(archiveDir, filename)

		content, err := safeReadFile(workItem)
		if err != nil {
			return "", fmt.Errorf("failed to read work item: %w", err)
		}

		if err := os.WriteFile(archivePath, content, 0o600); err != nil {
			return "", fmt.Errorf("failed to write to archive: %w", err)
		}
	}

	return archiveDir, nil
}

// formatCommandPreview formats a command for dry-run output
func formatCommandPreview(name string, args []string) string {
	if len(args) == 0 {
		return fmt.Sprintf("[DRY RUN] %s", name)
	}
	return fmt.Sprintf("[DRY RUN] %s %s", name, strings.Join(args, " "))
}

// executeCommand executes a command with context and optional dry-run support.
// If dryRun is true, it prints what would be executed and returns empty string and nil.
// If dryRun is false, it executes the command and returns stdout output.
// If dir is non-empty, the command is executed in that directory.
// Errors include stderr output for debugging.
func executeCommand(ctx context.Context, name string, args []string, dir string, dryRun bool) (string, error) {
	if dryRun {
		preview := formatCommandPreview(name, args)
		if dir != "" {
			preview = fmt.Sprintf("%s (in %s)", preview, dir)
		}
		fmt.Println(preview)
		return "", nil
	}

	cmd := exec.CommandContext(ctx, name, args...)
	if dir != "" {
		cmd.Dir = dir
	}

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		stderrStr := strings.TrimSpace(stderr.String())
		if stderrStr != "" {
			return "", fmt.Errorf("%w: %s", err, stderrStr)
		}
		return "", err
	}

	return stdout.String(), nil
}

// executeCommandCombinedOutput executes a command and returns combined stdout+stderr.
// This is useful for commands where you need to see all output regardless of success/failure.
// If dryRun is true, it prints what would be executed and returns empty string and nil.
func executeCommandCombinedOutput(ctx context.Context, name string, args []string, dir string, dryRun bool) (string, error) {
	if dryRun {
		preview := formatCommandPreview(name, args)
		if dir != "" {
			preview = fmt.Sprintf("%s (in %s)", preview, dir)
		}
		fmt.Println(preview)
		return "", nil
	}

	cmd := exec.CommandContext(ctx, name, args...)
	if dir != "" {
		cmd.Dir = dir
	}

	output, err := cmd.CombinedOutput()
	if err != nil {
		outputStr := strings.TrimSpace(string(output))
		if outputStr != "" {
			return "", fmt.Errorf("%w: %s", err, outputStr)
		}
		return "", err
	}

	return string(output), nil
}
