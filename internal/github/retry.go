// Package github provides GitHub-related utilities for the kira tool.
package github

import (
	"context"
	"errors"
	"net/http"
	"time"

	apigh "github.com/google/go-github/v58/github"
)

const (
	// maxRetries is the maximum number of retry attempts for GitHub API calls (PRD: 3).
	maxRetries = 3
	// retryBackoffBase is the initial backoff duration (PRD: 1s, 2s, 4s).
	retryBackoffBase = 1 * time.Second
)

// isRetryable returns true if the error and response indicate a transient failure
// that should be retried: 429 (rate limit), 5xx (server errors), or nil resp (often network).
// Do not retry on 4xx except 429 (per PRD).
func isRetryable(resp *apigh.Response, err error) bool {
	if err == nil {
		return false
	}
	// No response often means network/connection failure
	if resp == nil {
		return true
	}
	// Retry on rate limit (429)
	if resp.StatusCode == http.StatusTooManyRequests {
		return true
	}
	// Retry on server errors (5xx)
	if resp.StatusCode >= 500 && resp.StatusCode < 600 {
		return true
	}
	// Context deadline exceeded / canceled can be retried
	if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
		return true
	}
	return false
}

// RetryWithBackoff runs fn up to maxRetries times with exponential backoff (1s, 2s, 4s).
// It retries only when fn returns an error that isRetryable(resp, err).
// Returns the first successful result, or the last error if all attempts fail.
func RetryWithBackoff[T any](fn func() (T, *apigh.Response, error)) (T, *apigh.Response, error) {
	var zero T
	var lastResp *apigh.Response
	var lastErr error
	for attempt := 0; attempt < maxRetries; attempt++ {
		result, resp, err := fn()
		if err == nil {
			return result, resp, nil
		}
		lastResp, lastErr = resp, err
		if !isRetryable(resp, err) {
			return zero, resp, err
		}
		if attempt == maxRetries-1 {
			break
		}
		// Exponential backoff: 1s, 2s, 4s (PRD)
		backoff := retryBackoffBase * time.Duration(1<<uint(attempt))
		time.Sleep(backoff)
	}
	return zero, lastResp, lastErr
}
