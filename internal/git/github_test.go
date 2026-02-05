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

func TestUpdateDraftToReady_resourceNotAccessibleError(t *testing.T) {
	ctx := context.Background()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"errors":[{"message":"Resource not accessible by personal access token"}]}`))
	}))
	defer server.Close()

	baseURL, err := url.Parse(server.URL + "/")
	require.NoError(t, err)
	client := github.NewClient(server.Client())
	client.BaseURL = baseURL
	pr := &github.PullRequest{NodeID: github.String("PR_node1")}

	err = UpdateDraftToReady(ctx, client, pr)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "Resource not accessible")
	assert.Contains(t, err.Error(), "KIRA_GITHUB_TOKEN")
	assert.Contains(t, err.Error(), "repo scope")
	assert.Contains(t, err.Error(), "Pull requests")
}

func TestIsPRClosedOrMerged(t *testing.T) {
	t.Run("nil pr returns true", func(t *testing.T) {
		assert.True(t, IsPRClosedOrMerged(nil))
	})
	t.Run("closed state returns true", func(t *testing.T) {
		pr := &github.PullRequest{State: github.String("closed")}
		assert.True(t, IsPRClosedOrMerged(pr))
	})
	t.Run("merged has MergedAt returns true", func(t *testing.T) {
		pr := &github.PullRequest{State: github.String("closed"), MergedAt: &github.Timestamp{}}
		assert.True(t, IsPRClosedOrMerged(pr))
	})
	t.Run("open state returns false", func(t *testing.T) {
		pr := &github.PullRequest{State: github.String("open")}
		assert.False(t, IsPRClosedOrMerged(pr))
	})
}

func TestFindPullRequestByWorkItemID(t *testing.T) {
	ctx := context.Background()

	t.Run("returns matching open PR by head ref prefix", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Contains(t, r.URL.Path, "/pulls")
			assert.Equal(t, "open", r.URL.Query().Get("state"))
			w.Header().Set("Content-Type", "application/json")
			// One PR with head ref 014-kira-done
			_, _ = w.Write([]byte(`[{"number":1,"state":"open","head":{"ref":"014-kira-done"},"base":{"ref":"main"}}]`))
		}))
		defer server.Close()

		baseURL, err := url.Parse(server.URL + "/")
		require.NoError(t, err)
		client := github.NewClient(server.Client())
		client.BaseURL = baseURL

		pr, err := FindPullRequestByWorkItemID(ctx, client, "owner", "repo", "014")
		require.NoError(t, err)
		require.NotNil(t, pr)
		assert.Equal(t, 1, pr.GetNumber())
		assert.Equal(t, "014-kira-done", pr.GetHead().GetRef())
	})

	t.Run("returns nil when no matching PR", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`[]`))
		}))
		defer server.Close()

		baseURL, err := url.Parse(server.URL + "/")
		require.NoError(t, err)
		client := github.NewClient(server.Client())
		client.BaseURL = baseURL

		pr, err := FindPullRequestByWorkItemID(ctx, client, "owner", "repo", "999")
		require.NoError(t, err)
		assert.Nil(t, pr)
	})

	t.Run("returns closed PR when no open match", func(t *testing.T) {
		callCount := 0
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			callCount++
			if r.URL.Query().Get("state") == "open" {
				_, _ = w.Write([]byte(`[]`))
				return
			}
			_, _ = w.Write([]byte(`[{"number":2,"state":"closed","head":{"ref":"014-feature"},"merged_at":"2024-01-01T00:00:00Z"}]`))
		}))
		defer server.Close()

		baseURL, err := url.Parse(server.URL + "/")
		require.NoError(t, err)
		client := github.NewClient(server.Client())
		client.BaseURL = baseURL

		pr, err := FindPullRequestByWorkItemID(ctx, client, "owner", "repo", "014")
		require.NoError(t, err)
		require.NotNil(t, pr)
		assert.Equal(t, 2, pr.GetNumber())
		assert.Equal(t, "closed", pr.GetState())
		assert.GreaterOrEqual(t, callCount, 2)
	})
}
