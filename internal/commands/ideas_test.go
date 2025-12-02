package commands

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	ideasHeaderWithIdeas = `# Ideas

## List

1. [2025-01-01] First idea
2. [2025-01-02] Second idea
3. [2025-01-03] Third idea
`
)

func TestParseIdeasFile(t *testing.T) {
	t.Run("parses numbered ideas correctly", func(t *testing.T) {
		tmpDir := t.TempDir()
		require.NoError(t, os.Chdir(tmpDir))
		defer func() { _ = os.Chdir("/") }()

		require.NoError(t, os.MkdirAll(".work", 0o700))
		ideasContent := `# Ideas

## List

1. [2025-01-01] First idea
2. [2025-01-02] Second idea
`
		require.NoError(t, os.WriteFile(".work/IDEAS.md", []byte(ideasContent), 0o600))

		ideasFile, err := parseIdeasFile()
		require.NoError(t, err)

		assert.Equal(t, 2, len(ideasFile.Ideas))
		assert.NotNil(t, ideasFile.Ideas[1])
		assert.NotNil(t, ideasFile.Ideas[2])
		assert.Equal(t, "First idea", ideasFile.Ideas[1].Text)
		assert.Equal(t, "Second idea", ideasFile.Ideas[2].Text)
		assert.Equal(t, "2025-01-01", ideasFile.Ideas[1].Timestamp)
		assert.Equal(t, "2025-01-02", ideasFile.Ideas[2].Timestamp)
	})

	t.Run("preserves content before ## List header", func(t *testing.T) {
		tmpDir := t.TempDir()
		require.NoError(t, os.Chdir(tmpDir))
		defer func() { _ = os.Chdir("/") }()

		require.NoError(t, os.MkdirAll(".work", 0o700))
		ideasContent := `# Ideas

This is some content before.

## List

1. [2025-01-01] First idea
`
		require.NoError(t, os.WriteFile(".work/IDEAS.md", []byte(ideasContent), 0o600))

		ideasFile, err := parseIdeasFile()
		require.NoError(t, err)

		assert.Contains(t, ideasFile.BeforeIdeas, "# Ideas")
		assert.Contains(t, ideasFile.BeforeIdeas, "This is some content before")
	})

	t.Run("returns error if IDEAS.md missing", func(t *testing.T) {
		tmpDir := t.TempDir()
		require.NoError(t, os.Chdir(tmpDir))
		defer func() { _ = os.Chdir("/") }()

		require.NoError(t, os.MkdirAll(".work", 0o700))

		_, err := parseIdeasFile()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})

	t.Run("returns error if ## List header missing", func(t *testing.T) {
		tmpDir := t.TempDir()
		require.NoError(t, os.Chdir(tmpDir))
		defer func() { _ = os.Chdir("/") }()

		require.NoError(t, os.MkdirAll(".work", 0o700))
		ideasContent := `# Ideas

1. [2025-01-01] First idea
`
		require.NoError(t, os.WriteFile(".work/IDEAS.md", []byte(ideasContent), 0o600))

		_, err := parseIdeasFile()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "missing the '## List' header")
	})

	t.Run("allows empty ideas section", func(t *testing.T) {
		tmpDir := t.TempDir()
		require.NoError(t, os.Chdir(tmpDir))
		defer func() { _ = os.Chdir("/") }()

		require.NoError(t, os.MkdirAll(".work", 0o700))
		ideasContent := `# Ideas

## List

`
		require.NoError(t, os.WriteFile(".work/IDEAS.md", []byte(ideasContent), 0o600))

		ideasFile, err := parseIdeasFile()
		require.NoError(t, err)
		assert.NotNil(t, ideasFile)
		assert.Equal(t, 0, len(ideasFile.Ideas))
	})
}

func TestGetIdeaByNumber(t *testing.T) {
	t.Run("retrieves idea by number", func(t *testing.T) {
		tmpDir := t.TempDir()
		require.NoError(t, os.Chdir(tmpDir))
		defer func() { _ = os.Chdir("/") }()

		require.NoError(t, os.MkdirAll(".work", 0o700))
		ideasContent := `# Ideas

## List

1. [2025-01-01] First idea
2. [2025-01-02] Second idea
`
		require.NoError(t, os.WriteFile(".work/IDEAS.md", []byte(ideasContent), 0o600))

		idea, err := getIdeaByNumber(2)
		require.NoError(t, err)

		assert.Equal(t, 2, idea.Number)
		assert.Equal(t, "Second idea", idea.Text)
	})

	t.Run("returns error if idea not found", func(t *testing.T) {
		tmpDir := t.TempDir()
		require.NoError(t, os.Chdir(tmpDir))
		defer func() { _ = os.Chdir("/") }()

		require.NoError(t, os.MkdirAll(".work", 0o700))
		require.NoError(t, os.WriteFile(".work/IDEAS.md", []byte(ideasHeaderWithOneIdea), 0o600))

		_, err := getIdeaByNumber(100)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "Idea 100 not found")
	})
}

