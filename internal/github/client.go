// Package github provides GitHub-related utilities for the kira tool.
package github

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/go-github/v58/github"
	"golang.org/x/oauth2"
)

// CreateGitHubClient creates an authenticated GitHub API client using the provided token.
// The token is used to create an OAuth2 client that authenticates all API requests.
// Returns an error if the token is empty or contains only whitespace.
func CreateGitHubClient(token string) (*github.Client, error) {
	// Validate token is not empty or whitespace-only
	token = strings.TrimSpace(token)
	if token == "" {
		return nil, fmt.Errorf("GitHub token cannot be empty")
	}

	// Create OAuth2 token source
	ctx := context.Background()
	ts := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: token})
	tc := oauth2.NewClient(ctx, ts)

	// Create and return GitHub client
	client := github.NewClient(tc)
	return client, nil
}
