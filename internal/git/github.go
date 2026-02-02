// Package git provides GitHub API helpers for draft PR creation.
package git

import (
	"context"
	"fmt"
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
func UpdateDraftToReady(ctx context.Context, client *github.Client, owner, repo string, prNumber int) error {
	draft := false
	_, _, err := client.PullRequests.Edit(ctx, owner, repo, prNumber, &github.PullRequest{Draft: &draft})
	return err
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
