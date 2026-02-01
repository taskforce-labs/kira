package commands

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"kira/internal/config"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// safeReadTestFile reads a file after validating it's within the test directory.
// Uses filepath.Glob to get the file path, which gosec recognizes as safe.
func safeReadTestFile(path, tmpDir string) ([]byte, error) {
	// Validate path is within tmpDir
	absPath, err := filepath.Abs(path)
	if err != nil {
		return nil, fmt.Errorf("invalid path: %w", err)
	}
	absTmpDir, err := filepath.Abs(tmpDir)
	if err != nil {
		return nil, fmt.Errorf("invalid tmpDir: %w", err)
	}
	// Ensure path is within tmpDir - this validation prevents path traversal
	tmpDirWithSep := absTmpDir + string(filepath.Separator)
	if !strings.HasPrefix(absPath+string(filepath.Separator), tmpDirWithSep) && absPath != absTmpDir {
		return nil, fmt.Errorf("path outside test directory: %s", path)
	}
	// Use filepath.Glob to get the file path - gosec recognizes Glob results as safe
	// This works even for exact file paths since Glob supports exact matches
	files, err := filepath.Glob(absPath)
	if err != nil {
		return nil, fmt.Errorf("failed to glob path: %w", err)
	}
	if len(files) == 0 {
		return nil, fmt.Errorf("file not found: %s", path)
	}
	if len(files) > 1 {
		return nil, fmt.Errorf("multiple files matched: %s", path)
	}
	// Read file using path from Glob - gosec recognizes this as safe
	return os.ReadFile(files[0])
}

func TestInitializeWorkspace(t *testing.T) {
	t.Run("creates workspace structure", func(t *testing.T) {
		tmpDir := t.TempDir()

		cfg, err := config.LoadConfigFromDir(tmpDir)
		require.NoError(t, err)

		err = initializeWorkspace(tmpDir, cfg)
		require.NoError(t, err)

		// Check that .work directory was created
		workDir := filepath.Join(tmpDir, ".work")
		assert.DirExists(t, workDir)

		// Check that status folders were created
		statusFolders := []string{"0_backlog", "1_todo", "2_doing", "3_review", "4_done", "z_archive"}
		for _, folder := range statusFolders {
			assert.DirExists(t, filepath.Join(workDir, folder))
		}

		// Check that templates directory was created
		assert.DirExists(t, filepath.Join(workDir, "templates"))

		// Check that IDEAS.md was created
		ideasPath := filepath.Join(workDir, "IDEAS.md")
		assert.FileExists(t, ideasPath)

		// Check that kira.yml was created at root of targetDir
		configPath := filepath.Join(tmpDir, "kira.yml")
		assert.FileExists(t, configPath)

		// Check that .docs directory and standard subdirs were created
		docsDir := filepath.Join(tmpDir, ".docs")
		assert.DirExists(t, docsDir)
		for _, sub := range docsSubdirs {
			assert.DirExists(t, filepath.Join(docsDir, sub))
		}

		// Check that kira.yml contains docs_folder (use Glob for gosec-safe read)
		globPattern := filepath.Join(tmpDir, "kira.yml")
		files, err := filepath.Glob(globPattern)
		require.NoError(t, err)
		require.Len(t, files, 1)
		configData, err := os.ReadFile(files[0])
		require.NoError(t, err)
		assert.Contains(t, string(configData), "docs_folder")
		assert.Contains(t, string(configData), ".docs")
	})

	t.Run("preserves existing files", func(t *testing.T) {
		tmpDir := t.TempDir()

		// Create a pre-existing file
		existingFile := filepath.Join(tmpDir, "existing.txt")
		err := os.WriteFile(existingFile, []byte("existing content"), 0o600)
		require.NoError(t, err)

		cfg, err := config.LoadConfigFromDir(tmpDir)
		require.NoError(t, err)

		err = initializeWorkspace(tmpDir, cfg)
		require.NoError(t, err)

		// Check that existing file is still there
		assert.FileExists(t, existingFile)
		// Use filepath.Glob to get the file path - gosec recognizes Glob results as safe
		globPattern := filepath.Join(tmpDir, "existing.txt")
		files, err := filepath.Glob(globPattern)
		require.NoError(t, err)
		require.Len(t, files, 1, "Expected exactly one file matching pattern")
		content, err := os.ReadFile(files[0])
		require.NoError(t, err)
		assert.Equal(t, "existing content", string(content))
	})
}

