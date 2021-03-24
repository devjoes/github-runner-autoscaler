package gitclient

import (
	"context"
	"strconv"
	"time"

	"github.com/devjoes/github-runner-autoscaler/apiserver/pkg/state"
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
	GetQueueLength(ctx context.Context) (int, error)
	GetState() (*state.ClientState, error) //TODO: This should probably be loaded (and saved) from redis, memcached or something if we want multiple replicas
	SaveState(state *state.ClientState) error
}

type IStatelessClient interface {
	GetQueueLength(ctx context.Context) (int, error)
}

type Client struct {
	innerClient          IStatelessClient
	cacheWindow          time.Duration
	cacheWindowWhenEmpty time.Duration
	stateProvider        state.IStateProvider
	name                 string
}

func (c *Client) GetQueueLength(ctx context.Context) (int, error) {
	var length int
	cached := true
	var err error
	defer c.instrument(&length, &cached, &err)
	s, err := c.GetState()
	if err != nil {
		return 0, err
	}
	cacheUntil := s.LastRequest.Add(c.cacheWindow)
	if s.LastValue == 0 {
		cacheUntil = s.LastRequest.Add(c.cacheWindowWhenEmpty)
	}

	if s.Status != state.Valid || time.Now().After(cacheUntil) {
		cached = false
		length, err = c.innerClient.GetQueueLength(ctx)
		if err != nil {
			s.Status = state.Errored
		} else {
			s.LastRequest = time.Now()
			s.LastValue = length
			s.Status = state.Valid
		}
		if saveErr := c.SaveState(s); saveErr != nil {
			if err != nil {
				saveErr = errors.Wrapf(err, "Encountered error %s. Also errored on save %s", err.Error(), saveErr.Error())
			}
			err = saveErr
		}
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

func (c *Client) instrument(length *int, cached *bool, err *error) {
	labels := []string{c.name, strconv.FormatBool(*cached), strconv.FormatBool(*err != nil)}
	guageQueueLength.WithLabelValues(labels...).Set(float64(*length))
	counterQueries.WithLabelValues(labels...).Inc()
}

var guageQueueLength *prometheus.GaugeVec
var counterQueries *prometheus.CounterVec

func init() {
	labelNames := []string{"name",
		"cache_hit",
		"failed"}
	guageQueueLength = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "workflow_queue_length",
		Help: "Number of jobs in queue when queried",
	}, labelNames)
	counterQueries = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "workflow_queue_queries",
		Help: "Number of times a workflow queue is queried",
	}, labelNames)
}
