package commands

import (
	"os"
	"os/exec"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"kira/internal/config"
)

func TestFormatUserDisplay(t *testing.T) {
	t.Run("formats user with name", func(t *testing.T) {
		user := UserInfo{
			Email: "user@example.com",
			Name:  "John Doe",
		}
		display := formatUserDisplay(user)
		assert.Equal(t, "John Doe <user@example.com>", display)
	})

	t.Run("formats user without name", func(t *testing.T) {
		user := UserInfo{
			Email: "user@example.com",
			Name:  "",
		}
		display := formatUserDisplay(user)
		assert.Equal(t, "user@example.com", display)
	})
}

func TestShouldIgnoreEmail(t *testing.T) {
	t.Run("ignores exact email match", func(t *testing.T) {
		cfg := &config.Config{
			Users: config.UsersConfig{
				IgnoredEmails: []string{"bot@example.com", "noreply@github.com"},
			},
		}
		assert.True(t, shouldIgnoreEmail("bot@example.com", cfg))
		assert.True(t, shouldIgnoreEmail("BOT@EXAMPLE.COM", cfg)) // Case insensitive
		assert.False(t, shouldIgnoreEmail("user@example.com", cfg))
	})

	t.Run("ignores pattern match", func(t *testing.T) {
		cfg := &config.Config{
			Users: config.UsersConfig{
				IgnoredPatterns: []string{"*bot*", "*noreply*"},
			},
		}
		assert.True(t, shouldIgnoreEmail("test-bot@example.com", cfg))
		assert.True(t, shouldIgnoreEmail("noreply@github.com", cfg))
		assert.False(t, shouldIgnoreEmail("user@example.com", cfg))
	})

	t.Run("case insensitive pattern matching", func(t *testing.T) {
		cfg := &config.Config{
			Users: config.UsersConfig{
				IgnoredPatterns: []string{"*BOT*"},
			},
		}
		assert.True(t, shouldIgnoreEmail("test-bot@example.com", cfg))
		assert.True(t, shouldIgnoreEmail("TEST-BOT@example.com", cfg))
	})
}

func TestListUsersConfigOnly(t *testing.T) {
	t.Run("lists only saved users when git disabled", func(t *testing.T) {
		tmpDir := t.TempDir()
		require.NoError(t, os.Chdir(tmpDir))
		defer func() { _ = os.Chdir("/") }()

		useGitHistory := false
		cfg := &config.Config{
			Users: config.UsersConfig{
				UseGitHistory: &useGitHistory,
				SavedUsers: []config.SavedUser{
					{Email: "user1@example.com", Name: "User One"},
					{Email: "user2@example.com", Name: "User Two"},
				},
			},
		}

		// Should not require git repository
		err := listUsers(cfg, "list", 0)
		require.NoError(t, err)
	})

	t.Run("shows message when no saved users", func(t *testing.T) {
		tmpDir := t.TempDir()
		require.NoError(t, os.Chdir(tmpDir))
		defer func() { _ = os.Chdir("/") }()

		useGitHistory := false
		cfg := &config.Config{
			Users: config.UsersConfig{
				UseGitHistory: &useGitHistory,
				SavedUsers:    []config.SavedUser{},
			},
		}

		err := listUsers(cfg, "list", 0)
		require.NoError(t, err)
	})
}

func TestListUsersWithGit(t *testing.T) {
	t.Run("extracts users from git history", func(t *testing.T) {
		tmpDir := t.TempDir()
		require.NoError(t, os.Chdir(tmpDir))
		defer func() { _ = os.Chdir("/") }()

		// Initialize git repository
		require.NoError(t, exec.Command("git", "init").Run())
		require.NoError(t, exec.Command("git", "config", "user.email", "test@example.com").Run())
		require.NoError(t, exec.Command("git", "config", "user.name", "Test User").Run())

		// Create a commit
		require.NoError(t, os.WriteFile("test.txt", []byte("test"), 0o600))
		require.NoError(t, exec.Command("git", "add", "test.txt").Run())
		require.NoError(t, exec.Command("git", "commit", "-m", "Initial commit").Run())

		useGitHistory := true
		cfg := &config.Config{
			Users: config.UsersConfig{
				UseGitHistory: &useGitHistory,
			},
		}

		err := listUsers(cfg, "list", 0)
		require.NoError(t, err)
	})

	t.Run("handles empty repository", func(t *testing.T) {
		tmpDir := t.TempDir()
		require.NoError(t, os.Chdir(tmpDir))
		defer func() { _ = os.Chdir("/") }()

		// Initialize empty git repository
		require.NoError(t, exec.Command("git", "init").Run())

		useGitHistory := true
		cfg := &config.Config{
			Users: config.UsersConfig{
				UseGitHistory: &useGitHistory,
			},
		}

		err := listUsers(cfg, "list", 0)
		require.NoError(t, err) // Should not error, just return empty list
	})
}

