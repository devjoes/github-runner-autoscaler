package armgenerator

import (
	"crypto/sha1"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"math/rand"
	"os"
	"strings"

	runnerv1alpha1 "github.com/devjoes/github-runner-autoscaler/operator/api/v1alpha1"
	keda "github.com/kedacore/keda/v2/api/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/api/core/v1"
	rbac "k8s.io/api/rbac/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/apimachinery/pkg/util/yaml"
)

const CrdKey = "crd_key"

func getLabels(res metav1.Object) map[string]string {
	ls := res.GetLabels()
	if ls == nil {
		ls = map[string]string{}
	}
	ls["product"] = "github_actions_operator"
	return ls
}
func GenerateMemcachedResources(c *runnerv1alpha1.ActionRunnerMetrics) ([]client.Object, error) {
	if !*c.Spec.CreateMemcached {
		return []client.Object{}, nil
	}
	ls := getLabels(c)
	annotations := map[string]string{
		CrdKey: getKey(c),
	}
	name := fmt.Sprintf("%s-cache", c.Spec.Name)
	ls["app"] = "memcached"

	var ss appsv1.StatefulSet
	err := yaml.Unmarshal([]byte(JsonMemcached), &ss)
	if err != nil {
		return nil, err
	}

	ss.Labels = ls
	ss.SetAnnotations(annotations)
	ss.Spec.Template.Labels = ls
	ss.Spec.Selector.MatchLabels = ls
	ss.Spec.Template.Spec.Affinity.PodAntiAffinity.PreferredDuringSchedulingIgnoredDuringExecution[0].PodAffinityTerm.LabelSelector.MatchLabels = ls
	ss.Spec.Template.Spec.Affinity.PodAntiAffinity.PreferredDuringSchedulingIgnoredDuringExecution[0].PodAffinityTerm.Namespaces = []string{c.Spec.Namespace}
	ss.Name = name
	ss.Spec.Template.Name = name
	ss.Spec.ServiceName = name
	ss.Namespace = c.Spec.Namespace
	ss.Spec.Template.Spec.Containers[0].Env[0].Value = "user"
	if c.Spec.MemcachedUser != nil {
		ss.Spec.Template.Spec.Containers[0].Env[0].Value = *c.Spec.MemcachedUser
	}
	ss.Spec.Template.Spec.Containers[0].Env[1].ValueFrom.SecretKeyRef.Name = name
	if c.Spec.ExistingMemcacheCredsSecret != "" {
		ss.Spec.Template.Spec.Containers[0].Env[1].ValueFrom.SecretKeyRef.Name = c.Spec.ExistingMemcacheCredsSecret
	}
	ss.Spec.Replicas = &c.Spec.MemcachedReplicas
	ss.Spec.Template.Spec.Containers[0].Image = c.Spec.MemcachedImage

	svc := v1.Service{
		ObjectMeta: metav1.ObjectMeta{Name: name,
			Namespace:   c.Spec.Namespace,
			Labels:      ls,
			Annotations: annotations,
		},
		Spec: v1.ServiceSpec{
			Ports: []v1.ServicePort{
				v1.ServicePort{
					Name:       "memcache",
					Port:       11211,
					TargetPort: intstr.FromString("memcache"),
				},
			},
			Selector: ls,
		},
	}
	svc.TypeMeta.SetGroupVersionKind(schema.FromAPIVersionAndKind("v1", "Service"))

	var resources []client.Object
	resources = append(resources, &ss, &svc)

	if c.Spec.ExistingMemcacheCredsSecret == "" {
		secret := v1.Secret{
			TypeMeta: metav1.TypeMeta{
				Kind:       "Secret",
				APIVersion: "apps/v1",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:        name,
				Namespace:   c.Spec.Namespace,
				Labels:      ls,
				Annotations: annotations,
			},
			StringData: map[string]string{
				"memcached-password": getPass(),
			},
		}
		secret.TypeMeta.SetGroupVersionKind(schema.FromAPIVersionAndKind("v1", "Secret"))
		resources = append(resources, &secret)
	}
	return resources, nil
}

