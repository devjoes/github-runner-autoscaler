package gitclient

import (
	"context"

	"github.com/google/go-github/v33/github"
	"golang.org/x/oauth2"
)

type GithubClient struct {
	Owner      string
	Repository string
	client     *github.Client
}

type result struct {
	Length int
	Error  error
}

func NewGitHubClient(token string, owner string, repository string) GithubClient {
	ctx := context.Background()
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: token},
	)
	tc := oauth2.NewClient(ctx, ts)
	client := github.NewClient(tc)
	return GithubClient{
		client:     client,
		Owner:      owner,
		Repository: repository,
	}
}

func (c *GithubClient) getQueueLengthByStatus(status string, ctx context.Context, cResults chan result) {
	runs, _, err := c.client.Actions.ListRepositoryWorkflowRuns(ctx, c.Owner, c.Repository, &github.ListWorkflowRunsOptions{
		Status: status,
	})
	length := 0
	if err == nil {
		length = *runs.TotalCount
	}
	cResults <- result{
		Length: length,
		Error:  err,
	}
}

func (c *GithubClient) GetQueueLength(ctx context.Context) (int, error) {
	statuses := []string{"queued",
		"waiting",
		"requested",
		"in_progress"}
	cResults := make(chan result, len(statuses))
	for _, s := range statuses {
		go c.getQueueLengthByStatus(s, ctx, cResults)
	}

	length := 0
	for i := 0; i < len(statuses); i++ {
		res := <-cResults
		if res.Error != nil {
			return 0, res.Error
		}
		length += res.Length
	}
	return length, nil
}
