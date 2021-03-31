package k8sprovider

import (
	"fmt"
	"time"

	"github.com/devjoes/github-runner-autoscaler/apiserver/pkg/config"
	"github.com/devjoes/github-runner-autoscaler/apiserver/pkg/host"
	"github.com/devjoes/github-runner-autoscaler/apiserver/pkg/utils"
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

func (p *workflowQueueProvider) valueFor(info provider.CustomMetricInfo, name types.NamespacedName, metricSelector labels.Selector) (resource.Quantity, *config.GithubWorkflowConfig, error) {
	var err error
	if metricSelector == nil {
		metricSelector, err = labels.Parse(info.Metric)
		fmt.Println(err)
	}
	fmt.Println(metricSelector)
	jobs, wf, err := p.orchestrator.QueryMetric(name.Name)
	fmt.Println(jobs)
	if err != nil && err.Error() == host.MetricErrNotFound {
		return resource.Quantity{}, nil, provider.NewMetricNotFoundForError(info.GroupResource, info.Metric, name.Name)
	}

	var metricLabels []string
	shouldFilterByLabel := info.Metric != "" && info.Metric != "all"
	if shouldFilterByLabel {
		//TODO: doesnt filter by label
		jobs, err = filterQueuedJobs(jobs, metricSelector)
		metricLabels = utils.GetJobLabelDataForMetrics(&jobs)
	} else {
		for i := 0; i < len(utils.JobLabelsToIncludeInMetrics); i++ {
			metricLabels = append(metricLabels, "")
		}

	}
	metricLabels = append([]string{name.Name, metricSelector.String()}, metricLabels...)
	if err != nil {
		return resource.Quantity{}, wf, err
	}
	total := len(jobs)
	// if total == 0 {
	// 	return resource.Quantity{}, nil, provider.NewMetricNotFoundForSelectorError(info.GroupResource, info.Metric, name.Name, metricSelector)
	// }

	scaledTotal := int(wf.Scaling.GetOutput(int32(total)))
	defer guageFilteredQueueLength.WithLabelValues(metricLabels...).Set(float64(total))
	defer guageFilteredScaledQueueLength.WithLabelValues(metricLabels...).Set(float64(scaledTotal))
	return *resource.NewQuantity(int64(scaledTotal), resource.DecimalSI), wf, nil
}

func filterQueuedJobs(labeledQueuedJobs map[int64]map[string]string, metricSelector labels.Selector) (map[int64]map[string]string, error) {
	matched := map[int64]map[string]string{}
	for jobId, l := range labeledQueuedJobs {
		var lbls labels.Set
		var s labels.Set = l

		lbls, err := labels.ConvertSelectorToLabelsMap(s.String())
		if err != nil {
			return nil, err
		}

		include := metricSelector.Matches(lbls)
		fmt.Printf("selector curLabelStr:%s include:%t curLabel:%v metricSelector:%v err:%v\n", l, include, s, metricSelector, err)
		if include {
			matched[jobId] = l
		}
	}
	return matched, nil
}

func (p *workflowQueueProvider) GetMetricByName(name types.NamespacedName, info provider.CustomMetricInfo, metricSelector labels.Selector) (*custom_metrics.MetricValue, error) {

	fmt.Printf("GetMetricByName '%s/%s' '%s' '%s' '%v'\n", name.Namespace, name.Name, info.Metric, info.String(), metricSelector)
	value, wf, err := p.valueFor(info, name, metricSelector)

	fmt.Printf("%v %v %v\n", value, wf, err)
	if err != nil {
		if err.Error() == host.MetricErrNotFound {
			return nil, errors.NewNotFound(external_metrics.Resource("GithubWorkflowConfig"), name.Name)
		}
		fmt.Println(err.Error())
		return nil, errors.NewBadRequest("Error getting metric")
	}

	metric, err := custom_metrics.MetricValue{
		Metric: custom_metrics.MetricIdentifier{
			Name: name.Name,
		},
		DescribedObject: custom_metrics.ObjectReference{Kind: "ScaledActionRunner", APIVersion: "v1alpha1", Name: wf.Name, Namespace: wf.Namespace},
		Value:           value,
		Timestamp:       metav1.Time{time.Now()},
	}, nil
	fmt.Println(metric)
	return &metric, err
}

func (p *workflowQueueProvider) GetMetricBySelector(namespace string, selector labels.Selector, info provider.CustomMetricInfo, metricSelector labels.Selector) (*custom_metrics.MetricValueList, error) {
	fmt.Println("GetMetricBySelector")
	fmt.Printf("GetMetricBySelector %s %v %v\n", namespace, info, metricSelector)

	metrics := custom_metrics.MetricValueList{}
	for _, lbls := range p.orchestrator.GetAllMetricLabels() {
		if selector.Matches(lbls) {
			nsName := types.NamespacedName{Namespace: namespace, Name: lbls["name"]}
			v, wf, err := p.valueFor(info, nsName, metricSelector)
			if wf.Namespace != namespace {
				continue
			}
			if err != nil {
				return nil, err
			}
			metric := custom_metrics.MetricValue{
				Metric: custom_metrics.MetricIdentifier{
					Name:     nsName.Name,
					Selector: &v1.LabelSelector{MatchLabels: lbls},
				},
				DescribedObject: custom_metrics.ObjectReference{Kind: "ScaledActionRunner", APIVersion: "v1alpha1", Name: wf.Name, Namespace: wf.Namespace},
				Value:           v,
				Timestamp:       metav1.Now(),
			}
			metrics.Items = append(metrics.Items, metric)
		}
	}
	return &metrics, nil
}

func (p *workflowQueueProvider) ListAllMetrics() []provider.CustomMetricInfo {
	fmt.Println("ListAllMetrics")
	infos := make(map[provider.CustomMetricInfo]struct{})
	metricData, err := p.orchestrator.GetAllMetricNames()
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
	labelNames := append([]string{"name", "selector"}, utils.JobLabelsToIncludeInMetrics...)
	guageFilteredQueueLength = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "workflow_queue_length_filtered",
		Help: "The number of queued jobs filtered by labels",
	}, labelNames)
	guageFilteredScaledQueueLength = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "workflow_queue_length_filtered_scaled",
		Help: "The scaled up/down number of queued jobs filtered by labels",
	}, labelNames)

}
