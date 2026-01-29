package commands

import (
	"os"
	"strings"
	"testing"
	"time"

	"kira/internal/config"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	ideasHeaderContent = `# Ideas

## List

`
	ideasHeaderWithOneIdea = `# Ideas

## List

1. [2025-01-01] First idea
`
	ideasHeaderWithTwoIdeas = `# Ideas

## List

1. [2025-01-01] First idea
2. [2025-01-02] Second idea
`
)

func TestAddIdeaWithNumber(t *testing.T) {
	t.Run("adds numbered idea to IDEAS.md", func(t *testing.T) {
		tmpDir := t.TempDir()
		require.NoError(t, os.Chdir(tmpDir))
		defer func() { _ = os.Chdir("/") }()

		// Create .work directory and IDEAS.md
		require.NoError(t, os.MkdirAll(".work", 0o700))
		require.NoError(t, os.WriteFile(".work/IDEAS.md", []byte(ideasHeaderContent), 0o600))

		cfg, err := config.LoadConfig()
		require.NoError(t, err)

		// Add an idea
		err = addIdeaWithNumber(cfg, "Test idea for testing")
		require.NoError(t, err)

		// Check that the idea was added
		content, err := os.ReadFile(".work/IDEAS.md")
		require.NoError(t, err)

		contentStr := string(content)
		assert.Contains(t, contentStr, "Test idea for testing")
		assert.Contains(t, contentStr, "## List")
		assert.Contains(t, contentStr, "1. [")
	})

	t.Run("adds timestamp to idea", func(t *testing.T) {
		tmpDir := t.TempDir()
		require.NoError(t, os.Chdir(tmpDir))
		defer func() { _ = os.Chdir("/") }()

		// Create .work directory and IDEAS.md
		require.NoError(t, os.MkdirAll(".work", 0o700))
		require.NoError(t, os.WriteFile(".work/IDEAS.md", []byte(ideasHeaderContent), 0o600))

		cfg, err := config.LoadConfig()
		require.NoError(t, err)

		// Add an idea
		beforeTime := time.Now()
		err = addIdeaWithNumber(cfg, "Timestamped idea")
		require.NoError(t, err)
		afterTime := time.Now()

		// Check that the idea was added with timestamp
		content, err := os.ReadFile(".work/IDEAS.md")
		require.NoError(t, err)

		contentStr := string(content)
		assert.Contains(t, contentStr, "Timestamped idea")

		// Check that timestamp is in the expected format (date only)
		lines := strings.Split(contentStr, "\n")
		var ideaLine string
		for _, line := range lines {
			if strings.Contains(line, "Timestamped idea") {
				ideaLine = line
				break
			}
		}

		assert.NotEmpty(t, ideaLine)
		assert.Contains(t, ideaLine, "1. [")
		assert.Contains(t, ideaLine, "]")

		// Parse timestamp and verify it's within expected range
		startIdx := strings.Index(ideaLine, "[") + 1
		endIdx := strings.Index(ideaLine, "]")
		timestampStr := ideaLine[startIdx:endIdx]

		_, parseErr := time.Parse("2006-01-02", timestampStr)
		require.NoError(t, parseErr)

		beforeDate := beforeTime.UTC().Format("2006-01-02")
		afterDate := afterTime.UTC().Format("2006-01-02")
		assert.True(t, timestampStr == beforeDate || timestampStr == afterDate)
	})

	t.Run("numbers ideas sequentially", func(t *testing.T) {
		tmpDir := t.TempDir()
		require.NoError(t, os.Chdir(tmpDir))
		defer func() { _ = os.Chdir("/") }()

		// Create .work directory and IDEAS.md
		require.NoError(t, os.MkdirAll(".work", 0o700))
		require.NoError(t, os.WriteFile(".work/IDEAS.md", []byte(ideasHeaderWithOneIdea), 0o600))

		cfg, err := config.LoadConfig()
		require.NoError(t, err)

		// Add second idea
		err = addIdeaWithNumber(cfg, "Second idea")
		require.NoError(t, err)

		// Check that the idea was numbered correctly
		content, err := os.ReadFile(".work/IDEAS.md")
		require.NoError(t, err)

		contentStr := string(content)
		assert.Contains(t, contentStr, "2. [")
		assert.Contains(t, contentStr, "Second idea")
	})

	t.Run("creates IDEAS.md if it doesn't exist", func(t *testing.T) {
		tmpDir := t.TempDir()
		require.NoError(t, os.Chdir(tmpDir))
		defer func() { _ = os.Chdir("/") }()

		// Create .work directory but no IDEAS.md
		require.NoError(t, os.MkdirAll(".work", 0o700))

		cfg, err := config.LoadConfig()
		require.NoError(t, err)

		// Add an idea
		err = addIdeaWithNumber(cfg, "New idea")
		require.NoError(t, err)

		// Check that IDEAS.md was created
		content, err := os.ReadFile(".work/IDEAS.md")
		require.NoError(t, err)

		assert.Contains(t, string(content), "New idea")
		assert.Contains(t, string(content), "## List")
	})
}

func TestListIdeas(t *testing.T) {
	t.Run("lists ideas with numbers", func(t *testing.T) {
		tmpDir := t.TempDir()
		require.NoError(t, os.Chdir(tmpDir))
		defer func() { _ = os.Chdir("/") }()

		// Create .work directory and IDEAS.md with ideas
		require.NoError(t, os.MkdirAll(".work", 0o700))
		require.NoError(t, os.WriteFile(".work/IDEAS.md", []byte(ideasHeaderWithTwoIdeas), 0o600))

		cfg, err := config.LoadConfig()
		require.NoError(t, err)

		// This test would need to capture stdout, so we'll test the parsing instead
		ideasFile, err := parseIdeasFile(cfg)
		require.NoError(t, err)

		assert.Equal(t, 2, len(ideasFile.Ideas))
		assert.NotNil(t, ideasFile.Ideas[1])
		assert.NotNil(t, ideasFile.Ideas[2])
		assert.Equal(t, "First idea", ideasFile.Ideas[1].Text)
		assert.Equal(t, "Second idea", ideasFile.Ideas[2].Text)
	})

	t.Run("handles empty IDEAS.md gracefully", func(t *testing.T) {
		tmpDir := t.TempDir()
		require.NoError(t, os.Chdir(tmpDir))
		defer func() { _ = os.Chdir("/") }()

		// Create .work directory and empty IDEAS.md
		require.NoError(t, os.MkdirAll(".work", 0o700))
		require.NoError(t, os.WriteFile(".work/IDEAS.md", []byte(ideasHeaderContent), 0o600))

		cfg, err := config.LoadConfig()
		require.NoError(t, err)

		err = listIdeas(cfg)
		require.NoError(t, err)
	})

	t.Run("handles missing IDEAS.md gracefully", func(t *testing.T) {
		tmpDir := t.TempDir()
		require.NoError(t, os.Chdir(tmpDir))
		defer func() { _ = os.Chdir("/") }()

		// Create .work directory but no IDEAS.md
		require.NoError(t, os.MkdirAll(".work", 0o700))

		cfg, err := config.LoadConfig()
		require.NoError(t, err)

		err = listIdeas(cfg)
		require.NoError(t, err)
	})
}
