// Package git provides GitHub API helpers for draft PR creation.
package git

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/google/go-github/v61/github"
	"golang.org/x/oauth2"
)

// ParseGitHubOwnerRepo extracts owner and repo from a GitHub remote URL.
// Supports https://github.com/owner/repo, https://host/owner/repo, and git@host:owner/repo.
func ParseGitHubOwnerRepo(remoteURL string) (owner, repo string, err error) {
	u := remoteURL
	// Handle git@host:owner/repo[.git]
	if strings.HasPrefix(u, "git@") {
		u = strings.TrimPrefix(u, "git@")
		parts := strings.SplitN(u, ":", 2)
		if len(parts) != 2 {
			return "", "", fmt.Errorf("invalid git SSH URL: %s", remoteURL)
		}
		path := strings.TrimSuffix(parts[1], ".git")
		segments := strings.Split(strings.Trim(path, "/"), "/")
		if len(segments) < 2 {
			return "", "", fmt.Errorf("invalid GitHub path: %s", path)
		}
		return segments[0], segments[1], nil
	}
	// Handle https://host/owner/repo[.git]
	parsed, err := url.Parse(u)
	if err != nil {
		return "", "", fmt.Errorf("invalid URL: %w", err)
	}
	if parsed.Host == "" && strings.Contains(u, "://") {
		return "", "", fmt.Errorf("invalid URL: missing host")
	}
	path := strings.TrimSuffix(strings.Trim(parsed.Path, "/"), ".git")
	segments := strings.Split(path, "/")
	if len(segments) < 2 {
		return "", "", fmt.Errorf("invalid GitHub path: %s", path)
	}
	return segments[0], segments[1], nil
}

// NewClient creates a GitHub API client. token must be non-empty.
// baseURL is optional: empty means github.com; set for GitHub Enterprise (e.g. https://ghe.example.com).
// Never log or expose token.
func NewClient(ctx context.Context, token, baseURL string) (*github.Client, error) {
	if token == "" {
		return nil, fmt.Errorf("token is required")
	}
	ts := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: token})
	hc := oauth2.NewClient(ctx, ts)
	client := github.NewClient(hc).WithAuthToken(token)
	if baseURL == "" {
		return client, nil
	}
	baseURL = strings.TrimSuffix(baseURL, "/")
	apiBase := baseURL + "/api/v3"
	client, err := client.WithEnterpriseURLs(apiBase, apiBase)
	if err != nil {
		return nil, err
	}
	return client, nil
}

// CreateDraftPR creates a draft pull request and returns its HTML URL.
// base is the target branch (e.g. main), head is the source branch.
func CreateDraftPR(ctx context.Context, client *github.Client, owner, repo, base, head, title, body string) (prURL string, err error) {
	return CreatePR(ctx, client, owner, repo, base, head, title, body, true, nil)
}

// ListPullRequestsByHead lists open pull requests with head = owner:headBranch.
// headBranch is the branch name (e.g. 012-submit-for-review); head filter is owner:headBranch.
func ListPullRequestsByHead(ctx context.Context, client *github.Client, owner, repo, headBranch string) ([]*github.PullRequest, error) {
	headFilter := owner + ":" + headBranch
	opts := &github.PullRequestListOptions{State: "open", Head: headFilter, ListOptions: github.ListOptions{PerPage: 10}}
	prs, _, err := client.PullRequests.List(ctx, owner, repo, opts)
	if err != nil {
		return nil, err
	}
	return prs, nil
}

// UpdateDraftToReady updates a pull request from draft to ready for review.
// The REST API does not support changing draft status; we use the GraphQL mutation markPullRequestReadyForReview.
func UpdateDraftToReady(ctx context.Context, client *github.Client, pr *github.PullRequest) error {
	if pr == nil || pr.NodeID == nil || *pr.NodeID == "" {
		return fmt.Errorf("pull request has no node ID (required for GraphQL)")
	}
	graphqlURL, err := graphQLURL(client)
	if err != nil {
		return err
	}
	return graphQLMarkPullRequestReadyForReview(ctx, client, graphqlURL, *pr.NodeID)
}

