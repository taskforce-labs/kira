package commands

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"kira/internal/config"
)

func TestConfigGetUnknownKey(t *testing.T) {
	tmpDir := t.TempDir()
	originalDir, err := os.Getwd()
	require.NoError(t, err)
	defer func() {
		_ = os.Chdir(originalDir)
	}()

	err = os.Chdir(tmpDir)
	require.NoError(t, err)

	configContent := `version: "1.0"
`
	err = os.WriteFile("kira.yml", []byte(configContent), 0o600)
	require.NoError(t, err)

	err = runConfigGet(configGetCmd, []string{"unknown_key"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unknown key: 'unknown_key'")
	assert.Contains(t, err.Error(), "Valid keys:")
}

func TestConfigGetHelp(t *testing.T) {
	cmd := configGetCmd
	assert.NotNil(t, cmd)
	assert.Equal(t, "get", cmd.Name())
	assert.NotEmpty(t, cmd.Long)
	assert.Contains(t, cmd.Long, "trunk_branch")
	assert.Contains(t, cmd.Long, "Path syntax")
	assert.Contains(t, cmd.Long, "--project")
}

func TestConfigGetInvalidOutputFormat(t *testing.T) {
	tmpDir := t.TempDir()
	originalDir, err := os.Getwd()
	require.NoError(t, err)
	defer func() {
		_ = os.Chdir(originalDir)
	}()

	err = os.Chdir(tmpDir)
	require.NoError(t, err)

	configContent := `version: "1.0"
`
	err = os.WriteFile("kira.yml", []byte(configContent), 0o600)
	require.NoError(t, err)

	cmd := configGetCmd
	cmd.SetArgs([]string{"--output", "invalid", "work_folder"})
	err = cmd.ParseFlags([]string{"--output", "invalid"})
	require.NoError(t, err)

	err = runConfigGet(cmd, []string{"work_folder"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid output format")
}

func TestConfigGetScalarKeys(t *testing.T) {
	tests := []struct {
		name     string
		key      string
		config   string
		expected string
		wantErr  bool
	}{
		{
			name:     "work_folder with default",
			key:      "work_folder",
			config:   `version: "1.0"`,
			expected: ".work\n",
		},
		{
			name: "work_folder with custom value",
			key:  "work_folder",
			config: `version: "1.0"
workspace:
  work_folder: ".custom-work"`,
			expected: ".custom-work\n",
		},
		{
			name:     "docs_folder with default",
			key:      "docs_folder",
			config:   `version: "1.0"`,
			expected: ".docs\n",
		},
		{
			name: "docs_folder with custom value",
			key:  "docs_folder",
			config: `version: "1.0"
docs_folder: ".custom-docs"`,
			expected: ".custom-docs\n",
		},
		{
			name:     "remote with default",
			key:      "remote",
			config:   `version: "1.0"`,
			expected: "origin\n",
		},
		{
			name: "remote with custom value",
			key:  "remote",
			config: `version: "1.0"
git:
  remote: "upstream"`,
			expected: "upstream\n",
		},
		{
			name:     "config_dir",
			key:      "config_dir",
			config:   `version: "1.0"`,
			expected: "", // Will be set to tmpDir
		},
		{
			name:     "work_folder_abs",
			key:      "work_folder_abs",
			config:   `version: "1.0"`,
			expected: "", // Will be set to absolute path
		},
		{
			name:     "docs_folder_abs",
			key:      "docs_folder_abs",
			config:   `version: "1.0"`,
			expected: "", // Will be set to absolute path
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			originalDir, err := os.Getwd()
			require.NoError(t, err)
			defer func() {
				_ = os.Chdir(originalDir)
			}()

			err = os.Chdir(tmpDir)
			require.NoError(t, err)

			err = os.WriteFile("kira.yml", []byte(tt.config), 0o600)
			require.NoError(t, err)

			oldStdout := os.Stdout
			r, w, pipeErr := os.Pipe()
			require.NoError(t, pipeErr)
			os.Stdout = w

			cmd := configGetCmd
			err = cmd.Flags().Set("output", "text")
			require.NoError(t, err)
			err = cmd.Flags().Set("project", "")
			require.NoError(t, err)
			cmd.SetArgs([]string{tt.key})
			err = cmd.ParseFlags([]string{})
			require.NoError(t, err)

			errChan := make(chan error, 1)
			go func() {
				errChan <- runConfigGet(cmd, []string{tt.key})
				_ = w.Close()
			}()

			var outputBytes bytes.Buffer
			_, _ = outputBytes.ReadFrom(r)
			os.Stdout = oldStdout
			output := outputBytes.String()

			err = <-errChan
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			if tt.expected != "" {
				assert.Equal(t, tt.expected, output)
			} else {
				assert.NotEmpty(t, output)
				if tt.key == "config_dir" || tt.key == "work_folder_abs" || tt.key == "docs_folder_abs" {
					assert.True(t, filepath.IsAbs(strings.TrimSpace(output)), "expected absolute path, got: %s", output)
				}
			}
		})
	}
}

func TestConfigGetTrunkBranch(t *testing.T) {
	tests := []struct {
		name         string
		config       string
		setupGit     bool
		createMain   bool
		createMaster bool
		wantErr      bool
		errContains  string
		expected     string
	}{
		{
			name:         "configured trunk_branch",
			config:       "version: \"1.0\"\ngit:\n  trunk_branch: \"develop\"",
			setupGit:     true,
			createMain:   false,
			createMaster: false,
			wantErr:      true,
			errContains:  "not found",
		},
		{
			name:         "auto-detect main",
			config:       "version: \"1.0\"",
			setupGit:     true,
			createMain:   true,
			createMaster: false,
			expected:     "main\n",
		},
		{
			name:         "auto-detect master",
			config:       "version: \"1.0\"",
			setupGit:     true,
			createMain:   false,
			createMaster: true,
			expected:     "master\n",
		},
		{
			name:         "not a git repo",
			config:       "version: \"1.0\"",
			setupGit:     false,
			createMain:   false,
			createMaster: false,
			wantErr:      true,
			errContains:  "not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			originalDir, err := os.Getwd()
			require.NoError(t, err)
			defer func() {
				_ = os.Chdir(originalDir)
			}()

			err = os.Chdir(tmpDir)
			require.NoError(t, err)

			err = os.WriteFile("kira.yml", []byte(tt.config), 0o600)
			require.NoError(t, err)

			if tt.setupGit {
				defaultBranch := "temp"
				if tt.createMain {
					defaultBranch = defaultTrunkBranch
				} else if tt.createMaster {
					defaultBranch = defaultMasterBranch
				}

				cmd := exec.Command("git", "init", "-b", defaultBranch)
				cmd.Dir = tmpDir
				err = cmd.Run()
				require.NoError(t, err)

				cmd = exec.Command("git", "config", "user.name", "test")
				cmd.Dir = tmpDir
				err = cmd.Run()
				require.NoError(t, err)

				cmd = exec.Command("git", "config", "user.email", "test@test.com")
				cmd.Dir = tmpDir
				err = cmd.Run()
				require.NoError(t, err)

				cmd = exec.Command("git", "commit", "--allow-empty", "-m", "initial")
				cmd.Dir = tmpDir
				err = cmd.Run()
				require.NoError(t, err)

				if tt.createMain && defaultBranch != defaultTrunkBranch {
					cmd = exec.Command("git", "branch", "-m", defaultTrunkBranch)
					cmd.Dir = tmpDir
					err = cmd.Run()
					require.NoError(t, err)
				}

				if tt.createMaster && defaultBranch != "master" {
					cmd = exec.Command("git", "branch", "-D", "main")
					cmd.Dir = tmpDir
					_ = cmd.Run()
					cmd = exec.Command("git", "branch", "-m", "master")
					cmd.Dir = tmpDir
					err = cmd.Run()
					require.NoError(t, err)
				}
			}

			oldStdout := os.Stdout
			r, w, pipeErr := os.Pipe()
			require.NoError(t, pipeErr)
			os.Stdout = w

			cmd := configGetCmd
			cmd.SetArgs([]string{"trunk_branch"})

			errChan := make(chan error, 1)
			go func() {
				errChan <- runConfigGet(cmd, []string{"trunk_branch"})
				_ = w.Close()
			}()

			var outputBytes bytes.Buffer
			_, _ = outputBytes.ReadFrom(r)
			os.Stdout = oldStdout
			output := outputBytes.String()

			err = <-errChan
			if tt.wantErr {
				require.Error(t, err)
				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains)
				}
				return
			}
			require.NoError(t, err)
			if tt.expected != "" {
				assert.Equal(t, tt.expected, output)
			}
		})
	}
}

