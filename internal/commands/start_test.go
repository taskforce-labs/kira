package commands

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"kira/internal/config"
)

func TestValidateWorkItemID(t *testing.T) {
	cfg := &config.Config{
		Validation: config.ValidationConfig{
			IDFormat: "^\\d{3}$", // Three digits
		},
	}

	t.Run("accepts valid three-digit ID", func(t *testing.T) {
		err := validateWorkItemID("001", cfg)
		assert.NoError(t, err)

		err = validateWorkItemID("123", cfg)
		assert.NoError(t, err)

		err = validateWorkItemID("999", cfg)
		assert.NoError(t, err)
	})

	t.Run("rejects path traversal attempts", func(t *testing.T) {
		err := validateWorkItemID("../001", cfg)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid work item ID")

		err = validateWorkItemID("001/../002", cfg)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid work item ID")

		err = validateWorkItemID("001/subdir", cfg)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid work item ID")

		err = validateWorkItemID("001\\subdir", cfg)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid work item ID")
	})

	t.Run("rejects invalid ID formats", func(t *testing.T) {
		err := validateWorkItemID("1", cfg)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid work item ID")

		err = validateWorkItemID("12", cfg)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid work item ID")

		err = validateWorkItemID("1234", cfg)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid work item ID")

		err = validateWorkItemID("abc", cfg)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid work item ID")

		err = validateWorkItemID("12a", cfg)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid work item ID")
	})

	t.Run("works with custom ID format", func(t *testing.T) {
		customCfg := &config.Config{
			Validation: config.ValidationConfig{
				IDFormat: "^[A-Z]{2}-\\d{4}$", // e.g., AB-1234
			},
		}

		err := validateWorkItemID("AB-1234", customCfg)
		assert.NoError(t, err)

		err = validateWorkItemID("XY-9999", customCfg)
		assert.NoError(t, err)

		err = validateWorkItemID("123", customCfg)
		require.Error(t, err)
	})
}

func TestSanitizeTitle(t *testing.T) {
	t.Run("converts spaces to hyphens", func(t *testing.T) {
		result, err := sanitizeTitle("hello world", "001")
		require.NoError(t, err)
		assert.Equal(t, "hello-world", result)
	})

	t.Run("converts underscores to hyphens", func(t *testing.T) {
		result, err := sanitizeTitle("hello_world", "001")
		require.NoError(t, err)
		assert.Equal(t, "hello-world", result)
	})

	t.Run("converts to lowercase", func(t *testing.T) {
		result, err := sanitizeTitle("Hello World", "001")
		require.NoError(t, err)
		assert.Equal(t, "hello-world", result)
	})

	t.Run("handles mixed input", func(t *testing.T) {
		result, err := sanitizeTitle("Fix Bug_In Feature", "001")
		require.NoError(t, err)
		assert.Equal(t, "fix-bug-in-feature", result)
	})

	t.Run("handles empty title by returning empty string", func(t *testing.T) {
		result, err := sanitizeTitle("", "001")
		require.NoError(t, err)
		assert.Equal(t, "", result)
	})

	t.Run("handles unknown title by returning empty string", func(t *testing.T) {
		result, err := sanitizeTitle("unknown", "001")
		require.NoError(t, err)
		assert.Equal(t, "", result)
	})

	t.Run("rejects title that sanitizes to only hyphens", func(t *testing.T) {
		_, err := sanitizeTitle("---", "001")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "sanitization resulted in empty string")
	})

	t.Run("rejects title that sanitizes to empty", func(t *testing.T) {
		// A title with only special characters that get removed
		_, err := sanitizeTitle("   ", "001")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "sanitization resulted in empty string")
	})

	t.Run("truncates long titles with hash suffix", func(t *testing.T) {
		// Create a title longer than maxTitleLength (100 chars)
		longTitle := "this-is-a-very-long-title-that-exceeds-the-maximum-allowed-length-for-branch-names-and-worktree-directories-which-should-be-truncated"
		result, err := sanitizeTitle(longTitle, "001")
		require.NoError(t, err)
		assert.LessOrEqual(t, len(result), maxTitleLength)
		// Should end with a hash suffix (6 hex chars)
		assert.Regexp(t, `-[a-f0-9]{6}$`, result)
	})

	t.Run("preserves unicode characters", func(t *testing.T) {
		result, err := sanitizeTitle("Café Feature", "001")
		require.NoError(t, err)
		assert.Equal(t, "café-feature", result)
	})

	t.Run("removes leading and trailing hyphens", func(t *testing.T) {
		result, err := sanitizeTitle("  hello world  ", "001")
		require.NoError(t, err)
		assert.Equal(t, "hello-world", result)
	})
}

