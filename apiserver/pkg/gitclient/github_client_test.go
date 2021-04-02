package gitclient

import (
	"context"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAPIAccess(t *testing.T) {
	token := os.Getenv("GITHUB_TOKEN")
	if token == "" {
		t.Skip("Skipping TestAPIAccess because GITHUB_TOKEN environment variable was not set")
	}
	client := NewGitHubClient(token, "devjoes", "test")
	_, err := client.GetQueuedJobs(context.Background())
	assert.Nil(t, err)
}