func getPass() string {
	const (
		chars  = "1234567890qwertyuiopasdfghjklzxcvbnmQWERTYUIOPASDFGHJKLZXCVBNM"
		length = 32
	)
	pass := strings.Builder{}

	for i := 0; i < length; i++ {
		rand := rand.Intn(len(chars))
		newChar := chars[rand]
		pass.WriteByte(newChar)
	}
	return pass.String()
}

func generateExternalMetricsDeployment(c *runnerv1alpha1.ActionRunnerMetrics, ls map[string]string) *appsv1.Deployment {
	var dep appsv1.Deployment
	err := yaml.Unmarshal([]byte(JsonApiServer), &dep)
	if err != nil {
		fmt.Println(err)
	}
	dep.Name = c.Spec.Name
	dep.Spec.Template.Name = c.Spec.Name
	dep.Labels = ls
	dep.Spec.Template.Labels = ls
	dep.Spec.Selector.MatchLabels = ls

	dep.Namespace = c.Spec.Namespace
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
	if *c.Spec.CreateMemcached {
		for i := 0; i < int(c.Spec.MemcachedReplicas); i++ {
			mcServers = fmt.Sprintf("%s%s-cache-%d.%s-cache:11211", mcServers, c.Spec.Name, i, c.Spec.Name)
			if i+1 < int(c.Spec.MemcachedReplicas) {
				mcServers = fmt.Sprintf("%s;", mcServers)
			}
		}
	}
	if c.Spec.ExistingMemcacheServers != "" {
		mcServers = c.Spec.ExistingMemcacheServers
	}
	if mcServers != "" {
		args = append(args, fmt.Sprintf("--memcached-servers=%s", mcServers))
		if c.Spec.MemcachedUser != nil {
			args = append(args, fmt.Sprintf("--memcached-user=%s", *c.Spec.MemcachedUser))
		}
	}
	dep.Spec.Template.Spec.Containers[0].Args = append(dep.Spec.Template.Spec.Containers[0].Args, args...)
	dep.Spec.Template.Spec.ServiceAccountName = c.Spec.Name
	dep.Spec.Template.Spec.Volumes[1].Secret.SecretName = fmt.Sprintf("%s-cert", c.Spec.Name)
	if c.Spec.ExistingSslCertSecret != "" {
		dep.Spec.Template.Spec.Volumes[1].Secret.SecretName = c.Spec.ExistingSslCertSecret
	}

	dep.Spec.Template.Spec.Containers[0].Env[0].ValueFrom.SecretKeyRef.Name = fmt.Sprintf("%s-cache", c.Spec.Name)
	if c.Spec.ExistingMemcacheCredsSecret != "" {
		dep.Spec.Template.Spec.Containers[0].Env[0].ValueFrom.SecretKeyRef.Name = c.Spec.ExistingMemcacheCredsSecret
	} else if !*c.Spec.CreateMemcached {
		dep.Spec.Template.Spec.Containers[0].Env = []corev1.EnvVar{}
	}
	return &dep
}

