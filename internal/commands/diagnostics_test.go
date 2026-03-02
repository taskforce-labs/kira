package commands

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// TestEnvironmentDiagnostics logs software that kira relies on (Go, Git, etc.).
// Run with -v to see output; useful for comparing local vs CI (e.g. go test -run TestEnvironmentDiagnostics ./internal/commands/ -v).
func TestEnvironmentDiagnostics(t *testing.T) {
	t.Logf("go_version=%s", runtime.Version())
	t.Logf("go_os=%s go_arch=%s", runtime.GOOS, runtime.GOARCH)

	if out, err := execCommandOutput("git", "version"); err == nil {
		t.Logf("git_version=%s", strings.TrimSpace(out))
	} else {
		t.Logf("git_version=error:%v", err)
	}
	// Git default branch for new repos (unset -> "master" on many systems; "main" if configured).
	if out, err := execCommandOutput("git", "config", "--global", "init.defaultBranch"); err == nil {
		t.Logf("git_init_default_branch=%s", strings.TrimSpace(out))
	} else {
		t.Logf("git_init_default_branch=unset (git uses built-in default)")
	}

	// Required Go version from go.mod (best-effort: find module root and read go.mod).
	if goModPath, err := findGoMod(); err == nil {
		if b, err := os.ReadFile(goModPath); err == nil { // #nosec G304 -- goModPath is from findGoMod (module root only), test-only
			for _, line := range strings.Split(string(b), "\n") {
				line = strings.TrimSpace(line)
				if strings.HasPrefix(line, "go ") && !strings.HasPrefix(line, "go mod") {
					t.Logf("go_mod_require=%s", strings.TrimSpace(line))
					break
				}
			}
		}
	}
}

func execCommandOutput(name string, args ...string) (string, error) {
	cmd := exec.Command(name, args...) // #nosec G204 -- test-only: name/args are fixed (git version, git config)
	out, err := cmd.Output()
	return string(out), err
}

func findGoMod() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for {
		p := filepath.Join(dir, "go.mod")
		if _, err := os.Stat(p); err == nil {
			return p, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", os.ErrNotExist
		}
		dir = parent
	}
}
