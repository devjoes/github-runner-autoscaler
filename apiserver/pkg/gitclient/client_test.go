package gitclient

import (
	"context"
	"testing"
	"time"

	"github.com/devjoes/github-runner-autoscaler/apiserver/internal/testutils"
	"github.com/devjoes/github-runner-autoscaler/apiserver/pkg/state"
	"github.com/stretchr/testify/assert"
)

const (
	StateName      = "foo"
	GetQueueLength = "GetQueueLength"
)

func TestCallsInnerClientIfLastRequestInvalid(t *testing.T) {
	test := func(status state.Status) {
		queueLength := 321
		stateProvider := state.NewInMemoryStateProvider()
		stateProvider.SetState(StateName, &state.ClientState{
			LastValue: 123,
			Status:    status,
		})
		innerClient := testutils.ClientMock{
			QueueLength: queueLength,
			State:       state.ClientState{}}
		client := NewClient(&innerClient, StateName, time.Hour, time.Hour, stateProvider)
		result, err := client.GetQueueLength(context.TODO())
		assert.Nil(t, err)
		assert.Equal(t, queueLength, result)
		s, _ := client.GetState()
		assert.Equal(t, queueLength, s.LastValue)
	}
	test(state.Unset)
	test(state.Errored)
}

func callEvery100Ms(t *testing.T, lastValue int, cacheWindowMs int, cacheWindowWhenEmptyMs int, callCount int) int {
	stateProvider := state.NewInMemoryStateProvider()
	stateProvider.SetState(StateName, &state.ClientState{
		LastValue: lastValue,
		Status:    state.Valid,
	})
	innerClient := testutils.ClientMock{
		RecordGetWorkQueueLength: true,
		Delay:                    time.Millisecond * 100,
		State:                    state.ClientState{},
		QueueLength:              lastValue}
	client := NewClient(&innerClient, StateName, time.Duration(cacheWindowMs)*time.Millisecond, time.Duration(cacheWindowWhenEmptyMs)*time.Millisecond, stateProvider)

	innerClient.On(GetQueueLength).Return(lastValue, nil)
	for i := 0; i < callCount; i++ {
		length, err := client.GetQueueLength(context.TODO())
		assert.Nil(t, err)
		assert.Equal(t, lastValue, length)
		time.Sleep(100 * time.Millisecond)
	}
	innerClient.AssertCalled(t, GetQueueLength)
	return len(innerClient.Calls)
}

func TestCachesDataFor500MsIfQueueIsEmpty(t *testing.T) {
	cacheMisses := callEvery100Ms(t, 0, 200, 500, 10)
	assert.Equal(t, 2, cacheMisses)
}

func TestCachesDataFor200MsIfQueueIsNotEmpty(t *testing.T) {
	cacheMisses := callEvery100Ms(t, 123, 200, 500, 10)
	assert.Equal(t, 5, cacheMisses)
}