func TestInferWorkspaceBehavior(t *testing.T) {
	t.Run("returns standalone when no workspace config", func(t *testing.T) {
		cfg := &config.Config{}
		behavior := inferWorkspaceBehavior(cfg)
		assert.Equal(t, WorkspaceBehaviorStandalone, behavior)
	})

	t.Run("returns standalone when workspace has no projects", func(t *testing.T) {
		cfg := &config.Config{
			Workspace: &config.WorkspaceConfig{},
		}
		behavior := inferWorkspaceBehavior(cfg)
		assert.Equal(t, WorkspaceBehaviorStandalone, behavior)
	})

	t.Run("returns standalone when workspace projects is empty", func(t *testing.T) {
		cfg := &config.Config{
			Workspace: &config.WorkspaceConfig{
				Projects: []config.ProjectConfig{},
			},
		}
		behavior := inferWorkspaceBehavior(cfg)
		assert.Equal(t, WorkspaceBehaviorStandalone, behavior)
	})

	t.Run("returns polyrepo when any project has repo_root", func(t *testing.T) {
		cfg := &config.Config{
			Workspace: &config.WorkspaceConfig{
				Projects: []config.ProjectConfig{
					{Name: "frontend", Path: "../frontend"},
					{Name: "backend", Path: "../backend", RepoRoot: "../monorepo"},
				},
			},
		}
		behavior := inferWorkspaceBehavior(cfg)
		assert.Equal(t, WorkspaceBehaviorPolyrepo, behavior)
	})

	t.Run("returns monorepo when projects have no path fields", func(t *testing.T) {
		cfg := &config.Config{
			Workspace: &config.WorkspaceConfig{
				Projects: []config.ProjectConfig{
					{Name: "frontend", Kind: "app"},
					{Name: "backend", Kind: "service"},
				},
			},
		}
		behavior := inferWorkspaceBehavior(cfg)
		assert.Equal(t, WorkspaceBehaviorMonorepo, behavior)
	})

	t.Run("returns polyrepo when project path is external git repo", func(t *testing.T) {
		// Create a temporary directory structure to test
		tmpDir := t.TempDir()
		externalRepo := filepath.Join(tmpDir, "external-repo")
		require.NoError(t, os.MkdirAll(filepath.Join(externalRepo, ".git"), 0o700))

		cfg := &config.Config{
			Workspace: &config.WorkspaceConfig{
				Projects: []config.ProjectConfig{
					{Name: "external", Path: externalRepo},
				},
			},
		}
		behavior := inferWorkspaceBehavior(cfg)
		assert.Equal(t, WorkspaceBehaviorPolyrepo, behavior)
	})
}

func TestWorkspaceBehaviorString(t *testing.T) {
	assert.Equal(t, "standalone", WorkspaceBehaviorStandalone.String())
	assert.Equal(t, "monorepo", WorkspaceBehaviorMonorepo.String())
	assert.Equal(t, "polyrepo", WorkspaceBehaviorPolyrepo.String())
}

