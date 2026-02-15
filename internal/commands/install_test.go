package commands

import (
	"os"
	"path/filepath"
	"testing"

	"kira/internal/config"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateSkillFrontmatter(t *testing.T) {
	t.Run("valid frontmatter and folder name", func(t *testing.T) {
		content := []byte(`---
name: product-discovery
description: Guide through product discovery.
---
# Skill
`)
		err := validateSkillFrontmatter("kira-product-discovery", content)
		require.NoError(t, err)
	})

	t.Run("frontmatter name with kira- prefix", func(t *testing.T) {
		content := []byte(`---
name: kira-product-discovery
description: Guide.
---
# Skill
`)
		err := validateSkillFrontmatter("kira-product-discovery", content)
		require.NoError(t, err)
	})

	t.Run("missing name", func(t *testing.T) {
		content := []byte(`---
description: Guide only.
---
# Skill
`)
		err := validateSkillFrontmatter("kira-product-discovery", content)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "name")
	})

	t.Run("missing description", func(t *testing.T) {
		content := []byte(`---
name: product-discovery
---
# Skill
`)
		err := validateSkillFrontmatter("kira-product-discovery", content)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "description")
	})

	t.Run("folder name mismatch", func(t *testing.T) {
		content := []byte(`---
name: other-skill
description: Other.
---
# Skill
`)
		err := validateSkillFrontmatter("kira-product-discovery", content)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "does not match")
	})

	t.Run("no frontmatter", func(t *testing.T) {
		content := []byte("# No frontmatter\n")
		err := validateSkillFrontmatter("kira-product-discovery", content)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "no frontmatter")
	})
}

func TestValidatePathUnder(t *testing.T) {
	tmpDir := t.TempDir()
	baseAbs, err := filepath.Abs(tmpDir)
	require.NoError(t, err)

	t.Run("target under base", func(t *testing.T) {
		target := filepath.Join(tmpDir, "sub", "file")
		err := validatePathUnder(baseAbs, target)
		require.NoError(t, err)
	})

	t.Run("target is base", func(t *testing.T) {
		err := validatePathUnder(baseAbs, tmpDir)
		require.NoError(t, err)
	})

	t.Run("target outside base", func(t *testing.T) {
		err := validatePathUnder(baseAbs, filepath.Join(tmpDir, "..", "other"))
		require.Error(t, err)
		assert.Contains(t, err.Error(), "outside")
	})
}

func TestValidateCommandMarkdown(t *testing.T) {
	t.Run("valid content", func(t *testing.T) {
		err := validateCommandMarkdown([]byte("# Command\n\nContent here."))
		require.NoError(t, err)
	})
	t.Run("empty file", func(t *testing.T) {
		err := validateCommandMarkdown([]byte{})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "empty")
	})
	t.Run("whitespace only", func(t *testing.T) {
		err := validateCommandMarkdown([]byte("   \n\t  "))
		require.Error(t, err)
		assert.Contains(t, err.Error(), "no content")
	})
}

func TestRunInstallCursorSkills(t *testing.T) {
	t.Run("installs to configured path and creates skills", func(t *testing.T) {
		tmpDir := t.TempDir()
		cfgDir := t.TempDir()
		kiraYml := `version: "1.0"
cursor_install:
  base_path: ` + "\"" + tmpDir + "\"" + "\n"
		require.NoError(t, os.WriteFile(filepath.Join(cfgDir, "kira.yml"), []byte(kiraYml), 0o600))
		origWd, err := os.Getwd()
		require.NoError(t, err)
		defer func() { _ = os.Chdir(origWd) }()
		require.NoError(t, os.Chdir(cfgDir))

		installCmd := installCursorSkillsCmd
		require.NoError(t, installCmd.Flags().Set("force", "true"))
		err = runInstallCursorSkills(installCmd, nil)
		require.NoError(t, err)

		skillsPath := filepath.Join(tmpDir, ".agent", "skills")
		entries, err := os.ReadDir(skillsPath)
		require.NoError(t, err)
		var skillDirs []string
		for _, e := range entries {
			if e.IsDir() {
				skillDirs = append(skillDirs, e.Name())
			}
		}
		require.Contains(t, skillDirs, "kira-work-item-elaboration")
		skillPath := filepath.Join(skillsPath, "kira-work-item-elaboration", "SKILL.md")
		// #nosec G304 - path is built from test temp dir and fixed segments
		data, err := os.ReadFile(skillPath)
		require.NoError(t, err)
		assert.Contains(t, string(data), "name: work-item-elaboration")
	})
}

