package k8sprovider

import (
	"fmt"

	"github.com/devjoes/github-runner-autoscaler/apiserver/pkg/config"
	"github.com/devjoes/github-runner-autoscaler/apiserver/pkg/host"
	labeling "github.com/devjoes/github-runner-autoscaler/apiserver/pkg/labeling"
	"github.com/kubernetes-sigs/custom-metrics-apiserver/pkg/provider"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/klog/v2"
	"k8s.io/metrics/pkg/apis/custom_metrics"
	"k8s.io/metrics/pkg/apis/external_metrics"
)

//TODO: rediness probe

type metricValue struct {
	labels labels.Set
	value  resource.Quantity
}
type CustomMetricResource struct {
	provider.CustomMetricInfo
	types.NamespacedName
}

type workflowQueueProvider struct {
	orchestrator *host.Host
}

func NewProvider(orchestrator *host.Host) provider.CustomMetricsProvider {
	klog.Infof("NewProvider")
	provider := &workflowQueueProvider{
		orchestrator: orchestrator,
	}
	return provider
}

func metricFor(value resource.Quantity, wf config.GithubWorkflowConfig, lbls map[string]string) *custom_metrics.MetricValue {
	return &custom_metrics.MetricValue{
		Metric: custom_metrics.MetricIdentifier{
			Name:     wf.Name,
			Selector: &v1.LabelSelector{MatchLabels: lbls},
		},
		DescribedObject: custom_metrics.ObjectReference{Kind: "ScaledActionRunner", APIVersion: "v1alpha1", Name: wf.Name, Namespace: wf.Namespace},
		Value:           value,
		Timestamp:       metav1.Now(),
	}
}

func (p *workflowQueueProvider) valueFor(info provider.CustomMetricInfo, name types.NamespacedName, metricSelector labels.Selector) (resource.Quantity, *config.GithubWorkflowConfig, map[string]string, error) {
	var err error
	if metricSelector == nil {
		metricSelector, err = labels.Parse(info.Metric)
		fmt.Println(err)
	}
	fmt.Println(metricSelector)
	total, lbls, wf, err := p.orchestrator.QueryMetric(name.Name, metricSelector)
	fmt.Println(total)
	if err != nil && err.Error() == host.MetricErrNotFound {
		return resource.Quantity{}, nil, nil, provider.NewMetricNotFoundForError(info.GroupResource, info.Metric, name.Name)
	}

	promLabels, allLabels := labeling.GetLabelsForOutput(lbls)

	if err != nil {
		return resource.Quantity{}, wf, nil, err
	}
	// if total == 0 {
	// 	return resource.Quantity{}, nil, provider.NewMetricNotFoundForSelectorError(info.GroupResource, info.Metric, name.Name, metricSelector)
	// }
	promLabels = append([]string{name.String(), metricSelector.String()}, promLabels...)
	scaledTotal := int(wf.Scaling.GetOutput(int32(total)))
	//TODO: Get the labels out of QueryMetric, maybe move all this instrumentation stuff in to one place
	fmt.Println(promLabels)
	guageFilteredQueueLength.WithLabelValues(promLabels...).Set(float64(total))
	guageFilteredScaledQueueLength.WithLabelValues(promLabels...).Set(float64(scaledTotal))
	return *resource.NewQuantity(int64(scaledTotal), resource.DecimalSI), wf, allLabels, nil
}

func (p *workflowQueueProvider) GetMetricByName(name types.NamespacedName, info provider.CustomMetricInfo, metricSelector labels.Selector) (*custom_metrics.MetricValue, error) {

	fmt.Printf("GetMetricByName '%s/%s' '%s' '%s' '%v'\n", name.Namespace, name.Name, info.Metric, info.String(), metricSelector)
	var err error
	selector := metricSelector
	if metricSelector.String() == "" && info.Metric != "*" {
		selector, err = labels.Parse(info.Metric)
		if err != nil {
			return nil, err
		}
	}
	fmt.Println(selector)
	value, wf, lbls, err := p.valueFor(info, name, selector)

	fmt.Printf("%v %v %v\n", value, wf, err)
	if err != nil {
		if err.Error() == host.MetricErrNotFound {
			return nil, errors.NewNotFound(external_metrics.Resource("GithubWorkflowConfig"), name.Name)
		}
		fmt.Println(err.Error())
		return nil, errors.NewBadRequest("Error getting metric")
	}
	metric := metricFor(value, *wf, lbls)

	fmt.Println(metric)
	return metric, err
}

func (p *workflowQueueProvider) GetMetricBySelector(namespace string, selector labels.Selector, info provider.CustomMetricInfo, metricSelector labels.Selector) (*custom_metrics.MetricValueList, error) {
	fmt.Println("GetMetricBySelector")
	fmt.Printf("GetMetricBySelector %s %v %v\n", namespace, info, metricSelector)

	metrics := custom_metrics.MetricValueList{}
	names, err := p.orchestrator.GetAllMetricNames(namespace)
	if err != nil {
		return nil, err
	}

	for _, name := range names {
		v, wf, lbls, err := p.valueFor(info, types.NamespacedName{Namespace: namespace, Name: name}, metricSelector)
		if err != nil {
			return nil, err
		}
		if len(lbls) == 0 {
			continue
		}
		metrics.Items = append(metrics.Items, *metricFor(v, *wf, lbls))
	}
	return &metrics, nil
}

func (p *workflowQueueProvider) ListAllMetrics() []provider.CustomMetricInfo {
	fmt.Println("ListAllMetrics")
	infos := make(map[provider.CustomMetricInfo]struct{})
	metricData, err := p.orchestrator.GetAllMetricNames("")
	if err != nil {
		klog.Error(err)
	}
	for _, name := range metricData {
		infos[provider.CustomMetricInfo{Metric: name, Namespaced: true}] = struct{}{}
	}
	metrics := make([]provider.CustomMetricInfo, 0, len(infos))
	for info := range infos {
		metrics = append(metrics, info)
	}
	fmt.Println(metrics)
	return metrics
}

var guageFilteredQueueLength *prometheus.GaugeVec
var guageFilteredScaledQueueLength *prometheus.GaugeVec

func init() {
	labelNames := append([]string{"name", "selector"}, labeling.JobLabelsForPrometheus...)
	guageFilteredQueueLength = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "workflow_queue_length_filtered",
		Help: "The number of queued jobs filtered by labels",
	}, labelNames)
	guageFilteredScaledQueueLength = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "workflow_queue_length_filtered_scaled",
		Help: "The scaled up/down number of queued jobs filtered by labels",
	}, labelNames)

}
