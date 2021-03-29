package generators

import (
	"fmt"
	"reflect"

	runnerv1alpha1 "github.com/devjoes/github-runner-autoscaler/operator/api/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/api/core/v1"
	rbac "k8s.io/api/rbac/v1"

	keda "github.com/kedacore/keda/v2/api/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/apimachinery/pkg/util/yaml"
)

const AnnotationRunnerRef = "runner-ref"
const AnnotationSecretsHash = "runner-secrets-hash"

func getLabels(res metav1.Object) map[string]string {
	ls := res.GetLabels()
	if ls == nil {
		ls = map[string]string{}
	}
	ls["product"] = "github_actions_operator"
	return ls
}

//TODO: Take this approach for StatefulSets too
func SarUpdateScaledObjectSpec(c *runnerv1alpha1.ScaledActionRunner, url string, spec *keda.ScaledObjectSpec) {
	spec.ScaleTargetRef = &keda.ScaleTarget{
		Kind:       "StatefulSet",
		Name:       c.ObjectMeta.Name,
		APIVersion: "apps/v1",
	}
	spec.MinReplicaCount = &c.Spec.MinRunners
	spec.MaxReplicaCount = &c.Spec.MaxRunners
	spec.Triggers = []keda.ScaleTriggers{
		{
			Type: "metrics-api",
			AuthenticationRef: &keda.ScaledObjectAuthRef{
				Name: "certs",
			},
			Metadata: map[string]string{
				"targetValue":   "1",
				"url":           url,
				"valueLocation": "items.0.value",
				"authMode":      "tls",
			},
		},
	}
	if c.Spec.Scaling != nil {
		spec.CooldownPeriod = c.Spec.Scaling.CooldownPeriod
		spec.PollingInterval = c.Spec.Scaling.PollingInterval
		if c.Spec.Scaling.Behavior != nil {
			spec.Advanced = &keda.AdvancedConfig{
				HorizontalPodAutoscalerConfig: &keda.HorizontalPodAutoscalerConfig{
					Behavior: c.Spec.Scaling.Behavior,
				},
			}
		}
	}
}

func SarGenerateScaledObject(c *runnerv1alpha1.ScaledActionRunner, url string) *keda.ScaledObject {
	ls := getLabels(c)
	spec := keda.ScaledObjectSpec{}
	SarUpdateScaledObjectSpec(c, url, &spec)

	resource := keda.ScaledObject{
		ObjectMeta: metav1.ObjectMeta{Name: c.ObjectMeta.Name, Namespace: c.ObjectMeta.Namespace, Labels: ls},
		Spec:       spec,
	}
	return &resource
}

func SarGenerateStatefulSet(c *runnerv1alpha1.ScaledActionRunner, secretsHash string) *appsv1.StatefulSet {
	ls := getLabels(c)
	ls["app"] = "action-runner"
	as := map[string]string{
		AnnotationSecretsHash: secretsHash,
	}
	resource := appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:        c.ObjectMeta.Name,
			Namespace:   c.ObjectMeta.Namespace,
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
						Image:        c.Spec.Runner.Image,
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
					Spec: *c.Spec.Runner.WorkVolumeClaimTemplate,
				},
			},
		},
	}
	SarSetEnvVars(c, &resource)

	volumes, volumeMounts := SarGetVolumes(c)
	resource.Spec.Template.Spec.Volumes = volumes
	resource.Spec.Template.Spec.Containers[0].VolumeMounts = volumeMounts

	return &resource
}

