package gitclient

import (
	"bytes"
	"context"
	"io/ioutil"
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

func TestLabelsExtractionEmpty(t *testing.T) {
	wfYaml :=
		`
name: CI
jobs:
  build:
    steps:
      - uses: actions/checkout@v2`

	client := GithubClient{}
	reader := ioutil.NopCloser(bytes.NewReader([]byte(wfYaml)))
	labels, err := client.processWorkflow(reader)
	assert.Nil(t, err)
	assert.Empty(t, labels)
}

func TestLabelsExtraction(t *testing.T) {
	wfYaml :=
		`
name: CI
jobs:
  build:
    runs-on: [foo,bar]
    steps:
      - uses: actions/checkout@v2
  deploy:
    runs-on: [foo,baz]
    steps:
    - uses: actions/checkout@v2`

	client := GithubClient{}
	reader := ioutil.NopCloser(bytes.NewReader([]byte(wfYaml)))
	labels, err := client.processWorkflow(reader)
	assert.Nil(t, err)
	assert.Equal(t, []string{"bar", "baz", "foo"}, labels)
}