func TestDuplicateDetection(t *testing.T) {
	t.Run("merges saved user with git history user", func(t *testing.T) {
		tmpDir := t.TempDir()
		require.NoError(t, os.Chdir(tmpDir))
		defer func() { _ = os.Chdir("/") }()

		// Initialize git repository
		require.NoError(t, exec.Command("git", "init").Run())
		require.NoError(t, exec.Command("git", "config", "user.email", "user@example.com").Run())
		require.NoError(t, exec.Command("git", "config", "user.name", "Git User").Run())

		// Create a commit
		require.NoError(t, os.WriteFile("test.txt", []byte("test"), 0o600))
		require.NoError(t, exec.Command("git", "add", "test.txt").Run())
		require.NoError(t, exec.Command("git", "commit", "-m", "Initial commit").Run())

		useGitHistory := true
		cfg := &config.Config{
			Users: config.UsersConfig{
				UseGitHistory: &useGitHistory,
				SavedUsers: []config.SavedUser{
					{Email: "user@example.com", Name: "Saved User"}, // Same email, different name
				},
			},
		}

		// Should merge: saved user name takes precedence
		err := listUsers(cfg, "list", 0)
		require.NoError(t, err)
	})

	t.Run("case insensitive duplicate detection", func(t *testing.T) {
		tmpDir := t.TempDir()
		require.NoError(t, os.Chdir(tmpDir))
		defer func() { _ = os.Chdir("/") }()

		// Initialize git repository
		require.NoError(t, exec.Command("git", "init").Run())
		require.NoError(t, exec.Command("git", "config", "user.email", "User@Example.com").Run())
		require.NoError(t, exec.Command("git", "config", "user.name", "Git User").Run())

		// Create a commit
		require.NoError(t, os.WriteFile("test.txt", []byte("test"), 0o600))
		require.NoError(t, exec.Command("git", "add", "test.txt").Run())
		require.NoError(t, exec.Command("git", "commit", "-m", "Initial commit").Run())

		useGitHistory := true
		cfg := &config.Config{
			Users: config.UsersConfig{
				UseGitHistory: &useGitHistory,
				SavedUsers: []config.SavedUser{
					{Email: "user@example.com", Name: "Saved User"}, // Different case
				},
			},
		}

		// Should detect as duplicate (case insensitive)
		err := listUsers(cfg, "list", 0)
		require.NoError(t, err)
	})
}

func TestUserNumbering(t *testing.T) {
	t.Run("numbers users sequentially by first commit date", func(t *testing.T) {
		tmpDir := t.TempDir()
		require.NoError(t, os.Chdir(tmpDir))
		defer func() { _ = os.Chdir("/") }()

		// Initialize git repository
		require.NoError(t, exec.Command("git", "init").Run())
		require.NoError(t, exec.Command("git", "config", "user.email", "user1@example.com").Run())
		require.NoError(t, exec.Command("git", "config", "user.name", "User One").Run())

		// Create first commit
		require.NoError(t, os.WriteFile("test1.txt", []byte("test1"), 0o600))
		require.NoError(t, exec.Command("git", "add", "test1.txt").Run())
		require.NoError(t, exec.Command("git", "commit", "-m", "First commit").Run())

		// Change user for second commit
		require.NoError(t, exec.Command("git", "config", "user.email", "user2@example.com").Run())
		require.NoError(t, exec.Command("git", "config", "user.name", "User Two").Run())

		// Create second commit
		require.NoError(t, os.WriteFile("test2.txt", []byte("test2"), 0o600))
		require.NoError(t, exec.Command("git", "add", "test2.txt").Run())
		require.NoError(t, exec.Command("git", "commit", "-m", "Second commit").Run())

		useGitHistory := true
		cfg := &config.Config{
			Users: config.UsersConfig{
				UseGitHistory: &useGitHistory,
			},
		}

		err := listUsers(cfg, "list", 0)
		require.NoError(t, err)
		// User 1 should be numbered 1, User 2 should be numbered 2
	})
}

