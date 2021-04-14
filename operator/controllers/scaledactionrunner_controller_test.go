package controllers

import (
	"context"
	"strings"
	"time"

	runnerv1alpha1 "github.com/devjoes/github-runner-autoscaler/operator/api/v1alpha1"
	generators "github.com/devjoes/github-runner-autoscaler/operator/sargenerator"
	keda "github.com/kedacore/keda/v2/api/v1alpha1"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/api/autoscaling/v2beta2"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	testTimeoutSecs          = 60
	testSarName              = "test-name"
	testSarImage             = "foo"
	testSarNamespace         = "test-sar-namespace"
	testSarMaxRunners        = 2
	testSarMinRunners        = 0
	testSarRunnerSecrets     = "foo,bar"
	testSarGithubTokenSecret = "github"
	testSarOwner             = "testCorp"
	testSarRepo              = "test"

	testSarRunnerLabels            = "foo,bar"
	testSarRequests                = "123"
	testSarLimits                  = "234"
	testSarPollingIntervalSecs     = int32(30)
	testSarStabalizationWindowSecs = int32(60)

	testSarWorkVolumeSizeGigs = "123G"
)

var annotations map[string]string = map[string]string{
	"OverrideMetricsName":      "testname",
	"OverrideMetricsNamespace": "testns",
}