func TestRunInstallCursorCommands(t *testing.T) {
	t.Run("installs to configured path and creates commands", func(t *testing.T) {
		tmpDir := t.TempDir()
		cfgDir := t.TempDir()
		kiraYml := `version: "1.0"
cursor_install:
  base_path: ` + "\"" + tmpDir + "\"" + "\n"
		require.NoError(t, os.WriteFile(filepath.Join(cfgDir, "kira.yml"), []byte(kiraYml), 0o600))
		origWd, err := os.Getwd()
		require.NoError(t, err)
		defer func() { _ = os.Chdir(origWd) }()
		require.NoError(t, os.Chdir(cfgDir))

		cmd := installCursorCommandsCmd
		require.NoError(t, cmd.Flags().Set("force", "true"))
		err = runInstallCursorCommands(cmd, nil)
		require.NoError(t, err)

		commandsPath := filepath.Join(tmpDir, ".cursor", "commands")
		entries, err := os.ReadDir(commandsPath)
		require.NoError(t, err)
		var files []string
		for _, e := range entries {
			if !e.IsDir() {
				files = append(files, e.Name())
			}
		}
		require.Contains(t, files, "kira-elaborate-work-item.md")
		cmdPath := filepath.Join(commandsPath, "kira-elaborate-work-item.md")
		// #nosec G304 - path is from test temp dir and fixed segment
		data, err := os.ReadFile(cmdPath)
		require.NoError(t, err)
		assert.Contains(t, string(data), "# Elaborate Work Item")
	})
}

func TestEnsureCursorSkillsInstalled(t *testing.T) {
	t.Run("installs when skills path is empty", func(t *testing.T) {
		tmpDir := t.TempDir()
		cfg := &config.Config{CursorInstall: &config.CursorInstallConfig{BasePath: tmpDir}}
		err := EnsureCursorSkillsInstalled(cfg)
		require.NoError(t, err)
		skillsPath := filepath.Join(tmpDir, ".agent", "skills")
		entries, err := os.ReadDir(skillsPath)
		require.NoError(t, err)
		var dirs []string
		for _, e := range entries {
			if e.IsDir() {
				dirs = append(dirs, e.Name())
			}
		}
		require.Contains(t, dirs, "kira-work-item-elaboration")
	})
	t.Run("no-op when all skills already present", func(t *testing.T) {
		tmpDir := t.TempDir()
		cfg := &config.Config{CursorInstall: &config.CursorInstallConfig{BasePath: tmpDir}}
		err := EnsureCursorSkillsInstalled(cfg)
		require.NoError(t, err)
		// run again; should be no-op (skills already there)
		err = EnsureCursorSkillsInstalled(cfg)
		require.NoError(t, err)
	})
	t.Run("repairs when SKILL.md is missing from a skill directory", func(t *testing.T) {
		tmpDir := t.TempDir()
		cfg := &config.Config{CursorInstall: &config.CursorInstallConfig{BasePath: tmpDir}}
		// Do a valid install first
		err := EnsureCursorSkillsInstalled(cfg)
		require.NoError(t, err)
		// Remove SKILL.md from one skill to simulate corruption
		skillMDPath := filepath.Join(tmpDir, ".agent", "skills", "kira-work-item-elaboration", "SKILL.md")
		require.NoError(t, os.Remove(skillMDPath))
		// Ensure detects the corruption and repairs it
		err = EnsureCursorSkillsInstalled(cfg)
		require.NoError(t, err)
		// Verify SKILL.md was restored
		// #nosec G304 - path is built from test temp dir and fixed segments
		data, err := os.ReadFile(skillMDPath)
		require.NoError(t, err)
		assert.Contains(t, string(data), "name: work-item-elaboration")
	})
	t.Run("repairs when SKILL.md is empty", func(t *testing.T) {
		tmpDir := t.TempDir()
		cfg := &config.Config{CursorInstall: &config.CursorInstallConfig{BasePath: tmpDir}}
		// Do a valid install first
		err := EnsureCursorSkillsInstalled(cfg)
		require.NoError(t, err)
		// Truncate SKILL.md to simulate corruption
		skillMDPath := filepath.Join(tmpDir, ".agent", "skills", "kira-work-item-elaboration", "SKILL.md")
		require.NoError(t, os.WriteFile(skillMDPath, []byte{}, 0o600))
		// Ensure detects the corruption and repairs it
		err = EnsureCursorSkillsInstalled(cfg)
		require.NoError(t, err)
		// Verify SKILL.md was restored with valid content
		// #nosec G304 - path is built from test temp dir and fixed segments
		data, err := os.ReadFile(skillMDPath)
		require.NoError(t, err)
		assert.Contains(t, string(data), "name: work-item-elaboration")
	})
	t.Run("repairs when SKILL.md has invalid frontmatter", func(t *testing.T) {
		tmpDir := t.TempDir()
		cfg := &config.Config{CursorInstall: &config.CursorInstallConfig{BasePath: tmpDir}}
		// Do a valid install first
		err := EnsureCursorSkillsInstalled(cfg)
		require.NoError(t, err)
		// Overwrite SKILL.md with invalid frontmatter
		skillMDPath := filepath.Join(tmpDir, ".agent", "skills", "kira-work-item-elaboration", "SKILL.md")
		require.NoError(t, os.WriteFile(skillMDPath, []byte("# No frontmatter here\n"), 0o600))
		// Ensure detects the corruption and repairs it
		err = EnsureCursorSkillsInstalled(cfg)
		require.NoError(t, err)
		// Verify SKILL.md was restored with valid content
		// #nosec G304 - path is built from test temp dir and fixed segments
		data, err := os.ReadFile(skillMDPath)
		require.NoError(t, err)
		assert.Contains(t, string(data), "name: work-item-elaboration")
	})
}