func generateExternalMetricsRbac(c *runnerv1alpha1.ActionRunnerMetrics, ls map[string]string) ([]*rbac.ClusterRole, []*rbac.ClusterRoleBinding, []*rbac.Role, []*rbac.RoleBinding) {
	scaledactionrunnerViewer := rbac.ClusterRoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:   fmt.Sprintf("%s:operator-scaledactionrunner-viewer-role", c.Spec.Name),
			Labels: ls,
		},
		RoleRef: rbac.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "ClusterRole",
			Name:     "operator-scaledactionrunner-viewer-role",
		},
		Subjects: []rbac.Subject{
			{
				Kind:      "ServiceAccount",
				Name:      c.Spec.Name,
				Namespace: c.Spec.Namespace,
			}},
	}
	authDelegator := rbac.ClusterRoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:   fmt.Sprintf("%s:system:auth-delegator", c.Spec.Name),
			Labels: ls,
		},
		RoleRef: rbac.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "ClusterRole",
			Name:     "system:auth-delegator",
		},
		Subjects: []rbac.Subject{
			{
				Kind:      "ServiceAccount",
				Name:      c.Spec.Name,
				Namespace: c.Spec.Namespace,
			}},
	}

	apiserver := rbac.ClusterRoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:   fmt.Sprintf("%s:apiserver-clusterrolebinding", c.Spec.Name),
			Labels: ls,
		},
		RoleRef: rbac.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "ClusterRole",
			Name:     "aggregated-apiserver-clusterrole",
		},
		Subjects: []rbac.Subject{
			{
				Kind:      "ServiceAccount",
				Name:      c.Spec.Name,
				Namespace: c.Spec.Namespace,
			}},
	}

	authReader := rbac.RoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s:extension-apiserver-authentication-reader", c.Spec.Name),
			Labels:    ls,
			Namespace: "kube-system",
		},
		RoleRef: rbac.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "Role",
			Name:     "extension-apiserver-authentication-reader",
		},
		Subjects: []rbac.Subject{
			{
				Kind:      "ServiceAccount",
				Name:      c.Spec.Name,
				Namespace: c.Spec.Namespace,
			}},
	}
	aggApiserverClusterRole := rbac.ClusterRole{
		ObjectMeta: metav1.ObjectMeta{
			Name:   "aggregated-apiserver-clusterrole",
			Labels: ls,
		},
		Rules: []rbac.PolicyRule{
			rbac.PolicyRule{
				APIGroups: []string{""},
				Resources: []string{"namespaces"},
				Verbs:     []string{"get", "watch", "list"},
			},
			rbac.PolicyRule{
				APIGroups: []string{"admissionregistration.k8s.io"},
				Resources: []string{"mutatingwebhookconfigurations", "validatingwebhookconfigurations"},
				Verbs:     []string{"get", "watch", "list"},
			}},
	}
	aggApiserverClusterRole.TypeMeta.SetGroupVersionKind(schema.FromAPIVersionAndKind("rbac.authorization.k8s.io/v1", "ClusterRole"))
	authDelegator.TypeMeta.SetGroupVersionKind(schema.FromAPIVersionAndKind("rbac.authorization.k8s.io/v1", "ClusterRoleBinding"))
	apiserver.TypeMeta.SetGroupVersionKind(schema.FromAPIVersionAndKind("rbac.authorization.k8s.io/v1", "ClusterRoleBinding"))
	scaledactionrunnerViewer.TypeMeta.SetGroupVersionKind(schema.FromAPIVersionAndKind("rbac.authorization.k8s.io/v1", "ClusterRoleBinding"))
	authReader.TypeMeta.SetGroupVersionKind(schema.FromAPIVersionAndKind("rbac.authorization.k8s.io/v1", "RoleBinding"))
	return []*rbac.ClusterRole{&aggApiserverClusterRole}, []*rbac.ClusterRoleBinding{&authDelegator, &apiserver, &scaledactionrunnerViewer}, []*rbac.Role{}, []*rbac.RoleBinding{&authReader}
}

