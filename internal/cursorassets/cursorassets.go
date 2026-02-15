// Package cursorassets provides access to bundled Cursor skills and commands embedded in the binary.
package cursorassets

import (
	"embed"
	"fmt"
	"io/fs"
	"path"
	"strings"
)

//go:embed skills commands
var assetsFS embed.FS

const (
	skillsDir   = "skills"
	commandsDir = "commands"
	skillFile   = "SKILL.md"
)

// ListSkills returns the names of bundled skills (directory names under skills/, e.g. kira-product-discovery).
func ListSkills() ([]string, error) {
	entries, err := fs.ReadDir(assetsFS, skillsDir)
	if err != nil {
		return nil, fmt.Errorf("list skills: %w", err)
	}
	var names []string
	for _, e := range entries {
		if e.IsDir() && strings.HasPrefix(e.Name(), "kira-") {
			names = append(names, e.Name())
		}
	}
	return names, nil
}

// ListCommands returns the names of bundled commands (base names of .md files under commands/, e.g. kira-product-discovery).
func ListCommands() ([]string, error) {
	entries, err := fs.ReadDir(assetsFS, commandsDir)
	if err != nil {
		return nil, fmt.Errorf("list commands: %w", err)
	}
	var names []string
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".md") && strings.HasPrefix(e.Name(), "kira-") {
			names = append(names, strings.TrimSuffix(e.Name(), ".md"))
		}
	}
	return names, nil
}

// ReadSkillFile reads a file from a skill directory. name is the skill directory name (e.g. kira-product-discovery).
// relPath is the path relative to the skill dir (e.g. SKILL.md or scripts/foo.sh).
func ReadSkillFile(name, relPath string) ([]byte, error) {
	if name == "" || strings.Contains(name, "..") || strings.Contains(name, "\x00") {
		return nil, fmt.Errorf("invalid skill name: %q", name)
	}
	if strings.Contains(relPath, "..") || strings.Contains(relPath, "\x00") {
		return nil, fmt.Errorf("invalid relative path: %q", relPath)
	}
	fullPath := path.Join(skillsDir, name, relPath)
	// Ensure path stays under skills/name
	if !strings.HasPrefix(fullPath, skillsDir+"/") {
		return nil, fmt.Errorf("invalid skill path: %s", relPath)
	}
	data, err := fs.ReadFile(assetsFS, fullPath)
	if err != nil {
		return nil, fmt.Errorf("read skill file %s: %w", relPath, err)
	}
	return data, nil
}

// ReadSkillSKILL returns the SKILL.md content for a skill.
func ReadSkillSKILL(name string) ([]byte, error) {
	return ReadSkillFile(name, skillFile)
}

// ReadCommand returns the content of a bundled command file. name is the command name without .md (e.g. kira-product-discovery).
func ReadCommand(name string) ([]byte, error) {
	if name == "" || strings.Contains(name, "..") || strings.Contains(name, "\x00") {
		return nil, fmt.Errorf("invalid command name: %q", name)
	}
	filename := name + ".md"
	fullPath := path.Join(commandsDir, filename)
	if !strings.HasPrefix(fullPath, commandsDir+"/") {
		return nil, fmt.Errorf("invalid command path: %s", filename)
	}
	data, err := fs.ReadFile(assetsFS, fullPath)
	if err != nil {
		return nil, fmt.Errorf("read command %s: %w", name, err)
	}
	return data, nil
}

// skillEntries returns all entries (files and dirs) under a skill directory for copying.
func skillEntries(name string) ([]fs.DirEntry, error) {
	if name == "" || strings.Contains(name, "..") || strings.Contains(name, "\x00") {
		return nil, fmt.Errorf("invalid skill name: %q", name)
	}
	dir := path.Join(skillsDir, name)
	if !strings.HasPrefix(dir, skillsDir+"/") {
		return nil, fmt.Errorf("invalid skill path: %s", name)
	}
	entries, err := fs.ReadDir(assetsFS, dir)
	if err != nil {
		return nil, fmt.Errorf("read skill dir %s: %w", name, err)
	}
	return entries, nil
}

// ListSkillFilePaths returns relative paths of all files under a skill directory (e.g. SKILL.md, scripts/foo.sh).
func ListSkillFilePaths(name string) ([]string, error) {
	if name == "" || strings.Contains(name, "..") || strings.Contains(name, "\x00") {
		return nil, fmt.Errorf("invalid skill name: %q", name)
	}
	dir := path.Join(skillsDir, name)
	if !strings.HasPrefix(dir, skillsDir+"/") {
		return nil, fmt.Errorf("invalid skill path: %s", name)
	}
	var paths []string
	dirWithSep := dir + "/"
	err := fs.WalkDir(assetsFS, dir, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		if !strings.HasPrefix(p, dirWithSep) && p != dir {
			return nil
		}
		rel := p
		if strings.HasPrefix(p, dirWithSep) {
			rel = p[len(dirWithSep):]
		}
		paths = append(paths, rel)
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("walk skill %s: %w", name, err)
	}
	return paths, nil
}
