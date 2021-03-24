package generators

import (
	"fmt"
	"reflect"

	runnerv1alpha1 "github.com/devjoes/github-runner-autoscaler/operator/api/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"

	keda "github.com/kedacore/keda/v2/api/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const AnnotationRunnerRef = "runner-ref"
const AnnotationSecretsHash = "runner-secrets-hash"

func getLabels(name string) map[string]string {
	return map[string]string{"app": "github_runner", "github_runner_cr": name}
}

func GenerateScaledObject(c *runnerv1alpha1.ScaledActionRunner, url string) *keda.ScaledObject {
	ls := getLabels(c.Name)
	resource := keda.ScaledObject{
		ObjectMeta: metav1.ObjectMeta{Name: c.Spec.Name, Namespace: c.Spec.Namespace, Labels: ls},
		Spec: keda.ScaledObjectSpec{
			MinReplicaCount: &c.Spec.MinRunners,
			MaxReplicaCount: &c.Spec.MaxRunners,
			ScaleTargetRef: &keda.ScaleTarget{
				Kind:       "StatefulSet",
				Name:       c.Spec.Name,
				APIVersion: "apps/v1",
			},
			Triggers: []keda.ScaleTriggers{
				{
					Type: "metrics-api",
					AuthenticationRef: &keda.ScaledObjectAuthRef{
						Name: "certs",
					},
					Metadata: map[string]string{
						"targetValue":   "1", //TODO: Should this be 0? it errors
						"url":           url,
						"valueLocation": "items.0.value",
						"authMode":      "tls",
					},
				},
			},
			//TODO: add - or just merge from own config
			// pollingInterval: 1
			// cooldownPeriod: 1800 # Wait 30 mins before scaling to 0
			// advanced:
			//   horizontalPodAutoscalerConfig:
			// 	behavior:
			// 	  scaleDown:
			// 		stabilizationWindowSeconds: 300
		},
	}
	return &resource
}

func GenerateStatefulSet(c *runnerv1alpha1.ScaledActionRunner, secretsHash string) *appsv1.StatefulSet {
	ls := getLabels(c.Name)
	as := map[string]string{
		AnnotationSecretsHash: secretsHash,
	}

	resource := appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:        c.Spec.Name,
			Namespace:   c.Spec.Namespace,
			Labels:      ls,
			Annotations: as,
		},
		Spec: appsv1.StatefulSetSpec{
			Replicas: &c.Spec.MinRunners,
			Selector: &metav1.LabelSelector{
				MatchLabels: ls,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: ls,
				},
				Spec: corev1.PodSpec{
					Volumes: []corev1.Volume{},
					Containers: []corev1.Container{{
						Image:        c.Spec.Image,
						Name:         "runner",
						Env:          []corev1.EnvVar{},
						VolumeMounts: []corev1.VolumeMount{},
					}},
				},
			}, VolumeClaimTemplates: []corev1.PersistentVolumeClaim{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "workdir",
					},
					Spec: corev1.PersistentVolumeClaimSpec{
						AccessModes: []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce},
						Resources: corev1.ResourceRequirements{
							Requests: corev1.ResourceList{
								corev1.ResourceStorage: *c.Spec.WorkVolumeSize,
							},
						},
					},
				},
			},
		},
	}
	SetEnvVars(c, &resource)

	volumes, volumeMounts := GetVolumes(c)
	resource.Spec.Template.Spec.Volumes = volumes
	resource.Spec.Template.Spec.Containers[0].VolumeMounts = volumeMounts

	return &resource
}

func GetVolumes(c *runnerv1alpha1.ScaledActionRunner) ([]corev1.Volume, []corev1.VolumeMount) {
	emptyString := ""
	var volumes []corev1.Volume = []corev1.Volume{
		{
			Name: "dockersock",
			VolumeSource: corev1.VolumeSource{
				HostPath: &corev1.HostPathVolumeSource{
					Path: "/var/run/docker.sock",
					Type: (*corev1.HostPathType)(&emptyString), // Required when comparing with deepeequals
				},
			},
		},
	}
	var volumeMounts []corev1.VolumeMount = []corev1.VolumeMount{
		{
			Name:      "dockersock",
			MountPath: "/var/run/docker.sock",
		},
		{
			Name:      "workdir",
			MountPath: "/work",
		},
	}

	for i := 0; i < int(c.Spec.MaxRunners); i++ {
		name := fmt.Sprintf("%s-%d", c.Spec.Name, i)
		if i >= len(c.Spec.RunnerSecrets) {
			break
		}

		mode := int32(420)
		volumes = append(volumes, corev1.Volume{
			Name: name,
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName:  c.Spec.RunnerSecrets[i],
					DefaultMode: &mode,
				},
				HostPath: nil,
			},
		})

		volumeMounts = append(volumeMounts,
			corev1.VolumeMount{
				Name:      name,
				ReadOnly:  true,
				MountPath: fmt.Sprintf("/actions-creds/%s", name),
			})
	}
	return volumes, volumeMounts
}

func SetEnvVars(c *runnerv1alpha1.ScaledActionRunner, statefulSet *appsv1.StatefulSet) bool {
	modified := false
	toSet := map[string]corev1.EnvVar{
		"REPO_URL": {
			Name:  "REPO_URL",
			Value: fmt.Sprintf("git@github.com:%s/%s.git", c.Spec.Owner, c.Spec.Repo),
		},
		"RUNNER_NAME": {
			Name: "RUNNER_NAME",
			ValueFrom: &corev1.EnvVarSource{
				FieldRef: &corev1.ObjectFieldSelector{APIVersion: "v1", FieldPath: "metadata.name"},
			},
		},
		"RUNNER_WORKDIR": {
			Name:  "RUNNER_WORKDIR",
			Value: "/work",
		},
	}
	if c.Spec.RunnerLabels != "" {
		toSet["LABELS"] = corev1.EnvVar{Name: "LABELS", Value: c.Spec.RunnerLabels}
	}
	for i, e := range statefulSet.Spec.Template.Spec.Containers[0].Env {
		if newVal, found := toSet[e.Name]; found {
			if !reflect.DeepEqual(e, newVal) {
				modified = true
				statefulSet.Spec.Template.Spec.Containers[0].Env[i] = newVal
			}
			delete(toSet, e.Name)
		}
	}
	for _, e := range toSet {
		modified = true
		statefulSet.Spec.Template.Spec.Containers[0].Env = append(statefulSet.Spec.Template.Spec.Containers[0].Env, e)
	}
	return modified
}