func TestEnsureCursorCommandsInstalled(t *testing.T) {
	t.Run("installs when commands path is empty", func(t *testing.T) {
		tmpDir := t.TempDir()
		cfg := &config.Config{CursorInstall: &config.CursorInstallConfig{BasePath: tmpDir}}
		err := EnsureCursorCommandsInstalled(cfg)
		require.NoError(t, err)
		commandsPath := filepath.Join(tmpDir, ".cursor", "commands")
		entries, err := os.ReadDir(commandsPath)
		require.NoError(t, err)
		var files []string
		for _, e := range entries {
			if !e.IsDir() {
				files = append(files, e.Name())
			}
		}
		require.Contains(t, files, "kira-elaborate-work-item.md")
	})
	t.Run("no-op when all commands already present", func(t *testing.T) {
		tmpDir := t.TempDir()
		cfg := &config.Config{CursorInstall: &config.CursorInstallConfig{BasePath: tmpDir}}
		err := EnsureCursorCommandsInstalled(cfg)
		require.NoError(t, err)
		err = EnsureCursorCommandsInstalled(cfg)
		require.NoError(t, err)
	})
	t.Run("repairs when command file is empty", func(t *testing.T) {
		tmpDir := t.TempDir()
		cfg := &config.Config{CursorInstall: &config.CursorInstallConfig{BasePath: tmpDir}}
		// Do a valid install first
		err := EnsureCursorCommandsInstalled(cfg)
		require.NoError(t, err)
		// Truncate a command file to simulate corruption
		cmdPath := filepath.Join(tmpDir, ".cursor", "commands", "kira-elaborate-work-item.md")
		require.NoError(t, os.WriteFile(cmdPath, []byte{}, 0o600))
		// Ensure detects the corruption and repairs it
		err = EnsureCursorCommandsInstalled(cfg)
		require.NoError(t, err)
		// Verify command file was restored with valid content
		// #nosec G304 - path is built from test temp dir and fixed segments
		data, err := os.ReadFile(cmdPath)
		require.NoError(t, err)
		assert.Contains(t, string(data), "# Elaborate Work Item")
	})
	t.Run("repairs when command file is whitespace only", func(t *testing.T) {
		tmpDir := t.TempDir()
		cfg := &config.Config{CursorInstall: &config.CursorInstallConfig{BasePath: tmpDir}}
		// Do a valid install first
		err := EnsureCursorCommandsInstalled(cfg)
		require.NoError(t, err)
		// Write whitespace-only content to simulate corruption
		cmdPath := filepath.Join(tmpDir, ".cursor", "commands", "kira-elaborate-work-item.md")
		require.NoError(t, os.WriteFile(cmdPath, []byte("   \n\t  \n"), 0o600))
		// Ensure detects the corruption and repairs it
		err = EnsureCursorCommandsInstalled(cfg)
		require.NoError(t, err)
		// Verify command file was restored with valid content
		// #nosec G304 - path is built from test temp dir and fixed segments
		data, err := os.ReadFile(cmdPath)
		require.NoError(t, err)
		assert.Contains(t, string(data), "# Elaborate Work Item")
	})
}

