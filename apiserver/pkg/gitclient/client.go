package gitclient

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/devjoes/github-runner-autoscaler/apiserver/pkg/state"
	"github.com/google/go-github/v33/github"
	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
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
}

func (c *Client) GetQueuedJobs(ctx context.Context) ([]*github.WorkflowRun, error) {
	var jobQueue []*github.WorkflowRun
	cached := true
	var err error
	defer c.instrument(&jobQueue, &cached, &err)
	s, err := c.GetState()
	if err != nil {
		return nil, err
	}
	cacheUntil := s.LastRequest.Add(c.cacheWindow)
	if s.LastValue == nil || len(s.LastValue) == 0 {
		cacheUntil = s.LastRequest.Add(c.cacheWindowWhenEmpty)
		fmt.Println("empty")
	}

	if s.Status != state.Valid || time.Now().After(cacheUntil) {
		cached = false
		fmt.Printf("Cache miss %d %s %s %v\n", s.Status, cacheUntil.String(), time.Now().String(), s.LastValue)
		jobQueue, err = c.innerClient.GetQueuedJobs(ctx)
		if err != nil {
			s.Status = state.Errored
		} else {
			s.LastRequest = time.Now()
			s.LastValue = jobQueue
			s.Status = state.Valid
		}
		if saveErr := c.SaveState(s); saveErr != nil {
			if err != nil {
				saveErr = errors.Wrapf(err, "Encountered error %s. Also errored on save %s", err.Error(), saveErr.Error())
			}
			err = saveErr
		}
		fmt.Printf("Cached %v\n", err)
	} else {
		fmt.Println("Cache hit")
	}

	return s.LastValue, err
}

func (c *Client) GetState() (*state.ClientState, error) {
	return c.stateProvider.GetState(c.name)
}
func (c *Client) SaveState(state *state.ClientState) error {
	return c.stateProvider.SetState(c.name, state)
}

func NewClient(innerClient IStatelessClient, name string, cacheWindow time.Duration, cacheWindowWhenEmpty time.Duration, stateProvider state.IStateProvider) Client {
	return Client{
		innerClient:          innerClient,
		cacheWindow:          cacheWindow,
		cacheWindowWhenEmpty: cacheWindowWhenEmpty,
		name:                 name,
		stateProvider:        stateProvider,
	}
}

func (c *Client) instrument(labeledJobIds *[]*github.WorkflowRun, cached *bool, err *error) {
	labels := append([]string{c.name, strconv.FormatBool(*cached), strconv.FormatBool(*err != nil)})

	guageQueueLength.WithLabelValues(labels...).Set(float64(len(*labeledJobIds)))
	counterQueries.WithLabelValues(labels...).Inc()
}

var guageQueueLength *prometheus.GaugeVec
var counterQueries *prometheus.CounterVec

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
}
