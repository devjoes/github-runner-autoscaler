package gitclient

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/devjoes/github-runner-autoscaler/apiserver/pkg/state"
	utils "github.com/devjoes/github-runner-autoscaler/apiserver/pkg/utils"
	"github.com/google/go-github/v33/github"
	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"k8s.io/klog/v2"
)

type Status int8

const (
	Unset Status = iota
	Valid
	Errored
)

type IClient interface {
	GetQueuedJobs(ctx context.Context) (int, error)
	GetState() (*state.ClientState, error)
	SaveState(state *state.ClientState) error
}

type Client struct {
	innerClient          IStatelessClient
	cacheWindow          time.Duration
	cacheWindowWhenEmpty time.Duration
	stateProvider        state.IStateProvider
	name                 string
	gitOwnerRepo         string
}

func (c *Client) GetWorkflowInfo(ctx context.Context) (map[int64]utils.WorkflowInfo, error) {
	wfData, err := c.stateProvider.GetWorkflowInfo(c.gitOwnerRepo)
	if err != nil {
		return nil, err
	}
	if wfData == nil {
		wfData, err = c.innerClient.GetWorkflowData(ctx)
		if err != nil {
			return nil, err
		}
	}
	err = c.stateProvider.SetWorkflowInfo(c.gitOwnerRepo, wfData)
	if err != nil {
		return nil, err
	}
	return *wfData, err
}

func (c *Client) GetQueuedJobs(ctx context.Context) ([]*github.WorkflowRun, *time.Time, error) {
	var jobQueue []*github.WorkflowRun
	cached := true
	var err error
	defer c.instrument(&jobQueue, &cached, &err)
	s, err := c.GetState()
	if err != nil {
		return nil, nil, err
	}
	cacheUntil := s.LastRequest.Add(c.cacheWindow)
	if s.LastValue == nil || len(s.LastValue) == 0 {
		cacheUntil = s.LastRequest.Add(c.cacheWindowWhenEmpty)
	}

	if s.Status != state.Valid || time.Now().UTC().After(cacheUntil) {
		cached = false
		klog.V(5).Infof("Cache miss %d %s %s %v", s.Status, cacheUntil.String(), time.Now().UTC().String(), s.LastValue)

		jobQueue, err = c.innerClient.GetQueuedJobs(ctx)
		if err != nil {
			s.Status = state.Errored
		} else {
			s.LastRequest = time.Now().UTC()
			s.LastValue = jobQueue
			s.Status = state.Valid
		}
		if saveErr := c.SaveState(s); saveErr != nil {
			if err != nil {
				saveErr = errors.Wrapf(err, "Encountered error %s. Also errored on save %s", err.Error(), saveErr.Error())
			}
			err = saveErr
		}
		if err != nil {
			klog.Warningf("Error whilst processing %s %s", c.name, err.Error())
		}
	}

	return s.LastValue, &s.LastRequest, err
}

func (c *Client) GetState() (*state.ClientState, error) {
	return c.stateProvider.GetState(c.name)
}
func (c *Client) SaveState(state *state.ClientState) error {
	return c.stateProvider.SetState(c.name, state)
}

func NewClient(innerClient IStatelessClient, name string, gitOwnerRepo string, cacheWindow time.Duration, cacheWindowWhenEmpty time.Duration, stateProvider state.IStateProvider) Client {
	return Client{
		innerClient:          innerClient,
		cacheWindow:          cacheWindow,
		cacheWindowWhenEmpty: cacheWindowWhenEmpty,
		name:                 name,
		gitOwnerRepo:         gitOwnerRepo,
		stateProvider:        stateProvider,
	}
}

func (c *Client) instrument(labeledJobIds *[]*github.WorkflowRun, cached *bool, err *error) {
	labels := append([]string{c.name, strconv.FormatBool(*cached), strconv.FormatBool(*err != nil)})

	guageQueueLength.WithLabelValues(labels...).Set(float64(len(*labeledJobIds)))
	counterQueries.WithLabelValues(labels...).Inc()

	if !*cached {
		key, name, val, e := c.innerClient.GetRemainingCreditsForToken(context.Background())
		if e != nil {
			fmt.Printf("Error getting credit count: %s\n", e.Error())
			return
		}
		guageGithubCredits.WithLabelValues(key, name).Set(float64(val))
	}
}

var guageQueueLength *prometheus.GaugeVec
var counterQueries *prometheus.CounterVec
var guageGithubCredits *prometheus.GaugeVec

func init() {
	labelNames := []string{
		"name",
		"cache_hit",
		"failed"}
	labelNames = append(labelNames)
	guageQueueLength = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "workflow_queue_length",
		Help: "Number of jobs in queue when queried",
	}, labelNames)
	counterQueries = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "workflow_queue_queries",
		Help: "Number of times a workflow queue is queried",
	}, labelNames)
	guageGithubCredits = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "github_credits",
		Help: "Remaining rate limit creds by token",
	}, []string{"token_id", "token_name"})

}
