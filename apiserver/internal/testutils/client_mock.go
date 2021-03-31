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

func (c *ClientMock) GetQueueLength(ctx context.Context) (map[int64]map[string]string, error) {
	if c.ErrorOnGetQueueLength {
		return nil, errors.New(fmt.Sprintf("%s Bang!", c.GetState("foo").Name))
	}
	time.Sleep(c.Delay)
	if c.RecordGetWorkQueueLength {
		c.Called()
	}
	return FakeQueueData(c.QueueLength), nil
}
func (c *ClientMock) GetState(name string) *state.ClientState { return &c.State }
func (c *ClientMock) SaveState(state *state.ClientState)      {}

const WfIdLabel = "wf_id"
const JobStatusLabel = "job_status"

func FakeQueueData(size int) map[int64]map[string]string {
	data := map[int64]map[string]string{}
	for i := 0; i < size; i++ {
		data[int64(i)] = map[string]string{JobStatusLabel: "queued", WfIdLabel: "123"}
	}
	return data
}
