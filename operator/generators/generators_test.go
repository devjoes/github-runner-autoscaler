package generators

import (
	"testing"

	"github.com/devjoes/github-runner-autoscaler/operator/api/v1alpha1"
	"github.com/stretchr/testify/assert"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestCreatesScaledObject(t *testing.T) {
	co := GenerateScaledObject(&v1alpha1.ScaledActionRunner{
		ObjectMeta: v1.ObjectMeta{
			Name:      "Foo",
			Namespace: "Bar",
		},
		Spec: v1alpha1.ScaledActionRunnerSpec{
			MaxRunners: 10,
		},
	}, "https://foo/bar")
	assert.NotNil(t, co)
}
