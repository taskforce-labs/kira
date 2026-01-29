package github

import (
	"net/http"
	"testing"

	apigh "github.com/google/go-github/v58/github"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func respWithStatus(code int) *apigh.Response {
	return &apigh.Response{Response: &http.Response{StatusCode: code}}
}

func TestIsRetryable(t *testing.T) {
	t.Run("nil resp and err is retryable (network)", func(t *testing.T) {
		assert.True(t, isRetryable(nil, assert.AnError))
	})
	t.Run("429 is retryable", func(t *testing.T) {
		assert.True(t, isRetryable(respWithStatus(http.StatusTooManyRequests), assert.AnError))
	})
	t.Run("5xx is retryable", func(t *testing.T) {
		for _, code := range []int{500, 502, 503} {
			assert.True(t, isRetryable(respWithStatus(code), assert.AnError), "code %d", code)
		}
	})
	t.Run("4xx except 429 is not retryable", func(t *testing.T) {
		for _, code := range []int{400, 401, 403, 404, 422} {
			assert.False(t, isRetryable(respWithStatus(code), assert.AnError), "code %d", code)
		}
	})
	t.Run("nil err is not retryable", func(t *testing.T) {
		assert.False(t, isRetryable(respWithStatus(500), nil))
	})
}

func TestRetryWithBackoff_SuccessFirstTry(t *testing.T) {
	calls := 0
	result, resp, err := RetryWithBackoff(func() (string, *apigh.Response, error) {
		calls++
		return "ok", respWithStatus(200), nil
	})
	require.NoError(t, err)
	assert.Equal(t, "ok", result)
	assert.Equal(t, 200, resp.StatusCode)
	assert.Equal(t, 1, calls)
}

func TestRetryWithBackoff_NonRetryableError(t *testing.T) {
	calls := 0
	_, _, err := RetryWithBackoff(func() (string, *apigh.Response, error) {
		calls++
		return "", respWithStatus(404), assert.AnError
	})
	require.Error(t, err)
	assert.Equal(t, 1, calls)
}