func SarGetVolumes(c *runnerv1alpha1.ScaledActionRunner) ([]corev1.Volume, []corev1.VolumeMount) {
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
		name := fmt.Sprintf("%s-%d", c.ObjectMeta.Name, i)
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

func SarSetEnvVars(c *runnerv1alpha1.ScaledActionRunner, statefulSet *appsv1.StatefulSet) bool {
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
	if c.Spec.Runner.Labels != "" {
		toSet["LABELS"] = corev1.EnvVar{Name: "LABELS", Value: c.Spec.Runner.Labels}
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
	statefulSet.Spec.Template.Spec.Containers[0].Resources.Requests = *c.Spec.Runner.Requests
	statefulSet.Spec.Template.Spec.Containers[0].Resources.Limits = *c.Spec.Runner.Limits
	for _, e := range toSet {
		modified = true
		statefulSet.Spec.Template.Spec.Containers[0].Env = append(statefulSet.Spec.Template.Spec.Containers[0].Env, e)
	}
	return modified
}

func ArmGenerateExternalMetrics(c *runnerv1alpha1.ActionRunnerMetrics) (*appsv1.Deployment, *v1.Service, *v1.ServiceAccount, []rbac.ClusterRole, []rbac.ClusterRoleBinding) {
	ls := getLabels(c)
	ls["app"] = "external-metrics-apiserver"
	jsonRes := `{
		"apiVersion": "apps/v1",
		"kind": "Deployment",
		"metadata": {
			"labels": {
				"app": "external-metrics-apiserver"
			},
			"name": "external-metrics-apiserver",
			"namespace": "runners"
		},
		"spec": {
			"replicas": 2,
			"selector": {
				"matchLabels": {
					"app": "external-metrics-apiserver"
				}
			},
			"template": {
				"metadata": {
					"labels": {
						"app": "external-metrics-apiserver"
					},
					"name": "external-metrics-apiserver"
				},
				"spec": {
					"containers": [
						{
							"args": [
								"--secure-port=6443",
								"--logtostderr=true",
								"--incluster",
								"--tls-cert-file=/apiserver.local.config/certificates/cert",
								"--tls-private-key-file=/apiserver.local.config/certificates/key"
							],
							"image": "joeshearn/github-runner-autoscaler-apiserver:000021",
							"imagePullPolicy": "IfNotPresent",
							"name": "external-metrics-apiserver",
							"ports": [
								{
									"containerPort": 6443,
									"name": "https",
									"protocol": "TCP"
								},
								{
									"containerPort": 2112,
									"name": "metrics",
									"protocol": "TCP"
								}
							],
							"env": [
								{
									"name": "MEMCACHEDPASS",
									"valueFrom": {
										"secretKeyRef": {
											"name": "memcache",
											"key": "password"
										}
									}
								}
							],
							"resources": {},
							"terminationMessagePath": "/dev/termination-log",
							"terminationMessagePolicy": "File",
							"volumeMounts": [
								{
									"mountPath": "/tmp",
									"name": "temp-vol"
								},
								{
									"mountPath": "/apiserver.local.config/certificates",
									"name": "cert",
									"readOnly": true
								}
							]
						}
					],
					"dnsPolicy": "ClusterFirst",
					"restartPolicy": "Always",
					"schedulerName": "default-scheduler",
					"securityContext": {},
					"serviceAccount": "external-metrics-apiserver",
					"serviceAccountName": "external-metrics-apiserver",
					"terminationGracePeriodSeconds": 30,
					"volumes": [
						{
							"emptyDir": {},
							"name": "temp-vol"
						},
						{
							"name": "cert",
							"secret": {
								"defaultMode": 420,
								"secretName": "cert"
							}
						}
					]
				}
			}
		}
	}`
	var dep appsv1.Deployment
	err := yaml.Unmarshal([]byte(jsonRes), &dep)
	if err != nil {
		fmt.Println(err)
	}
	dep.Name = c.Name
	dep.Spec.Template.Name = c.Name
	dep.Labels = ls
	dep.Spec.Template.Labels = ls
	dep.Spec.Selector.MatchLabels = ls

	dep.Namespace = c.ObjectMeta.Namespace
	dep.Spec.Replicas = &c.Spec.Replicas
	dep.Spec.Template.Spec.Containers[0].Image = c.Spec.Image

	var args []string
	if len(c.Spec.Namespaces) == 0 {
		args = append(args, "--allnamespaces")
	}
	for _, n := range c.Spec.Namespaces {
		args = append(args, "--namespace='%s'", n)
	}

	mcServers := ""
	if c.Spec.CreateMemcached {
		for i := 1; i <= int(c.Spec.MemcachedReplicas); i++ {
			mcServers = fmt.Sprintf("%s%s-cache-%d:11211", mcServers, c.Name, i)
			if i < int(c.Spec.MemcachedReplicas) {
				mcServers = fmt.Sprintf("%s,", mcServers)
			}
		}
	}
	if c.Spec.ExistingMemcacheServers != "" {
		mcServers = c.Spec.ExistingMemcacheServers
	}
	if mcServers != "" {
		args = append(args, fmt.Sprintf("--memcached-servers='%s'", mcServers))
	}
	dep.Spec.Template.Spec.Containers[0].Args = append(dep.Spec.Template.Spec.Containers[0].Args, args...)
	dep.Spec.Template.Spec.ServiceAccountName = c.Name
	dep.Spec.Template.Spec.Volumes[1].Secret.SecretName = fmt.Sprintf("%s-cert", c.Name)
	if c.Spec.ExistingSslCertSecret != "" {
		dep.Spec.Template.Spec.Volumes[1].Secret.SecretName = c.Spec.ExistingSslCertSecret
	}

	dep.Spec.Template.Spec.Containers[0].Env[0].ValueFrom.SecretKeyRef.Name = fmt.Sprintf("%s-cache", c.Name)
	if c.Spec.ExistingMemcacheCredsSecret != "" {
		dep.Spec.Template.Spec.Containers[0].Env[0].ValueFrom.SecretKeyRef.Name = c.Spec.ExistingMemcacheCredsSecret
	} else if !c.Spec.CreateMemcached {
		dep.Spec.Template.Spec.Containers[0].Env = []corev1.EnvVar{}
	}
	svc := v1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      c.Name,
			Namespace: c.Namespace,
			Labels:    c.Labels,
		},
		Spec: v1.ServiceSpec{
			Ports: []v1.ServicePort{
				v1.ServicePort{
					Name:       "https",
					Protocol:   corev1.ProtocolTCP,
					Port:       443,
					TargetPort: intstr.FromInt(6443),
				},
				v1.ServicePort{
					Name:       "metrics",
					Protocol:   corev1.ProtocolTCP,
					Port:       2112,
					TargetPort: intstr.FromInt(2112),
				},
			},
			Selector: map[string]string{
				"app": c.Name,
			},
		},
	}
	sa := v1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name:      c.Name,
			Namespace: c.Namespace,
			Labels:    c.Labels,
		},
	}
	r1:=rbac.ClusterRole
	rb1 := rbac.ClusterRoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:      c.Name,
			Namespace: c.Namespace,
			Labels:    c.Labels,
		},
		RoleRef: ,
	}
	return &dep, &svc, &sa, nil, nil
}
