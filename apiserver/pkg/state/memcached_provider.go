package state

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/memcachier/mc"
	"google.golang.org/appengine/memcache"
)

type MemcachedStateProvider struct {
	cache *mc.Client
}

func NewMemcachedStateProvider(servers string, username string, password string) (*MemcachedStateProvider, error) {
	cache := mc.NewMC(servers, username, password)
	return &MemcachedStateProvider{cache: cache}, nil
}

func (p *MemcachedStateProvider) GetState(key string) (*ClientState, error) {
	val, _, _, err := p.cache.Get(key)
	if err == nil {
		var state ClientState
		err = json.Unmarshal([]byte(val), &state)
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
	_, err = p.cache.Set(key, string(data), 0, 60*60, 0)
	return err
}
