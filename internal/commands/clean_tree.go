// Package commands implements the CLI commands for the kira tool.
// This file provides shared helpers for running operations with a clean git working tree (stash → run → pop).
package commands

import (
	"context"
	"errors"
	"fmt"
	"strings"
)

// ErrKeepStashOnFailure is returned by a RunWithCleanTree callback when the operation failed
// but the stash should not be popped (e.g. rebase had conflicts and user will resolve manually).
// RunWithCleanTree will return the error without popping the stash.
var ErrKeepStashOnFailure = errors.New("keep stash on failure")

// HasUncommitted reports whether the working tree in dir has uncommitted changes.
// If dryRun is true, returns (false, nil) to match start's dry-run behavior.
func HasUncommitted(dir string, dryRun bool) (bool, error) {
	if dryRun {
		return false, nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), gitCommandTimeout)
	defer cancel()

	output, err := executeCommand(ctx, "git", []string{"status", "--porcelain"}, dir, false)
	if err != nil {
		return false, fmt.Errorf("failed to check git status: %w", err)
	}
	return strings.TrimSpace(output) != "", nil
}

// Stash stashes uncommitted changes in dir with the given message.
// Treats "No local changes to save" as success (no-op).
func Stash(dir, message string) error {
	ctx, cancel := context.WithTimeout(context.Background(), gitCommandTimeout)
	defer cancel()

	_, err := executeCommand(ctx, "git", []string{"stash", "push", "-m", message, "--include-untracked"}, dir, false)
	if err != nil {
		errStr := err.Error()
		if strings.Contains(errStr, "No local changes to save") {
			return nil
		}
		return fmt.Errorf("failed to stash changes: %w", err)
	}
	return nil
}

// Pop pops the most recent stash in dir.
// Treats "No stash entries found" as success (no-op). Conflicts are reported as errors.
func Pop(dir string) error {
	ctx, cancel := context.WithTimeout(context.Background(), gitCommandTimeout)
	defer cancel()

	_, err := executeCommand(ctx, "git", []string{"stash", "pop"}, dir, false)
	if err != nil {
		errStr := err.Error()
		if strings.Contains(errStr, "No stash entries found") {
			return nil
		}
		if strings.Contains(errStr, "CONFLICT") || strings.Contains(errStr, "conflict") {
			return fmt.Errorf("stash pop failed due to conflicts. Resolve conflicts manually: %w", err)
		}
		return fmt.Errorf("failed to pop stash: %w", err)
	}
	return nil
}

// RunWithCleanTree runs fn after ensuring a clean working tree, stashing if needed.
// If the tree was dirty, it stashes with message "kira <opName>: auto-stash before operation on <repoName>",
// runs fn(), then pops (unless noPopStash or fn failed). On fn() failure it pops before returning (restore)
// unless the error wraps ErrKeepStashOnFailure (e.g. latest's "rebase had conflicts, keep stash" case).
func RunWithCleanTree(dir, opName, repoName string, noPopStash bool, fn func() error) (hadStash bool, err error) {
	dirty, err := HasUncommitted(dir, false)
	if err != nil {
		return false, err
	}
	if !dirty {
		return false, fn()
	}

	msg := fmt.Sprintf("kira %s: auto-stash before operation on %s", opName, repoName)
	if err := Stash(dir, msg); err != nil {
		return false, err
	}
	hadStash = true

	opErr := fn()
	if opErr != nil {
		if errors.Is(opErr, ErrKeepStashOnFailure) {
			return hadStash, opErr
		}
		_ = Pop(dir) // Best effort to restore working tree
		return hadStash, opErr
	}

	if !noPopStash {
		if err := Pop(dir); err != nil {
			return hadStash, fmt.Errorf("operation succeeded but failed to pop stash: %w. Use 'git stash pop' to restore your changes", err)
		}
	}
	return hadStash, nil
}