func TestCheckWorktreeExists(t *testing.T) {
	t.Run("returns NotExists for non-existent path", func(t *testing.T) {
		status, err := checkWorktreeExists("/non/existent/path", "001")
		require.NoError(t, err)
		assert.Equal(t, WorktreeNotExists, status)
	})

	t.Run("returns InvalidPath for file (not directory)", func(t *testing.T) {
		tmpFile, err := os.CreateTemp("", "test")
		require.NoError(t, err)
		defer func() { _ = os.Remove(tmpFile.Name()) }()
		_ = tmpFile.Close()

		status, err := checkWorktreeExists(tmpFile.Name(), "001")
		require.NoError(t, err)
		assert.Equal(t, WorktreeInvalidPath, status)
	})

	t.Run("returns InvalidPath for directory without .git", func(t *testing.T) {
		tmpDir := t.TempDir()

		status, err := checkWorktreeExists(tmpDir, "001")
		require.NoError(t, err)
		assert.Equal(t, WorktreeInvalidPath, status)
	})

	t.Run("returns InvalidPath for directory with .git directory (regular repo)", func(t *testing.T) {
		tmpDir := t.TempDir()
		gitDir := filepath.Join(tmpDir, ".git")
		require.NoError(t, os.MkdirAll(gitDir, 0o700))

		status, err := checkWorktreeExists(tmpDir, "001")
		require.NoError(t, err)
		assert.Equal(t, WorktreeInvalidPath, status)
	})

	t.Run("returns ValidSameItem for worktree with matching ID in path", func(t *testing.T) {
		tmpDir := t.TempDir()
		worktreePath := filepath.Join(tmpDir, "001-my-feature")
		require.NoError(t, os.MkdirAll(worktreePath, 0o700))

		// Create .git as a file (worktree indicator)
		gitFile := filepath.Join(worktreePath, ".git")
		require.NoError(t, os.WriteFile(gitFile, []byte("gitdir: /some/path"), 0o600))

		status, err := checkWorktreeExists(worktreePath, "001")
		require.NoError(t, err)
		assert.Equal(t, WorktreeValidSameItem, status)
	})

	t.Run("returns ValidDifferentItem for worktree with different ID", func(t *testing.T) {
		tmpDir := t.TempDir()
		worktreePath := filepath.Join(tmpDir, "002-other-feature")
		require.NoError(t, os.MkdirAll(worktreePath, 0o700))

		// Create .git as a file (worktree indicator)
		gitFile := filepath.Join(worktreePath, ".git")
		require.NoError(t, os.WriteFile(gitFile, []byte("gitdir: /some/path"), 0o600))

		status, err := checkWorktreeExists(worktreePath, "001")
		require.NoError(t, err)
		assert.Equal(t, WorktreeValidDifferentItem, status)
	})
}

func TestValidateAndCleanPath(t *testing.T) {
	t.Run("accepts simple relative path", func(t *testing.T) {
		result, err := validateAndCleanPath("../worktrees")
		require.NoError(t, err)
		assert.Equal(t, "../worktrees", result)
	})

	t.Run("cleans redundant slashes", func(t *testing.T) {
		result, err := validateAndCleanPath("../worktrees//subdir")
		require.NoError(t, err)
		assert.NotContains(t, result, "//")
	})

	t.Run("cleans current directory references", func(t *testing.T) {
		result, err := validateAndCleanPath("./worktrees/./subdir")
		require.NoError(t, err)
		assert.Equal(t, "worktrees/subdir", result)
	})

	t.Run("accepts absolute path", func(t *testing.T) {
		result, err := validateAndCleanPath("/Users/test/worktrees")
		require.NoError(t, err)
		assert.Equal(t, "/Users/test/worktrees", result)
	})
}

func TestFindCommonPathPrefix(t *testing.T) {
	t.Run("returns empty for empty input", func(t *testing.T) {
		result := findCommonPathPrefix([]string{})
		assert.Equal(t, "", result)
	})

	t.Run("returns parent dir for single path", func(t *testing.T) {
		result := findCommonPathPrefix([]string{"/Users/test/project"})
		assert.Equal(t, "/Users/test", result)
	})

	t.Run("finds common prefix for sibling directories", func(t *testing.T) {
		paths := []string{
			"/Users/test/repos/frontend",
			"/Users/test/repos/backend",
		}
		result := findCommonPathPrefix(paths)
		assert.Equal(t, "/Users/test/repos", result)
	})

	t.Run("finds common prefix for nested directories", func(t *testing.T) {
		paths := []string{
			"/Users/test/repos/project/frontend",
			"/Users/test/repos/project/backend",
			"/Users/test/repos/project/shared",
		}
		result := findCommonPathPrefix(paths)
		assert.Equal(t, "/Users/test/repos/project", result)
	})

	t.Run("returns root for paths with only root in common", func(t *testing.T) {
		paths := []string{
			"/Users/alice/project",
			"/opt/project",
		}
		result := findCommonPathPrefix(paths)
		// Both have "/" as common prefix (empty string after the leading slash)
		// filepath.Join of an empty first component gives just the separator
		assert.Equal(t, "/", result)
	})
}

