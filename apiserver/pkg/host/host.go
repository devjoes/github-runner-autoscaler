package host

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"

	"github.com/devjoes/github-runner-autoscaler/apiserver/pkg/config"
	"github.com/devjoes/github-runner-autoscaler/apiserver/pkg/gitclient"
	client "github.com/devjoes/github-runner-autoscaler/apiserver/pkg/gitclient"
	"github.com/devjoes/github-runner-autoscaler/apiserver/pkg/state"
)

type Host struct {
	config        config.Config
	stateProvider state.IStateProvider
}

func (h *Host) GetAllMetricNames() ([]string, error) {
	wfs := h.config.GetAllWorkflows()
	metrics := make([]string, len(wfs))
	for i, wf := range wfs {
		metrics[i] = wf.Name
	}
	return metrics, nil
}

const MetricErrNotFound string = "metric not found"

func (h *Host) QueryMetric(key string) (int32, *config.GithubWorkflowConfig, error) {
	wf, err := h.config.GetWorkflow(key)
	if err != nil {
		return 0, nil, err
	}
	if wf == nil {
		return 0, nil, errors.New(MetricErrNotFound)
	}
	client := h.getClient(wf)
	length, err := client.GetQueueLength(context.Background()) //TODO: context
	return int32(length), wf, err
}

func (h *Host) getClient(wf *config.GithubWorkflowConfig) client.Client {
	githubClient := client.NewGitHubClient(wf.Token, wf.Owner, wf.Repository)
	key := wf.Name
	return client.NewClient(&githubClient, key, h.config.CacheWindow, h.config.CacheWindowWhenEmpty, h.stateProvider)
}

func (h *Host) getClients() ([]client.Client, error) {
	clients := []client.Client{}
	wfs := h.config.GetAllWorkflows()
	for _, wf := range wfs {
		innerClient := client.NewGitHubClient(wf.Token, wf.Owner, wf.Repository)
		client := client.NewClient(&innerClient, wf.Name, h.config.CacheWindow, h.config.CacheWindowWhenEmpty, h.stateProvider)
		clients = append(clients, client)
	}
	return clients, nil
}

func NewHost(conf config.Config, params ...interface{}) (*Host, error) {
	var stateProvider state.IStateProvider
	var err error
	if len(conf.MemcachedServers) > 0 {
		stateProvider, err = state.NewMemcachedStateProvider(conf.MemcachedServers, conf.MemcachedUser, conf.MemcachedPass)
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
	var wfs []config.GithubWorkflowConfig
	if len(params) > 0 {
		wfs = params[0].([]config.GithubWorkflowConfig)
	} else {
		wfs = conf.GetAllWorkflows()
	}
	var wg sync.WaitGroup
	cErr := make(chan error, len(wfs))
	for _, c := range wfs {
		wg.Add(1)
		go h.intializeClient(c, &wg, cErr)
	}
	wg.Wait()
	close(cErr)

	var errMsgs []string
	for err := range cErr {
		errMsgs = append(errMsgs, err.Error())
	}

	if len(errMsgs) > 0 {
		return nil, errors.New(strings.Join(errMsgs, "\n"))
	}
	return &h, nil
}

func (h *Host) intializeClient(wf config.GithubWorkflowConfig, wg *sync.WaitGroup, cErr chan error) {
	defer wg.Done()
	client := gitclient.NewGitHubClient(wf.Token, wf.Owner, wf.Repository)
	title := fmt.Sprintf("%s/%s (%s in %s)", wf.Owner, wf.Repository, wf.Name, wf.Namespace)
	fmt.Printf("Initializing client %s\n", title)
	queueLength, err := client.GetQueueLength(context.Background()) //TODO: context
	if err != nil {
		cErr <- fmt.Errorf("Error loading queue for client '%s': %v", title, err)
	} else {
		fmt.Printf("Client %s initialized (%d items in queue)\n", title, queueLength)
	}
}
