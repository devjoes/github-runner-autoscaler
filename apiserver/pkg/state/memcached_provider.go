package state

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"

	"github.com/devjoes/github-runner-autoscaler/apiserver/pkg/utils"
	"github.com/memcachier/mc/v3"
	"k8s.io/apimachinery/pkg/util/rand"
)

const (
	stateCacheTime  = 60 * 60
	wfInfoCacheTime = 5 * 60
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

	key := fmt.Sprintf("test_%s", rand.String(5))
	// Internally the client picks a server based on the hash of the key
	// so it is a good test to make the key dynamic
	_, err := cache.Set(key, "ok", 0, 1, 0)
	if err != nil {
		pass := password
		if len(pass) > 3 {
			pass = fmt.Sprintf("%s******", password[:3])
		}
		return nil, fmt.Errorf("Could not connect to cache with '%s' '%s' '%s': %s", servers, username, pass, err.Error())
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
	_, err = p.cache.Set(key, string(data), 0, stateCacheTime, 0)
	return err
}

func (p *MemcachedStateProvider) GetWorkflowInfo(key string) (*map[int64]utils.WorkflowInfo, error) {
	val, _, _, err := p.cache.Get(key)
	if err == nil {
		var wfInfo map[int64]utils.WorkflowInfo
		err = json.Unmarshal([]byte(val), &wfInfo)
		if err == nil {
			return &wfInfo, nil
		}
	}
	if errors.Is(err, mc.ErrNotFound) {
		return nil, nil
	}
	return nil, err
}

func (p *MemcachedStateProvider) SetWorkflowInfo(key string, wfInfo *map[int64]utils.WorkflowInfo) error {
	data, err := json.Marshal(wfInfo)
	if err != nil {
		return err
	}
	_, err = p.cache.Set(key, string(data), 0, wfInfoCacheTime, 0)
	return err
}
