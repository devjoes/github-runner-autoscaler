package state

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"

	"github.com/memcachier/mc"
)

type MemcachedStateProvider struct {
	cache *mc.Client
}

func NewMemcachedStateProvider(servers string, username string, argPassword string) (*MemcachedStateProvider, error) {
	password := argPassword
	if argPassword == "" && os.Getenv("MEMCACHED_PASSWORD") != "" {
		password = os.Getenv("MEMCACHED_PASSWORD")
	}
	cache := mc.NewMC(servers, username, password)
	_, err := cache.Set("ok", "ok", 0, 10, 0)
	if err != nil {
		return nil, fmt.Errorf("Could not connect to cache with '%s' '%s' '%s': %s", servers, username, password, err.Error())
	}
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
	if errors.Is(err, mc.ErrNotFound) {
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
