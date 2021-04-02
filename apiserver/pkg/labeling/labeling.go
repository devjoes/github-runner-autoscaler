package labeling

import (
	"fmt"
	"sort"
	"strings"

	"github.com/devjoes/github-runner-autoscaler/apiserver/pkg/config"
	"github.com/google/go-github/v33/github"
	"k8s.io/apimachinery/pkg/labels"
)

const WfIdLabel = "wf_id"

//const WfNameLabel = "wf_name"//TODO: get name and runner labels
const JobStatusLabel = "job_status"

var JobLabelsForPrometheus []string = []string{JobStatusLabel, WfIdLabel}

func containsStr(arr []string, i string) bool {
	for _, x := range arr {
		if x == i {
			return true
		}
	}
	return false
}

// func GetJobLabelDataForMetrics(labeledJobIds *map[int64]map[string]string) []string {
// 	allJobLabels := map[string][]string{}
// 	for _, jobLabels := range *labeledJobIds {
// 		for _, lKey := range JobLabelsForPrometheus {
// 			curVal, exists := allJobLabels[lKey]
// 			if !exists {
// 				allJobLabels[lKey] = []string{}
// 			}
// 			lValue := jobLabels[lKey]
// 			if !containsStr(allJobLabels[lKey], lValue) {
// 				allJobLabels[lKey] = append(curVal, lValue)
// 			}
// 		}
// 	}
// 	commaSeperated := []string{}
// 	for _, label := range JobLabelsForPrometheus {
// 		lbls := allJobLabels[label]
// 		sort.Strings(lbls)
// 		commaSeperated = append(commaSeperated, strings.Join(lbls, ","))
// 	}
// 	return commaSeperated
// }

func GetLabelsForOutput(lbls map[string][]string) ([]string, map[string]string) {
	prometheusLabels := make([]string, len(JobLabelsForPrometheus))
	allLabelStrings := make(map[string]string, len(lbls))
	for l, v := range lbls {
		ls := v
		sort.Strings(ls)
		str := strings.Join(ls, ",")
		allLabelStrings[l] = str
	}
	for i, l := range JobLabelsForPrometheus {
		prometheusLabels[i] = ""
		ls, found := allLabelStrings[l]
		if found {
			prometheusLabels[i] = ls
		}
	}
	return prometheusLabels, allLabelStrings
}

func getLabels(r *github.WorkflowRun, wf *config.GithubWorkflowConfig) labels.Set {
	var lbls labels.Set = map[string]string{
		WfIdLabel:      fmt.Sprintf("%d", *r.WorkflowID),
		JobStatusLabel: *r.Status,
		"name":         wf.Name,
		"namespace":    wf.Namespace,
		"repo":         wf.Repository,
		"owner":        wf.Owner,
	}
	return lbls
}

func FilterBySelector(runs []*github.WorkflowRun, wf *config.GithubWorkflowConfig, selector labels.Selector) ([]*github.WorkflowRun, map[string][]string) {
	filtered := []*github.WorkflowRun{}
	matchedLabels := map[string][]string{}
	//wfExtra := getExtraWfInfo(resp)
	for _, r := range runs {
		lbls := getLabels(r, wf)
		if selector.Matches(lbls) {
			for l, v := range lbls {
				if !containsStr(matchedLabels[l], v) {
					matchedLabels[l] = append(matchedLabels[l], v)
				}
			}
			filtered = append(filtered, r)
		}
	}

	return filtered, matchedLabels
}

// func filterQueuedJobs(labeledQueuedJobs map[int64]map[string]string, metricSelector labels.Selector) (map[int64]map[string]string, error) {
// 	matched := map[int64]map[string]string{}
// 	for jobId, l := range labeledQueuedJobs {
// 		var lbls labels.Set
// 		var s labels.Set = l

// 		lbls, err := labels.ConvertSelectorToLabelsMap(s.String())
// 		if err != nil {
// 			return nil, err
// 		}

// 		include := metricSelector.Matches(lbls)
// 		fmt.Printf("selector curLabelStr:%s include:%t curLabel:%v metricSelector:%v err:%v\n", l, include, s, metricSelector, err)
// 		if include {
// 			matched[jobId] = l
// 		}
// 	}
// 	return matched, nil
// }