func TestConfigGetIDE(t *testing.T) {
	tests := []struct {
		name     string
		config   string
		key      string
		format   string
		expected string
		wantErr  bool
	}{
		{
			name:     "ide.command with value",
			config:   "version: \"1.0\"\nide:\n  command: \"cursor\"",
			key:      "ide.command",
			format:   "text",
			expected: "cursor\n",
		},
		{
			name:     "ide.command without config",
			config:   "version: \"1.0\"",
			key:      "ide.command",
			format:   "text",
			expected: "\n",
		},
		{
			name:     "ide.args with text output",
			config:   "version: \"1.0\"\nide:\n  args:\n    - \"--new-window\"\n    - \"--wait\"",
			key:      "ide.args",
			format:   "text",
			expected: "--new-window\n--wait\n",
		},
		{
			name:     "ide.args with json output",
			config:   "version: \"1.0\"\nide:\n  args:\n    - \"--new-window\"\n    - \"--wait\"",
			key:      "ide.args",
			format:   "json",
			expected: "[\"--new-window\",\"--wait\"]\n",
		},
		{
			name:     "ide.args without config",
			config:   "version: \"1.0\"",
			key:      "ide.args",
			format:   "text",
			expected: "",
		},
		{
			name:     "ide.args empty array",
			config:   "version: \"1.0\"\nide:\n  args: []",
			key:      "ide.args",
			format:   "text",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			originalDir, err := os.Getwd()
			require.NoError(t, err)
			defer func() {
				_ = os.Chdir(originalDir)
			}()

			err = os.Chdir(tmpDir)
			require.NoError(t, err)

			err = os.WriteFile("kira.yml", []byte(tt.config), 0o600)
			require.NoError(t, err)

			oldStdout := os.Stdout
			r, w, pipeErr := os.Pipe()
			require.NoError(t, pipeErr)
			os.Stdout = w

			cmd := configGetCmd
			cmd.SetArgs([]string{"--output", tt.format, tt.key})
			err = cmd.ParseFlags([]string{"--output", tt.format})
			require.NoError(t, err)

			errChan := make(chan error, 1)
			go func() {
				errChan <- runConfigGet(cmd, []string{tt.key})
				_ = w.Close()
			}()

			var outputBytes bytes.Buffer
			_, _ = outputBytes.ReadFrom(r)
			os.Stdout = oldStdout
			output := outputBytes.String()

			err = <-errChan
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.expected, output)
		})
	}
}

