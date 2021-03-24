package runnerclient

import (
	"context"
	"fmt"

	"k8s.io/apimachinery/pkg/runtime/serializer"

	runnerv1alpha1 "github.com/devjoes/github-runner-autoscaler/operator/api/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
)

const scaledactionrunners = "scaledactionrunners"

type IRunnersV1Alpha1Client interface {
	ScaledActionRunners(namespace string) IScaledActionRunnerClient
}
type RunnersV1Alpha1Client struct {
	restClient rest.Interface
}

func (c *RunnersV1Alpha1Client) ScaledActionRunners(namespace string) IScaledActionRunnerClient {
	return &scaledActionRunnerClient{restClient: c.restClient, ns: namespace}
}

func NewForConfig(config *rest.Config) (*RunnersV1Alpha1Client, error) {
	runnerv1alpha1.AddToScheme(scheme.Scheme)

	crdConfig := *config
	crdConfig.ContentConfig.GroupVersion = &runnerv1alpha1.GroupVersion
	crdConfig.APIPath = "/apis"
	crdConfig.NegotiatedSerializer = serializer.NewCodecFactory(scheme.Scheme)
	crdConfig.UserAgent = rest.DefaultKubernetesUserAgent()

	restClient, err := rest.RESTClientFor(&crdConfig)
	if err != nil {
		return nil, err
	}
	return &RunnersV1Alpha1Client{restClient: restClient}, nil
}

type IScaledActionRunnerClient interface {
	List(ctx context.Context, opts metav1.ListOptions) (*runnerv1alpha1.ScaledActionRunnerList, error)
	Get(ctx context.Context, name string, options metav1.GetOptions) (*runnerv1alpha1.ScaledActionRunner, error)
	Watch(ctx context.Context, opts metav1.ListOptions) (watch.Interface, error)
	GetNs() string
}

type scaledActionRunnerClient struct {
	restClient rest.Interface
	ns         string
}

func (c *scaledActionRunnerClient) List(ctx context.Context, opts metav1.ListOptions) (*runnerv1alpha1.ScaledActionRunnerList, error) {
	result := runnerv1alpha1.ScaledActionRunnerList{}
	err := c.restClient.
		Get().
		Namespace(c.ns).
		Resource(scaledactionrunners).
		VersionedParams(&opts, scheme.ParameterCodec).
		Do(ctx).
		Into(&result)

	return &result, err
}

func (c *scaledActionRunnerClient) Get(ctx context.Context, name string, opts metav1.GetOptions) (*runnerv1alpha1.ScaledActionRunner, error) {
	result := runnerv1alpha1.ScaledActionRunner{}
	err := c.restClient.
		Get().
		Namespace(c.ns).
		Resource(scaledactionrunners).
		Name(name).
		VersionedParams(&opts, scheme.ParameterCodec).
		Do(ctx).
		Into(&result)

	return &result, err
}

func (c *scaledActionRunnerClient) Watch(ctx context.Context, opts metav1.ListOptions) (watch.Interface, error) {
	opts.Watch = true
	result := runnerv1alpha1.ScaledActionRunnerList{}
	err := c.restClient.
		Get().
		Namespace(c.ns).
		Resource(scaledactionrunners).
		VersionedParams(&opts, scheme.ParameterCodec).
		Do(ctx).
		Into(&result)
	if err != nil {
		fmt.Println(err)
	}
	return c.restClient.
		Get().
		Namespace(c.ns).
		Resource(scaledactionrunners).
		VersionedParams(&opts, scheme.ParameterCodec).
		Watch(ctx)
}

func (c *scaledActionRunnerClient) GetNs() string {
	return c.ns
}
