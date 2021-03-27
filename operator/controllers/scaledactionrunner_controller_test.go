package controllers

import (
	"context"
	"fmt"
	"strings"
	"time"

	runnerv1alpha1 "github.com/devjoes/github-runner-autoscaler/operator/api/v1alpha1"
	"github.com/devjoes/github-runner-autoscaler/operator/generators"
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

//todo: poss get rid of spec.namespace since everything has to be in the same namespace.
//TODO: poss just use meta.name. or splice spec.name up
const (
	testName                = "test-name"
	testNamespace           = "test-namespace"
	testMaxRunners          = 2
	testMinRunners          = 0
	testRunnerSecrets       = "foo,bar"
	testGithubTokenSecret   = "github"
	testOwner               = "testCorp"
	testRepo                = "test"
	testImage               = "foo"
	testRunnerLabels        = "foo,bar"
	testPollingInterval     = int32(30)
	testStabalizationWindow = int32(60)

	testWorkVolumeSizeGigs = "123G"
)

var _ = Describe("ScaledActionRunner controller", func() {
	Context("ScaledActionRunner CRD", func() {
		ctx := context.Background()
		var sar *runnerv1alpha1.ScaledActionRunner
		var ssVer, soVer string
		It("Should set up dependant StatefulSet and ScaledObject when created", func() {
			Expect(k8sClient.Create(ctx, &corev1.Namespace{ObjectMeta: v1.ObjectMeta{Name: testNamespace}})).Should(Succeed())

			sar, _ = createTestScaledActionRunner(ctx, k8sClient)

			testResults(ctx, func(ss *appsv1.StatefulSet, so *keda.ScaledObject) bool {
				if ss == nil || so == nil {
					return false
				}
				ssVer = ss.ObjectMeta.ResourceVersion
				soVer = so.ObjectMeta.ResourceVersion
				Expect(ss.ObjectMeta.Name).To(Equal(testName))
				Expect(ss.ObjectMeta.Namespace).To(Equal(testNamespace))
				vols := ss.Spec.Template.Spec.Volumes
				Expect(len(vols)).To(Equal(len(strings.Split(testRunnerSecrets, ",")) + 1))
				var secs []string
				for _, v := range vols {
					if v.Secret != nil {
						secs = append(secs, v.Secret.SecretName)
					}
				}
				Expect(secs).To(ConsistOf(append(strings.Split(testRunnerSecrets, ","))))
				Expect(ss.Spec.Template.Spec.Containers[0].Env).To(ContainElement(corev1.EnvVar{Name: "LABELS", Value: testRunnerLabels}))
				Expect(ss.Annotations[generators.AnnotationSecretsHash]).NotTo(BeNil())
				Expect(*so.Spec.MaxReplicaCount).To(BeEquivalentTo(testMaxRunners))
				Expect(*so.Spec.MinReplicaCount).To(BeEquivalentTo(testMinRunners))
				Expect(*so.Spec.PollingInterval).To(BeEquivalentTo(testPollingInterval))
				Expect(*so.Spec.Advanced.HorizontalPodAutoscalerConfig.Behavior.ScaleUp.StabilizationWindowSeconds).To(BeEquivalentTo(testStabalizationWindow))
				Expect(so.Spec.ScaleTargetRef.Name).To(Equal(ss.Name))
				Expect(so.Spec.Triggers[0].Type).To(Equal("metrics-api"))
				Expect(strings.HasSuffix(so.Spec.Triggers[0].Metadata["url"], testNamespace+"/"+testName)).To(BeTrue())
				return true
			})
		})
		time.Sleep(10 * time.Second)
		It("Should update resources when updated", func() {
			updateTestScaledActionRunner(context.TODO(), k8sClient, sar.ResourceVersion)
			testResults(ctx, func(ss *appsv1.StatefulSet, so *keda.ScaledObject) bool {
				if ss == nil || so == nil {
					return false
				}
				fmt.Println(ss.ObjectMeta.ResourceVersion)
				if ss.ObjectMeta.ResourceVersion == ssVer || so.ObjectMeta.ResourceVersion == soVer {
					return false
				}
				Expect(ss.ObjectMeta.Name).To(Equal(testName))
				Expect(ss.ObjectMeta.Namespace).To(Equal(testNamespace))
				vols := ss.Spec.Template.Spec.Volumes
				//Expect(len(vols)).To(Equal(len(strings.Split(testRunnerSecrets, ",")) + 1))
				var secs []string
				for _, v := range vols {
					if v.Secret != nil {
						secs = append(secs, v.Secret.SecretName)
					}
				}
				Expect(secs).To(Not(ConsistOf(append(strings.Split(testRunnerSecrets, ",")))))
				Expect(ss.Spec.Template.Spec.Containers[0].Env).To(Not(ContainElement(corev1.EnvVar{Name: "LABELS", Value: testRunnerLabels})))
				Expect(ss.Annotations[generators.AnnotationSecretsHash]).NotTo(BeNil())
				Expect(*so.Spec.MaxReplicaCount).To(Not(BeEquivalentTo(testMaxRunners)))
				Expect(*so.Spec.MinReplicaCount).To(Not(BeEquivalentTo(testMinRunners)))
				Expect(*so.Spec.PollingInterval).To(Not(BeEquivalentTo(testPollingInterval)))
				Expect(*so.Spec.Advanced.HorizontalPodAutoscalerConfig.Behavior.ScaleUp.StabilizationWindowSeconds).To(Not(BeEquivalentTo(testStabalizationWindow)))
				Expect(so.Spec.ScaleTargetRef.Name).To(Equal(ss.Name))
				Expect(so.Spec.Triggers[0].Type).To(Equal("metrics-api"))
				Expect(strings.HasSuffix(so.Spec.Triggers[0].Metadata["url"], testNamespace+"/"+testName)).To(BeTrue())
				return true
			})
		})
		time.Sleep(10 * time.Second)
		nsName := types.NamespacedName{Namespace: testNamespace, Name: testName}
		It("Should delete resources when deleted", func() {
			Expect(sar).ToNot(BeNil())
			k8sClient.Delete(ctx, sar)
			Eventually(func() bool {
				return k8sClient.Get(ctx, nsName, &appsv1.StatefulSet{}) == nil &&
					k8sClient.Get(ctx, nsName, &keda.ScaledObject{}) == nil
			}, time.Minute, time.Second).Should(BeTrue())
		})

	})
})

func testResults(ctx context.Context, test func(*appsv1.StatefulSet, *keda.ScaledObject) bool) {
	Eventually(func() bool {
		sSet := appsv1.StatefulSet{}
		sObj := keda.ScaledObject{}
		nsName := types.NamespacedName{Namespace: testNamespace, Name: testName}
		e1 := k8sClient.Get(ctx, nsName, &sSet)
		e2 := k8sClient.Get(ctx, nsName, &sObj)
		if e1 != nil || e2 != nil {
			return false
		}
		return test(&sSet, &sObj)
	}, time.Minute, time.Second).Should(BeTrue())
}

func createTestScaledActionRunner(ctx context.Context, k8sClient client.Client) (*runnerv1alpha1.ScaledActionRunner, []corev1.Secret) {
	var pollingInterval int32 = testPollingInterval
	var stabalizationWindow int32 = testStabalizationWindow
	var filesystem corev1.PersistentVolumeMode = "Filesystem"
	runner := runnerv1alpha1.ScaledActionRunner{
		ObjectMeta: v1.ObjectMeta{
			Name:      testName,
			Namespace: testNamespace,
		},
		Spec: runnerv1alpha1.ScaledActionRunnerSpec{
			MaxRunners:        testMaxRunners,
			MinRunners:        testMinRunners,
			RunnerSecrets:     strings.Split(testRunnerSecrets, ","),
			GithubTokenSecret: testGithubTokenSecret,
			Owner:             testOwner,
			Repo:              testRepo,
			Runner: &runnerv1alpha1.Runner{
				Image:        testImage,
				RunnerLabels: testRunnerLabels,
				WorkVolumeClaimTemplate: &corev1.PersistentVolumeClaimSpec{
					AccessModes: []corev1.PersistentVolumeAccessMode{corev1.ReadWriteMany},
					VolumeMode:  &filesystem,
					Resources: corev1.ResourceRequirements{
						Requests: corev1.ResourceList{
							corev1.ResourceStorage: resource.MustParse(testWorkVolumeSizeGigs),
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
	for _, s := range append(strings.Split(testRunnerSecrets, ","), testGithubTokenSecret) {
		sec := corev1.Secret{
			ObjectMeta: v1.ObjectMeta{
				Namespace:   testNamespace,
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
	var pollingInterval int32 = testPollingInterval * 10
	var stabalizationWindow int32 = testStabalizationWindow * 10
	var filesystem corev1.PersistentVolumeMode = "Filesystem"
	runner := runnerv1alpha1.ScaledActionRunner{
		ObjectMeta: v1.ObjectMeta{
			Name:      testName,
			Namespace: testNamespace,
		},
		Spec: runnerv1alpha1.ScaledActionRunnerSpec{
			MaxRunners:        1,
			MinRunners:        1,
			RunnerSecrets:     strings.Split(reverse(testRunnerSecrets), ","),
			GithubTokenSecret: reverse(testGithubTokenSecret),
			Owner:             reverse(testOwner),
			Repo:              reverse(testRepo),
			Runner: &runnerv1alpha1.Runner{
				Image:        reverse(testImage),
				RunnerLabels: reverse(testRunnerLabels),
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
	for _, s := range append(strings.Split(testRunnerSecrets, ","), testGithubTokenSecret) {
		sec := corev1.Secret{
			ObjectMeta: v1.ObjectMeta{
				Namespace:   testNamespace,
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