func TestConfigGetProjectNames(t *testing.T) {
	tests := []struct {
		name     string
		config   string
		format   string
		expected string
	}{
		{
			name:     "project_names standalone",
			config:   "version: \"1.0\"",
			format:   "text",
			expected: "",
		},
		{
			name:     "project_names polyrepo",
			config:   "version: \"1.0\"\nworkspace:\n  projects:\n    - name: frontend\n      path: ../frontend\n    - name: api\n      path: ../api",
			format:   "text",
			expected: "frontend\napi\n",
		},
		{
			name:     "project_names json",
			config:   "version: \"1.0\"\nworkspace:\n  projects:\n    - name: frontend\n      path: ../frontend\n    - name: api\n      path: ../api",
			format:   "json",
			expected: "[\"frontend\",\"api\"]\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			originalDir, err := os.Getwd()
			require.NoError(t, err)
			defer func() {
				_ = os.Chdir(originalDir)
			}()

			err = os.Chdir(tmpDir)
			require.NoError(t, err)

			err = os.WriteFile("kira.yml", []byte(tt.config), 0o600)
			require.NoError(t, err)

			oldStdout := os.Stdout
			r, w, pipeErr := os.Pipe()
			require.NoError(t, pipeErr)
			os.Stdout = w

			cmd := configGetCmd
			cmd.SetArgs([]string{"--output", tt.format, "project_names"})
			err = cmd.ParseFlags([]string{"--output", tt.format})
			require.NoError(t, err)

			errChan := make(chan error, 1)
			go func() {
				errChan <- runConfigGet(cmd, []string{"project_names"})
				_ = w.Close()
			}()

			var outputBytes bytes.Buffer
			_, _ = outputBytes.ReadFrom(r)
			os.Stdout = oldStdout
			output := outputBytes.String()

			err = <-errChan
			require.NoError(t, err)
			assert.Equal(t, tt.expected, output)
		})
	}
}

