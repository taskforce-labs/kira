package commands

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFillGapsInIdeas(t *testing.T) {
	t.Run("fills gaps before highest sequential number", func(t *testing.T) {
		tmpDir := t.TempDir()
		require.NoError(t, os.Chdir(tmpDir))
		defer func() { _ = os.Chdir("/") }()

		require.NoError(t, os.MkdirAll(".work", 0o700))
		// Create IDEAS.md with gaps: 1-10, 12, 13, 15, 16, 21, 22
		ideasContent := `# Ideas

## List

1. [2025-01-01] First idea
2. [2025-01-02] Second idea
3. [2025-01-03] Third idea
4. [2025-01-04] Fourth idea
5. [2025-01-05] Fifth idea
6. [2025-01-06] Sixth idea
7. [2025-01-07] Seventh idea
8. [2025-01-08] Eighth idea
9. [2025-01-09] Ninth idea
10. [2025-01-10] Tenth idea
12. [2025-01-12] Twelfth idea
13. [2025-01-13] Thirteenth idea
15. [2025-01-15] Fifteenth idea
16. [2025-01-16] Sixteenth idea
21. [2025-01-21] Twenty-first idea
22. [2025-01-22] Twenty-second idea
`
		require.NoError(t, os.WriteFile(".work/IDEAS.md", []byte(ideasContent), 0o600))

		ideasFile, err := parseIdeasFile()
		require.NoError(t, err)

		// Fill gaps
		err = fillGapsInIdeas(ideasFile)
		require.NoError(t, err)

		// Verify renumbering: 12→11, 13→12, 15→13, 16→14, 21→15, 22→16
		assert.Equal(t, 1, ideasFile.Ideas[1].Number)
		assert.Equal(t, 10, ideasFile.Ideas[10].Number)
		assert.Equal(t, 11, ideasFile.Ideas[11].Number)
		assert.Equal(t, "Twelfth idea", ideasFile.Ideas[11].Text)
		assert.Equal(t, 12, ideasFile.Ideas[12].Number)
		assert.Equal(t, "Thirteenth idea", ideasFile.Ideas[12].Text)
		assert.Equal(t, 13, ideasFile.Ideas[13].Number)
		assert.Equal(t, "Fifteenth idea", ideasFile.Ideas[13].Text)
		assert.Equal(t, 14, ideasFile.Ideas[14].Number)
		assert.Equal(t, "Sixteenth idea", ideasFile.Ideas[14].Text)
		assert.Equal(t, 15, ideasFile.Ideas[15].Number)
		assert.Equal(t, "Twenty-first idea", ideasFile.Ideas[15].Text)
		assert.Equal(t, 16, ideasFile.Ideas[16].Number)
		assert.Contains(t, ideasFile.Ideas[16].Text, "Twenty-second idea")

		// Now test that writing and reading preserves the renumbering
		err = writeIdeasFile(ideasFile)
		require.NoError(t, err)

		// Re-parse and verify
		ideasFile2, err := parseIdeasFile()
		require.NoError(t, err)

		// Verify the renumbered ideas are still there
		assert.Equal(t, 11, ideasFile2.Ideas[11].Number)
		assert.Equal(t, "Twelfth idea", ideasFile2.Ideas[11].Text)
		assert.Equal(t, 16, ideasFile2.Ideas[16].Number)
		assert.Contains(t, ideasFile2.Ideas[16].Text, "Twenty-second idea")
		// Verify old numbers are gone
		_, exists := ideasFile2.Ideas[21]
		assert.False(t, exists, "Old idea number 21 should not exist")
		_, exists = ideasFile2.Ideas[22]
		assert.False(t, exists, "Old idea number 22 should not exist")
	})
}