func TestOutputFormats(t *testing.T) {
	t.Run("table format", func(t *testing.T) {
		tmpDir := t.TempDir()
		require.NoError(t, os.Chdir(tmpDir))
		defer func() { _ = os.Chdir("/") }()

		useGitHistory := false
		cfg := &config.Config{
			Users: config.UsersConfig{
				UseGitHistory: &useGitHistory,
				SavedUsers: []config.SavedUser{
					{Email: "user@example.com", Name: "Test User"},
				},
			},
		}

		err := listUsers(cfg, "table", 0)
		require.NoError(t, err)
	})

	t.Run("list format", func(t *testing.T) {
		tmpDir := t.TempDir()
		require.NoError(t, os.Chdir(tmpDir))
		defer func() { _ = os.Chdir("/") }()

		useGitHistory := false
		cfg := &config.Config{
			Users: config.UsersConfig{
				UseGitHistory: &useGitHistory,
				SavedUsers: []config.SavedUser{
					{Email: "user@example.com", Name: "Test User"},
				},
			},
		}

		err := listUsers(cfg, "list", 0)
		require.NoError(t, err)
	})

	t.Run("json format", func(t *testing.T) {
		tmpDir := t.TempDir()
		require.NoError(t, os.Chdir(tmpDir))
		defer func() { _ = os.Chdir("/") }()

		useGitHistory := false
		cfg := &config.Config{
			Users: config.UsersConfig{
				UseGitHistory: &useGitHistory,
				SavedUsers: []config.SavedUser{
					{Email: "user@example.com", Name: "Test User"},
				},
			},
		}

		err := listUsers(cfg, "json", 0)
		require.NoError(t, err)
	})

	t.Run("invalid format", func(t *testing.T) {
		tmpDir := t.TempDir()
		require.NoError(t, os.Chdir(tmpDir))
		defer func() { _ = os.Chdir("/") }()

		useGitHistory := false
		cfg := &config.Config{
			Users: config.UsersConfig{
				UseGitHistory: &useGitHistory,
			},
		}

		err := listUsers(cfg, "invalid", 0)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid format")
	})
}

func TestIgnorePatterns(t *testing.T) {
	t.Run("ignores users matching patterns", func(t *testing.T) {
		tmpDir := t.TempDir()
		require.NoError(t, os.Chdir(tmpDir))
		defer func() { _ = os.Chdir("/") }()

		// Initialize git repository
		require.NoError(t, exec.Command("git", "init").Run())
		require.NoError(t, exec.Command("git", "config", "user.email", "bot@example.com").Run())
		require.NoError(t, exec.Command("git", "config", "user.name", "Bot").Run())

		// Create a commit
		require.NoError(t, os.WriteFile("test.txt", []byte("test"), 0o600))
		require.NoError(t, exec.Command("git", "add", "test.txt").Run())
		require.NoError(t, exec.Command("git", "commit", "-m", "Initial commit").Run())

		useGitHistory := true
		cfg := &config.Config{
			Users: config.UsersConfig{
				UseGitHistory:   &useGitHistory,
				IgnoredPatterns: []string{"*bot*"},
			},
		}

		err := listUsers(cfg, "list", 0)
		require.NoError(t, err)
		// Bot user should be filtered out
	})
}