func TestConfigGetProjectFlag(t *testing.T) {
	tests := []struct {
		name        string
		config      string
		key         string
		projectFlag string
		wantErr     bool
		errContains string
	}{
		{
			name:        "project_path without --project",
			config:      "version: \"1.0\"",
			key:         "project_path",
			projectFlag: "",
			wantErr:     true,
			errContains: "requires --project",
		},
		{
			name:        "--project with unsupported key",
			config:      "version: \"1.0\"\nworkspace:\n  projects:\n    - name: frontend\n      path: ../frontend",
			key:         "work_folder",
			projectFlag: "frontend",
			wantErr:     true,
			errContains: "does not support --project",
		},
		{
			name:        "--project with no projects",
			config:      "version: \"1.0\"",
			key:         "trunk_branch",
			projectFlag: "frontend",
			wantErr:     true,
			errContains: "no projects defined",
		},
		{
			name:        "--project with unknown project",
			config:      "version: \"1.0\"\nworkspace:\n  projects:\n    - name: frontend\n      path: ../frontend",
			key:         "trunk_branch",
			projectFlag: "unknown",
			wantErr:     true,
			errContains: "unknown project",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			originalDir, err := os.Getwd()
			require.NoError(t, err)
			defer func() {
				_ = os.Chdir(originalDir)
			}()

			err = os.Chdir(tmpDir)
			require.NoError(t, err)

			err = os.WriteFile("kira.yml", []byte(tt.config), 0o600)
			require.NoError(t, err)

			cmd := configGetCmd
			args := []string{tt.key}
			if tt.projectFlag != "" {
				args = append([]string{"--project", tt.projectFlag}, args...)
				cmd.SetArgs(args)
				err = cmd.ParseFlags([]string{"--project", tt.projectFlag})
				require.NoError(t, err)
			} else {
				cmd.SetArgs(args)
			}

			err = runConfigGet(cmd, []string{tt.key})
			if tt.wantErr {
				require.Error(t, err)
				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains)
				}
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestConfigGetPathSyntax(t *testing.T) {
	tests := []struct {
		name        string
		config      string
		path        string
		format      string
		expected    string
		wantErr     bool
		errContains string
	}{
		{
			name:     "git.trunk_branch path",
			config:   "version: \"1.0\"\ngit:\n  trunk_branch: \"develop\"",
			path:     "git.trunk_branch",
			format:   "text",
			expected: "develop\n",
		},
		{
			name:     "workspace.work_folder path",
			config:   "version: \"1.0\"\nworkspace:\n  work_folder: \".custom-work\"",
			path:     "workspace.work_folder",
			format:   "text",
			expected: ".custom-work\n",
		},
		{
			name:     "workspace.projects.0.name path",
			config:   "version: \"1.0\"\nworkspace:\n  projects:\n    - name: frontend\n      path: ../frontend",
			path:     "workspace.projects.0.name",
			format:   "text",
			expected: "frontend\n",
		},
		{
			name:     "workspace.projects.0.remote path",
			config:   "version: \"1.0\"\nworkspace:\n  projects:\n    - name: frontend\n      path: ../frontend\n      remote: \"upstream\"",
			path:     "workspace.projects.0.remote",
			format:   "text",
			expected: "upstream\n",
		},
		{
			name:     "ide.args path json",
			config:   "version: \"1.0\"\nide:\n  args:\n    - \"--new-window\"\n    - \"--wait\"",
			path:     "ide.args",
			format:   "json",
			expected: "[\"--new-window\",\"--wait\"]\n",
		},
		{
			name:        "missing path",
			config:      "version: \"1.0\"",
			path:        "git.foo",
			format:      "text",
			wantErr:     true,
			errContains: "path not found",
		},
		{
			name:        "invalid index",
			config:      "version: \"1.0\"\nworkspace:\n  projects:\n    - name: frontend",
			path:        "workspace.projects.99.name",
			format:      "text",
			wantErr:     true,
			errContains: "invalid index",
		},
		{
			name:     "object path json",
			config:   "version: \"1.0\"\ngit:\n  trunk_branch: \"main\"\n  remote: \"origin\"",
			path:     "git",
			format:   "json",
			expected: "", // Will be JSON object
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			originalDir, err := os.Getwd()
			require.NoError(t, err)
			defer func() {
				_ = os.Chdir(originalDir)
			}()

			err = os.Chdir(tmpDir)
			require.NoError(t, err)

			err = os.WriteFile("kira.yml", []byte(tt.config), 0o600)
			require.NoError(t, err)

			oldStdout := os.Stdout
			r, w, pipeErr := os.Pipe()
			require.NoError(t, pipeErr)
			os.Stdout = w

			cmd := configGetCmd
			err = cmd.Flags().Set("output", tt.format)
			require.NoError(t, err)
			err = cmd.Flags().Set("project", "")
			require.NoError(t, err)
			cmd.SetArgs([]string{"--output", tt.format, tt.path})
			err = cmd.ParseFlags([]string{"--output", tt.format})
			require.NoError(t, err)

			errChan := make(chan error, 1)
			go func() {
				errChan <- runConfigGet(cmd, []string{tt.path})
				_ = w.Close()
			}()

			var outputBytes bytes.Buffer
			_, _ = outputBytes.ReadFrom(r)
			os.Stdout = oldStdout
			output := outputBytes.String()

			err = <-errChan
			if tt.wantErr {
				require.Error(t, err)
				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains)
				}
				return
			}
			require.NoError(t, err)
			if tt.expected != "" {
				assert.Equal(t, tt.expected, output)
			} else {
				assert.NotEmpty(t, output)
				if tt.format == sliceLintOutputJSON {
					var jsonVal interface{}
					err = json.Unmarshal([]byte(strings.TrimSpace(output)), &jsonVal)
					assert.NoError(t, err, "output should be valid JSON")
				}
			}
		})
	}
}

