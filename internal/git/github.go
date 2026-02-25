// Package git provides GitHub API helpers for draft PR creation.
package git

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

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

// IsRateLimitError reports whether err is a GitHub API rate-limit error (403 or 429).
func IsRateLimitError(err error) bool {
	var errResp *github.ErrorResponse
	if !errors.As(err, &errResp) || errResp.Response == nil {
		return false
	}
	code := errResp.Response.StatusCode
	return code == http.StatusForbidden || code == 429
}

// rateLimitRetryAfter returns how long to wait before retrying after a rate-limit error.
// Uses Retry-After header (seconds) or x-ratelimit-reset (epoch), otherwise 60s.
func rateLimitRetryAfter(err error) time.Duration {
	var errResp *github.ErrorResponse
	if !errors.As(err, &errResp) || errResp.Response == nil {
		return 60 * time.Second
	}
	r := errResp.Response
	if s := r.Header.Get("Retry-After"); s != "" {
		if sec, parseErr := strconv.Atoi(s); parseErr == nil && sec > 0 {
			return time.Duration(sec) * time.Second
		}
	}
	if reset := r.Header.Get("X-RateLimit-Reset"); reset != "" {
		if epoch, parseErr := strconv.ParseInt(reset, 10, 64); parseErr == nil {
			wait := time.Until(time.Unix(epoch, 0))
			if wait > 0 && wait < 10*time.Minute {
				return wait
			}
		}
	}
	return 60 * time.Second
}

// WithRateLimitRetry runs fn and on rate-limit error waits and retries up to maxRetries times.
// Returns the last error if all retries fail or context is cancelled.
func WithRateLimitRetry(ctx context.Context, maxRetries int, fn func() error) error {
	var lastErr error
	for i := 0; i <= maxRetries; i++ {
		lastErr = fn()
		if lastErr == nil {
			return nil
		}
		if !IsRateLimitError(lastErr) || i == maxRetries {
			return lastErr
		}
		wait := rateLimitRetryAfter(lastErr)
		t := time.NewTimer(wait)
		select {
		case <-ctx.Done():
			t.Stop()
			return ctx.Err()
		case <-t.C:
		}
	}
	return lastErr
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
	return graphQLMarkPullRequestReadyForReview(ctx, client, *pr.NodeID)
}

// graphQLEndpointURL returns the validated GraphQL endpoint for the client's base URL.
func graphQLEndpointURL(client *github.Client) (string, error) {
	if client.BaseURL == nil {
		return "https://api.github.com/graphql", nil
	}
	u := client.BaseURL
	if strings.Contains(u.Host, "api.github.com") {
		return "https://api.github.com/graphql", nil
	}
	rawURL := u.Scheme + "://" + u.Host + "/api/graphql"
	parsed, err := url.ParseRequestURI(rawURL)
	if err != nil {
		return "", fmt.Errorf("invalid graphql URL: %w", err)
	}
	if parsed.Scheme != "https" && parsed.Scheme != "http" {
		return "", fmt.Errorf("URL must use https or http scheme: %s", rawURL)
	}
	return parsed.String(), nil
}

// extractHTTPClient returns the underlying http.Client from a github.Client.
func extractHTTPClient(client *github.Client) *http.Client {
	return client.Client()
}

// buildGraphQLRequestBody constructs the JSON body for a GraphQL mutation.
func buildGraphQLRequestBody(pullRequestNodeID string) ([]byte, error) {
	query := `mutation($id: ID!) { markPullRequestReadyForReview(input: { pullRequestId: $id }) { pullRequest { id isDraft } } }`
	body := map[string]interface{}{
		"query": query,
		"variables": map[string]string{
			"id": pullRequestNodeID,
		},
	}
	return json.Marshal(body)
}