func TestCheckWorkItemStatus(t *testing.T) {
	t.Run("returns nil when skipCheck is true", func(t *testing.T) {
		err := checkWorkItemStatus("doing", "doing", true)
		assert.NoError(t, err)
	})

	t.Run("returns nil when status differs from target", func(t *testing.T) {
		err := checkWorkItemStatus("backlog", "doing", false)
		assert.NoError(t, err)
	})

	t.Run("returns error when status matches target", func(t *testing.T) {
		err := checkWorkItemStatus("doing", "doing", false)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "status already matches target")
	})
}

func TestDeriveWorktreeRoot(t *testing.T) {
	t.Run("uses configured worktree_root when set", func(t *testing.T) {
		cfg := &config.Config{
			Workspace: &config.WorkspaceConfig{
				WorktreeRoot: "/custom/worktrees",
			},
		}

		result, err := deriveWorktreeRoot(cfg, WorkspaceBehaviorStandalone)
		require.NoError(t, err)
		assert.Equal(t, "/custom/worktrees", result)
	})

	t.Run("derives worktree root for standalone", func(t *testing.T) {
		cfg := &config.Config{}

		// This will use current directory as fallback
		result, err := deriveWorktreeRoot(cfg, WorkspaceBehaviorStandalone)
		require.NoError(t, err)
		assert.Contains(t, result, "_worktrees")
	})

	t.Run("derives worktree root for monorepo same as standalone", func(t *testing.T) {
		cfg := &config.Config{}

		standalonePath, err := deriveWorktreeRoot(cfg, WorkspaceBehaviorStandalone)
		require.NoError(t, err)

		monorepoPath, err := deriveWorktreeRoot(cfg, WorkspaceBehaviorMonorepo)
		require.NoError(t, err)

		assert.Equal(t, standalonePath, monorepoPath)
	})
}

func TestIsExternalGitRepo(t *testing.T) {
	t.Run("returns false for non-existent path", func(t *testing.T) {
		result := isExternalGitRepo("/non/existent/path")
		assert.False(t, result)
	})

	t.Run("returns true for directory with .git directory", func(t *testing.T) {
		tmpDir := t.TempDir()
		gitDir := filepath.Join(tmpDir, ".git")
		require.NoError(t, os.MkdirAll(gitDir, 0o700))

		result := isExternalGitRepo(tmpDir)
		assert.True(t, result)
	})

	t.Run("returns true for directory with .git file (worktree)", func(t *testing.T) {
		tmpDir := t.TempDir()
		gitFile := filepath.Join(tmpDir, ".git")
		require.NoError(t, os.WriteFile(gitFile, []byte("gitdir: /some/path"), 0o600))

		result := isExternalGitRepo(tmpDir)
		assert.True(t, result)
	})

	t.Run("returns false for directory without .git", func(t *testing.T) {
		tmpDir := t.TempDir()

		result := isExternalGitRepo(tmpDir)
		assert.False(t, result)
	})
}

func TestGetValidStatuses(t *testing.T) {
	cfg := &config.Config{
		StatusFolders: map[string]string{
			"done":    "4_done",
			"backlog": "0_backlog",
			"doing":   "2_doing",
			"todo":    "1_todo",
		},
	}

	result := getValidStatuses(cfg)

	// Should be sorted alphabetically
	assert.Equal(t, []string{"backlog", "doing", "done", "todo"}, result)
}

func TestStartFlags(t *testing.T) {
	t.Run("all flags have correct types", func(t *testing.T) {
		flags := StartFlags{
			DryRun:          true,
			Override:        true,
			SkipStatusCheck: true,
			ReuseBranch:     true,
			NoIDE:           true,
			IDECommand:      "cursor",
			TrunkBranch:     "develop",
			StatusAction:    "commit_only",
		}

		assert.True(t, flags.DryRun)
		assert.True(t, flags.Override)
		assert.True(t, flags.SkipStatusCheck)
		assert.True(t, flags.ReuseBranch)
		assert.True(t, flags.NoIDE)
		assert.Equal(t, "cursor", flags.IDECommand)
		assert.Equal(t, "develop", flags.TrunkBranch)
		assert.Equal(t, "commit_only", flags.StatusAction)
	})
}