func TestConfigGetAllProjects(t *testing.T) {
	tests := []struct {
		name     string
		config   string
		key      string
		format   string
		setupGit bool
		wantErr  bool
	}{
		{
			name:     "all projects trunk_branch text",
			config:   "version: \"1.0\"\nworkspace:\n  projects:\n    - name: frontend\n      path: ../frontend\n    - name: api\n      path: ../api",
			key:      "trunk_branch",
			format:   "text",
			setupGit: true,
		},
		{
			name:     "all projects trunk_branch json",
			config:   "version: \"1.0\"\nworkspace:\n  projects:\n    - name: frontend\n      path: ../frontend\n    - name: api\n      path: ../api",
			key:      "trunk_branch",
			format:   "json",
			setupGit: true,
		},
		{
			name:     "all projects remote text",
			config:   "version: \"1.0\"\nworkspace:\n  projects:\n    - name: frontend\n      path: ../frontend\n      remote: upstream\n    - name: api\n      path: ../api",
			key:      "remote",
			format:   "text",
			setupGit: true,
		},
		{
			name:     "all projects project_path text",
			config:   "version: \"1.0\"\nworkspace:\n  projects:\n    - name: frontend\n      path: ../frontend\n    - name: api\n      path: ../api",
			key:      "project_path",
			format:   "text",
			setupGit: true,
		},
		{
			name:     "all projects empty",
			config:   "version: \"1.0\"",
			key:      "trunk_branch",
			format:   "text",
			setupGit: false,
		},
		{
			name:     "all projects empty json",
			config:   "version: \"1.0\"",
			key:      "trunk_branch",
			format:   "json",
			setupGit: false,
		},
		{
			name:     "all projects unsupported key",
			config:   "version: \"1.0\"\nworkspace:\n  projects:\n    - name: frontend\n      path: ../frontend",
			key:      "work_folder",
			format:   "text",
			setupGit: false,
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			originalDir, err := os.Getwd()
			require.NoError(t, err)
			defer func() {
				_ = os.Chdir(originalDir)
			}()

			err = os.Chdir(tmpDir)
			require.NoError(t, err)

			err = os.WriteFile("kira.yml", []byte(tt.config), 0o600)
			require.NoError(t, err)

			if tt.setupGit {
				cmd := exec.Command("git", "init", "-b", "main")
				cmd.Dir = tmpDir
				err = cmd.Run()
				require.NoError(t, err)

				cmd = exec.Command("git", "config", "user.name", "test")
				cmd.Dir = tmpDir
				err = cmd.Run()
				require.NoError(t, err)

				cmd = exec.Command("git", "config", "user.email", "test@test.com")
				cmd.Dir = tmpDir
				err = cmd.Run()
				require.NoError(t, err)

				cmd = exec.Command("git", "commit", "--allow-empty", "-m", "initial")
				cmd.Dir = tmpDir
				err = cmd.Run()
				require.NoError(t, err)

				cfg, err := config.LoadConfig()
				require.NoError(t, err)
				if cfg.Workspace != nil {
					for _, p := range cfg.Workspace.Projects {
						if p.Path != "" {
							projectPath := p.Path
							if !filepath.IsAbs(projectPath) {
								projectPath = filepath.Join(tmpDir, projectPath)
							}
							projectPath = filepath.Clean(projectPath)
							require.NoError(t, os.MkdirAll(projectPath, 0o700))

							cmd := exec.Command("git", "init", "-b", "main")
							cmd.Dir = projectPath
							err = cmd.Run()
							require.NoError(t, err)

							cmd = exec.Command("git", "config", "user.name", "test")
							cmd.Dir = projectPath
							err = cmd.Run()
							require.NoError(t, err)

							cmd = exec.Command("git", "config", "user.email", "test@test.com")
							cmd.Dir = projectPath
							err = cmd.Run()
							require.NoError(t, err)

							cmd = exec.Command("git", "commit", "--allow-empty", "-m", "initial")
							cmd.Dir = projectPath
							err = cmd.Run()
							require.NoError(t, err)
						}
					}
				}
			}

			oldStdout := os.Stdout
			r, w, pipeErr := os.Pipe()
			require.NoError(t, pipeErr)
			os.Stdout = w

			cmd := configGetCmd
			cmd.SetArgs([]string{"--project", "*", "--output", tt.format, tt.key})
			err = cmd.ParseFlags([]string{"--project", "*", "--output", tt.format})
			require.NoError(t, err)

			errChan := make(chan error, 1)
			go func() {
				errChan <- runConfigGet(cmd, []string{tt.key})
				_ = w.Close()
			}()

			var outputBytes bytes.Buffer
			_, _ = outputBytes.ReadFrom(r)
			os.Stdout = oldStdout
			output := outputBytes.String()

			err = <-errChan
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)

			if tt.format == sliceLintOutputJSON {
				if strings.Contains(tt.config, "projects:") {
					var jsonVal interface{}
					err = json.Unmarshal([]byte(strings.TrimSpace(output)), &jsonVal)
					assert.NoError(t, err, "output should be valid JSON")
				} else {
					assert.Equal(t, "{}\n", output)
				}
			} else {
				if strings.Contains(tt.config, "projects:") {
					assert.NotEmpty(t, output)
				}
			}
		})
	}
}

