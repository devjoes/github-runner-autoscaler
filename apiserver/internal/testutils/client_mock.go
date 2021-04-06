package testutils

import (
	"context"
	"fmt"
	"time"

	"github.com/devjoes/github-runner-autoscaler/apiserver/pkg/state"
	"github.com/google/go-github/v33/github"
	"github.com/stretchr/testify/mock"
)

type ClientMock struct {
	mock.Mock
	QueueLength              int
	Delay                    time.Duration
	State                    state.ClientState
	RecordGetWorkQueueLength bool
	RecordRefreshAccessToken bool
	ErrorOnGetQueuedJobs     bool
}

func (c *ClientMock) GetQueuedJobs(ctx context.Context) ([]*github.WorkflowRun, error) {
	if c.ErrorOnGetQueuedJobs {
		return nil, fmt.Errorf("%s Bang", c.GetState("foo").Name)
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

func FakeQueueData(size int) []*github.WorkflowRun {
	queued := "queued"
	wfId := int64(123)
	data := make([]*github.WorkflowRun, size)
	for i := 0; i < size; i++ {
		data[int64(i)] = &github.WorkflowRun{Status: &queued, WorkflowID: &wfId}
	}
	return data
}
