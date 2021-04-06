package labeling

import (
	"testing"

	"github.com/devjoes/github-runner-autoscaler/apiserver/pkg/config"
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
	assert.Len(t, promLabels, 2)
	assert.Equal(t, "", promLabels[0])
	assert.Equal(t, "aaa,z", promLabels[1])
	assert.Len(t, allLabels, 2)
	assert.Equal(t, "aaa,z", allLabels[JobLabelsForPrometheus[1]])
	assert.Equal(t, "prometheus", allLabels["not"])
}

func TestFilterBySelectorMatchesAll(t *testing.T) {
	jobs, wf := getTestData()
	matched, labels := FilterBySelector(jobs, wf, labels.Everything())
	assert.Equal(t, jobs, matched)
	assert.Len(t, labels[JobStatusLabel], 4)
	assert.Len(t, labels[WfIdLabel], 10)
}

func TestFilterBySelectorMatchesSelector(t *testing.T) {
	jobs, wf := getTestData()
	matched, labels := FilterBySelector(jobs, wf, labels.SelectorFromSet(labels.Set{"job_status": "queued"}))
	assert.Len(t, matched, 20)
	assert.Len(t, labels[JobStatusLabel], 1)
}

func getTestData() ([]*github.WorkflowRun, *config.GithubWorkflowConfig) {
	statuses := []string{"queued", "waiting", "requested", "in_progress"}
	jobs := []*github.WorkflowRun{}
	for i := int64(0); i < 80; i++ {
		id := i
		wfId := id % 10
		jobs = append(jobs, &github.WorkflowRun{
			ID:         &id,
			WorkflowID: &wfId,
			Status:     &statuses[id%4],
		})
	}
	wf := config.GithubWorkflowConfig{
		Name:      testName,
		Namespace: testNamespace,
		Owner:     testOwner,
	}
	return jobs, &wf
}