func TestConfigGetPerProjectResolution(t *testing.T) {
	tests := []struct {
		name        string
		config      string
		key         string
		projectFlag string
		setupGit    bool
		wantErr     bool
		errContains string
		expected    string
	}{
		{
			name:        "per-project trunk_branch",
			config:      "version: \"1.0\"\nworkspace:\n  projects:\n    - name: frontend\n      path: ../frontend\n      trunk_branch: develop",
			key:         "trunk_branch",
			projectFlag: "frontend",
			setupGit:    true,
			expected:    "develop\n",
		},
		{
			name:        "per-project remote",
			config:      "version: \"1.0\"\nworkspace:\n  projects:\n    - name: frontend\n      path: ../frontend\n      remote: upstream",
			key:         "remote",
			projectFlag: "frontend",
			setupGit:    false,
			expected:    "upstream\n",
		},
		{
			name:        "per-project project_path",
			config:      "version: \"1.0\"\nworkspace:\n  projects:\n    - name: frontend\n      path: ../frontend",
			key:         "project_path",
			projectFlag: "frontend",
			setupGit:    true,
			expected:    "", // Will be absolute path
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			originalDir, err := os.Getwd()
			require.NoError(t, err)
			defer func() {
				_ = os.Chdir(originalDir)
			}()

			err = os.Chdir(tmpDir)
			require.NoError(t, err)

			err = os.WriteFile("kira.yml", []byte(tt.config), 0o600)
			require.NoError(t, err)

			if tt.setupGit {
				cmd := exec.Command("git", "init", "-b", "main")
				cmd.Dir = tmpDir
				err = cmd.Run()
				require.NoError(t, err)

				cmd = exec.Command("git", "config", "user.name", "test")
				cmd.Dir = tmpDir
				err = cmd.Run()
				require.NoError(t, err)

				cmd = exec.Command("git", "config", "user.email", "test@test.com")
				cmd.Dir = tmpDir
				err = cmd.Run()
				require.NoError(t, err)

				cmd = exec.Command("git", "commit", "--allow-empty", "-m", "initial")
				cmd.Dir = tmpDir
				err = cmd.Run()
				require.NoError(t, err)

				cfg, err := config.LoadConfig()
				require.NoError(t, err)
				if cfg.Workspace != nil {
					for _, p := range cfg.Workspace.Projects {
						if p.Path != "" && p.Name == tt.projectFlag {
							projectPath := p.Path
							if !filepath.IsAbs(projectPath) {
								projectPath = filepath.Join(tmpDir, projectPath)
							}
							projectPath = filepath.Clean(projectPath)
							require.NoError(t, os.MkdirAll(projectPath, 0o700))

							branchName := defaultTrunkBranch
							if p.TrunkBranch != "" {
								branchName = p.TrunkBranch
							}

							cmd := exec.Command("git", "init", "-b", branchName)
							cmd.Dir = projectPath
							err = cmd.Run()
							require.NoError(t, err)

							cmd = exec.Command("git", "config", "user.name", "test")
							cmd.Dir = projectPath
							err = cmd.Run()
							require.NoError(t, err)

							cmd = exec.Command("git", "config", "user.email", "test@test.com")
							cmd.Dir = projectPath
							err = cmd.Run()
							require.NoError(t, err)

							cmd = exec.Command("git", "commit", "--allow-empty", "-m", "initial")
							cmd.Dir = projectPath
							err = cmd.Run()
							require.NoError(t, err)
							break
						}
					}
				}
			}

			oldStdout := os.Stdout
			r, w, pipeErr := os.Pipe()
			require.NoError(t, pipeErr)
			os.Stdout = w

			cmd := configGetCmd
			cmd.SetArgs([]string{"--project", tt.projectFlag, tt.key})
			err = cmd.ParseFlags([]string{"--project", tt.projectFlag})
			require.NoError(t, err)

			errChan := make(chan error, 1)
			go func() {
				errChan <- runConfigGet(cmd, []string{tt.key})
				_ = w.Close()
			}()

			var outputBytes bytes.Buffer
			_, _ = outputBytes.ReadFrom(r)
			os.Stdout = oldStdout
			output := outputBytes.String()

			err = <-errChan
			if tt.wantErr {
				require.Error(t, err)
				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains)
				}
				return
			}
			require.NoError(t, err)

			if tt.expected != "" {
				assert.Equal(t, tt.expected, output)
			} else {
				if tt.key == "project_path" {
					assert.True(t, filepath.IsAbs(strings.TrimSpace(output)), "expected absolute path, got: %s", output)
				}
			}
		})
	}
}