func TestStartContext(t *testing.T) {
	t.Run("holds all required fields", func(t *testing.T) {
		cfg := &config.Config{}
		ctx := &StartContext{
			WorkItemID:     "001",
			WorkItemPath:   ".work/0_backlog/001-test.md",
			SanitizedTitle: "test-feature",
			BranchName:     "001-test-feature",
			WorktreeRoot:   "/path/to/worktrees",
			Behavior:       WorkspaceBehaviorStandalone,
			Config:         cfg,
			Flags: StartFlags{
				DryRun: true,
			},
		}

		assert.Equal(t, "001", ctx.WorkItemID)
		assert.Equal(t, ".work/0_backlog/001-test.md", ctx.WorkItemPath)
		assert.Equal(t, "test-feature", ctx.SanitizedTitle)
		assert.Equal(t, "001-test-feature", ctx.BranchName)
		assert.Equal(t, "/path/to/worktrees", ctx.WorktreeRoot)
		assert.Equal(t, WorkspaceBehaviorStandalone, ctx.Behavior)
		assert.True(t, ctx.Flags.DryRun)
	})
}

// ============================================================================
// Phase 2: Git Operations Tests
// ============================================================================

func TestBranchStatus(t *testing.T) {
	t.Run("BranchNotExists is default", func(t *testing.T) {
		var status BranchStatus
		assert.Equal(t, BranchNotExists, status)
	})

	t.Run("BranchStatus values are distinct", func(t *testing.T) {
		assert.NotEqual(t, BranchNotExists, BranchPointsToTrunk)
		assert.NotEqual(t, BranchNotExists, BranchHasCommits)
		assert.NotEqual(t, BranchPointsToTrunk, BranchHasCommits)
	})
}

func TestResolveRemoteName(t *testing.T) {
	t.Run("returns origin when no config", func(t *testing.T) {
		cfg := &config.Config{}
		result := resolveRemoteName(cfg, nil)
		assert.Equal(t, "origin", result)
	})

	t.Run("returns git.remote when configured", func(t *testing.T) {
		cfg := &config.Config{
			Git: &config.GitConfig{
				Remote: "upstream",
			},
		}
		result := resolveRemoteName(cfg, nil)
		assert.Equal(t, "upstream", result)
	})

	t.Run("returns project.remote when project has remote", func(t *testing.T) {
		cfg := &config.Config{
			Git: &config.GitConfig{
				Remote: "upstream",
			},
		}
		project := &config.ProjectConfig{
			Remote: "github",
		}
		result := resolveRemoteName(cfg, project)
		assert.Equal(t, "github", result)
	})

	t.Run("falls back to git.remote when project has no remote", func(t *testing.T) {
		cfg := &config.Config{
			Git: &config.GitConfig{
				Remote: "upstream",
			},
		}
		project := &config.ProjectConfig{
			Name: "test",
		}
		result := resolveRemoteName(cfg, project)
		assert.Equal(t, "upstream", result)
	})

	t.Run("falls back to origin when project and git have no remote", func(t *testing.T) {
		cfg := &config.Config{}
		project := &config.ProjectConfig{
			Name: "test",
		}
		result := resolveRemoteName(cfg, project)
		assert.Equal(t, "origin", result)
	})
}

func TestDetermineTrunkBranch(t *testing.T) {
	t.Run("uses flag value when provided in dry-run", func(t *testing.T) {
		cfg := &config.Config{
			Git: &config.GitConfig{
				TrunkBranch: "main",
			},
		}
		result, err := determineTrunkBranch(cfg, "develop", "", true)
		require.NoError(t, err)
		assert.Equal(t, "develop", result)
	})

	t.Run("uses config value when no flag in dry-run", func(t *testing.T) {
		cfg := &config.Config{
			Git: &config.GitConfig{
				TrunkBranch: "production",
			},
		}
		result, err := determineTrunkBranch(cfg, "", "", true)
		require.NoError(t, err)
		assert.Equal(t, "production", result)
	})

	t.Run("auto-detects main in dry-run", func(t *testing.T) {
		cfg := &config.Config{}
		result, err := determineTrunkBranch(cfg, "", "", true)
		require.NoError(t, err)
		assert.Equal(t, "main", result)
	})
}

func TestAutoDetectTrunkBranch(t *testing.T) {
	t.Run("returns main in dry-run mode", func(t *testing.T) {
		result, err := autoDetectTrunkBranch("", true)
		require.NoError(t, err)
		assert.Equal(t, "main", result)
	})
}

