package commands

import (
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"kira/internal/config"
	"kira/internal/roadmap"
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

func TestRoadmapApply_DryRun(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.Chdir(dir))
	defer func() { _ = os.Chdir(t.TempDir()) }()

	require.NoError(t, os.MkdirAll(filepath.Join(dir, ".work"), 0o700))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "kira.yml"), []byte("version: \"1.0\"\ntemplates:\n  task: templates/template.task.md\nstatus_folders:\n  backlog: 0_backlog\n  todo: 1_todo\n  doing: 2_doing\n  review: 3_review\n  done: 4_done\n  archived: z_archive\n"), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "ROADMAP.yml"), []byte("roadmap:\n  - title: My ad-hoc item\n    meta:\n      period: Q1-26\n"), 0o600))

	rootCmd.SetArgs([]string{"roadmap", "apply", "--dry-run"})
	err := rootCmd.Execute()
	require.NoError(t, err)
}

func TestRoadmapApply_Promote(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.Chdir(dir))
	defer func() { _ = os.Chdir(t.TempDir()) }()

	workDir := filepath.Join(dir, ".work")
	backlogDir := filepath.Join(workDir, "0_backlog")
	templateDir := filepath.Join(workDir, "templates")
	require.NoError(t, os.MkdirAll(backlogDir, 0o700))
	require.NoError(t, os.MkdirAll(templateDir, 0o700))
	// Minimal task template (only required inputs for ProcessTemplate)
	minimalTask := `---
id: <!--input-number:id:"Task ID"-->
title: <!--input-string:title:"Task title"-->
status: <!--input-string[backlog,todo,doing,review,done,released,abandoned,archived]:status:"Current status"-->
kind: task
created: <!--input-datetime[yyyy-mm-dd]:created:"Creation date"-->
---
# <!--input-string:title:"Task title"`
	require.NoError(t, os.WriteFile(filepath.Join(templateDir, "template.task.md"), []byte(minimalTask), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "kira.yml"), []byte("version: \"1.0\"\ntemplates:\n  task: templates/template.task.md\nstatus_folders:\n  backlog: 0_backlog\n  todo: 1_todo\n  doing: 2_doing\n  review: 3_review\n  done: 4_done\n  archived: z_archive\n"), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "ROADMAP.yml"), []byte("roadmap:\n  - title: Promoted item\n    meta:\n      workstream: auth\n"), 0o600))

	rootCmd.SetArgs([]string{"roadmap", "apply"})
	_ = roadmapApplyCmd.Flags().Set("dry-run", "false")
	err := rootCmd.Execute()
	require.NoError(t, err)

	// Check work item was created in backlog
	entries, _ := os.ReadDir(backlogDir)
	require.GreaterOrEqual(t, len(entries), 1)
	// Check ROADMAP.yml was updated (ad-hoc replaced with id)
	// #nosec G304 - dir is from t.TempDir(), path is under it
	data, _ := os.ReadFile(filepath.Join(dir, "ROADMAP.yml"))
	assert.Contains(t, string(data), "id:")
	assert.NotContains(t, string(data), "title: Promoted item")
}

func TestRoadmapDraft_Empty(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.Chdir(dir))
	defer func() { _ = os.Chdir(t.TempDir()) }()

	require.NoError(t, os.MkdirAll(filepath.Join(dir, ".work"), 0o700))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "kira.yml"), []byte("version: \"1.0\"\nstatus_folders:\n  backlog: 0_backlog\n  todo: 1_todo\n  doing: 2_doing\n  review: 3_review\n  done: 4_done\n  archived: z_archive\n"), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "ROADMAP.yml"), []byte("roadmap:\n  - id: \"001\"\n"), 0o600))

	rootCmd.SetArgs([]string{"roadmap", "draft", "empty-test", "--empty"})
	err := rootCmd.Execute()
	require.NoError(t, err)

	draftPath := filepath.Join(dir, "ROADMAP-empty-test.yml")
	require.FileExists(t, draftPath)
	// #nosec G304 - dir is from t.TempDir()
	data, err := os.ReadFile(draftPath)
	require.NoError(t, err)
	assert.Contains(t, string(data), "roadmap:")
	// Empty draft has roadmap key; content may be null or []
	assert.Contains(t, string(data), "roadmap")
}

