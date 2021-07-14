package sargenerator

import (
	"testing"

	"github.com/devjoes/github-runner-autoscaler/operator/api/v1alpha1"
	"github.com/stretchr/testify/assert"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
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

func TestDoesNothingIfPatchIsMissing(t *testing.T) {
	ss := getTestSs()
	result, hash, err := PatchStatefulSet(ss, &v1alpha1.ScaledActionRunner{
		Spec: v1alpha1.ScaledActionRunnerSpec{},
	})
	assert.Nil(t, result)
	assert.Equal(t, "", hash)
	assert.Nil(t, err)

	result, hash, err = PatchStatefulSet(ss, &v1alpha1.ScaledActionRunner{
		Spec: v1alpha1.ScaledActionRunnerSpec{
			Runner: &v1alpha1.Runner{},
		},
	})
	assert.Nil(t, result)
	assert.Equal(t, "", hash)
	assert.Nil(t, err)
}

func TestAppliesPatchAndSetsAnnotation(t *testing.T) {
	ss := getTestSs()
	result, hash, err := PatchStatefulSet(ss, &v1alpha1.ScaledActionRunner{
		Spec: v1alpha1.ScaledActionRunnerSpec{
			Runner: &v1alpha1.Runner{
				Patch: `[{"op": "replace", "path": "/serviceName", "value": "replaced"}]`,
			},
		},
	})
	assert.NotNil(t, result)
	assert.Nil(t, err)
	assert.Equal(t, "replaced", result.Spec.ServiceName)
	assert.NotEqual(t, "", result.Annotations[AnnotationRunnerPatchHash])
	assert.NotEqual(t, "", hash)
}

func TestDoesNotApplyPatchTwice(t *testing.T) {
	ss := getTestSs()

	result, hash1, err := PatchStatefulSet(ss, &v1alpha1.ScaledActionRunner{
		Spec: v1alpha1.ScaledActionRunnerSpec{
			Runner: &v1alpha1.Runner{
				Patch: `[{"op": "replace", "path": "/serviceName", "value": "replaced"}]`,
			},
		},
	})
	assert.Nil(t, err)
	assert.NotEqual(t, hash1, ss.ObjectMeta.Annotations[AnnotationRunnerPatchHash])
	assert.Equal(t, hash1, result.ObjectMeta.Annotations[AnnotationRunnerPatchHash])
	assert.NotEqual(t, "", hash1)

	ss = getTestSs()
	ss.ObjectMeta.Annotations[AnnotationRunnerPatchHash] = hash1

	result, hash2, err := PatchStatefulSet(ss, &v1alpha1.ScaledActionRunner{
		Spec: v1alpha1.ScaledActionRunnerSpec{
			Runner: &v1alpha1.Runner{
				Patch: `[{"op": "replace", "path": "/serviceName", "value": "replaced"}]`,
			},
		},
	})
	assert.Nil(t, result)
	assert.Equal(t, hash1, hash2)
	assert.Nil(t, err)
}

func TestReplicasDoesNotAffectHash(t *testing.T) {
	ss := getTestSs()

	_, hash1, err := PatchStatefulSet(ss, &v1alpha1.ScaledActionRunner{
		Spec: v1alpha1.ScaledActionRunnerSpec{
			Runner: &v1alpha1.Runner{
				Patch: `[{"op": "replace", "path": "/serviceName", "value": "replaced"}]`,
			},
		},
	})
	assert.Nil(t, err)

	ss = getTestSs()
	moreReplicas := (*ss.Spec.Replicas + 10)
	ss.Spec.Replicas = &moreReplicas
	_, hash2, err := PatchStatefulSet(ss, &v1alpha1.ScaledActionRunner{
		Spec: v1alpha1.ScaledActionRunnerSpec{
			Runner: &v1alpha1.Runner{
				Patch: `[{"op": "replace", "path": "/serviceName", "value": "replaced"}]`,
			},
		},
	})
	assert.Equal(t, hash1, hash2)
	assert.Nil(t, err)
}

func TestTriggersDeletionIfPatchChanges(t *testing.T) {
	ss := getTestSs()
	_, hash1, err := PatchStatefulSet(ss, &v1alpha1.ScaledActionRunner{
		Spec: v1alpha1.ScaledActionRunnerSpec{
			Runner: &v1alpha1.Runner{
				Patch: `[{"op": "replace", "path": "/serviceName", "value": "replaced"}]`,
			},
		},
	})
	assert.Nil(t, err)
	_, hash2, err := PatchStatefulSet(ss, &v1alpha1.ScaledActionRunner{
		Spec: v1alpha1.ScaledActionRunnerSpec{
			Runner: &v1alpha1.Runner{
				Patch: `[{"op": "replace", "path": "/serviceName", "value": "different"}]`,
			},
		},
	})
	assert.NotEqual(t, "", hash1)
	assert.NotEqual(t, "", hash2)
	assert.NotEqual(t, hash1, hash2)
	assert.Nil(t, err)
}

func getTestSs() *appsv1.StatefulSet {
	var replicas int32 = 2
	ss := appsv1.StatefulSet{
		TypeMeta:   v1.TypeMeta{},
		ObjectMeta: v1.ObjectMeta{Name: "foo", Namespace: "bar", Annotations: map[string]string{}},
		Spec: appsv1.StatefulSetSpec{
			Replicas: &replicas,
			Selector: &v1.LabelSelector{MatchLabels: map[string]string{"a": "b"}},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: v1.ObjectMeta{Name: "foo", Namespace: "bar"},
				Spec: corev1.PodSpec{
					Volumes:        []corev1.Volume{},
					InitContainers: []corev1.Container{},
					Containers:     []corev1.Container{corev1.Container{Name: "baz"}},
				},
			},
			VolumeClaimTemplates: []corev1.PersistentVolumeClaim{},
			ServiceName:          "foo",
			PodManagementPolicy:  "",
			UpdateStrategy:       appsv1.StatefulSetUpdateStrategy{},
			RevisionHistoryLimit: new(int32),
		},
		Status: appsv1.StatefulSetStatus{},
	}
	return &ss
}

// This is mostly tested in scaledactionrunner_controller_test