func TestHandleExistingWorktree(t *testing.T) {
	t.Run("returns nil for non-existent path", func(t *testing.T) {
		err := handleExistingWorktree("/non/existent/path", "001", false, false)
		assert.NoError(t, err)
	})

	t.Run("returns error for existing worktree without override", func(t *testing.T) {
		tmpDir := t.TempDir()
		worktreePath := filepath.Join(tmpDir, "001-my-feature")
		require.NoError(t, os.MkdirAll(worktreePath, 0o700))

		// Create .git as a file (worktree indicator)
		gitFile := filepath.Join(worktreePath, ".git")
		require.NoError(t, os.WriteFile(gitFile, []byte("gitdir: /some/path"), 0o600))

		err := handleExistingWorktree(worktreePath, "001", false, false)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "worktree already exists")
	})

	t.Run("returns error for different work item worktree without override", func(t *testing.T) {
		tmpDir := t.TempDir()
		worktreePath := filepath.Join(tmpDir, "002-other-feature")
		require.NoError(t, os.MkdirAll(worktreePath, 0o700))

		// Create .git as a file (worktree indicator)
		gitFile := filepath.Join(worktreePath, ".git")
		require.NoError(t, os.WriteFile(gitFile, []byte("gitdir: /some/path"), 0o600))

		err := handleExistingWorktree(worktreePath, "001", false, false)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "different work item")
	})

	t.Run("returns error for invalid path without override", func(t *testing.T) {
		tmpDir := t.TempDir()
		invalidPath := filepath.Join(tmpDir, "invalid-path")
		require.NoError(t, os.MkdirAll(invalidPath, 0o700))
		// No .git file/dir - invalid worktree

		err := handleExistingWorktree(invalidPath, "001", false, false)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "not a valid git worktree")
	})

	t.Run("removes invalid path with override in dry-run", func(t *testing.T) {
		tmpDir := t.TempDir()
		invalidPath := filepath.Join(tmpDir, "invalid-path")
		require.NoError(t, os.MkdirAll(invalidPath, 0o700))

		// With dry-run, should not actually remove
		err := handleExistingWorktree(invalidPath, "001", true, true)
		assert.NoError(t, err)

		// Path should still exist (dry-run)
		_, statErr := os.Stat(invalidPath)
		assert.NoError(t, statErr)
	})
}

func TestResolvePolyrepoProjects(t *testing.T) {
	t.Run("returns nil when no workspace config", func(t *testing.T) {
		cfg := &config.Config{}
		result, err := resolvePolyrepoProjects(cfg, "/repo")
		require.NoError(t, err)
		assert.Nil(t, result)
	})

	t.Run("returns nil when no projects", func(t *testing.T) {
		cfg := &config.Config{
			Workspace: &config.WorkspaceConfig{},
		}
		result, err := resolvePolyrepoProjects(cfg, "/repo")
		require.NoError(t, err)
		assert.Nil(t, result)
	})

	t.Run("resolves relative paths", func(t *testing.T) {
		cfg := &config.Config{
			Workspace: &config.WorkspaceConfig{
				Projects: []config.ProjectConfig{
					{Name: "frontend", Path: "../frontend", Mount: "fe"},
				},
			},
		}
		result, err := resolvePolyrepoProjects(cfg, "/Users/test/main")
		require.NoError(t, err)
		require.Len(t, result, 1)
		assert.Equal(t, "frontend", result[0].Name)
		assert.Equal(t, "/Users/test/frontend", result[0].Path)
		assert.Equal(t, "fe", result[0].Mount)
	})

	t.Run("preserves absolute paths", func(t *testing.T) {
		cfg := &config.Config{
			Workspace: &config.WorkspaceConfig{
				Projects: []config.ProjectConfig{
					{Name: "frontend", Path: "/absolute/frontend", Mount: "fe"},
				},
			},
		}
		result, err := resolvePolyrepoProjects(cfg, "/Users/test/main")
		require.NoError(t, err)
		require.Len(t, result, 1)
		assert.Equal(t, "/absolute/frontend", result[0].Path)
	})

	t.Run("uses project trunk_branch when set", func(t *testing.T) {
		cfg := &config.Config{
			Git: &config.GitConfig{
				TrunkBranch: "main",
			},
			Workspace: &config.WorkspaceConfig{
				Projects: []config.ProjectConfig{
					{Name: "frontend", Path: "../frontend", TrunkBranch: "develop"},
				},
			},
		}
		result, err := resolvePolyrepoProjects(cfg, "/Users/test/main")
		require.NoError(t, err)
		require.Len(t, result, 1)
		assert.Equal(t, "develop", result[0].TrunkBranch)
	})

	t.Run("falls back to git.trunk_branch when project has none", func(t *testing.T) {
		cfg := &config.Config{
			Git: &config.GitConfig{
				TrunkBranch: "production",
			},
			Workspace: &config.WorkspaceConfig{
				Projects: []config.ProjectConfig{
					{Name: "frontend", Path: "../frontend"},
				},
			},
		}
		result, err := resolvePolyrepoProjects(cfg, "/Users/test/main")
		require.NoError(t, err)
		require.Len(t, result, 1)
		assert.Equal(t, "production", result[0].TrunkBranch)
	})

	t.Run("uses project remote when set", func(t *testing.T) {
		cfg := &config.Config{
			Git: &config.GitConfig{
				Remote: "upstream",
			},
			Workspace: &config.WorkspaceConfig{
				Projects: []config.ProjectConfig{
					{Name: "frontend", Path: "../frontend", Remote: "github"},
				},
			},
		}
		result, err := resolvePolyrepoProjects(cfg, "/Users/test/main")
		require.NoError(t, err)
		require.Len(t, result, 1)
		assert.Equal(t, "github", result[0].Remote)
	})
}

