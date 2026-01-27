package github

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestCreateGitHubClient tests the CreateGitHubClient function
func TestCreateGitHubClient(t *testing.T) {
	t.Run("creates client with valid token", func(t *testing.T) {
		client, err := CreateGitHubClient("test-token-12345")
		require.NoError(t, err)
		require.NotNil(t, client)
		// Verify client is properly configured by checking it's not nil
		// The actual authentication will be tested when the client is used
		assert.NotNil(t, client)
	})

	t.Run("trims whitespace from token", func(t *testing.T) {
		// Token with whitespace should be trimmed and still work
		client, err := CreateGitHubClient("  test-token-12345  ")
		require.NoError(t, err)
		require.NotNil(t, client)
		assert.NotNil(t, client)
	})

	t.Run("returns error for empty token", func(t *testing.T) {
		client, err := CreateGitHubClient("")
		require.Error(t, err)
		assert.Nil(t, client)
		assert.Contains(t, err.Error(), "GitHub token cannot be empty")
	})

	t.Run("returns error for whitespace-only token", func(t *testing.T) {
		testCases := []string{
			"   ",
			"\n\t",
			"  \n\t  ",
		}
		for _, token := range testCases {
			client, err := CreateGitHubClient(token)
			require.Error(t, err, "should return error for whitespace-only token: %q", token)
			assert.Nil(t, client)
			assert.Contains(t, err.Error(), "GitHub token cannot be empty")
		}
	})

	t.Run("creates client with realistic token format", func(t *testing.T) {
		// Test with a token that looks like a real GitHub token (ghp_ prefix)
		client, err := CreateGitHubClient("ghp_test1234567890abcdefghijklmnopqrstuvwxyz")
		require.NoError(t, err)
		require.NotNil(t, client)
		assert.NotNil(t, client)
	})

	// Note: We don't test actual API calls here because:
	// 1. We don't want to require a real GitHub token in unit tests
	// 2. We don't want to expose tokens in test code
	// 3. The actual API functionality will be tested in integration tests
	// The unit tests verify that the client is created correctly and error handling works
}
