package generators

import (
	"fmt"
	"testing"

	"github.com/devjoes/github-runner-autoscaler/operator/api/v1alpha1"
	"github.com/stretchr/testify/assert"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestCreatesScaledObject(t *testing.T) {
	a := ArmGenerateDeployment(&v1alpha1.ActionRunnerMetrics{
		ObjectMeta: v1.ObjectMeta{
			Name:      "a",
			Namespace: "b",
		},
		Spec: v1alpha1.ActionRunnerMetricsSpec{
			Image:                       "a",
			CreateMemcached:             true,
			Replicas:                    3,
			ExistingMemcacheUser:        "a",
			ExistingMemcacheCredsSecret: "b",
			CacheWindow:                 123,
			CacheWindowWhenEmpty:        234,
			ResyncInterval:              345,
			ExistingMemcacheServers:     "foo",
			MemcachedReplicas:           3,
			ExistingSslCertSecret:       "cert",
			Namespaces:                  []string{"a", "b"},
		},
	})
	fmt.Println(a)
	co := SarGenerateScaledObject(&v1alpha1.ScaledActionRunner{
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