var _ = Describe("ScaledActionRunner controller", func() {
	Context("ScaledActionRunner CRD", func() {
		ctx := context.Background()
		var sar *runnerv1alpha1.ScaledActionRunner
		var ssVer, soVer string
		It("Should set up dependant StatefulSet and ScaledObject when created", func() {
			Expect(k8sClient.Create(ctx, &corev1.Namespace{ObjectMeta: v1.ObjectMeta{Name: testSarNamespace}})).Should(Succeed())

			sar, _ = createTestScaledActionRunner(ctx, k8sClient)

			testSarResults(ctx, func(ss *appsv1.StatefulSet, so *keda.ScaledObject) bool {
				if ss == nil || so == nil {
					return false
				}
				fooBar := make(map[string]string)
				fooBar["foo"] = "bar"
				ssVer = ss.ObjectMeta.ResourceVersion
				soVer = so.ObjectMeta.ResourceVersion
				Expect(ss.ObjectMeta.Name).To(Equal(testSarName))
				Expect(ss.ObjectMeta.Namespace).To(Equal(testSarNamespace))
				vols := ss.Spec.Template.Spec.Volumes
				Expect(len(vols)).To(Equal(len(strings.Split(testSarRunnerSecrets, ",")) + 1))
				var secs []string
				for _, v := range vols {
					if v.Secret != nil {
						secs = append(secs, v.Secret.SecretName)
					}
				}
				Expect(secs).To(ConsistOf(append(strings.Split(testSarRunnerSecrets, ","))))
				Expect(ss.Spec.Template.Spec.Containers[0].Env).To(ContainElement(corev1.EnvVar{Name: "LABELS", Value: testSarRunnerLabels}))
				Expect(ss.Spec.Template.Spec.Containers[0].Resources.Requests.Cpu().String()).To((Equal(testSarRequests + "m")))
				Expect(ss.Spec.Template.Spec.Containers[0].Resources.Requests.Memory().String()).To((Equal(testSarRequests + "Mi")))
				Expect(ss.Spec.Template.Spec.Containers[0].Resources.Limits.Cpu().String()).To((Equal(testSarLimits + "m")))
				Expect(ss.Spec.Template.Spec.Containers[0].Resources.Limits.Memory().String()).To((Equal(testSarLimits + "Mi")))
				Expect(ss.Spec.Template.Annotations).To(Equal(fooBar))
				Expect(ss.Spec.Template.Spec.NodeSelector).To(Equal(fooBar))
				Expect(ss.Annotations[generators.AnnotationSecretsHash]).NotTo(BeNil())
				Expect(*so.Spec.MaxReplicaCount).To(BeEquivalentTo(testSarMaxRunners))
				Expect(*so.Spec.MinReplicaCount).To(BeEquivalentTo(testSarMinRunners))
				Expect(*so.Spec.PollingInterval).To(BeEquivalentTo(testSarPollingIntervalSecs))
				Expect(*so.Spec.Advanced.HorizontalPodAutoscalerConfig.Behavior.ScaleUp.StabilizationWindowSeconds).To(BeEquivalentTo(testSarStabalizationWindowSecs))
				Expect(so.Spec.ScaleTargetRef.Name).To(Equal(ss.Name))
				Expect(so.Spec.Triggers[0].Type).To(Equal("metrics-api"))
				Expect(strings.Contains(so.Spec.Triggers[0].Metadata["url"], testSarNamespace+"/Scaledactionrunners/"+testSarName+"/*")).To(BeTrue())
				return true
			})
		})
		time.Sleep(10 * time.Second)
		It("Should update resources when updated", func() {
			updateTestScaledActionRunner(context.TODO(), k8sClient, sar.ResourceVersion)
			testSarResults(ctx, func(ss *appsv1.StatefulSet, so *keda.ScaledObject) bool {
				if ss == nil || so == nil {
					return false
				}
				if ss.ObjectMeta.ResourceVersion == ssVer || so.ObjectMeta.ResourceVersion == soVer {
					return false
				}
				Expect(ss.ObjectMeta.Name).To(Equal(testSarName))
				Expect(ss.ObjectMeta.Namespace).To(Equal(testSarNamespace))
				vols := ss.Spec.Template.Spec.Volumes
				//Expect(len(vols)).To(Equal(len(strings.Split(testRunnerSecrets, ",")) + 1))
				var secs []string
				for _, v := range vols {
					if v.Secret != nil {
						secs = append(secs, v.Secret.SecretName)
					}
				}
				Expect(secs).To(Not(ConsistOf(append(strings.Split(testSarRunnerSecrets, ",")))))
				Expect(ss.Spec.Template.Spec.Containers[0].Env).To(Not(ContainElement(corev1.EnvVar{Name: "LABELS", Value: testSarRunnerLabels})))
				Expect(ss.Spec.Template.Spec.Containers[0].Resources.Requests.Cpu().String()).To(Not(Equal(testSarRequests + "m")))
				Expect(ss.Spec.Template.Spec.Containers[0].Resources.Requests.Memory().String()).To(Not(Equal(testSarRequests + "Mi")))
				Expect(ss.Spec.Template.Spec.Containers[0].Resources.Limits.Cpu().String()).To(Not(Equal(testSarLimits + "m")))
				Expect(ss.Spec.Template.Spec.Containers[0].Resources.Limits.Memory().String()).To(Not(Equal(testSarLimits + "Mi")))
				Expect(ss.Annotations[generators.AnnotationSecretsHash]).NotTo(BeNil())
				Expect(*so.Spec.MaxReplicaCount).To(Not(BeEquivalentTo(testSarMaxRunners)))
				Expect(*so.Spec.MinReplicaCount).To(Not(BeEquivalentTo(testSarMinRunners)))
				Expect(*so.Spec.PollingInterval).To(Not(BeEquivalentTo(testSarPollingIntervalSecs)))
				Expect(*so.Spec.Advanced.HorizontalPodAutoscalerConfig.Behavior.ScaleUp.StabilizationWindowSeconds).To(Not(BeEquivalentTo(testSarStabalizationWindowSecs)))
				Expect(so.Spec.ScaleTargetRef.Name).To(Equal(ss.Name))
				Expect(so.Spec.Triggers[0].Type).To(Equal("metrics-api"))
				Expect(strings.Contains(so.Spec.Triggers[0].Metadata["url"], testSarNamespace+"/Scaledactionrunners/"+testSarName+"/*")).To(BeTrue())
				return true
			})
		})
		time.Sleep(10 * time.Second)
		nsName := types.NamespacedName{Namespace: testSarNamespace, Name: testSarName}
		It("Should delete resources when deleted", func() {
			Expect(sar).ToNot(BeNil())
			k8sClient.Delete(ctx, sar)
			Eventually(func() bool {
				return k8sClient.Get(ctx, nsName, &appsv1.StatefulSet{}) == nil &&
					k8sClient.Get(ctx, nsName, &keda.ScaledObject{}) == nil
			}, testTimeoutSecs*time.Second, time.Second).Should(BeTrue())
		})

	})
})

