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

	"kira/internal/config"
)

// gitCommandTimeout is the default timeout for git commands
const gitCommandTimeout = 30 * time.Second

// getCurrentBranch returns the current branch name
func getCurrentBranch(dir string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), gitCommandTimeout)
	defer cancel()

	output, err := executeCommand(ctx, "git", []string{"rev-parse", "--abbrev-ref", "HEAD"}, dir, false)
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(output), nil
}

// validateWorkPath ensures a path is safe and within the work directory (from config).
func validateWorkPath(path string, cfg *config.Config) error {
	cleanPath := filepath.Clean(path)
	absPath, err := filepath.Abs(cleanPath)
	if err != nil {
		return fmt.Errorf("invalid path: %w", err)
	}
	workDir, err := config.GetWorkFolderAbsPath(cfg)
	if err != nil {
		return fmt.Errorf("failed to resolve work directory: %w", err)
	}
	workDirWithSep := workDir + string(filepath.Separator)
	if !strings.HasPrefix(absPath+string(filepath.Separator), workDirWithSep) && absPath != workDir {
		return fmt.Errorf("path outside work directory: %s", path)
	}
	return nil
}

// safeReadFile reads a file after validating the path is within the work directory.
func safeReadFile(filePath string, cfg *config.Config) ([]byte, error) {
	if err := validateWorkPath(filePath, cfg); err != nil {
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

// findWorkItemFile searches for a work item file by ID in the configured work folder.
func findWorkItemFile(workItemID string, cfg *config.Config) (string, error) {
	var foundPath string
	workFolder := config.GetWorkFolderPath(cfg)

	err := filepath.Walk(workFolder, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		// Check if this is a work item file with the matching ID
		if strings.HasSuffix(path, ".md") && !strings.Contains(path, "template") && !strings.HasSuffix(path, "IDEAS.md") {
			// Read the file to check the ID
			content, err := safeReadFile(path, cfg)
			if err != nil {
				return err
			}

			// Simple check for ID in front matter (unquoted, double-quoted, or single-quoted)
			s := string(content)
			if strings.Contains(s, fmt.Sprintf("id: %s", workItemID)) ||
				strings.Contains(s, fmt.Sprintf("id: %q", workItemID)) ||
				strings.Contains(s, fmt.Sprintf("id: '%s'", workItemID)) {
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
func updateWorkItemStatus(filePath, newStatus string, cfg *config.Config) error {
	content, err := safeReadFile(filePath, cfg)
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

// archiveWorkItems archives work items to the archive directory in the configured work folder.
func archiveWorkItems(workItems []string, sourcePath string, cfg *config.Config) (string, error) {
	workFolder := config.GetWorkFolderPath(cfg)
	date := time.Now().Format("2006-01-02")
	archiveDir := filepath.Join(workFolder, "z_archive", date, filepath.Base(sourcePath))

	if err := os.MkdirAll(archiveDir, 0o700); err != nil {
		return "", fmt.Errorf("failed to create archive directory: %w", err)
	}

	// Copy work items to archive
	for _, workItem := range workItems {
		filename := filepath.Base(workItem)
		archivePath := filepath.Join(archiveDir, filename)

		content, err := safeReadFile(workItem, cfg)
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

// gitConfigEnv returns env vars so git subprocesses see GIT_CONFIG_GLOBAL when set (e.g. in CI tests).
// This ensures temp repos are trusted (safe.directory) when tests set GIT_CONFIG_GLOBAL.
func gitConfigEnv() []string {
	if v := os.Getenv("GIT_CONFIG_GLOBAL"); v != "" {
		return []string{"GIT_CONFIG_GLOBAL=" + v}
	}
	return nil
}

// executeCommand executes a command with context and optional dry-run support.
// If dryRun is true, it prints what would be executed and returns empty string and nil.
// If dryRun is false, it executes the command and returns stdout output.
// If dir is non-empty, the command is executed in that directory.
// Errors include stderr output for debugging.
// Git commands inherit GIT_CONFIG_GLOBAL when set (for CI).
func executeCommand(ctx context.Context, name string, args []string, dir string, dryRun bool) (string, error) {
	env := gitConfigEnv()
	return executeCommandWithEnv(ctx, name, args, dir, env, dryRun)
}

// executeCommandWithEnv is like executeCommand but allows passing extra environment variables
// (e.g. GIT_CONFIG_GLOBAL for CI). When extraEnv is nil or empty, behavior matches executeCommand.
func executeCommandWithEnv(ctx context.Context, name string, args []string, dir string, extraEnv []string, dryRun bool) (string, error) {
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
	if len(extraEnv) > 0 {
		cmd.Env = append(os.Environ(), extraEnv...)
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
	if env := gitConfigEnv(); len(env) > 0 {
		cmd.Env = append(os.Environ(), env...)
	}

	output, err := cmd.CombinedOutput()
	if err != nil {
		outputStr := strings.TrimSpace(string(output))
		if outputStr == "" {
			outputStr = "(no output)"
		}
		return "", fmt.Errorf("%w: %s", err, outputStr)
	}

	return string(output), nil
}

// executeCommandCombinedOutputWithEnv executes a command with additional environment variables
// and returns combined stdout+stderr. It is useful for commands (like git rebase) that may
// otherwise try to open an editor in non-interactive environments.
// If dryRun is true, it prints what would be executed and returns empty string and nil.
func executeCommandCombinedOutputWithEnv(ctx context.Context, name string, args []string, dir string, env []string, dryRun bool) (string, error) {
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
	if len(env) > 0 || len(gitConfigEnv()) > 0 {
		cmd.Env = append(os.Environ(), append(gitConfigEnv(), env...)...)
	}

	output, err := cmd.CombinedOutput()
	if err != nil {
		outputStr := strings.TrimSpace(string(output))
		if outputStr == "" {
			outputStr = "(no output)"
		}
		return "", fmt.Errorf("%w: %s", err, outputStr)
	}

	return string(output), nil
}
