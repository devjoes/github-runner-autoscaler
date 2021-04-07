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

func TestTokenization(t *testing.T) {
	token := "324hj324h3h32234hj34hj3h23hj2323h3h23hj234hj23"
	key, name := tokenizeToken(token)
	assert.Equal(t, "Qgw+4u9Aw0jqoYTfJwVFLsjW067wO4YwXLYCNw", key)
	assert.Equal(t, "324****************************************j23", name)
}