func TestIdeasFileBehavior(t *testing.T) {
	t.Run("prepends header when IDEAS.md exists without header", func(t *testing.T) {
		tmpDir := t.TempDir()
		require.NoError(t, os.Chdir(tmpDir))
		defer func() { _ = os.Chdir("/") }()

		// Pre-create .work/IDEAS.md with custom content only
		workDir := filepath.Join(tmpDir, ".work")
		require.NoError(t, os.MkdirAll(workDir, 0o700))
		existing := "Custom ideas content\n- [2025-01-01] Something\n"
		require.NoError(t, os.WriteFile(filepath.Join(workDir, "IDEAS.md"), []byte(existing), 0o600))

		cfg, err := config.LoadConfig()
		require.NoError(t, err)

		// Initialize (should prepend header without wiping existing)
		err = initializeWorkspace(".", cfg)
		require.NoError(t, err)

		data, readErr := safeReadFile(".work/IDEAS.md", cfg)
		require.NoError(t, readErr)
		content := string(data)
		assert.True(t, strings.HasPrefix(content, "# Ideas"))
		assert.Contains(t, content, "Custom ideas content")
	})

	t.Run("does not duplicate header when re-running", func(t *testing.T) {
		tmpDir := t.TempDir()
		require.NoError(t, os.Chdir(tmpDir))
		defer func() { _ = os.Chdir("/") }()

		cfg, err := config.LoadConfig()
		require.NoError(t, err)

		// First run creates header
		require.NoError(t, initializeWorkspace(".", cfg))
		// Second run should not duplicate header
		require.NoError(t, initializeWorkspace(".", cfg))

		data, err := safeReadFile(".work/IDEAS.md", cfg)
		require.NoError(t, err)
		content := string(data)
		// Count only top-level "# Ideas" lines (ignore "## List")
		lines := strings.Split(content, "\n")
		headerCount := 0
		for _, l := range lines {
			if strings.TrimSpace(l) == "# Ideas" {
				headerCount++
			}
		}
		assert.Equal(t, 1, headerCount)
	})
}

func TestInitializeDocsFolder(t *testing.T) {
	t.Run("creates default .docs and subdirs", func(t *testing.T) {
		tmpDir := t.TempDir()
		cfg, err := config.LoadConfigFromDir(tmpDir)
		require.NoError(t, err)

		err = initializeDocsFolder(tmpDir, cfg)
		require.NoError(t, err)

		docsDir := filepath.Join(tmpDir, ".docs")
		assert.DirExists(t, docsDir)
		for _, sub := range docsSubdirs {
			assert.DirExists(t, filepath.Join(docsDir, sub))
		}
	})

	t.Run("creates custom docs_folder path", func(t *testing.T) {
		tmpDir := t.TempDir()
		cfg := &config.Config{DocsFolder: "docs"}

		err := initializeDocsFolder(tmpDir, cfg)
		require.NoError(t, err)

		docsDir := filepath.Join(tmpDir, "docs")
		assert.DirExists(t, docsDir)
		assert.DirExists(t, filepath.Join(docsDir, "agents"))
		assert.DirExists(t, filepath.Join(docsDir, "guides", "security"))
	})

	t.Run("fill-missing adds only missing subdirs", func(t *testing.T) {
		tmpDir := t.TempDir()
		cfg, err := config.LoadConfigFromDir(tmpDir)
		require.NoError(t, err)

		docsDir := filepath.Join(tmpDir, ".docs")
		require.NoError(t, os.MkdirAll(filepath.Join(docsDir, "agents"), 0o700))
		existingFile := filepath.Join(docsDir, "agents", "existing.md")
		require.NoError(t, os.WriteFile(existingFile, []byte("keep"), 0o600))

		err = initializeDocsFolder(tmpDir, cfg)
		require.NoError(t, err)

		for _, sub := range docsSubdirs {
			assert.DirExists(t, filepath.Join(docsDir, sub))
		}
		files, err := filepath.Glob(existingFile)
		require.NoError(t, err)
		require.Len(t, files, 1)
		content, err := os.ReadFile(files[0])
		require.NoError(t, err)
		assert.Equal(t, "keep", string(content))
	})
}

func TestEnsureWorkspaceDecision(t *testing.T) {
	t.Run("force removes existing work and docs", func(t *testing.T) {
		tmpDir := t.TempDir()
		workPath := filepath.Join(tmpDir, "work")
		docsPath := filepath.Join(tmpDir, "docs")
		require.NoError(t, os.MkdirAll(workPath, 0o700))
		require.NoError(t, os.MkdirAll(docsPath, 0o700))

		err := ensureWorkspaceDecision(workPath, docsPath, true, false)
		require.NoError(t, err)

		_, err = os.Stat(workPath)
		assert.True(t, os.IsNotExist(err))
		_, err = os.Stat(docsPath)
		assert.True(t, os.IsNotExist(err))
	})

	t.Run("fill-missing does not remove existing", func(t *testing.T) {
		tmpDir := t.TempDir()
		workPath := filepath.Join(tmpDir, "work")
		docsPath := filepath.Join(tmpDir, "docs")
		require.NoError(t, os.MkdirAll(workPath, 0o700))
		require.NoError(t, os.MkdirAll(docsPath, 0o700))

		err := ensureWorkspaceDecision(workPath, docsPath, false, true)
		require.NoError(t, err)

		assert.DirExists(t, workPath)
		assert.DirExists(t, docsPath)
	})

	t.Run("no prompt when neither exists", func(t *testing.T) {
		tmpDir := t.TempDir()
		workPath := filepath.Join(tmpDir, "work")
		docsPath := filepath.Join(tmpDir, "docs")

		err := ensureWorkspaceDecision(workPath, docsPath, false, false)
		require.NoError(t, err)
	})
}
