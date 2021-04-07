package host

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/devjoes/github-runner-autoscaler/apiserver/pkg/config"
	client "github.com/devjoes/github-runner-autoscaler/apiserver/pkg/gitclient"
	labeling "github.com/devjoes/github-runner-autoscaler/apiserver/pkg/labeling"
	"github.com/devjoes/github-runner-autoscaler/apiserver/pkg/state"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/klog/v2"
)

type Host struct {
	config        config.Config
	stateProvider state.IStateProvider
}

func (h *Host) GetAllMetricNames(namespace string) ([]string, error) {
	wfs := h.config.GetAllWorkflows()
	metrics := make([]string, len(wfs))
	for i, wf := range wfs {
		if namespace == "" || wf.Namespace == namespace {
			metrics[i] = wf.Name
		}
	}
	return metrics, nil
}

const MetricErrNotFound string = "metric not found"

func (h *Host) QueryMetric(key string, selector labels.Selector) (int, map[string][]string, *config.GithubWorkflowConfig, error) {
	wf, err := h.config.GetWorkflow(key)
	if err != nil {
		return 0, nil, nil, err
	}
	if wf == nil {
		return 0, nil, nil, errors.New(MetricErrNotFound)
	}
	client := h.getClient(wf)
	jobs, err := client.GetQueuedJobs(context.Background())
	filteredJobs, matchedLabels := labeling.FilterBySelector(jobs, wf, selector)
	return len(filteredJobs), matchedLabels, wf, err
}

func (h *Host) getClient(wf *config.GithubWorkflowConfig) client.Client {
	githubClient := client.NewGitHubClient(wf.Token, wf.Owner, wf.Repository)
	key := wf.Name
	return client.NewClient(&githubClient, key, h.config.CacheWindow, h.config.CacheWindowWhenEmpty, h.stateProvider)
}

func NewHost(conf config.Config, params ...interface{}) (*Host, error) {
	var stateProvider state.IStateProvider
	var err error
	if len(conf.MemcachedServers) > 0 {
		attempts := 0
		for stateProvider == nil && attempts < 120 {
			stateProvider, err = state.NewMemcachedStateProvider(conf.MemcachedServers, conf.MemcachedUser, conf.MemcachedPass)
			attempts++
			if err != nil {
				klog.Warningf("Attempt %d - Error connecting to memcached: %s", attempts, err.Error())
				time.Sleep(time.Second)
			}
		}
		if err != nil {
			return nil, err
		}
	} else {
		stateProvider = state.NewInMemoryStateProvider()
	}
	h := Host{
		config:        conf,
		stateProvider: stateProvider,
	}
	for _, wf := range h.config.GetAllWorkflows() {
		c := h.getClient(&wf)
		jobs, err := c.GetQueuedJobs(context.Background())
		name := fmt.Sprintf("%s/%s (%s/%s)", wf.Namespace, wf.Name, wf.Owner, wf.Repository)
		if err != nil {
			klog.Errorf("Error whilst getting jobs for %s: %s", name, err.Error())
			return nil, err
		}
		klog.Infof("Initialized %s: %d jobs", name, len(jobs))
	}
	return &h, nil
}
