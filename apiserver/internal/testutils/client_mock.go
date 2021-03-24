package testutils

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/devjoes/github-runner-autoscaler/apiserver/pkg/state"
	"github.com/stretchr/testify/mock"
)

type ClientMock struct {
	mock.Mock
	QueueLength              int
	Delay                    time.Duration
	State                    state.ClientState
	RecordGetWorkQueueLength bool
	RecordRefreshAccessToken bool
	ErrorOnGetQueueLength    bool
}

func (c *ClientMock) GetQueueLength(ctx context.Context) (int, error) {
	if c.ErrorOnGetQueueLength {
		return 0, errors.New(fmt.Sprintf("%s Bang!", c.GetState("foo").Name))
	}
	time.Sleep(c.Delay)
	if c.RecordGetWorkQueueLength {
		c.Called()
	}
	return c.QueueLength, nil
}
func (c *ClientMock) GetState(name string) *state.ClientState { return &c.State }
func (c *ClientMock) SaveState(state *state.ClientState)      {}
