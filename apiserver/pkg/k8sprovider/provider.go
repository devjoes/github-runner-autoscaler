package k8sprovider

import (
	"time"

	"github.com/devjoes/github-runner-autoscaler/apiserver/pkg/config"
	"github.com/devjoes/github-runner-autoscaler/apiserver/pkg/host"
	labeling "github.com/devjoes/github-runner-autoscaler/apiserver/pkg/labeling"
	"github.com/kubernetes-sigs/custom-metrics-apiserver/pkg/provider"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/klog/v2"
	"k8s.io/metrics/pkg/apis/custom_metrics"
	"k8s.io/metrics/pkg/apis/external_metrics"
)

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
	klog.V(5).Infof("NewProvider")
	provider := &workflowQueueProvider{
		orchestrator: orchestrator,
	}
	return provider
}

func metricFor(value resource.Quantity, retrievalTime time.Time, wf config.GithubWorkflowConfig, lbls map[string]string) *custom_metrics.MetricValue {
	return &custom_metrics.MetricValue{
		Metric: custom_metrics.MetricIdentifier{
			Name:     wf.Name,
			Selector: &v1.LabelSelector{MatchLabels: lbls},
		},
		DescribedObject: custom_metrics.ObjectReference{Kind: "ScaledActionRunner", APIVersion: "v1alpha1", Name: wf.Name, Namespace: wf.Namespace},
		Value:           value,
		Timestamp:       v1.Time{Time: retrievalTime},
	}
}

func (p *workflowQueueProvider) valueFor(info provider.CustomMetricInfo, name types.NamespacedName, metricSelector labels.Selector) (resource.Quantity, time.Time, *config.GithubWorkflowConfig, map[string]string, error) {
	var err error
	if metricSelector == nil {
		metricSelector, err = labels.Parse(info.Metric)
		if err != nil {
			klog.Warningf("Invalid selector '%s' in %s. %s", info.Metric, name.String(), err.Error())
		}
	}
	total, retrievalTime, lbls, wf, forceScale, err := p.orchestrator.QueryMetric(name.Name, metricSelector)
	if err != nil && err.Error() == host.MetricErrNotFound {
		return resource.Quantity{}, time.Time{}, nil, nil, provider.NewMetricNotFoundForError(info.GroupResource, info.Metric, name.Name)
	}

	promLabels, allLabels := labeling.GetLabelsForOutput(lbls)

	if err != nil {
		return resource.Quantity{}, time.Time{}, wf, nil, err
	}
	scaledTotal := int(wf.Scaling.GetOutput(int32(total)))
	promLabels = append([]string{name.String(), metricSelector.String()}, promLabels...)

	if forceScale {
		promLabels = append(promLabels, "forciblyScaledUp")
		scaledTotal = int(wf.Scaling.MaxWorkers)
	}

	//TODO: Get the labels out of QueryMetric, maybe move all this instrumentation stuff in to one place

	guageFilteredQueueLength.WithLabelValues(promLabels...).Set(float64(total))
	guageFilteredScaledQueueLength.WithLabelValues(promLabels...).Set(float64(scaledTotal))
	return *resource.NewQuantity(int64(scaledTotal), resource.DecimalSI), *retrievalTime, wf, allLabels, nil
}

func (p *workflowQueueProvider) GetMetricByName(name types.NamespacedName, info provider.CustomMetricInfo, metricSelector labels.Selector) (*custom_metrics.MetricValue, error) {
	klog.V(5).Infof("GetMetricByName '%s/%s' '%s' '%s' '%s'", name.Namespace, name.Name, info.Metric, info.String(), metricSelector.String())
	var err error
	selector := metricSelector
	if metricSelector.String() == "" && info.Metric != "*" {
		selector, err = labels.Parse(info.Metric)
		if err != nil {
			klog.Warningf("Invalid selector '%s' in %s. %s", info.Metric, name.String(), err.Error())
			return nil, err
		}
	}
	value, tm, wf, lbls, err := p.valueFor(info, name, selector)

	if err != nil {
		if err.Error() == host.MetricErrNotFound {
			klog.Warningf("Metric not found with %s %s %s", name, info.String(), selector.String())
			return nil, errors.NewNotFound(external_metrics.Resource("GithubWorkflowConfig"), name.Name)
		}
		klog.Warningf("Error getting metric with %s %s %s. %s", name, info.String(), selector.String(), err.Error())
		return nil, errors.NewBadRequest("Error getting metric")
	}
	metric := metricFor(value, tm, *wf, lbls)
	return metric, err
}

func (p *workflowQueueProvider) GetMetricBySelector(namespace string, selector labels.Selector, info provider.CustomMetricInfo, metricSelector labels.Selector) (*custom_metrics.MetricValueList, error) {
	klog.V(5).Infof("GetMetricBySelector %s %s %s", namespace, info.Metric, metricSelector.String())

	metrics := custom_metrics.MetricValueList{}
	names, err := p.orchestrator.GetAllMetricNames(namespace)
	if err != nil {
		return nil, err
	}

	for _, name := range names {
		v, tm, wf, lbls, err := p.valueFor(info, types.NamespacedName{Namespace: namespace, Name: name}, metricSelector)
		if err != nil {
			return nil, err
		}
		if len(lbls) == 0 {
			continue
		}
		metrics.Items = append(metrics.Items, *metricFor(v, tm, *wf, lbls))
	}
	return &metrics, nil
}

func (p *workflowQueueProvider) ListAllMetrics() []provider.CustomMetricInfo {
	klog.V(5).Info("ListAllMetrics")
	infos := make(map[provider.CustomMetricInfo]struct{})
	metricData, err := p.orchestrator.GetAllMetricNames("")
	if err != nil {
		klog.Errorf("Error listing all metrics. %s", err)
	}
	for _, name := range metricData {
		infos[provider.CustomMetricInfo{Metric: name, Namespaced: true}] = struct{}{}
	}
	metrics := make([]provider.CustomMetricInfo, 0, len(infos))
	for info := range infos {
		metrics = append(metrics, info)
	}
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
