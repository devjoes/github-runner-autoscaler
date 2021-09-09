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

//TODO: Wrap all of these returned vars up in ot a struct
func (h *Host) QueryMetric(key string, selector labels.Selector) (int, *time.Time, map[string][]string, *config.GithubWorkflowConfig, bool, error) {
	wf, err := h.config.GetWorkflow(key)
	if err != nil {
		return 0, nil, nil, nil, false, err
	}
	if wf == nil {
		return 0, nil, nil, nil, false, errors.New(MetricErrNotFound)
	}
	client := h.getClient(wf)
	ctx := context.Background()
	jobs, retrievalTime, err := client.GetQueuedJobs(ctx)
	if err != nil {
		return 0, nil, nil, wf, false, err
	}
	wfInfo, err := client.GetWorkflowInfo(ctx)
	if err != nil {
		return 0, nil, nil, wf, false, err
	}
	clientState, err := client.GetState()
	if err != nil {
		return 0, nil, nil, wf, false, err
	}
	forceScaleNow, nextForceScale := wf.Scaling.CalculateForcedScale(clientState.NextForcedScale)
	if clientState.NextForcedScale == nil || nextForceScale != *clientState.NextForcedScale {
		clientState.NextForcedScale = &nextForceScale
		client.SaveState(clientState)
	}
	filteredJobs, matchedLabels := labeling.FilterBySelector(jobs, wf, wfInfo, selector)

	return len(filteredJobs), retrievalTime, matchedLabels, wf, forceScaleNow, err
}

func (h *Host) getClient(wf *config.GithubWorkflowConfig) client.Client {
	githubClient := client.NewGitHubClient(wf.Token, wf.Owner, wf.Repository)
	gitOwnerRepo := fmt.Sprintf("%s/%s", wf.Owner, wf.Repository)
	return client.NewClient(&githubClient, wf.Name, gitOwnerRepo, h.config.CacheWindow, h.config.CacheWindowWhenEmpty, h.stateProvider)
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
				stateProvider = nil
				klog.Warningf("Attempt %d - Error connecting to memcached: %s", attempts, err.Error())
				time.Sleep(time.Second)
			}
			fmt.Println(err)
			fmt.Println(stateProvider)
			fmt.Println(attempts)
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
	err = h.config.InitWorkflows()
	if err != nil {
		return &h, err
	}
	for _, wf := range h.config.GetAllWorkflows() {
		c := h.getClient(&wf)
		jobs, retrievalTime, err := c.GetQueuedJobs(context.Background())
		name := fmt.Sprintf("%s/%s (%s/%s) @%s", wf.Namespace, wf.Name, wf.Owner, wf.Repository, retrievalTime.String())
		if err != nil {
			klog.Errorf("Error whilst getting jobs for %s: %s", name, err.Error())
		}
		klog.Infof("Initialized %s: %d jobs", name, len(jobs))
	}
	return &h, nil
}
