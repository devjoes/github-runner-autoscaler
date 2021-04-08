package labeling

import (
	"fmt"
	"regexp"
	"sort"
	"strings"

	"github.com/devjoes/github-runner-autoscaler/apiserver/pkg/config"
	utils "github.com/devjoes/github-runner-autoscaler/apiserver/pkg/utils"
	"github.com/google/go-github/v33/github"
	"k8s.io/apimachinery/pkg/labels"
)

const (
	WfIdLabel        = "wf_id"
	WfNameLabel      = "wf_name"
	WfAllRunsOn      = "wf_runs_on"
	CrNameLabel      = "cr_name"
	CrNamespaceLabel = "cr_namespace"
	CrRepoLabel      = "cr_repo"
	CrOwnerLabel     = "cr_owner"
)

var JobLabelsForPrometheus []string = []string{WfIdLabel, WfNameLabel, WfAllRunsOn}

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

var rx *regexp.Regexp = regexp.MustCompile("(?i)[^-a-z\\d_.]")
var rxEdges *regexp.Regexp = regexp.MustCompile("(?i)(^[^a-z\\d]+)|([^a-z\\d]+$)")

func sanitizeLabelValue(val string) string {
	return rxEdges.ReplaceAllString(rx.ReplaceAllString(val, "_"), "")
}

func getLabels(r *github.WorkflowRun, wf *config.GithubWorkflowConfig, wfInfo map[int64]utils.WorkflowInfo) labels.Set {
	var lbls labels.Set = map[string]string{
		WfIdLabel:        fmt.Sprintf("%d", *r.WorkflowID),
		CrNameLabel:      sanitizeLabelValue(wf.Name),
		CrNamespaceLabel: sanitizeLabelValue(wf.Namespace),
		CrRepoLabel:      sanitizeLabelValue(wf.Repository),
		CrOwnerLabel:     sanitizeLabelValue(wf.Owner),
	}
	lbls[WfNameLabel] = "unknown"
	info, found := wfInfo[*r.WorkflowID]
	if found {
		lbls[WfNameLabel] = sanitizeLabelValue(info.Name)
		allRunsOn := strings.Builder{}
		sort.Strings(info.Labels)
		for i, rl := range info.Labels {
			if i > 0 {
				allRunsOn.WriteString(".")
			}
			allRunsOn.WriteString(strings.ReplaceAll(rl, ".", "_"))
			lbls[fmt.Sprintf("wf_runs_on_%s", rl)] = rl
		}
		lbls[WfAllRunsOn] = allRunsOn.String()
	}
	return lbls
}

func FilterBySelector(runs []*github.WorkflowRun, wf *config.GithubWorkflowConfig, wfInfo map[int64]utils.WorkflowInfo, selector labels.Selector) ([]*github.WorkflowRun, map[string][]string) {
	filtered := []*github.WorkflowRun{}
	matchedLabels := map[string][]string{}
	for _, r := range runs {
		lbls := getLabels(r, wf, wfInfo)
		if selector.Matches(lbls) {
			for l, v := range lbls {
				if !utils.ContainsStr(matchedLabels[l], v) {
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
