package utils

import (
	"sort"
	"strings"
)

const WfIdLabel = "wf_id"

//const WfNameLabel = "wf_name"//TODO: get name and labels
const JobStatusLabel = "job_status"

var JobLabelsToIncludeInMetrics []string = []string{JobStatusLabel, WfIdLabel}

func ContainsStr(arr []string, i string) bool {
	for _, x := range arr {
		if x == i {
			return true
		}
	}
	return false
}

func GetJobLabelDataForMetrics(labeledJobIds *map[int64]map[string]string) []string {
	allJobLabels := map[string][]string{}
	for _, jobLabels := range *labeledJobIds {
		for _, lKey := range JobLabelsToIncludeInMetrics {
			curVal, exists := allJobLabels[lKey]
			if !exists {
				allJobLabels[lKey] = []string{}
			}
			lValue := jobLabels[lKey]
			if !ContainsStr(allJobLabels[lKey], lValue) {
				allJobLabels[lKey] = append(curVal, lValue)
			}
		}
	}
	commaSeperated := []string{}
	for _, label := range JobLabelsToIncludeInMetrics {
		lbls := allJobLabels[label]
		sort.Strings(lbls)
		commaSeperated = append(commaSeperated, strings.Join(lbls, ","))
	}
	return commaSeperated
}