func TestConfigGetPathIgnoresProject(t *testing.T) {
	tests := []struct {
		name     string
		config   string
		path     string
		project  string
		expected string
		wantErr  bool
	}{
		{
			name:     "path syntax ignores --project",
			config:   "version: \"1.0\"\ngit:\n  trunk_branch: \"develop\"\nworkspace:\n  projects:\n    - name: frontend\n      path: ../frontend",
			path:     "git.trunk_branch",
			project:  "frontend",
			expected: "develop\n",
		},
		{
			name:     "path syntax ignores --project all",
			config:   "version: \"1.0\"\nworkspace:\n  work_folder: \".custom-work\"\n  projects:\n    - name: frontend\n      path: ../frontend",
			path:     "workspace.work_folder",
			project:  "*",
			expected: ".custom-work\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			originalDir, err := os.Getwd()
			require.NoError(t, err)
			defer func() {
				_ = os.Chdir(originalDir)
			}()

			err = os.Chdir(tmpDir)
			require.NoError(t, err)

			err = os.WriteFile("kira.yml", []byte(tt.config), 0o600)
			require.NoError(t, err)

			oldStdout := os.Stdout
			r, w, pipeErr := os.Pipe()
			require.NoError(t, pipeErr)
			os.Stdout = w

			cmd := configGetCmd
			args := []string{"--project", tt.project, "--output", "text", tt.path}
			cmd.SetArgs(args)
			err = cmd.ParseFlags([]string{"--project", tt.project, "--output", "text"})
			require.NoError(t, err)

			errChan := make(chan error, 1)
			go func() {
				errChan <- runConfigGet(cmd, []string{tt.path})
				_ = w.Close()
			}()

			var outputBytes bytes.Buffer
			_, _ = outputBytes.ReadFrom(r)
			os.Stdout = oldStdout
			output := outputBytes.String()

			err = <-errChan
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.expected, output)
		})
	}
}
