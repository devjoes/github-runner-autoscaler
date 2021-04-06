package config

//TODO: Poss seperate this from the config class
import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	runnerclient "github.com/devjoes/github-runner-autoscaler/apiserver/pkg/runnerclient"
	"github.com/devjoes/github-runner-autoscaler/apiserver/pkg/scaling"
	runnerv1alpha1 "github.com/devjoes/github-runner-autoscaler/operator/api/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
)

func (c *Config) initWorkflows(params []interface{}) error {
	c.store = cache.NewStore(getKey)
	var k8sClient kubernetes.Interface
	var runnerClient runnerclient.IRunnersV1Alpha1Client
	if len(params) == 0 {
		var err error
		k8sClient, runnerClient, err = getClients(*c.flagInClusterConfig, *c.flagKubeconfig)
		if err != nil {
			return err
		}
	} else {
		k8sClient = params[0].(kubernetes.Interface)
		runnerClient = params[1].(runnerclient.IRunnersV1Alpha1Client)
	}

	err := c.syncWorkflows(k8sClient, runnerClient, c.RunnerNSs)
	if err != nil {
		return err
	}
	return nil
}

func (c *Config) GetAllWorkflows() []GithubWorkflowConfig {
	cached := c.store.List()
	wfs := make([]GithubWorkflowConfig, len(cached))
	for i, c := range cached {
		wfs[i] = c.(GithubWorkflowConfig)
	}
	return wfs
}

func (c *Config) GetWorkflow(key string) (*GithubWorkflowConfig, error) {
	item, found, err := c.store.GetByKey(key)
	fmt.Printf("GetWorkflow %s %t %v %v", key, found, item, err)
	if !found {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	wf := item.(GithubWorkflowConfig)
	return &wf, nil
}

func getKey(obj interface{}) (string, error) {
	wfc := obj.(GithubWorkflowConfig)
	return wfc.Name, nil
}

func getNamespacedClients(runnerClient runnerclient.IRunnersV1Alpha1Client, runnerNSs []string) []runnerclient.IScaledActionRunnerClient {
	var clients []runnerclient.IScaledActionRunnerClient
	if len(runnerNSs) == 0 {
		clients = append(clients, runnerClient.ScaledActionRunners(""))
	} else {
		for _, ns := range runnerNSs {
			clients = append(clients, runnerClient.ScaledActionRunners(ns))
		}
	}
	return clients
}
func (c *Config) copyAllWorkflows(ctx context.Context, k8sClient kubernetes.Interface, runnerclient runnerclient.IRunnersV1Alpha1Client, runnerNSs []string) {
	var runners []runnerv1alpha1.ScaledActionRunner
	var r *runnerv1alpha1.ScaledActionRunnerList
	var err error
	for _, client := range getNamespacedClients(runnerclient, runnerNSs) {
		r, err = client.List(ctx, metav1.ListOptions{})
		if err != nil {
			fmt.Printf("Error getting runners: %v", err)
			continue
		}
		runners = append(runners, r.Items...)
	}

	purgeOld := true
	var toCache []interface{}
	for _, r := range runners {
		wf, err := workflowFromScaledActionRunner(ctx, k8sClient, r)
		if err != nil {
			fmt.Printf("Failed to copy workflow from runner %s/%s: %s", r.ObjectMeta.Namespace, r.ObjectMeta.Name, err.Error())
			purgeOld = false
		} else {
			toCache = append(toCache, *wf)
		}
	}

	if purgeOld {
		c.store.Replace(toCache, "v1")
	} else {
		fmt.Println("Some workflows failed to load - not purging old config")
		for _, tc := range toCache {
			c.store.Update(tc)
		}
	}
}

func (c *Config) syncWorkflows(k8sClient kubernetes.Interface, runnerclient runnerclient.IRunnersV1Alpha1Client, runnerNSs []string) error {
	ctx := context.Background()
	c.copyAllWorkflows(ctx, k8sClient, runnerclient, runnerNSs)
	if c.ResyncInterval > 0 {
		ticker := time.NewTicker(c.ResyncInterval)
		go func() {
			for {
				now := <-ticker.C
				fmt.Printf("Resyncing all workflows @ %s\n", now.String())
				c.copyAllWorkflows(ctx, k8sClient, runnerclient, runnerNSs)
			}
		}()
	}
	for _, client := range getNamespacedClients(runnerclient, runnerNSs) {
		go c.setupWatcher(k8sClient, client)
	}
	return nil
}

func (c *Config) setupWatcher(k8sClient kubernetes.Interface, runnerClient runnerclient.IScaledActionRunnerClient) {
	for {
		w, err := runnerClient.Watch(context.TODO(), metav1.ListOptions{})
		if err != nil {
			fmt.Printf("Error whilst watching namespace %s: %s\n", runnerClient.GetNs(), err.Error())
			return
		}
		for {
			var event watch.Event
			event, ok := <-w.ResultChan()
			if !ok {
				break
			}
			if event.Object.GetObjectKind().GroupVersionKind().Kind != "ScaledActionRunner" {
				fmt.Printf("Error from watch on namespace %s:\n", runnerClient.GetNs())
				d, _ := json.Marshal(event)
				fmt.Println(string(d))
				time.Sleep(time.Second)
				continue
			}
			var runner *runnerv1alpha1.ScaledActionRunner
			obj := event.Object
			runner = obj.(*runnerv1alpha1.ScaledActionRunner)

			wf, err := workflowFromScaledActionRunner(context.TODO(), k8sClient, *runner)
			if err != nil {
				fmt.Printf("Error %s from watch. %v %v\n", event.Type, event, err)
				continue
			}

			fmt.Println(event.Type)
			fmt.Println(c.store.ListKeys())
			switch event.Type {
			case watch.Added:
				{
					err = c.store.Add(*wf)
				}
			case watch.Modified:
				{
					err = c.store.Update(*wf)
				}
			case watch.Deleted:
				{
					err = c.store.Delete(*wf)
					fmt.Printf("del %v", event.Object)
					//fmt.Println(getKey(event.Object))
				}
			}
			if err != nil {
				fmt.Println(err)
			}
			fmt.Println(c.store.ListKeys())
		}
	}
}

func workflowFromScaledActionRunner(ctx context.Context, client kubernetes.Interface, crd runnerv1alpha1.ScaledActionRunner) (*GithubWorkflowConfig, error) {
	ns := crd.ObjectMeta.Namespace
	a, _ := client.CoreV1().Secrets("ns").List(ctx, metav1.ListOptions{})
	fmt.Println(a)
	secret, err := client.CoreV1().Secrets(ns).Get(ctx, crd.Spec.GithubTokenSecret, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("Error reading secret %s in namespace %s. %s", crd.Spec.GithubTokenSecret, ns, err.Error())
	}
	if crd.Spec.ScaleFactor == nil {
		one := "1"
		crd.Spec.ScaleFactor = &one
	}

	return &GithubWorkflowConfig{
		Name:       crd.ObjectMeta.Name,
		Namespace:  ns,
		Token:      string(secret.Data["token"]),
		Owner:      crd.Spec.Owner,
		Repository: crd.Spec.Repo,
		Scaling:    scaling.NewScaling(&crd),
	}, nil
}
