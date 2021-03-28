package k8sprovider

import (
	"fmt"

	"github.com/devjoes/github-runner-autoscaler/apiserver/pkg/host"
	"github.com/kubernetes-sigs/custom-metrics-apiserver/pkg/provider"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/klog/v2"
	"k8s.io/metrics/pkg/apis/external_metrics"
)

type ExternalMetricInfo struct {
	Metric string
}

//TODO: rediness probe
type externalMetric struct {
	info   provider.ExternalMetricInfo
	labels map[string]string
	value  external_metrics.ExternalMetricValue
}

type metricValue struct {
	labels labels.Set
	value  resource.Quantity
}

type workflowQueueProvider struct {
	orchestrator *host.Host
}

func NewProvider(orchestrator *host.Host) provider.ExternalMetricsProvider {
	klog.Infof("NewProvider")
	provider := &workflowQueueProvider{
		orchestrator: orchestrator,
	}
	return provider
}
func (p *workflowQueueProvider) GetExternalMetric(namespace string, metricSelector labels.Selector, info provider.ExternalMetricInfo) (*external_metrics.ExternalMetricValueList, error) {
	name := info.Metric
	//TODO: Remove?
	nameLabel, found := metricSelector.RequiresExactMatch("name")
	if found {
		name = nameLabel
	}

	matchingMetrics := []external_metrics.ExternalMetricValue{}
	value, wfInfo, err := p.orchestrator.QueryMetric(name)
	if err != nil {
		if err.Error() == host.MetricErrNotFound {
			return nil, errors.NewNotFound(external_metrics.Resource("GithubWorkflowConfig"), name)
		}
		fmt.Println(err.Error())
		return nil, errors.NewBadRequest("Error getting metric")
	}
	value = wfInfo.Scaling.GetOutput(value)
	defer guageRequestedRunners.WithLabelValues(name).Set(float64(value))

	matchingMetrics = append(matchingMetrics, external_metrics.ExternalMetricValue{
		MetricName: name,
		MetricLabels: map[string]string{
			"owner":      wfInfo.Owner,
			"repository": wfInfo.Repository, //TODO: select by RunnerLabels
		},
		Value:     *resource.NewQuantity(int64(value), resource.BinarySI),
		Timestamp: metav1.Now(),
	})

	return &external_metrics.ExternalMetricValueList{
		Items: matchingMetrics,
	}, nil
}

func (p *workflowQueueProvider) ListAllExternalMetrics() []provider.ExternalMetricInfo {
	externalMetricsInfo := []provider.ExternalMetricInfo{}
	metrics, err := p.orchestrator.GetAllMetricNames()
	if err != nil {
		klog.Error(err)
	}
	for _, name := range metrics {
		externalMetricsInfo = append(externalMetricsInfo, provider.ExternalMetricInfo{Metric: name})
	}
	return externalMetricsInfo
}

var guageRequestedRunners *prometheus.GaugeVec

func init() {
	labelNames := []string{"name"}
	guageRequestedRunners = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "workflow_requested_runners",
		Help: "Number of runners requested",
	}, labelNames)

}
