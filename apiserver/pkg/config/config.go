package config

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	runnerClient "github.com/devjoes/github-runner-autoscaler/apiserver/pkg/runnerclient"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	cache "k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
)

const (
	weirdJsonMsgPrefix = "\xef\xbb\xbf"
)

type Config struct {
	CacheWindow          time.Duration `json:"cacheWindow"`
	CacheWindowWhenEmpty time.Duration `json:"cacheWindowWhenEmpty"`
	ResyncInterval       time.Duration `json:"resyncInterval"`
	MemcachedServers     []string      `json:"memcachedServers"`
	AllNs                bool          `json:"allNs"`
	InClusterConfig      bool          `json:"inClusterConfig"`
	Kubeconfig           string        `json:"kubeconfig"`
	RunnerNSs            []string      `json:"runnerNSs"`

	flagMemcachedServers     *ArrayFlags
	flagCacheWindow          *string
	flagCacheWindowWhenEmpty *string
	flagResyncIntervalStr    *string
	flagRunnerNSs            *ArrayFlags
	flagAllNs                *bool
	flagKubeconfig           *string
	flagInClusterConfig      *bool

	store cache.Store
}

type GithubWorkflowConfig struct {
	Name       string `json:"name"`
	Namespace  string `json:"namespace"`
	Token      string `json:"token"`
	Owner      string `json:"owner"`
	Repository string `json:"repository"`
}

type IWorkflowSource interface {
	GetAllWorkflows() []GithubWorkflowConfig
	GetWorkflow(key string) (*GithubWorkflowConfig, error)
}

func getClients(inCluster bool, kubeconfig string) (kubernetes.Interface, runnerClient.IRunnersV1Alpha1Client, error) {
	var config *rest.Config
	var err error
	if inCluster {
		config, err = rest.InClusterConfig()
	} else {
		config, err = clientcmd.BuildConfigFromFlags("", kubeconfig)
	}

	if err != nil {
		return nil, nil, err
	}

	k8sClient, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, nil, err
	}

	runnerClient, err := runnerClient.NewForConfig(config)
	if err != nil {
		return nil, nil, err
	}
	return k8sClient, runnerClient, err
}

type ArrayFlags []string

func (i *ArrayFlags) String() string {
	if i == nil {
		return ""
	}
	return strings.Join(*i, ",")
}

func (i *ArrayFlags) Set(value string) error {
	*i = append(*i, value)
	return nil
}
func (c *Config) AddFlags() {
	c.flagRunnerNSs = &ArrayFlags{}
	c.flagMemcachedServers = &ArrayFlags{}
	flag.Var(c.flagRunnerNSs, "namespace", "Namespace to find secrets in, can be specified multiple times.")
	c.flagAllNs = flag.Bool("allnamespaces", false, "Find secrets in all namespaces.")

	if home := homedir.HomeDir(); home != "" {
		c.flagKubeconfig = flag.String("kubeconfig", filepath.Join(home, ".kube", "config"), "(optional) absolute path to the kubeconfig file.")
	} else {
		c.flagKubeconfig = flag.String("kubeconfig", "", "absolute path to the kubeconfig file.")
	}

	c.flagInClusterConfig = flag.Bool("incluster", false, "Ignore kubeconfig, connecting from within a cluster.")
	c.flagCacheWindow = flag.String("cache-window", "1m", "How long to cache queue lengths for")
	c.flagCacheWindowWhenEmpty = flag.String("cache-windowwhen-empty", "30s", "How long to cache queue lengths for when all runners may be offline")
	c.flagResyncIntervalStr = flag.String("resync-interval", "5m", "How often to fully reload all ScaledActionRunner CRDs")
	flag.Var(c.flagMemcachedServers, "memcached-server", "Memcached servers to use. If unspecified a local in memory cache is used.")
}

func validateArgs(runnerNSs []string, allNs bool) error {
	if len(runnerNSs) == 0 && !allNs {
		return errors.New("Specify --namespace or --all-namespaces")
	}
	if len(runnerNSs) > 0 && allNs {
		return errors.New("Can't specify --namespaces and --all-namespaces")
	}
	return nil
}

func parseDuration(flag *string, value time.Duration) time.Duration {
	if flag == nil {
		return value
	}
	parsed, err := time.ParseDuration(*flag)
	if err != nil {
		if *flag != "" {
			fmt.Printf("Error parsing '%s': %s.\nUsing default: %s\n", *flag, err.Error(), value)
		}
		return value
	}
	return parsed
}
func (c *Config) SetupConfig(params ...interface{}) error {
	c.RunnerNSs = make([]string, 0)
	if c.flagRunnerNSs != nil && len((*c.flagRunnerNSs).String()) > 0 {
		c.RunnerNSs = strings.Split((*c.flagRunnerNSs).String(), ",")
	}
	c.MemcachedServers = make([]string, 0)
	if c.flagMemcachedServers != nil && len((*c.flagMemcachedServers).String()) > 0 {
		c.MemcachedServers = strings.Split((*c.flagMemcachedServers).String(), ",")
	}

	c.ResyncInterval = parseDuration(c.flagResyncIntervalStr, c.ResyncInterval)
	c.CacheWindow = parseDuration(c.flagCacheWindow, c.CacheWindow)
	c.CacheWindowWhenEmpty = parseDuration(c.flagCacheWindowWhenEmpty, c.CacheWindowWhenEmpty)
	c.AllNs = *c.flagAllNs
	c.InClusterConfig = *c.flagInClusterConfig
	c.Kubeconfig = *c.flagKubeconfig
	c.RunnerNSs = *c.flagRunnerNSs

	if err := validateArgs(c.RunnerNSs, c.AllNs); err != nil {
		return err
	}

	output, _ := json.Marshal(c)
	fmt.Printf("%s\n", string(output))

	return c.initWorkflows(params)
}

type Duration struct {
	time.Duration
}

func (d Duration) MarshalJSON() ([]byte, error) {
	return json.Marshal(d.String())
}

func (d *Duration) UnmarshalJSON(b []byte) error {
	var v interface{}
	if err := json.Unmarshal(b, &v); err != nil {
		return err
	}
	switch value := v.(type) {
	case float64:
		d.Duration = time.Duration(value)
		return nil
	case string:
		var err error
		d.Duration, err = time.ParseDuration(value)
		if err != nil {
			return err
		}
		return nil
	default:
		return errors.New("invalid duration")
	}
}