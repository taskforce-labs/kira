package git

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/google/go-github/v61/github"
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

func TestGraphQLURL(t *testing.T) {
	t.Run("nil BaseURL returns github.com graphql", func(t *testing.T) {
		client := github.NewClient(nil)
		// Client from NewClient(nil) may have BaseURL set; clear it for this test
		client.BaseURL = nil
		u, err := graphQLURL(client)
		require.NoError(t, err)
		assert.Equal(t, "https://api.github.com/graphql", u)
	})
	t.Run("api.github.com host returns github.com graphql", func(t *testing.T) {
		client := github.NewClient(nil)
		client.BaseURL, _ = url.Parse("https://api.github.com/")
		u, err := graphQLURL(client)
		require.NoError(t, err)
		assert.Equal(t, "https://api.github.com/graphql", u)
	})
	t.Run("enterprise host returns host api/graphql", func(t *testing.T) {
		client := github.NewClient(nil)
		client.BaseURL, _ = url.Parse("https://ghe.example.com/api/v3/")
		u, err := graphQLURL(client)
		require.NoError(t, err)
		assert.Equal(t, "https://ghe.example.com/api/graphql", u)
	})
}

func TestUpdateDraftToReady_validation(t *testing.T) {
	ctx := context.Background()
	client := github.NewClient(nil)

	t.Run("nil pr returns error", func(t *testing.T) {
		err := UpdateDraftToReady(ctx, client, nil)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "node ID")
	})
	t.Run("pr with nil NodeID returns error", func(t *testing.T) {
		pr := &github.PullRequest{Number: github.Int(1)}
		err := UpdateDraftToReady(ctx, client, pr)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "node ID")
	})
	t.Run("pr with empty NodeID returns error", func(t *testing.T) {
		pr := &github.PullRequest{Number: github.Int(1), NodeID: github.String("")}
		err := UpdateDraftToReady(ctx, client, pr)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "node ID")
	})
}

func TestUpdateDraftToReady_success(t *testing.T) {
	ctx := context.Background()
	var receivedBody map[string]interface{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))
		_ = json.NewDecoder(r.Body).Decode(&receivedBody)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":{"markPullRequestReadyForReview":{"pullRequest":{"id":"PR_1","isDraft":false}}}}`))
	}))
	defer server.Close()

	baseURL, err := url.Parse(server.URL + "/")
	require.NoError(t, err)
	client := github.NewClient(server.Client())
	client.BaseURL = baseURL

	pr := &github.PullRequest{NodeID: github.String("PR_node123")}
	err = UpdateDraftToReady(ctx, client, pr)
	require.NoError(t, err)
	require.NotNil(t, receivedBody)
	variables, _ := receivedBody["variables"].(map[string]interface{})
	require.NotNil(t, variables)
	assert.Equal(t, "PR_node123", variables["id"])
	query, _ := receivedBody["query"].(string)
	assert.Contains(t, query, "markPullRequestReadyForReview")
}