func TestGetNextIdeaNumber(t *testing.T) {
	t.Run("returns 1 for empty IDEAS.md", func(t *testing.T) {
		tmpDir := t.TempDir()
		require.NoError(t, os.Chdir(tmpDir))
		defer func() { _ = os.Chdir("/") }()

		require.NoError(t, os.MkdirAll(".work", 0o700))
		ideasContent := `# Ideas

## List

`
		require.NoError(t, os.WriteFile(".work/IDEAS.md", []byte(ideasContent), 0o600))

		nextNumber, err := getNextIdeaNumber()
		require.NoError(t, err)
		assert.Equal(t, 1, nextNumber)
	})

	t.Run("returns next number after highest", func(t *testing.T) {
		tmpDir := t.TempDir()
		require.NoError(t, os.Chdir(tmpDir))
		defer func() { _ = os.Chdir("/") }()

		require.NoError(t, os.MkdirAll(".work", 0o700))
		ideasContent := `# Ideas

## List

1. [2025-01-01] First idea
3. [2025-01-03] Third idea
5. [2025-01-05] Fifth idea
`
		require.NoError(t, os.WriteFile(".work/IDEAS.md", []byte(ideasContent), 0o600))

		nextNumber, err := getNextIdeaNumber()
		require.NoError(t, err)
		assert.Equal(t, 6, nextNumber)
	})

	t.Run("returns 1 if IDEAS.md doesn't exist", func(t *testing.T) {
		tmpDir := t.TempDir()
		require.NoError(t, os.Chdir(tmpDir))
		defer func() { _ = os.Chdir("/") }()

		require.NoError(t, os.MkdirAll(".work", 0o700))

		nextNumber, err := getNextIdeaNumber()
		require.NoError(t, err)
		assert.Equal(t, 1, nextNumber)
	})
}

func TestParseIdeaTitleDescription(t *testing.T) {
	t.Run("parses idea with colon", func(t *testing.T) {
		result := parseIdeaTitleDescription("dark mode: allow the user to toggle between light and dark mode")
		assert.Equal(t, "dark mode", result.Title)
		assert.Equal(t, "allow the user to toggle between light and dark mode", result.Description)
	})

	t.Run("parses idea without colon - 5+ words", func(t *testing.T) {
		result := parseIdeaTitleDescription("add user authentication requirements with OAuth support")
		assert.Equal(t, "add user authentication requirements with", result.Title)
		assert.Equal(t, "add user authentication requirements with OAuth support", result.Description)
	})

	t.Run("parses idea without colon - fewer than 5 words", func(t *testing.T) {
		result := parseIdeaTitleDescription("fix login bug")
		assert.Equal(t, "fix login bug", result.Title)
		assert.Equal(t, "", result.Description)
	})

	t.Run("parses idea without colon - exactly 4 words", func(t *testing.T) {
		result := parseIdeaTitleDescription("implement a new feature")
		assert.Equal(t, "implement a new feature", result.Title)
		assert.Equal(t, "", result.Description)
	})

	t.Run("handles empty title after colon", func(t *testing.T) {
		result := parseIdeaTitleDescription(": description only")
		assert.NotEmpty(t, result.Title)
		assert.Contains(t, result.Description, "description only")
	})

	t.Run("trims whitespace", func(t *testing.T) {
		result := parseIdeaTitleDescription("  dark mode  :  allow toggle  ")
		assert.Equal(t, "dark mode", result.Title)
		assert.Equal(t, "allow toggle", result.Description)
	})

	t.Run("handles multiple colons - uses first", func(t *testing.T) {
		result := parseIdeaTitleDescription("title: description: more text")
		assert.Equal(t, "title", result.Title)
		assert.Equal(t, "description: more text", result.Description)
	})
}

