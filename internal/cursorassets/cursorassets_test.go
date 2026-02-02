package cursorassets

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestListSkills(t *testing.T) {
	names, err := ListSkills()
	require.NoError(t, err)
	require.NotEmpty(t, names)
	assert.Contains(t, names, "kira-product-discovery")
	for _, n := range names {
		assert.True(t, strings.HasPrefix(n, "kira-"), "skill name should have kira- prefix: %s", n)
	}
}

func TestListCommands(t *testing.T) {
	names, err := ListCommands()
	require.NoError(t, err)
	require.NotEmpty(t, names)
	assert.Contains(t, names, "kira-product-discovery")
	for _, n := range names {
		assert.True(t, strings.HasPrefix(n, "kira-"), "command name should have kira- prefix: %s", n)
	}
}

func TestReadSkillSKILL(t *testing.T) {
	data, err := ReadSkillSKILL("kira-product-discovery")
	require.NoError(t, err)
	require.NotEmpty(t, data)
	content := string(data)
	assert.Contains(t, content, "name: product-discovery")
	assert.Contains(t, content, "description:")
}

func TestReadSkillFile_InvalidName(t *testing.T) {
	_, err := ReadSkillFile("..", "SKILL.md")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid skill name")

	_, err = ReadSkillFile("kira-good/../evil", "SKILL.md")
	require.Error(t, err)
}

func TestReadCommand(t *testing.T) {
	data, err := ReadCommand("kira-product-discovery")
	require.NoError(t, err)
	require.NotEmpty(t, data)
	content := string(data)
	assert.Contains(t, content, "# Product Discovery")
}

func TestReadCommand_InvalidName(t *testing.T) {
	_, err := ReadCommand("..")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid command name")

	_, err = ReadCommand("unknown-command")
	require.Error(t, err)
}

func TestSkillEntries(t *testing.T) {
	entries, err := SkillEntries("kira-product-discovery")
	require.NoError(t, err)
	require.NotEmpty(t, entries)
	var hasSKILL bool
	for _, e := range entries {
		if e.Name() == "SKILL.md" {
			hasSKILL = true
			break
		}
	}
	assert.True(t, hasSKILL, "skill should contain SKILL.md")
}

func TestSkillEntries_InvalidName(t *testing.T) {
	_, err := SkillEntries("..")
	require.Error(t, err)
}