func TestListExistingKiraSkills(t *testing.T) {
	t.Run("only detects bundled skills", func(t *testing.T) {
		tmpDir := t.TempDir()
		skillsPath := filepath.Join(tmpDir, ".agent", "skills")
		require.NoError(t, os.MkdirAll(skillsPath, 0o700))

		// Create an unrelated kira- directory that is not bundled
		unrelatedDir := filepath.Join(skillsPath, "kira-user-custom-skill")
		require.NoError(t, os.MkdirAll(unrelatedDir, 0o700))

		existing, err := listExistingKiraSkills(skillsPath)
		require.NoError(t, err)
		// Should not include the unrelated directory
		assert.NotContains(t, existing, "kira-user-custom-skill")
	})

	t.Run("detects bundled skills when present", func(t *testing.T) {
		tmpDir := t.TempDir()
		skillsPath := filepath.Join(tmpDir, ".agent", "skills")
		require.NoError(t, os.MkdirAll(skillsPath, 0o700))

		// Create a bundled skill directory
		bundledDir := filepath.Join(skillsPath, "kira-work-item-elaboration")
		require.NoError(t, os.MkdirAll(bundledDir, 0o700))
		require.NoError(t, os.WriteFile(filepath.Join(bundledDir, "SKILL.md"), []byte("---\nname: work-item-elaboration\n---\n"), 0o600))

		existing, err := listExistingKiraSkills(skillsPath)
		require.NoError(t, err)
		assert.Contains(t, existing, "kira-work-item-elaboration")
	})
}

func TestListExistingKiraCommands(t *testing.T) {
	t.Run("only detects bundled commands", func(t *testing.T) {
		tmpDir := t.TempDir()
		commandsPath := filepath.Join(tmpDir, ".cursor", "commands")
		require.NoError(t, os.MkdirAll(commandsPath, 0o700))

		// Create an unrelated kira-*.md file that is not bundled
		unrelatedFile := filepath.Join(commandsPath, "kira-user-custom-command.md")
		require.NoError(t, os.WriteFile(unrelatedFile, []byte("# Custom Command"), 0o600))

		existing, err := listExistingKiraCommands(commandsPath)
		require.NoError(t, err)
		// Should not include the unrelated file
		assert.NotContains(t, existing, "kira-user-custom-command.md")
	})

	t.Run("detects bundled commands when present", func(t *testing.T) {
		tmpDir := t.TempDir()
		commandsPath := filepath.Join(tmpDir, ".cursor", "commands")
		require.NoError(t, os.MkdirAll(commandsPath, 0o700))

		// Create a bundled command file
		bundledFile := filepath.Join(commandsPath, "kira-elaborate-work-item.md")
		require.NoError(t, os.WriteFile(bundledFile, []byte("# Elaborate Work Item"), 0o600))

		existing, err := listExistingKiraCommands(commandsPath)
		require.NoError(t, err)
		assert.Contains(t, existing, "kira-elaborate-work-item.md")
	})
}
