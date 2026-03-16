package commands

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRoadmapLint_ValidRoadmap(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.Chdir(dir))
	defer func() { _ = os.Chdir(t.TempDir()) }()

	// Minimal kira workspace
	workDir := filepath.Join(dir, ".work")
	require.NoError(t, os.MkdirAll(filepath.Join(workDir, "0_backlog"), 0o700))
	// Write a work item so 001 exists
	wiPath := filepath.Join(workDir, "0_backlog", "001-test.md")
	require.NoError(t, os.WriteFile(wiPath, []byte("---\nid: \"001\"\ntitle: Test\nstatus: backlog\nkind: task\ncreated: 2026-01-01\n---\n"), 0o600))
	// Write ROADMAP.yml that references 001
	roadmapPath := filepath.Join(dir, "ROADMAP.yml")
	require.NoError(t, os.WriteFile(roadmapPath, []byte("roadmap:\n  - id: \"001\"\n"), 0o600))

	// LoadConfig from dir (we need kira.yml for checkWorkDir)
	require.NoError(t, os.MkdirAll(workDir, 0o700))
	configPath := filepath.Join(dir, "kira.yml")
	require.NoError(t, os.WriteFile(configPath, []byte("version: \"1.0\"\nstatus_folders:\n  backlog: 0_backlog\n  todo: 1_todo\n  doing: 2_doing\n  review: 3_review\n  done: 4_done\n  archived: z_archive\n"), 0o600))

	// Run lint via root so config is loaded for the correct dir
	rootCmd.SetArgs([]string{"roadmap", "lint"})
	err := rootCmd.Execute()
	assert.NoError(t, err)
}

func TestRoadmapLint_BrokenRef(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.Chdir(dir))
	defer func() { _ = os.Chdir(t.TempDir()) }()

	workDir := filepath.Join(dir, ".work")
	require.NoError(t, os.MkdirAll(workDir, 0o700))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "kira.yml"), []byte("version: \"1.0\"\nstatus_folders:\n  backlog: 0_backlog\n  todo: 1_todo\n  doing: 2_doing\n  review: 3_review\n  done: 4_done\n  archived: z_archive\n"), 0o600))
	roadmapPath := filepath.Join(dir, "ROADMAP.yml")
	require.NoError(t, os.WriteFile(roadmapPath, []byte("roadmap:\n  - id: \"999\"\n"), 0o600))

	rootCmd.SetArgs([]string{"roadmap", "lint"})
	err := rootCmd.Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "broken references")
}

func TestRoadmapLint_SchemaError(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.Chdir(dir))
	defer func() { _ = os.Chdir(t.TempDir()) }()

	workDir := filepath.Join(dir, ".work")
	require.NoError(t, os.MkdirAll(workDir, 0o700))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "kira.yml"), []byte("version: \"1.0\"\nstatus_folders:\n  backlog: 0_backlog\n  todo: 1_todo\n  doing: 2_doing\n  review: 3_review\n  done: 4_done\n  archived: z_archive\n"), 0o600))
	roadmapPath := filepath.Join(dir, "ROADMAP.yml")
	// Empty entry is invalid
	require.NoError(t, os.WriteFile(roadmapPath, []byte("roadmap:\n  - {}\n"), 0o600))

	rootCmd.SetArgs([]string{"roadmap", "lint"})
	err := rootCmd.Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "schema validation failed")
}

func TestRoadmapLint_FileNotFound(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.Chdir(dir))
	defer func() { _ = os.Chdir(t.TempDir()) }()

	workDir := filepath.Join(dir, ".work")
	require.NoError(t, os.MkdirAll(workDir, 0o700))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "kira.yml"), []byte("version: \"1.0\"\nstatus_folders:\n  backlog: 0_backlog\n  todo: 1_todo\n  doing: 2_doing\n  review: 3_review\n  done: 4_done\n  archived: z_archive\n"), 0o600))
	// No ROADMAP.yml
	rootCmd.SetArgs([]string{"roadmap", "lint"})
	err := rootCmd.Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}
