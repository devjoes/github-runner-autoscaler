package controllers

import (
	"context"
	"strings"
	"time"

	runnerv1alpha1 "github.com/devjoes/github-runner-autoscaler/operator/api/v1alpha1"
	"github.com/devjoes/github-runner-autoscaler/operator/generators"
	keda "github.com/kedacore/keda/v2/api/v1alpha1"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

//todo: poss get rid of spec.namespace since everything has to be in the same namespace.
//TODO: poss just use meta.name. or splice spec.name up
const (
	testName            = "test-name"
	testNamespace       = "test-namespace"
	testRunnerName      = "test-runner-name"
	testRunnerNamespace = "test-runner-namespace"
	testMaxRunners      = 2
	//TODO: make 1.
	testMinRunners        = 0
	testRunnerSecrets     = "foo,bar"
	testGithubTokenSecret = "github"
	testOwner             = "testCorp"
	testRepo              = "test"
	testImage             = "foo"
	testRunnerLabels      = "foo,bar"

	testWorkVolumeSizeGigs = 123
)

var _ = Describe("CronJob controller", func() {

	// Define utility constants for object names and testing timeouts/durations and intervals.

	Context("ScaledActionRunner CRD", func() {
		ctx := context.Background()
		var sar *runnerv1alpha1.ScaledActionRunner
		It("Should set it up dependant StatefulSet and ScaledObject when created", func() {
			Expect(k8sClient.Create(ctx, &corev1.Namespace{ObjectMeta: v1.ObjectMeta{Name: testNamespace}})).Should(Succeed())
			Expect(k8sClient.Create(ctx, &corev1.Namespace{ObjectMeta: v1.ObjectMeta{Name: testRunnerNamespace}})).Should(Succeed())
			sar, _ = createTestScaledActionRunner(ctx, k8sClient)

			testResults(ctx, func(ss *appsv1.StatefulSet, so *keda.ScaledObject) bool {
				if ss == nil || so == nil {
					return false
				}
				Expect(ss.ObjectMeta.Name).To(Equal(testRunnerName))
				Expect(ss.ObjectMeta.Namespace).To(Equal(testRunnerNamespace))
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
				Expect(so.Spec.ScaleTargetRef.Name).To(Equal(ss.Name))
				Expect(so.Spec.Triggers[0].Type).To(Equal("metrics-api"))
				Expect(strings.HasSuffix(so.Spec.Triggers[0].Metadata["url"], testRunnerNamespace+"/"+testRunnerName)).To(BeTrue())
				return true
			})
		})
		//TODO: More tests
		nsName := types.NamespacedName{Namespace: testRunnerNamespace, Name: testRunnerName}
		It("Should delete resources when deleted", func() {
			Expect(sar).ToNot(BeNil())
			k8sClient.Delete(ctx, sar)
			Eventually(func() bool {
				return k8sClient.Get(ctx, nsName, &appsv1.StatefulSet{}) == nil &&
					k8sClient.Get(ctx, nsName, &keda.ScaledObject{}) == nil
			}, time.Hour, time.Second).Should(BeTrue())
		})

	})
})

func testResults(ctx context.Context, test func(*appsv1.StatefulSet, *keda.ScaledObject) bool) {
	Eventually(func() bool {
		sSet := appsv1.StatefulSet{}
		sObj := keda.ScaledObject{}
		nsName := types.NamespacedName{Namespace: testRunnerNamespace, Name: testRunnerName}
		e1 := k8sClient.Get(ctx, nsName, &sSet)
		e2 := k8sClient.Get(ctx, nsName, &sObj)
		if e1 != nil || e2 != nil {
			return false
		}
		return test(&sSet, &sObj)
	}, time.Hour, time.Second).Should(BeTrue())
}

func createTestScaledActionRunner(ctx context.Context, k8sClient client.Client) (*runnerv1alpha1.ScaledActionRunner, []corev1.Secret) {
	runner := runnerv1alpha1.ScaledActionRunner{
		ObjectMeta: v1.ObjectMeta{
			Name:      testName,
			Namespace: testNamespace,
		},
		Spec: runnerv1alpha1.ScaledActionRunnerSpec{
			Name:              testRunnerName,
			Namespace:         testRunnerNamespace,
			MaxRunners:        testMaxRunners,
			MinRunners:        testMinRunners,
			RunnerSecrets:     strings.Split(testRunnerSecrets, ","),
			GithubTokenSecret: testGithubTokenSecret,
			Owner:             testOwner,
			Repo:              testRepo,
			Image:             testImage,
			RunnerLabels:      testRunnerLabels,
			WorkVolumeSize:    resource.NewScaledQuantity(testWorkVolumeSizeGigs, resource.Giga),
		},
	}
	Expect(k8sClient.Create(ctx, &runner)).Should(Succeed())
	var secrets []corev1.Secret
	for _, s := range append(strings.Split(testRunnerSecrets, ","), testGithubTokenSecret) {
		sec := corev1.Secret{
			ObjectMeta: v1.ObjectMeta{
				Namespace:   testRunnerNamespace,
				Name:        s,
				Annotations: make(map[string]string)},
			Data: map[string][]byte{"name": []byte(s)},
		}
		Expect(k8sClient.Create(ctx, &sec)).Should(Succeed())
		secrets = append(secrets, sec)
	}
	return &runner, secrets
}