func TestRemoveIdeaByNumber(t *testing.T) {
	t.Run("removes idea and renumbers remaining", func(t *testing.T) {
		tmpDir := t.TempDir()
		require.NoError(t, os.Chdir(tmpDir))
		defer func() { _ = os.Chdir("/") }()

		require.NoError(t, os.MkdirAll(".work", 0o700))
		require.NoError(t, os.WriteFile(".work/IDEAS.md", []byte(ideasHeaderWithIdeas), 0o600))

		// Remove idea 2
		err := removeIdeaByNumber(2)
		require.NoError(t, err)

		// Verify remaining ideas are renumbered
		ideasFile, err := parseIdeasFile()
		require.NoError(t, err)

		assert.Equal(t, 2, len(ideasFile.Ideas))
		assert.NotNil(t, ideasFile.Ideas[1])
		assert.NotNil(t, ideasFile.Ideas[2])
		assert.Equal(t, "First idea", ideasFile.Ideas[1].Text)
		assert.Equal(t, "Third idea", ideasFile.Ideas[2].Text)
	})

	t.Run("removes first idea and renumbers", func(t *testing.T) {
		tmpDir := t.TempDir()
		require.NoError(t, os.Chdir(tmpDir))
		defer func() { _ = os.Chdir("/") }()

		require.NoError(t, os.MkdirAll(".work", 0o700))
		require.NoError(t, os.WriteFile(".work/IDEAS.md", []byte(ideasHeaderWithIdeas), 0o600))

		// Remove idea 1
		err := removeIdeaByNumber(1)
		require.NoError(t, err)

		// Verify remaining ideas are renumbered
		ideasFile, err := parseIdeasFile()
		require.NoError(t, err)

		assert.Equal(t, 2, len(ideasFile.Ideas))
		assert.NotNil(t, ideasFile.Ideas[1])
		assert.NotNil(t, ideasFile.Ideas[2])
		assert.Equal(t, "Second idea", ideasFile.Ideas[1].Text)
		assert.Equal(t, "Third idea", ideasFile.Ideas[2].Text)
	})

	t.Run("removes last idea and renumbers", func(t *testing.T) {
		tmpDir := t.TempDir()
		require.NoError(t, os.Chdir(tmpDir))
		defer func() { _ = os.Chdir("/") }()

		require.NoError(t, os.MkdirAll(".work", 0o700))
		require.NoError(t, os.WriteFile(".work/IDEAS.md", []byte(ideasHeaderWithIdeas), 0o600))

		// Remove idea 3
		err := removeIdeaByNumber(3)
		require.NoError(t, err)

		// Verify remaining ideas are renumbered
		ideasFile, err := parseIdeasFile()
		require.NoError(t, err)

		assert.Equal(t, 2, len(ideasFile.Ideas))
		assert.NotNil(t, ideasFile.Ideas[1])
		assert.NotNil(t, ideasFile.Ideas[2])
		assert.Equal(t, "First idea", ideasFile.Ideas[1].Text)
		assert.Equal(t, "Second idea", ideasFile.Ideas[2].Text)
	})

	t.Run("returns error if idea not found", func(t *testing.T) {
		tmpDir := t.TempDir()
		require.NoError(t, os.Chdir(tmpDir))
		defer func() { _ = os.Chdir("/") }()

		require.NoError(t, os.MkdirAll(".work", 0o700))
		require.NoError(t, os.WriteFile(".work/IDEAS.md", []byte(ideasHeaderWithOneIdea), 0o600))

		err := removeIdeaByNumber(100)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "Idea 100 not found")
	})
}

func TestWriteIdeasFile(t *testing.T) {
	t.Run("writes ideas file preserving structure", func(t *testing.T) {
		tmpDir := t.TempDir()
		require.NoError(t, os.Chdir(tmpDir))
		defer func() { _ = os.Chdir("/") }()

		require.NoError(t, os.MkdirAll(".work", 0o700))
		require.NoError(t, os.WriteFile(".work/IDEAS.md", []byte(ideasHeaderWithOneIdea), 0o600))

		ideasFile, err := parseIdeasFile()
		require.NoError(t, err)

		// Add a new idea
		ideasFile.Ideas[2] = &Idea{
			Number:    2,
			Timestamp: "2025-01-02",
			Text:      "Second idea",
			LineIndex: 5,
		}

		err = writeIdeasFile(ideasFile)
		require.NoError(t, err)

		// Verify file was written correctly
		content, err := os.ReadFile(".work/IDEAS.md")
		require.NoError(t, err)

		contentStr := string(content)
		assert.Contains(t, contentStr, "1. [2025-01-01] First idea")
		assert.Contains(t, contentStr, "2. [2025-01-02] Second idea")
		assert.Contains(t, contentStr, "## List")
	})
}
