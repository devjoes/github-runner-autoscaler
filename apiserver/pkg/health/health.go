package health

import (
	"net/http"

	"github.com/devjoes/github-runner-autoscaler/apiserver/pkg/config"
	"github.com/devjoes/github-runner-autoscaler/apiserver/pkg/state"
	"k8s.io/klog/v2"
)

type Health struct {
	conf config.Config
}

func NewHealth(conf config.Config) Health {
	return Health{conf: conf}
}

func (h *Health) Livez() http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}
}

func (h *Health) Readyz() http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		if len(h.conf.MemcachedServers) > 0 {
			_, err := state.NewMemcachedStateProvider(h.conf.MemcachedServers, h.conf.MemcachedUser, h.conf.MemcachedPass)
			if err != nil {
				klog.Errorf("Readiness probe failed with: %s", err.Error())
				http.Error(w, http.StatusText(http.StatusServiceUnavailable), http.StatusServiceUnavailable)
				return
			}
		}
		w.WriteHeader(http.StatusOK)
	}
}