func TestRoadmapDraft_IncludeAll(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.Chdir(dir))
	defer func() { _ = os.Chdir(t.TempDir()) }()

	workDir := filepath.Join(dir, ".work")
	backlogDir := filepath.Join(workDir, "0_backlog")
	doneDir := filepath.Join(workDir, "4_done")
	require.NoError(t, os.MkdirAll(backlogDir, 0o700))
	require.NoError(t, os.MkdirAll(doneDir, 0o700))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "kira.yml"), []byte("version: \"1.0\"\nstatus_folders:\n  backlog: 0_backlog\n  todo: 1_todo\n  doing: 2_doing\n  review: 3_review\n  done: 4_done\n  archived: z_archive\n"), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(backlogDir, "001-backlog.md"), []byte("---\nid: \"001\"\ntitle: Backlog\nstatus: backlog\n---\n"), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(doneDir, "002-done.md"), []byte("---\nid: \"002\"\ntitle: Done\nstatus: done\n---\n"), 0o600))
	roadmapPath := filepath.Join(dir, "ROADMAP.yml")
	require.NoError(t, os.WriteFile(roadmapPath, []byte("roadmap:\n  - id: \"001\"\n  - id: \"002\"\n"), 0o600))
	// Ensure roadmap parses to 2 entries (test runs from dir)
	f, parseErr := roadmap.LoadFile(dir, roadmapPath)
	require.NoError(t, parseErr)
	require.Len(t, f.Roadmap, 2, "ROADMAP.yml must have 2 entries")

	rootCmd.SetArgs([]string{"roadmap", "draft", "include-all-test", "--include-all"})
	err := rootCmd.Execute()
	require.NoError(t, err)

	draftPath := filepath.Join(dir, "ROADMAP-include-all-test.yml")
	require.FileExists(t, draftPath)
	// #nosec G304 - dir is from t.TempDir()
	data, err := os.ReadFile(draftPath)
	require.NoError(t, err)
	assert.Contains(t, string(data), "001")
	assert.Contains(t, string(data), "002")
}

func TestPreparePromotePaths_ArchivePath(t *testing.T) {
	dir := t.TempDir()
	cfg := &config.Config{
		ConfigDir:     dir,
		StatusFolders: map[string]string{"archived": "z_archive"},
	}
	// GetWorkFolderPath default is .work
	roadmapDir, draftPath, archivePath, err := preparePromotePaths(cfg, "mydraft")
	require.NoError(t, err)
	assert.Equal(t, filepath.Join(dir, "ROADMAP.yml"), roadmapDir)
	assert.Equal(t, filepath.Join(dir, "ROADMAP-mydraft.yml"), draftPath)
	assert.Contains(t, archivePath, filepath.Join(dir, ".work", "z_archive", "roadmap"))
	assert.Regexp(t, regexp.MustCompile(`ROADMAP-\d{4}-\d{2}-\d{2}T\d{6}Z\.yml$`), filepath.Base(archivePath))
}

func TestRoadmapPromote_Integration(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.Chdir(dir))
	defer func() { _ = os.Chdir(t.TempDir()) }()

	workDir := filepath.Join(dir, ".work")
	archiveDir := filepath.Join(workDir, "z_archive", "roadmap")
	require.NoError(t, os.MkdirAll(archiveDir, 0o700))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "kira.yml"), []byte("version: \"1.0\"\nstatus_folders:\n  backlog: 0_backlog\n  todo: 1_todo\n  doing: 2_doing\n  review: 3_review\n  done: 4_done\n  archived: z_archive\n"), 0o600))
	currentContent := "roadmap:\n  - id: \"001\"\n"
	draftContent := "roadmap:\n  - id: \"002\"\n"
	require.NoError(t, os.WriteFile(filepath.Join(dir, "ROADMAP.yml"), []byte(currentContent), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "ROADMAP-promo.yml"), []byte(draftContent), 0o600))

	// Init git and track files so promote can stage draft deletion
	// #nosec G204 - test helper
	initCmd := exec.Command("git", "init")
	initCmd.Dir = dir
	require.NoError(t, initCmd.Run())
	// #nosec G204 - test helper
	configCmd := exec.Command("git", "config", "user.email", "test@test.com")
	configCmd.Dir = dir
	require.NoError(t, configCmd.Run())
	// #nosec G204 - test helper
	configCmd = exec.Command("git", "config", "user.name", "Test")
	configCmd.Dir = dir
	require.NoError(t, configCmd.Run())
	// #nosec G204 - test helper
	addCmd := exec.Command("git", "add", "ROADMAP.yml", "ROADMAP-promo.yml")
	addCmd.Dir = dir
	require.NoError(t, addCmd.Run())

	rootCmd.SetArgs([]string{"roadmap", "promote", "promo", "--yes"})
	err := rootCmd.Execute()
	require.NoError(t, err)

	// Previous ROADMAP.yml should be in archive
	archives, err := os.ReadDir(archiveDir)
	require.NoError(t, err)
	require.GreaterOrEqual(t, len(archives), 1)
	var archivedData []byte
	for _, f := range archives {
		if filepath.Ext(f.Name()) == ".yml" {
			// #nosec G304 - archiveDir from t.TempDir(), f.Name() from ReadDir
			archivedData, _ = os.ReadFile(filepath.Join(archiveDir, f.Name()))
			break
		}
	}
	require.NotEmpty(t, archivedData)
	assert.Contains(t, string(archivedData), "001")
	// Current ROADMAP.yml should have draft content
	// #nosec G304 - dir is from t.TempDir()
	data, err := os.ReadFile(filepath.Join(dir, "ROADMAP.yml"))
	require.NoError(t, err)
	assert.Equal(t, draftContent, string(data))
	// Draft file should be gone
	_, err = os.Stat(filepath.Join(dir, "ROADMAP-promo.yml"))
	assert.True(t, os.IsNotExist(err))
}