func GenerateAuthTrigger(c *runnerv1alpha1.ActionRunnerMetrics) []client.Object {
	if !*c.Spec.CreateAuthentication {
		return []client.Object{}
	}
	ls := getLabels(c)
	annotations := map[string]string{
		CrdKey: getKey(c),
	}
	certName := c.Spec.ExistingSslCertSecret
	if certName == "" {
		//TODO: make cert in kedanamespace
	}
	authTrigger := keda.ClusterTriggerAuthentication{
		ObjectMeta: metav1.ObjectMeta{
			Name:        c.Spec.Name,
			Namespace:   c.Spec.KedaNamespace,
			Labels:      ls,
			Annotations: annotations,
		},
		Spec: keda.TriggerAuthenticationSpec{
			SecretTargetRef: []keda.AuthSecretTargetRef{
				keda.AuthSecretTargetRef{
					Name:      certName,
					Key:       "cert",
					Parameter: "cert",
				},
				keda.AuthSecretTargetRef{
					Name:      certName,
					Key:       "ca",
					Parameter: "ca",
				},
				keda.AuthSecretTargetRef{
					Name:      certName,
					Key:       "key",
					Parameter: "key",
				},
			},
		},
	}
	authTrigger.TypeMeta.SetGroupVersionKind(schema.FromAPIVersionAndKind("keda.sh/v1alpha1", "ClusterTriggerAuthentication"))
	return []client.Object{&authTrigger}
}

func GenerateMetricsApiServer(c *runnerv1alpha1.ActionRunnerMetrics) []client.Object {
	if !*c.Spec.CreateApiServer {
		return []client.Object{}
	}
	ls := getLabels(c)
	ls["app"] = "external-metrics-apiserver"
	dep := generateExternalMetricsDeployment(c, ls)
	dep.TypeMeta.SetGroupVersionKind(schema.FromAPIVersionAndKind("apps/v1", "Deployment"))
	svc := v1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      c.Spec.Name,
			Namespace: c.Spec.Namespace,
			Labels:    ls,
		},
		Spec: v1.ServiceSpec{
			Ports: []v1.ServicePort{
				{
					Name:       "https",
					Protocol:   corev1.ProtocolTCP,
					Port:       443,
					TargetPort: intstr.FromInt(6443),
				},
				{
					Name:       "metrics",
					Protocol:   corev1.ProtocolTCP,
					Port:       2112,
					TargetPort: intstr.FromInt(2112),
				},
			},
			Selector: ls,
		},
	}
	svc.TypeMeta.SetGroupVersionKind(schema.FromAPIVersionAndKind("v1", "Service"))
	sa := v1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name:      c.Spec.Name,
			Namespace: c.Spec.Namespace,
			Labels:    ls,
		},
	}
	sa.TypeMeta.SetGroupVersionKind(schema.FromAPIVersionAndKind("v1", "ServiceAccount"))

	apiservice := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "apiregistration.k8s.io/v1",
			"kind":       "APIService",
			"metadata": map[string]interface{}{
				"name": "v1beta1.custom.metrics.k8s.io",
			},
			"spec": map[string]interface{}{
				"insecureSkipTLSVerify": true,
				"group":                 "custom.metrics.k8s.io",
				"groupPriorityMinimum":  100,
				"versionPriority":       100,
				"service": map[string]interface{}{
					"name":      c.Spec.Name,
					"namespace": c.Spec.Namespace,
				},
				"version": "v1beta1",
			},
		},
	}
	cr, crb, r, rb := generateExternalMetricsRbac(c, ls)
	output := setKey(c, dep, &svc, &sa, cr, crb, r, rb, apiservice)
	return output
}