func testSarResults(ctx context.Context, test func(*appsv1.StatefulSet, *keda.ScaledObject) bool) {
	Eventually(func() bool {
		sSet := appsv1.StatefulSet{}
		sObj := keda.ScaledObject{}
		nsName := types.NamespacedName{Namespace: testSarNamespace, Name: testSarName}
		e1 := k8sClient.Get(ctx, nsName, &sSet)
		e2 := k8sClient.Get(ctx, nsName, &sObj)
		if e1 != nil || e2 != nil {
			return false
		}
		return test(&sSet, &sObj)
	}, testTimeoutSecs*time.Second, time.Second).Should(BeTrue())
}

func createTestScaledActionRunner(ctx context.Context, k8sClient client.Client) (*runnerv1alpha1.ScaledActionRunner, []corev1.Secret) {
	fooBar := make(map[string]string)
	fooBar["foo"] = "bar"
	var pollingInterval int32 = testSarPollingIntervalSecs
	var stabalizationWindow int32 = testSarStabalizationWindowSecs
	var filesystem corev1.PersistentVolumeMode = "Filesystem"
	runner := runnerv1alpha1.ScaledActionRunner{
		ObjectMeta: v1.ObjectMeta{
			Name:        testSarName,
			Namespace:   testSarNamespace,
			Annotations: annotations,
		},
		Spec: runnerv1alpha1.ScaledActionRunnerSpec{
			MaxRunners:        testSarMaxRunners,
			MinRunners:        testSarMinRunners,
			RunnerSecrets:     strings.Split(testSarRunnerSecrets, ","),
			GithubTokenSecret: testSarGithubTokenSecret,
			Owner:             testSarOwner,
			Repo:              testSarRepo,
			Runner: &runnerv1alpha1.Runner{
				Image:        testSarImage,
				RunnerLabels: testSarRunnerLabels,
				Annotations:  fooBar,
				NodeSelector: fooBar,
				Requests: &map[corev1.ResourceName]resource.Quantity{
					corev1.ResourceCPU:    resource.MustParse(testSarRequests + "m"),
					corev1.ResourceMemory: resource.MustParse(testSarRequests + "Mi"),
				},
				Limits: &map[corev1.ResourceName]resource.Quantity{
					corev1.ResourceCPU:    resource.MustParse(testSarLimits + "m"),
					corev1.ResourceMemory: resource.MustParse(testSarLimits + "Mi"),
				},
				WorkVolumeClaimTemplate: &corev1.PersistentVolumeClaimSpec{
					AccessModes: []corev1.PersistentVolumeAccessMode{corev1.ReadWriteMany},
					VolumeMode:  &filesystem,
					Resources: corev1.ResourceRequirements{
						Requests: corev1.ResourceList{
							corev1.ResourceStorage: resource.MustParse(testSarWorkVolumeSizeGigs),
						},
					},
				},
			},
			Scaling: &runnerv1alpha1.Scaling{
				PollingInterval: &pollingInterval,
				Behavior: &v2beta2.HorizontalPodAutoscalerBehavior{
					ScaleUp: &v2beta2.HPAScalingRules{
						StabilizationWindowSeconds: &stabalizationWindow,
					},
				},
			},
		},
	}

	Expect(k8sClient.Create(ctx, &runner)).Should(Succeed())
	var secrets []corev1.Secret
	for _, s := range append(strings.Split(testSarRunnerSecrets, ","), testSarGithubTokenSecret) {
		sec := corev1.Secret{
			TypeMeta: v1.TypeMeta{
				Kind:       "Secret",
				APIVersion: "apps/v1",
			},
			ObjectMeta: v1.ObjectMeta{
				Namespace:   testSarNamespace,
				Name:        s,
				Annotations: make(map[string]string)},
			Data: map[string][]byte{"name": []byte(s)},
		}
		Expect(k8sClient.Create(ctx, &sec)).Should(Succeed())
		secrets = append(secrets, sec)
	}
	return &runner, secrets
}