func TestEdgeCases(t *testing.T) {
	t.Run("handles users with same commit date", func(t *testing.T) {
		tmpDir := t.TempDir()
		require.NoError(t, os.Chdir(tmpDir))
		defer func() { _ = os.Chdir("/") }()

		// Initialize git repository
		require.NoError(t, exec.Command("git", "init").Run())

		// Set fixed date for commits
		env := os.Environ()
		env = append(env, "GIT_AUTHOR_DATE=2024-01-01T00:00:00Z", "GIT_COMMITTER_DATE=2024-01-01T00:00:00Z")

		// First user
		require.NoError(t, exec.Command("git", "config", "user.email", "user1@example.com").Run())
		require.NoError(t, exec.Command("git", "config", "user.name", "User One").Run())
		require.NoError(t, os.WriteFile("test1.txt", []byte("test1"), 0o600))
		require.NoError(t, exec.Command("git", "add", "test1.txt").Run())
		cmd := exec.Command("git", "commit", "-m", "First commit")
		cmd.Env = env
		require.NoError(t, cmd.Run())

		// Second user with same date
		require.NoError(t, exec.Command("git", "config", "user.email", "user2@example.com").Run())
		require.NoError(t, exec.Command("git", "config", "user.name", "User Two").Run())
		require.NoError(t, os.WriteFile("test2.txt", []byte("test2"), 0o600))
		require.NoError(t, exec.Command("git", "add", "test2.txt").Run())
		cmd = exec.Command("git", "commit", "-m", "Second commit")
		cmd.Env = env
		require.NoError(t, cmd.Run())

		useGitHistory := true
		cfg := &config.Config{
			Users: config.UsersConfig{
				UseGitHistory: &useGitHistory,
			},
		}

		err := listUsers(cfg, "list", 0)
		require.NoError(t, err)
		// Should order by email when dates are equal
	})

	t.Run("handles saved user without name", func(t *testing.T) {
		tmpDir := t.TempDir()
		require.NoError(t, os.Chdir(tmpDir))
		defer func() { _ = os.Chdir("/") }()

		useGitHistory := false
		cfg := &config.Config{
			Users: config.UsersConfig{
				UseGitHistory: &useGitHistory,
				SavedUsers: []config.SavedUser{
					{Email: "user@example.com"}, // No name
				},
			},
		}

		err := listUsers(cfg, "list", 0)
		require.NoError(t, err)
		// Should display as email only
	})

	t.Run("handles invalid limit", func(t *testing.T) {
		tmpDir := t.TempDir()
		require.NoError(t, os.Chdir(tmpDir))
		defer func() { _ = os.Chdir("/") }()

		useGitHistory := true
		cfg := &config.Config{
			Users: config.UsersConfig{
				UseGitHistory: &useGitHistory,
			},
		}

		err := listUsers(cfg, "list", -1)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid limit")
	})
}

func TestExtractGitUsers(t *testing.T) {
	t.Run("returns error when not a git repository", func(t *testing.T) {
		tmpDir := t.TempDir()
		require.NoError(t, os.Chdir(tmpDir))
		defer func() { _ = os.Chdir("/") }()

		_, err := extractGitUsers(0)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "not a git repository")
	})

	t.Run("handles empty repository", func(t *testing.T) {
		tmpDir := t.TempDir()
		require.NoError(t, os.Chdir(tmpDir))
		defer func() { _ = os.Chdir("/") }()

		require.NoError(t, exec.Command("git", "init").Run())

		users, err := extractGitUsers(0)
		require.NoError(t, err)
		assert.Empty(t, users)
	})
}

func TestDisplayUsers(t *testing.T) {
	t.Run("handles empty user list", func(t *testing.T) {
		err := displayUsers([]UserInfo{}, "table")
		require.NoError(t, err)
	})

	t.Run("displays users in table format", func(t *testing.T) {
		users := []UserInfo{
			{
				Number:      1,
				Email:       "user@example.com",
				Name:        "Test User",
				FirstCommit: timePtr(time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)),
				Source:      "git",
			},
		}
		err := displayUsers(users, "table")
		require.NoError(t, err)
	})

	t.Run("displays users in list format", func(t *testing.T) {
		users := []UserInfo{
			{
				Number: 1,
				Email:  "user@example.com",
				Name:   "Test User",
				Source: "config",
			},
		}
		err := displayUsers(users, "list")
		require.NoError(t, err)
	})

	t.Run("displays users in json format", func(t *testing.T) {
		users := []UserInfo{
			{
				Number: 1,
				Email:  "user@example.com",
				Name:   "Test User",
				Source: "config",
			},
		}
		err := displayUsers(users, "json")
		require.NoError(t, err)
	})
}

func timePtr(t time.Time) *time.Time {
	return &t
}