func getKey(c *runnerv1alpha1.ActionRunnerMetrics) string {
	bin := "unknown"
	if binPath, err := os.Readlink("/proc/self/exe"); err == nil {
		if stats, err := os.Stat(binPath); err == nil {
			bin = fmt.Sprintf("%d", stats.ModTime().Unix())
		}
	}

	j, _ := json.Marshal(c)
	b := sha1.Sum(j)
	return fmt.Sprintf("%s_%s_%s/%s%s", bin, base64.RawStdEncoding.EncodeToString(b[:]), c.Spec.Namespace, c.Spec.Name, c.ResourceVersion)
}
func setKey(c *runnerv1alpha1.ActionRunnerMetrics, dep *appsv1.Deployment, svc *v1.Service, sa *v1.ServiceAccount, cr []*rbac.ClusterRole, crb []*rbac.ClusterRoleBinding, r []*rbac.Role, rb []*rbac.RoleBinding, as *unstructured.Unstructured) []client.Object {
	key := getKey(c)
	process := func(o client.Object) client.Object {
		anns := o.GetAnnotations()
		if anns == nil {
			anns = map[string]string{}
		}
		anns[CrdKey] = key
		o.SetAnnotations(anns)
		return o
	}
	var out []client.Object
	out = append(out, process(dep))
	out = append(out, process(svc))
	out = append(out, process(sa))
	for _, o := range cr {
		out = append(out, process(o))
	}
	for _, o := range crb {
		out = append(out, process(o))
	}
	for _, o := range r {
		out = append(out, process(o))
	}
	for _, o := range rb {
		out = append(out, process(o))
	}
	out = append(out, process(as))
	return out
}

const JsonMemcached = `{
	"apiVersion": "apps/v1",
	"kind": "StatefulSet",
	"metadata": {
	  "name": "replaced-memcached",
	  "namespace": "replacedns"
	},
	"spec": {
	  "selector": {
		"matchLabels": {
		}
	  },
	  "replicas": 2,
	  "serviceName": "replaced-memcached",
	  "template": {
		"spec": {
		  "affinity": {
			"podAffinity": null,
			"podAntiAffinity": {
			  "preferredDuringSchedulingIgnoredDuringExecution": [
				{
				  "podAffinityTerm": {
					"labelSelector": {
					},
					"namespaces": [
					  "replacedns"
					],
					"topologyKey": "kubernetes.io/hostname"
				  },
				  "weight": 1
				}
			  ]
			},
			"nodeAffinity": null
		  },
		  "securityContext": {
			"fsGroup": 1001,
			"runAsUser": 1001
		  },
		  "containers": [
			{
			  "name": "memcached",
			  "image": "docker.io/bitnami/memcached:latest",
			  "imagePullPolicy": "IfNotPresent",
			  "args": [
				"/run.sh"
			  ],
			  "env": [
				{
				  "name": "MEMCACHED_USERNAME",
				  "value": "user"
				},
				{
				  "name": "MEMCACHED_PASSWORD",
				  "valueFrom": {
					"secretKeyRef": {
					  "name": "replaced-memcached",
					  "key": "memcached-password"
					}
				  }
				}
			  ],
			  "ports": [
				{
				  "name": "memcache",
				  "containerPort": 11211
				}
			  ],
			  "livenessProbe": {
				"tcpSocket": {
				  "port": "memcache"
				},
				"initialDelaySeconds": 30,
				"timeoutSeconds": 5,
				"failureThreshold": 6
			  },
			  "readinessProbe": {
				"tcpSocket": {
				  "port": "memcache"
				},
				"initialDelaySeconds": 5,
				"timeoutSeconds": 3,
				"periodSeconds": 5
			  },
			  "resources": {
				"limits": {},
				"requests": {
				  "cpu": "250m",
				  "memory": "256Mi"
				}
			  },
			  "securityContext": {
				"readOnlyRootFilesystem": false
			  },
			  "volumeMounts": [
				{
				  "name": "tmp",
				  "mountPath": "/tmp"
				}
			  ]
			}
		  ],
		  "volumes": [
			{
			  "name": "tmp",
			  "emptyDir": {}
			}
		  ]
		}
	  }
	}
  }`

const JsonApiServer = `{
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
								"name": "MEMCACHED_PASSWORD",
								"valueFrom": {
									"secretKeyRef": {
										"name": "memcache",
										"key": "memcached-password"
									}
								}
							}
						],
						"resources": {
							"requests": {
								"cpu": "200m",
								"memory": "50Mi"
							  },
							"limits": {
								"cpu": "300m",
								"memory": "200Mi"
							}
						},
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
