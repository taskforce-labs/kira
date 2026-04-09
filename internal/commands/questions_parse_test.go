package commands

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseQuestionsFromMarkdown(t *testing.T) {
	t.Run("returns nil when no Questions section", func(t *testing.T) {
		content := []byte("# Doc\n\n## Requirements\n\nSome text")
		assert.Nil(t, ParseQuestionsFromMarkdown(content))
	})

	t.Run("ignores ## Questions to Answer", func(t *testing.T) {
		content := []byte("# Doc\n\n## Questions to Answer\n\n### 1. Foo\n")
		assert.Nil(t, ParseQuestionsFromMarkdown(content))
	})

	t.Run("lists question with no #### Options", func(t *testing.T) {
		content := []byte(`## Questions

### 1. API shape

Some context.
`)
		entries := ParseQuestionsFromMarkdown(content)
		require.Len(t, entries, 1)
		assert.Equal(t, "API shape", entries[0].Title)
		assert.False(t, entries[0].Answered)
		require.Len(t, UnansweredQuestions(entries), 1)
	})

	t.Run("lists question when #### Options all unchecked", func(t *testing.T) {
		content := []byte(`## Questions

### 1. Pick one

#### Options

- [ ] A
- [ ] B
`)
		entries := ParseQuestionsFromMarkdown(content)
		require.Len(t, entries, 1)
		assert.False(t, entries[0].Answered)
		require.Len(t, UnansweredQuestions(entries), 1)
	})

	t.Run("excludes question when at least one [x]", func(t *testing.T) {
		content := []byte(`## Questions

### 1. Pick one

#### Options

- [ ] A
- [x] B
`)
		entries := ParseQuestionsFromMarkdown(content)
		require.Len(t, entries, 1)
		assert.True(t, entries[0].Answered)
		assert.Len(t, UnansweredQuestions(entries), 0)
	})

	t.Run("excludes question when [X] checked", func(t *testing.T) {
		content := []byte(`## Questions

### 1. Pick one

#### Options

- [X] A
`)
		entries := ParseQuestionsFromMarkdown(content)
		require.Len(t, entries, 1)
		assert.True(t, entries[0].Answered)
	})

	t.Run("parses multiple questions", func(t *testing.T) {
		content := []byte(`## Questions

### 1. First

#### Options
- [x] Done

### 2. Second

No options here.

### 3. Third

#### Options
- [ ] Open
`)
		entries := ParseQuestionsFromMarkdown(content)
		require.Len(t, entries, 3)
		assert.True(t, entries[0].Answered)
		assert.False(t, entries[1].Answered)
		assert.False(t, entries[2].Answered)
		un := UnansweredQuestions(entries)
		require.Len(t, un, 2)
		assert.Equal(t, "Second", un[0].Title)
		assert.Equal(t, "Third", un[1].Title)
	})

	t.Run("does not treat ## Questions inside fenced block as section", func(t *testing.T) {
		content := []byte("```\n## Questions\n\n### 1. Fake\n```\n\n## Real\n\nNo questions.")
		assert.Nil(t, ParseQuestionsFromMarkdown(content))
	})
}
