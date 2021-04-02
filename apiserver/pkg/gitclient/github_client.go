package gitclient

//TODO: record rate limits
import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/go-github/v33/github"
	"golang.org/x/oauth2"
)

type IStatelessClient interface {
	GetQueuedJobs(ctx context.Context) ([]*github.WorkflowRun, error)
}
type GithubClient struct {
	Owner      string
	Repository string
	client     *github.Client
}

type result struct {
	Error  error
	Labels map[string]int
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

type workflowRun struct {
	ID   *int64  `json:"id,omitempty"`
	Name *string `json:"name,omitempty"`
}

func getExtraWfInfo(resp *github.Response) map[int64]workflowRun {

	type workflowRuns struct {
		TotalCount   *int           `json:"total_count,omitempty"`
		WorkflowRuns []*workflowRun `json:"workflow_runs,omitempty"`
	}
	wfr := workflowRuns{}
	body := resp.Body
	err := json.NewDecoder(body).Decode(&wfr)
	fmt.Println(err)
	fmt.Println(wfr)
	wfrMap := map[int64]workflowRun{}
	for _, r := range wfr.WorkflowRuns {
		wfrMap[*r.ID] = *r
	}
	return wfrMap
}

// func (c *GithubClient) getQueueLengthByStatus(status string, ctx context.Context, cResults chan result) {
// 	runs, resp, err := c.client.Actions.ListRepositoryWorkflowRuns(ctx, c.Owner, c.Repository, &github.ListWorkflowRunsOptions{
// 		Status: status,
// 		ListOptions: github.ListOptions{
// 			PerPage: 100,
// 		},
// 	})
// 	length := 0
// 	if err == nil {
// 		length = *runs.TotalCount
// 	}
// 	labels := GetLabels(length, runs.WorkflowRuns, resp, )

// 	cResults <- result{
// 		Labels: labels,
// 		Error:  err,
// 	}
// }

func (c *GithubClient) GetQueuedJobs(ctx context.Context) ([]*github.WorkflowRun, error) {
	// This wastes credits - just getting the top 100 should work pretty much all of the time
	// statuses := []string{
	// 	"queued",
	// 	"waiting",
	// 	"requested",
	// 	"in_progress"}
	// cResults := make(chan result, len(statuses)+1)
	// for _, s := range statuses {
	// 	go c.getQueueLengthByStatus(s, ctx, cResults)
	// }

	// lengths := make(map[string]int)
	// lock := sync.Mutex{}
	// for i := 0; i < len(statuses); i++ {
	// 	res := <-cResults
	// 	if res.Error != nil {
	// 		return lengths, res.Error
	// 	}
	// 	lock.Lock()
	// 	for k := range res.Labels {
	// 		current := lengths[k]
	// 		lengths[k] = current + res.Labels[k]
	// 	}
	// 	lock.Unlock()
	// }
	runs, _, err := c.client.Actions.ListRepositoryWorkflowRuns(ctx, c.Owner, c.Repository, &github.ListWorkflowRunsOptions{
		ListOptions: github.ListOptions{
			PerPage: 100,
		},
	})
	if err != nil {
		return nil, err
	}
	jobs := filterJobsByStatus(runs.WorkflowRuns)
	return jobs, nil
}

func filterJobsByStatus(jobs []*github.WorkflowRun) []*github.WorkflowRun {
	filtered := []*github.WorkflowRun{}
	statuses := map[string]bool{
		"queued":      true,
		"waiting":     true,
		"requested":   true,
		"in_progress": true,
	}
	for _, j := range jobs {
		if statuses[*j.Status] {
			filtered = append(filtered, j)
		}
	}
	return filtered
}
