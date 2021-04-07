package gitclient

//TODO: record rate limits
import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"io"
	"io/ioutil"
	"sort"
	"strings"

	utils "github.com/devjoes/github-runner-autoscaler/apiserver/pkg/utils"
	"github.com/google/go-github/v33/github"
	"golang.org/x/oauth2"
	"k8s.io/apimachinery/pkg/util/yaml"
	"k8s.io/klog/v2"
)

type IStatelessClient interface {
	GetQueuedJobs(ctx context.Context) ([]*github.WorkflowRun, error)
	GetRemainingCreditsForToken(ctx context.Context) (string, string, int, error)
	GetWorkflowData(ctx context.Context) (*map[int64]utils.WorkflowInfo, error)
}
type GithubClient struct {
	Owner      string
	Repository string
	client     *github.Client
	token      string
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

func tokenizeToken(token string) (string, string) {
	hash := sha256.Sum224([]byte(token))
	key := base64.RawStdEncoding.EncodeToString(hash[:])
	const sideCharsRevealed = 3
	redacted := strings.Builder{}
	for i, c := range token {
		if i < sideCharsRevealed || i >= len(token)-sideCharsRevealed {
			redacted.WriteRune(c)
		} else {
			redacted.WriteRune('*')
		}
	}
	return key, redacted.String()
}

func (c *GithubClient) GetRemainingCreditsForToken(ctx context.Context) (string, string, int, error) {
	key, name := tokenizeToken(c.token)
	limits, _, err := c.client.RateLimits(ctx)
	if err != nil {
		return key, name, 0, err
	}
	return key, name, limits.Core.Remaining, nil
}

type workflow struct {
	Name string `json:"name"`
	Jobs map[string]struct {
		RunsOn []string `json:"runs-on"`
	} `json:"jobs"`
}

func (c *GithubClient) getLabels(ctx context.Context, path string) ([]string, error) {
	reader, _, err := c.client.Repositories.DownloadContents(ctx, c.Owner, c.Repository, path, &github.RepositoryContentGetOptions{})
	if err != nil {
		return nil, err
	}
	return c.processWorkflow(reader)
}
func (c *GithubClient) processWorkflow(reader io.ReadCloser) ([]string, error) {
	data, err := ioutil.ReadAll(reader)
	if err != nil {
		return nil, err
	}
	wf := workflow{}
	err = yaml.Unmarshal(data, &wf)
	if err != nil {
		return nil, err
	}
	var runsOn []string
	for _, j := range wf.Jobs {
		if j.RunsOn != nil {
			for _, l := range j.RunsOn {
				if !utils.ContainsStr(runsOn, l) {
					runsOn = append(runsOn, l)
				}
			}
		}
	}
	sort.Strings(runsOn)
	return runsOn, nil
}
func (c *GithubClient) GetWorkflowData(ctx context.Context) (*map[int64]utils.WorkflowInfo, error) {
	results := make(map[int64]utils.WorkflowInfo)
	wfs, _, err := c.client.Actions.ListWorkflows(ctx, c.Owner, c.Repository, &github.ListOptions{})
	if err != nil {
		return nil, err
	}
	for _, w := range wfs.Workflows {
		labels, err := c.getLabels(ctx, *w.Path)
		if err != nil {
			klog.Warningf("Failed to get workflow info for %s in %s/%s: %s", *w.Path, c.Owner, c.Repository, err.Error())
		}
		results[*w.ID] = utils.WorkflowInfo{
			ID:     *w.ID,
			Name:   *w.Name,
			Labels: labels,
		}
	}

	return &results, nil
}