// graphQLURL returns the GraphQL endpoint for the client's base URL (github.com or Enterprise).
func graphQLURL(client *github.Client) (string, error) {
	if client.BaseURL == nil {
		return "https://api.github.com/graphql", nil
	}
	u := client.BaseURL
	if strings.Contains(u.Host, "api.github.com") {
		return "https://api.github.com/graphql", nil
	}
	// GitHub Enterprise: base is typically https://hostname/api/v3, GraphQL is https://hostname/api/graphql
	return u.Scheme + "://" + u.Host + "/api/graphql", nil
}

// graphQLMarkPullRequestReadyForReview calls the GraphQL mutation to mark a draft PR as ready for review.
func graphQLMarkPullRequestReadyForReview(ctx context.Context, client *github.Client, graphqlURL, pullRequestNodeID string) error {
	query := `mutation($id: ID!) { markPullRequestReadyForReview(input: { pullRequestId: $id }) { pullRequest { id isDraft } } }`
	body := map[string]interface{}{
		"query": query,
		"variables": map[string]string{
			"id": pullRequestNodeID,
		},
	}
	enc, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("graphql request: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, graphqlURL, bytes.NewReader(enc))
	if err != nil {
		return fmt.Errorf("graphql request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := client.Client().Do(req)
	if err != nil {
		return fmt.Errorf("graphql request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("graphql returned status %d", resp.StatusCode)
	}
	var result struct {
		Data *struct {
			MarkPullRequestReadyForReview *struct {
				PullRequest *struct {
					ID      string `json:"id"`
					IsDraft bool   `json:"isDraft"`
				} `json:"pullRequest"`
			} `json:"markPullRequestReadyForReview"`
		} `json:"data"`
		Errors []struct {
			Message string `json:"message"`
		} `json:"errors"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return fmt.Errorf("graphql decode: %w", err)
	}
	if len(result.Errors) > 0 {
		return fmt.Errorf("graphql: %s", result.Errors[0].Message)
	}
	if result.Data == nil || result.Data.MarkPullRequestReadyForReview == nil {
		return fmt.Errorf("graphql: no pull request in response")
	}
	return nil
}

// SetReviewers requests reviewers on a pull request (user logins).
func SetReviewers(ctx context.Context, client *github.Client, owner, repo string, prNumber int, reviewers []string) error {
	if len(reviewers) == 0 {
		return nil
	}
	_, _, err := client.PullRequests.RequestReviewers(ctx, owner, repo, prNumber, github.ReviewersRequest{Reviewers: reviewers})
	return err
}

// CreatePR creates a pull request (draft or ready) and optionally sets reviewers.
// Returns the PR HTML URL.
func CreatePR(ctx context.Context, client *github.Client, owner, repo, base, head, title, body string, draft bool, reviewers []string) (prURL string, err error) {
	req := &github.NewPullRequest{
		Title: github.String(title),
		Head:  github.String(head),
		Base:  github.String(base),
		Body:  github.String(body),
		Draft: github.Bool(draft),
	}
	pr, _, err := client.PullRequests.Create(ctx, owner, repo, req)
	if err != nil {
		return "", err
	}
	if pr.HTMLURL != nil {
		prURL = *pr.HTMLURL
	} else {
		prURL = ""
	}
	if pr.Number != nil && len(reviewers) > 0 {
		if err := SetReviewers(ctx, client, owner, repo, *pr.Number, reviewers); err != nil {
			return prURL, fmt.Errorf("PR created but failed to set reviewers: %w", err)
		}
	}
	if prURL == "" {
		return "", fmt.Errorf("PR created but no HTML URL returned")
	}
	return prURL, nil
}