// graphQLMarkPullRequestReadyForReview calls the GraphQL mutation to mark a draft PR as ready for review.
func graphQLMarkPullRequestReadyForReview(ctx context.Context, client *github.Client, pullRequestNodeID string) error {
	endpointURL, err := graphQLEndpointURL(client)
	if err != nil {
		return fmt.Errorf("graphql request: %w", err)
	}
	httpClient := extractHTTPClient(client)
	enc, err := buildGraphQLRequestBody(pullRequestNodeID)
	if err != nil {
		return fmt.Errorf("graphql request: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpointURL, bytes.NewReader(enc))
	if err != nil {
		return fmt.Errorf("graphql request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	// #nosec G704 -- URL and client come from graphQLEndpointURL/extractHTTPClient (GitHub API only). Cannot restructure away: taint analysis flags any param-derived HTTP call; we must call Do(req) with our client. See .docs/guides/security/golang-secure-coding.md § Approved #nosec exceptions.
	resp, err := httpClient.Do(req)
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
		msg := result.Errors[0].Message
		if strings.Contains(msg, "Resource not accessible") {
			return fmt.Errorf("%s — KIRA_GITHUB_TOKEN needs permission to mark a draft PR as ready. "+
				"Classic tokens: use the repo scope. Fine-grained tokens: grant Repository permissions \"Pull requests\" (Read and Write) and \"Contents\" (Write), "+
				"include this repo in Repository access, and set the token's resource owner to the repo owner. If it still fails, use a classic token. "+
				"See https://docs.github.com/en/rest/authentication/permissions-required-for-fine-grained-personal-access-tokens", msg)
		}
		return fmt.Errorf("graphql: %s", msg)
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

// FindPullRequestByWorkItemID finds a PR whose head branch matches the work item ID pattern {id}-*.
// Lists PRs (open first, then all) and returns the first whose Head.Ref has prefix workItemID+"-".
func FindPullRequestByWorkItemID(ctx context.Context, client *github.Client, owner, repo, workItemID string) (*github.PullRequest, error) {
	prefix := workItemID + "-"
	opts := &github.PullRequestListOptions{State: "open", ListOptions: github.ListOptions{PerPage: 100}}
	prs, _, err := client.PullRequests.List(ctx, owner, repo, opts)
	if err != nil {
		return nil, err
	}
	for _, pr := range prs {
		if pr.Head != nil && pr.Head.Ref != nil && strings.HasPrefix(*pr.Head.Ref, prefix) {
			return pr, nil
		}
	}
	// Not open; try closed/merged (idempotent path)
	opts.State = "closed"
	prs, _, err = client.PullRequests.List(ctx, owner, repo, opts)
	if err != nil {
		return nil, err
	}
	for _, pr := range prs {
		if pr.Head != nil && pr.Head.Ref != nil && strings.HasPrefix(*pr.Head.Ref, prefix) {
			return pr, nil
		}
	}
	return nil, nil
}

// IsPRClosedOrMerged returns true if the PR is closed or merged (idempotent path).
func IsPRClosedOrMerged(pr *github.PullRequest) bool {
	if pr == nil {
		return true
	}
	if pr.State != nil && *pr.State == "closed" {
		return true
	}
	return pr.MergedAt != nil
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

// GetPullRequest fetches a single pull request by number.
func GetPullRequest(ctx context.Context, client *github.Client, owner, repo string, number int) (*github.PullRequest, error) {
	pr, _, err := client.PullRequests.Get(ctx, owner, repo, number)
	return pr, err
}

// MergePullRequest merges a pull request with the given strategy and commit message.
// mergeMethod must be "merge", "squash", or "rebase".
func MergePullRequest(ctx context.Context, client *github.Client, owner, repo string, number int, commitMessage, mergeMethod string) (*github.PullRequestMergeResult, error) {
	opts := &github.PullRequestOptions{
		MergeMethod: mergeMethod,
	}
	result, _, err := client.PullRequests.Merge(ctx, owner, repo, number, commitMessage, opts)
	return result, err
}

// GetCombinedStatus returns the combined status for a ref (e.g. branch name or SHA).
func GetCombinedStatus(ctx context.Context, client *github.Client, owner, repo, ref string) (*github.CombinedStatus, error) {
	status, _, err := client.Repositories.GetCombinedStatus(ctx, owner, repo, ref, nil)
	return status, err
}

// ListPullRequestReviewComments lists all review comments on the specified pull request.
func ListPullRequestReviewComments(ctx context.Context, client *github.Client, owner, repo string, number int) ([]*github.PullRequestComment, error) {
	comments, _, err := client.PullRequests.ListComments(ctx, owner, repo, number, nil)
	return comments, err
}

const reviewThreadsPageSize = 100

var reviewThreadsQuery = `query($id: ID!, $after: String) {
  node(id: $id) {
    ... on PullRequest {
      reviewThreads(first: ` + strconv.Itoa(reviewThreadsPageSize) + `, after: $after) {
        pageInfo { hasNextPage endCursor }
        nodes { isResolved }
      }
    }
  }
}`

// CountUnresolvedReviewThreads returns the number of unresolved review threads on the PR.
// Uses the GraphQL API because the REST API does not expose thread resolution state.
// If pr is nil or pr.NodeID is empty, returns 0, nil (skip check when node ID not available).
func CountUnresolvedReviewThreads(ctx context.Context, client *github.Client, pr *github.PullRequest) (int, error) {
	if pr == nil || pr.NodeID == nil || strings.TrimSpace(*pr.NodeID) == "" {
		return 0, nil
	}
	graphqlURL, err := graphQLURL(client)
	if err != nil {
		return 0, err
	}
	nodeID := *pr.NodeID
	var after *string
	var total int
	for {
		page, err := fetchReviewThreadsPage(ctx, client, graphqlURL, nodeID, after)
		if err != nil {
			return 0, err
		}
		if page == nil {
			break
		}
		for _, n := range page.Nodes {
			if !n.IsResolved {
				total++
			}
		}
		if page.PageInfo == nil || !page.PageInfo.HasNextPage {
			break
		}
		after = &page.PageInfo.EndCursor
	}
	return total, nil
}

// reviewThreadsPage holds one page of review thread nodes and pageInfo from GraphQL.
type reviewThreadsPage struct {
	Nodes    []struct{ IsResolved bool } `json:"nodes"`
	PageInfo *struct {
		HasNextPage bool   `json:"hasNextPage"`
		EndCursor   string `json:"endCursor"`
	} `json:"pageInfo"`
}

// fetchReviewThreadsPage runs one GraphQL request for reviewThreads and returns the page or an error.
func fetchReviewThreadsPage(ctx context.Context, client *github.Client, graphqlURL, nodeID string, after *string) (*reviewThreadsPage, error) {
	variables := map[string]interface{}{"id": nodeID}
	if after != nil {
		variables["after"] = *after
	} else {
		variables["after"] = nil
	}
	body := map[string]interface{}{"query": reviewThreadsQuery, "variables": variables}
	enc, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("graphql request: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, graphqlURL, bytes.NewReader(enc))
	if err != nil {
		return nil, fmt.Errorf("graphql request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	transport := client.Client().Transport
	if transport == nil {
		transport = http.DefaultTransport
	}
	resp, err := transport.RoundTrip(req)
	if err != nil {
		return nil, fmt.Errorf("graphql request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("graphql returned status %d", resp.StatusCode)
	}
	var result struct {
		Data *struct {
			Node *struct {
				ReviewThreads *reviewThreadsPage `json:"reviewThreads"`
			} `json:"node"`
		} `json:"data"`
		Errors []struct {
			Message string `json:"message"`
		} `json:"errors"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("graphql decode: %w", err)
	}
	if len(result.Errors) > 0 {
		return nil, fmt.Errorf("graphql: %s", result.Errors[0].Message)
	}
	if result.Data == nil || result.Data.Node == nil {
		return nil, nil
	}
	return result.Data.Node.ReviewThreads, nil
}

// DeleteRef deletes a git reference (e.g. "heads/feature-branch").
func DeleteRef(ctx context.Context, client *github.Client, owner, repo, ref string) error {
	_, err := client.Git.DeleteRef(ctx, owner, repo, ref)
	return err
}
