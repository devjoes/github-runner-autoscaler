package state

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/bradfitz/gomemcache/memcache"
)

type MemcachedStateProvider struct {
	mc *memcache.Client
}

func NewMemcachedStateProvider(servers []string) (*MemcachedStateProvider, error) {
	mc := memcache.New(servers...)
	return &MemcachedStateProvider{mc: mc}, nil
}

func (p *MemcachedStateProvider) GetState(key string) (*ClientState, error) {
	s, err := p.mc.Get(key)
	if err == nil {
		var state ClientState
		err = json.Unmarshal(s.Value, &state)
		if err == nil {
			return &state, nil
		}
	}
	if errors.Is(err, memcache.ErrCacheMiss) {
		return NewClientState(key), nil
	}
	//TODO: Configurable
	return nil, fmt.Errorf("memcache server unreachable. aborting to avoid potential rate limitting. %s", err.Error())
}

func (p *MemcachedStateProvider) SetState(key string, state *ClientState) error {
	data, err := json.Marshal(state)
	if err != nil {
		return err
	}
	return p.mc.Set(&memcache.Item{Key: key, Value: data, Expiration: 60 * 60})
}
