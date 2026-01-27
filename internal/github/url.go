// Package github provides GitHub-related utilities for the kira tool.
package github

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"kira/internal/config"
)

var gitCommandTimeout = 30 * time.Second

// isGitHubURL checks if a URL is a GitHub URL (including GitHub Enterprise).
// It validates that the hostname contains "github" (case-insensitive).
func isGitHubURL(remoteURL string) bool {
	if remoteURL == "" {
		return false
	}

	// Check for SSH format: git@github.com:owner/repo.git
	sshPattern := regexp.MustCompile(`^git@([^:]+):`)
	if matches := sshPattern.FindStringSubmatch(remoteURL); len(matches) > 1 {
		hostname := strings.ToLower(matches[1])
		return strings.Contains(hostname, "github")
	}

	// Check for HTTPS format: https://github.com/owner/repo.git
	parsedURL, err := url.Parse(remoteURL)
	if err != nil {
		return false
	}

	hostname := strings.ToLower(parsedURL.Hostname())
	return strings.Contains(hostname, "github")
}

// parseGitHubURL parses a GitHub remote URL and extracts the owner and repository name.
// It supports SSH, HTTPS, and GitHub Enterprise formats.
// Returns an error if the URL format is invalid or if owner/repo cannot be extracted.
func parseGitHubURL(remoteURL string) (owner, repo string, err error) {
	if remoteURL == "" {
		return "", "", fmt.Errorf("remote URL cannot be empty")
	}

	// Handle SSH format: git@github.com:owner/repo.git
	sshPattern := regexp.MustCompile(`^git@[^:]+:([^/]+)/([^/]+?)(?:\.git)?$`)
	if matches := sshPattern.FindStringSubmatch(remoteURL); len(matches) == 3 {
		owner = strings.TrimSpace(matches[1])
		repo = strings.TrimSpace(matches[2])
		if owner == "" || repo == "" {
			return "", "", fmt.Errorf("invalid GitHub URL: owner or repository name is missing")
		}
		return owner, repo, nil
	}

	// Handle HTTPS format: https://github.com/owner/repo.git
	parsedURL, err := url.Parse(remoteURL)
	if err != nil {
		return "", "", fmt.Errorf("failed to parse GitHub URL: %s. Expected format: git@github.com:owner/repo.git or https://github.com/owner/repo.git", remoteURL)
	}

	// If parsing succeeded but path is empty, it's likely not a valid GitHub URL format
	if parsedURL.Path == "" && parsedURL.Host == "" {
		return "", "", fmt.Errorf("failed to parse GitHub URL: %s. Expected format: git@github.com:owner/repo.git or https://github.com/owner/repo.git", remoteURL)
	}

	// Extract path and remove leading/trailing slashes
	path := strings.Trim(parsedURL.Path, "/")
	if path == "" {
		return "", "", fmt.Errorf("invalid GitHub URL: owner or repository name is missing")
	}

	// Split path into components
	parts := strings.Split(path, "/")
	if len(parts) < 2 {
		return "", "", fmt.Errorf("invalid GitHub URL: owner or repository name is missing")
	}

	owner = strings.TrimSpace(parts[0])
	repo = strings.TrimSpace(parts[1])

	// Remove .git suffix if present
	repo = strings.TrimSuffix(repo, ".git")
	repo = strings.TrimSpace(repo)

	if owner == "" || repo == "" {
		return "", "", fmt.Errorf("invalid GitHub URL: owner or repository name is missing")
	}

	return owner, repo, nil
}

// GetGitHubRepoInfo retrieves the GitHub repository owner and name from the configured git remote.
// It validates that the remote is a GitHub repository and extracts owner/repo information.
func GetGitHubRepoInfo(cfg *config.Config) (owner, repo string, err error) {
	if cfg == nil {
		return "", "", fmt.Errorf("configuration cannot be nil")
	}

	// Get remote name from config
	remoteName := resolveRemoteName(cfg)
	if remoteName == "" {
		remoteName = "origin"
	}

	// Get repository root
	repoRoot, err := getRepoRoot()
	if err != nil {
		return "", "", fmt.Errorf("failed to get repository root: %w", err)
	}

	// Get remote URL
	ctx, cancel := context.WithTimeout(context.Background(), gitCommandTimeout)
	defer cancel()

	remoteURL, err := executeCommand(ctx, "git", []string{"remote", "get-url", remoteName}, repoRoot)
	if err != nil {
		// Check if remote doesn't exist
		if strings.Contains(err.Error(), "No such remote") || strings.Contains(err.Error(), "does not exist") {
			return "", "", fmt.Errorf("GitHub remote '%s' not configured", remoteName)
		}
		return "", "", fmt.Errorf("failed to get remote URL: %w", err)
	}

	remoteURL = strings.TrimSpace(remoteURL)
	if remoteURL == "" {
		return "", "", fmt.Errorf("remote '%s' has no URL configured", remoteName)
	}

	// Validate it's a GitHub URL
	if !isGitHubURL(remoteURL) {
		return "", "", fmt.Errorf("remote '%s' is not a GitHub repository. This command only works with GitHub repositories", remoteName)
	}

	// Parse the URL
	owner, repo, err = parseGitHubURL(remoteURL)
	if err != nil {
		return "", "", fmt.Errorf("failed to parse GitHub URL for remote '%s': %w", remoteName, err)
	}

	return owner, repo, nil
}

// resolveRemoteName determines the remote name using priority order:
// git.remote from config or "origin" as default
func resolveRemoteName(cfg *config.Config) string {
	if cfg.Git != nil && cfg.Git.Remote != "" {
		return cfg.Git.Remote
	}
	return "origin"
}

// getRepoRoot returns the git repository root directory by walking up from current directory
func getRepoRoot() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}

	for {
		gitPath := filepath.Join(dir, ".git")
		if _, err := os.Stat(gitPath); err == nil {
			return dir, nil
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("not a git repository")
		}
		dir = parent
	}
}

// executeCommand executes a git command and returns the output
func executeCommand(ctx context.Context, name string, args []string, dir string) (string, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	if dir != "" {
		cmd.Dir = dir
	}

	output, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return "", fmt.Errorf("%w: %s", err, string(exitErr.Stderr))
		}
		return "", err
	}

	return strings.TrimSpace(string(output)), nil
}
