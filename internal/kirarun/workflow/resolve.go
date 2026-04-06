package workflow

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"kira/internal/config"
)

// Resolve turns a CLI selector into an absolute workflow path and a display name for run-id derivation.
func Resolve(cfg *config.Config, selector string) (abs, displayName string, err error) {
	if cfg == nil {
		return "", "", fmt.Errorf("nil config")
	}
	root, err := workflowsRootAbs(cfg)
	if err != nil {
		return "", "", err
	}
	sel := strings.TrimSpace(selector)
	if sel == "" {
		return "", "", fmt.Errorf("workflow selector is empty")
	}

	// Explicit path (absolute or relative to project root).
	if filepath.IsAbs(sel) {
		abs, err = ensureUnderRoot(root, sel)
		if err != nil {
			return "", "", err
		}
		return abs, displayBase(abs), nil
	}
	if strings.Contains(sel, "/") || strings.Contains(sel, string(filepath.Separator)) {
		joined := filepath.Join(cfg.ConfigDir, filepath.Clean(sel))
		abs, err = ensureUnderRoot(root, joined)
		if err != nil {
			return "", "", err
		}
		return abs, displayBase(abs), nil
	}

	// Named workflow from kira.yml.
	if cfg.Workflows != nil {
		if rel, ok := cfg.Workflows.Scripts[sel]; ok {
			joined := filepath.Join(root, filepath.Clean(rel))
			abs, err = ensureUnderRoot(root, joined)
			if err != nil {
				return "", "", err
			}
			return abs, sel, nil
		}
	}

	// Default: <name>.go under workflows root.
	base := sel
	if !strings.HasSuffix(base, ".go") {
		base += ".go"
	}
	joined := filepath.Join(root, filepath.Clean(base))
	abs, err = ensureUnderRoot(root, joined)
	if err != nil {
		return "", "", err
	}
	return abs, strings.TrimSuffix(filepath.Base(base), ".go"), nil
}

func workflowsRootAbs(cfg *config.Config) (string, error) {
	sub := ".workflows"
	if cfg.Workflows != nil && strings.TrimSpace(cfg.Workflows.Root) != "" {
		sub = cfg.Workflows.Root
	}
	return filepath.Abs(filepath.Join(cfg.ConfigDir, sub))
}

func ensureUnderRoot(root, candidate string) (string, error) {
	abs, err := filepath.Abs(candidate)
	if err != nil {
		return "", err
	}
	rootAbs, err := filepath.Abs(root)
	if err != nil {
		return "", err
	}
	st, err := os.Stat(abs)
	if err != nil {
		return "", fmt.Errorf("workflow file: %w", err)
	}
	if st.IsDir() {
		return "", fmt.Errorf("workflow path must be a file: %s", abs)
	}
	rel, err := filepath.Rel(rootAbs, abs)
	if err != nil {
		return "", err
	}
	if rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("workflow must be under %s", rootAbs)
	}
	return abs, nil
}

func displayBase(abs string) string {
	return strings.TrimSuffix(filepath.Base(abs), ".go")
}
