package state

import "sync"

type IStateProvider interface {
	GetState(key string) (*ClientState, error)
	SetState(key string, state *ClientState) error
}

type InMemoryStateProvider struct {
	Data  map[string]ClientState
	mutex *sync.RWMutex
}

func (p *InMemoryStateProvider) GetState(key string) (*ClientState, error) {
	p.mutex.RLock()
	defer p.mutex.RUnlock()
	s := p.Data[key]
	if s.Name == "" {
		return NewClientState(key), nil
	}
	return &s, nil
}

func (p *InMemoryStateProvider) SetState(key string, state *ClientState) error {
	p.mutex.Lock()
	defer p.mutex.Unlock()
	p.Data[key] = *state
	return nil
}

func NewInMemoryStateProvider() *InMemoryStateProvider {
	return NewInMemoryStateProviderWithData(make(map[string]ClientState))
}

func NewInMemoryStateProviderWithData(data map[string]ClientState) *InMemoryStateProvider {
	return &InMemoryStateProvider{
		mutex: &sync.RWMutex{},
		Data:  data,
	}
}
