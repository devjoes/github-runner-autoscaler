package state

import (
	"sync"

	"github.com/devjoes/github-runner-autoscaler/apiserver/pkg/utils"
)

type IStateProvider interface {
	GetState(key string) (*ClientState, error)
	SetState(key string, state *ClientState) error
	GetWorkflowInfo(key string) (*map[int64]utils.WorkflowInfo, error)
	SetWorkflowInfo(key string, wfInfo *map[int64]utils.WorkflowInfo) error
}

type InMemoryStateProvider struct {
	ClientStateData      map[string]ClientState
	clientStateDataMutex *sync.RWMutex
	WorkflowInfo         map[string]map[int64]utils.WorkflowInfo
	workflowInfoMutex    *sync.RWMutex
}

func (p *InMemoryStateProvider) GetState(key string) (*ClientState, error) {
	p.clientStateDataMutex.RLock()
	defer p.clientStateDataMutex.RUnlock()
	s, found := p.ClientStateData[key]
	if !found {
		return NewClientState(key), nil
	}
	return &s, nil
}

func (p *InMemoryStateProvider) SetState(key string, state *ClientState) error {
	p.clientStateDataMutex.Lock()
	defer p.clientStateDataMutex.Unlock()
	p.ClientStateData[key] = *state
	return nil
}

func (p *InMemoryStateProvider) GetWorkflowInfo(key string) (*map[int64]utils.WorkflowInfo, error) {
	p.workflowInfoMutex.RLock()
	defer p.workflowInfoMutex.RUnlock()
	s, found := p.WorkflowInfo[key]
	if !found {
		return nil, nil
	}
	return &s, nil
}

func (p *InMemoryStateProvider) SetWorkflowInfo(key string, state *map[int64]utils.WorkflowInfo) error {
	p.workflowInfoMutex.Lock()
	defer p.workflowInfoMutex.Unlock()
	p.WorkflowInfo[key] = *state
	return nil
}

func NewInMemoryStateProvider() *InMemoryStateProvider {
	return NewInMemoryStateProviderWithData(make(map[string]ClientState))
}

func NewInMemoryStateProviderWithData(data map[string]ClientState) *InMemoryStateProvider {
	return &InMemoryStateProvider{
		clientStateDataMutex: &sync.RWMutex{},
		workflowInfoMutex:    &sync.RWMutex{},
		ClientStateData:      data,
		WorkflowInfo:         make(map[string]map[int64]utils.WorkflowInfo),
	}
}
