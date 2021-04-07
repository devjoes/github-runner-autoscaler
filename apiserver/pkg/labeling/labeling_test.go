package labeling

import (
	"fmt"
	"testing"

	"github.com/devjoes/github-runner-autoscaler/apiserver/pkg/config"
	utils "github.com/devjoes/github-runner-autoscaler/apiserver/pkg/utils"
	"github.com/google/go-github/v33/github"
	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/labels"
)

const (
	testName      = "testName"
	testNamespace = "testNamespace"
	testOwner     = "testOwner"
)

func TestJoinsLabels(t *testing.T) {
	promLabels, allLabels := GetLabelsForOutput(map[string][]string{
		JobLabelsForPrometheus[1]: {"z", "aaa"},
		"not":                     {"prometheus"},
	})
	assert.Len(t, promLabels, 3)
	assert.Equal(t, "", promLabels[0])
	assert.Equal(t, "aaa,z", promLabels[1])
	assert.Len(t, allLabels, 2)
	assert.Equal(t, "aaa,z", allLabels[JobLabelsForPrometheus[1]])
	assert.Equal(t, "prometheus", allLabels["not"])
}

func TestFilterBySelectorMatchesAll(t *testing.T) {
	jobs, wf, wfInfo := getTestData()
	matched, labels := FilterBySelector(jobs, wf, wfInfo, labels.Everything())
	assert.Equal(t, jobs, matched)
	assert.Len(t, labels[WfNameLabel], 10)
	assert.Len(t, labels[WfIdLabel], 10)
}

func TestFilterBySelectorMatchesSelector(t *testing.T) {
	jobs, wf, wfInfo := getTestData()
	matched, labels := FilterBySelector(jobs, wf, wfInfo, labels.SelectorFromSet(labels.Set{WfRunsOnLabel: "runson_5"}))
	assert.Len(t, matched, 8)
	assert.Len(t, labels[WfRunsOnLabel], 1)
}

func getTestData() ([]*github.WorkflowRun, *config.GithubWorkflowConfig, map[int64]utils.WorkflowInfo) {
	jobs := []*github.WorkflowRun{}
	wfInfo := make(map[int64]utils.WorkflowInfo)
	for i := int64(0); i < 80; i++ {
		id := i
		wfId := id % 10
		jobs = append(jobs, &github.WorkflowRun{
			ID:         &id,
			WorkflowID: &wfId,
		})
		wfInfo[wfId] = utils.WorkflowInfo{
			ID:     wfId,
			Name:   fmt.Sprintf("wf %d", wfId),
			Labels: []string{fmt.Sprintf("runson_%d", wfId)},
		}
	}
	wf := config.GithubWorkflowConfig{
		Name:      testName,
		Namespace: testNamespace,
		Owner:     testOwner,
	}
	return jobs, &wf, wfInfo
}
