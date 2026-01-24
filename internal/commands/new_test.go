package commands

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	yaml "gopkg.in/yaml.v3"

	"kira/internal/config"
)

type arrayDefaultYAML struct {
	Tags []string `yaml:"tags"`
}

func TestConvertDefaultToString_ArrayValuesAreYAMLSafe(t *testing.T) {
	fieldConfig := &config.FieldConfig{
		Type: "array",
	}

	defaultValue := []interface{}{"tag:important", "another, tag"}

	value, err := convertDefaultToString(defaultValue, fieldConfig)
	require.NoError(t, err)

	var parsed arrayDefaultYAML
	err = yaml.Unmarshal([]byte("tags: "+value+"\n"), &parsed)
	require.NoError(t, err)

	assert.Equal(t, []string{"tag:important", "another, tag"}, parsed.Tags)
}

func TestConvertDefaultToString_ArraySingleValueIsWrapped(t *testing.T) {
	fieldConfig := &config.FieldConfig{
		Type: "array",
	}

	defaultValue := "tag:important"

	value, err := convertDefaultToString(defaultValue, fieldConfig)
	require.NoError(t, err)

	var parsed arrayDefaultYAML
	err = yaml.Unmarshal([]byte("tags: "+value+"\n"), &parsed)
	require.NoError(t, err)

	assert.Equal(t, []string{"tag:important"}, parsed.Tags)
}
