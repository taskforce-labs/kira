package git

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseGitHubOwnerRepo(t *testing.T) {
	tests := []struct {
		name      string
		remoteURL string
		owner     string
		repo      string
		wantErr   bool
	}{
		{"https github.com", "https://github.com/owner/repo", "owner", "repo", false},
		{"https with .git", "https://github.com/owner/repo.git", "owner", "repo", false},
		{"https GHE", "https://ghe.example.com/org/my-repo", "org", "my-repo", false},
		{"ssh github", "git@github.com:owner/repo.git", "owner", "repo", false},
		{"ssh without .git", "git@github.com:owner/repo", "owner", "repo", false},
		{"ssh GHE", "git@ghe.example.com:org/repo.git", "org", "repo", false},
		{"invalid ssh", "git@github.com", "", "", true},
		{"invalid path", "https://github.com/owner", "", "", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			owner, repo, err := ParseGitHubOwnerRepo(tt.remoteURL)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.owner, owner)
			assert.Equal(t, tt.repo, repo)
		})
	}
}

func TestNewClient_emptyToken(t *testing.T) {
	ctx := context.Background()
	_, err := NewClient(ctx, "", "")
	require.Error(t, err)
}