func TestGroupProjectsByRepoRoot(t *testing.T) {
	t.Run("groups projects with same repo_root", func(t *testing.T) {
		projects := []PolyrepoProject{
			{Name: "frontend", Path: "/monorepo/frontend", RepoRoot: "/monorepo"},
			{Name: "backend", Path: "/monorepo/backend", RepoRoot: "/monorepo"},
			{Name: "orders", Path: "/orders-service"},
		}

		result := groupProjectsByRepoRoot(projects)

		// Should have 2 groups: /monorepo and /orders-service
		assert.Len(t, result, 2)
		assert.Len(t, result["/monorepo"], 2)
		assert.Len(t, result["/orders-service"], 1)
	})

	t.Run("uses path as key for standalone projects", func(t *testing.T) {
		projects := []PolyrepoProject{
			{Name: "frontend", Path: "/frontend"},
			{Name: "backend", Path: "/backend"},
		}

		result := groupProjectsByRepoRoot(projects)

		assert.Len(t, result, 2)
		assert.Contains(t, result, "/frontend")
		assert.Contains(t, result, "/backend")
	})
}

func TestValidatePolyrepoProjects(t *testing.T) {
	t.Run("returns nil in dry-run mode", func(t *testing.T) {
		projects := []PolyrepoProject{
			{Name: "frontend", Path: "/non/existent"},
		}
		err := validatePolyrepoProjects(projects, true)
		assert.NoError(t, err)
	})

	t.Run("returns nil for projects without path", func(t *testing.T) {
		projects := []PolyrepoProject{
			{Name: "frontend"},
		}
		err := validatePolyrepoProjects(projects, false)
		assert.NoError(t, err)
	})

	t.Run("returns error for non-existent path", func(t *testing.T) {
		projects := []PolyrepoProject{
			{Name: "frontend", Path: "/non/existent/path"},
		}
		err := validatePolyrepoProjects(projects, false)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})

	t.Run("returns nil for valid git repo", func(t *testing.T) {
		tmpDir := t.TempDir()
		require.NoError(t, os.MkdirAll(filepath.Join(tmpDir, ".git"), 0o700))

		projects := []PolyrepoProject{
			{Name: "frontend", Path: tmpDir},
		}
		err := validatePolyrepoProjects(projects, false)
		assert.NoError(t, err)
	})
}

func TestPolyrepoProject(t *testing.T) {
	t.Run("holds all fields", func(t *testing.T) {
		project := PolyrepoProject{
			Name:        "frontend",
			Path:        "/path/to/frontend",
			Mount:       "fe",
			RepoRoot:    "/path/to/monorepo",
			TrunkBranch: "develop",
			Remote:      "upstream",
		}

		assert.Equal(t, "frontend", project.Name)
		assert.Equal(t, "/path/to/frontend", project.Path)
		assert.Equal(t, "fe", project.Mount)
		assert.Equal(t, "/path/to/monorepo", project.RepoRoot)
		assert.Equal(t, "develop", project.TrunkBranch)
		assert.Equal(t, "upstream", project.Remote)
	})
}
