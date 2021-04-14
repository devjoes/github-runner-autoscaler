package sargenerator

import (
	"testing"

	"github.com/devjoes/github-runner-autoscaler/operator/api/v1alpha1"
	"github.com/stretchr/testify/assert"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestCreatesScaledObject(t *testing.T) {
	sar := v1alpha1.ScaledActionRunner{
		ObjectMeta: v1.ObjectMeta{
			Name:      "Foo",
			Namespace: "Bar",
		},
		Spec: v1alpha1.ScaledActionRunnerSpec{
			MaxRunners: 10,
		},
	}
	so := GenerateScaledObject(&sar, "https://foo/bar", "baz")
	assert.NotNil(t, so)
	assert.Equal(t, sar.ObjectMeta.Name, so.Name)
	assert.Equal(t, sar.ObjectMeta.Namespace, so.Namespace)
}

// This is mostly tested in scaledactionrunner_controller_test
