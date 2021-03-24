package generators

import (
	"testing"

	"github.com/devjoes/github-runner-autoscaler/operator/api/v1alpha1"
	"github.com/stretchr/testify/assert"
)

func TestCreatesScaledObject(t *testing.T) {
	co := GenerateScaledObject(&v1alpha1.ScaledActionRunner{
		Spec: v1alpha1.ScaledActionRunnerSpec{
			Name:       "Foo",
			Namespace:  "Bar",
			MaxRunners: 10,
		},
	}, "https://foo/bar")
	assert.NotNil(t, co)
}