func reverse(s string) (result string) {
	for _, v := range s {
		result = string(v) + result
	}
	return
}
func updateTestScaledActionRunner(ctx context.Context, k8sClient client.Client, resourceVersion string) *runnerv1alpha1.ScaledActionRunner {
	var pollingInterval int32 = testSarPollingIntervalSecs * 10
	var stabalizationWindow int32 = testSarStabalizationWindowSecs * 10
	var filesystem corev1.PersistentVolumeMode = "Filesystem"
	runner := runnerv1alpha1.ScaledActionRunner{
		ObjectMeta: v1.ObjectMeta{
			Name:        testSarName,
			Namespace:   testSarNamespace,
			Annotations: annotations,
		},
		Spec: runnerv1alpha1.ScaledActionRunnerSpec{
			MaxRunners:        1,
			MinRunners:        1,
			RunnerSecrets:     strings.Split(reverse(testSarRunnerSecrets), ","),
			GithubTokenSecret: reverse(testSarGithubTokenSecret),
			Owner:             reverse(testSarOwner),
			Repo:              reverse(testSarRepo),
			Runner: &runnerv1alpha1.Runner{
				Image:        reverse(testSarImage),
				RunnerLabels: reverse(testSarRunnerLabels),
				Requests: &map[corev1.ResourceName]resource.Quantity{
					corev1.ResourceCPU:    resource.MustParse(reverse(testSarRequests) + "m"),
					corev1.ResourceMemory: resource.MustParse(reverse(testSarRequests) + "Mi"),
				},
				Limits: &map[corev1.ResourceName]resource.Quantity{
					corev1.ResourceCPU:    resource.MustParse(reverse(testSarLimits) + "m"),
					corev1.ResourceMemory: resource.MustParse(reverse(testSarLimits) + "Mi"),
				},
				WorkVolumeClaimTemplate: &corev1.PersistentVolumeClaimSpec{
					AccessModes: []corev1.PersistentVolumeAccessMode{corev1.ReadWriteMany},
					VolumeMode:  &filesystem,
					Resources: corev1.ResourceRequirements{
						Requests: corev1.ResourceList{
							corev1.ResourceStorage: resource.MustParse("1Gi"),
						},
					},
				},
			},
			Scaling: &runnerv1alpha1.Scaling{
				PollingInterval: &pollingInterval,
				Behavior: &v2beta2.HorizontalPodAutoscalerBehavior{
					ScaleUp: &v2beta2.HPAScalingRules{
						StabilizationWindowSeconds: &stabalizationWindow,
					},
				},
			},
		},
	}
	for _, s := range append(strings.Split(testSarRunnerSecrets, ","), testSarGithubTokenSecret) {
		sec := corev1.Secret{
			TypeMeta: v1.TypeMeta{
				Kind:       "Secret",
				APIVersion: "apps/v1",
			},
			ObjectMeta: v1.ObjectMeta{
				Namespace:   testSarNamespace,
				Name:        reverse(s),
				Annotations: make(map[string]string)},
			Data: map[string][]byte{"name": []byte(s)},
		}
		Expect(k8sClient.Create(ctx, &sec)).Should(Succeed())
	}

	runner.ResourceVersion = resourceVersion
	Expect(k8sClient.Update(ctx, &runner)).Should(Succeed())
	return &runner
}
