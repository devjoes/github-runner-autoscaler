package runnerclient

import (
	"context"
	"errors"

	runnerv1alpha1 "github.com/devjoes/github-runner-autoscaler/operator/api/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
)

type FakeRunnersV1Alpha1Client struct {
	Runners *[]runnerv1alpha1.ScaledActionRunner
	Watch   *watch.Interface
}

func NewFakeRunnersV1Alpha1Client(runners []runnerv1alpha1.ScaledActionRunner) (*FakeRunnersV1Alpha1Client, *watch.FakeWatcher) {
	fw := watch.NewFakeWithChanSize(2, false)
	var w watch.Interface = fw
	return &FakeRunnersV1Alpha1Client{Runners: &runners, Watch: &w}, fw
}

func (c *FakeRunnersV1Alpha1Client) ScaledActionRunners(namespace string) IScaledActionRunnerClient {
	return &fakeScaledActionRunnerClient{ns: namespace, runners: c.Runners, watch: c.Watch}
}

type fakeScaledActionRunnerClient struct {
	ns      string
	runners *[]runnerv1alpha1.ScaledActionRunner
	watch   *watch.Interface
}

func (c *fakeScaledActionRunnerClient) List(ctx context.Context, opts metav1.ListOptions) (*runnerv1alpha1.ScaledActionRunnerList, error) {
	result := runnerv1alpha1.ScaledActionRunnerList{}
	for _, r := range *c.runners {
		if r.ObjectMeta.Namespace == c.ns || c.ns == "" {
			result.Items = append(result.Items, r)
		}
	}
	return &result, nil
}
func (c *fakeScaledActionRunnerClient) Get(ctx context.Context, name string, opts metav1.GetOptions) (*runnerv1alpha1.ScaledActionRunner, error) {
	for _, r := range *c.runners {
		if (r.ObjectMeta.Namespace == c.ns || c.ns == "") && r.ObjectMeta.Name == name {
			return &r, nil
		}
	}
	return nil, errors.New("not found")
}

func (c *fakeScaledActionRunnerClient) Watch(ctx context.Context, opts metav1.ListOptions) (watch.Interface, error) {
	return *c.watch, nil
}
func (c *fakeScaledActionRunnerClient) GetNs() string {
	return c.ns
}
