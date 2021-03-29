package controllers

import (
	"context"
	"fmt"
	"strings"
	"time"

	runnerv1alpha1 "github.com/devjoes/github-runner-autoscaler/operator/api/v1alpha1"
	keda "github.com/kedacore/keda/v2/api/v1alpha1"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	testArmName                     = "test-name"
	testArmNamespace                = "test-arm-namespace"
	testArmReplicas                 = 3
	testArmCacheWindowSecs          = 123
	testArmCacheWindowWhenEmptySecs = 234
	testArmResyncIntervalSecs       = 345
	testArmUseExistingSslCertSecret = ""
	testArmNamespaces               = ""
)

var _ = Describe("ActionRunnerMetrics controller", func() {
	Context("ActionRunnerMetrics CRD", func() {
		ctx := context.Background()
		var arm *runnerv1alpha1.ActionRunnerMetrics
		It("Should set up dependant Deployment, Memcached and ClusterTriggerAuthentication when created", func() {
			Expect(k8sClient.Create(ctx, &corev1.Namespace{ObjectMeta: v1.ObjectMeta{Name: testArmNamespace}})).Should(Succeed())
			arm, _ = createTestActionRunnerMetrics(ctx, k8sClient, true, true, true)
			testArmResults(ctx, []bool{true, true, true}, func(dep, mc *appsv1.Deployment, auth *keda.ClusterTriggerAuthentication) bool {
				Expect(dep.Name).To(Equal(arm.ObjectMeta.Name))
				Expect(mc.Name).To(Equal(arm.ObjectMeta.Name + "-cache"))
				Expect(auth.Name).To(Equal(arm.ObjectMeta.Name))
				Expect(dep.Namespace).To(Equal(arm.ObjectMeta.Namespace))
				Expect(mc.Namespace).To(Equal(arm.ObjectMeta.Namespace))
				Expect(auth.Namespace).To(Equal(arm.ObjectMeta.Namespace))
				return true
			})
		})
	})
})

func createTestActionRunnerMetrics(ctx context.Context, k8sClient client.Client, createApiServer bool, createMemcached bool, createAuthentication bool) (*runnerv1alpha1.ActionRunnerMetrics, []corev1.Secret) {
	ns := make([]string, 0)
	if testArmNamespaces != "" {
		ns = strings.Split(testArmNamespaces, ",")
	}

	runner := runnerv1alpha1.ActionRunnerMetrics{
		ObjectMeta: v1.ObjectMeta{
			Name:      testArmName,
			Namespace: testArmNamespace,
		},
		Spec: runnerv1alpha1.ActionRunnerMetricsSpec{
			Image:                 testImage,
			Replicas:              testArmReplicas,
			CacheWindow:           testArmCacheWindowSecs,
			CacheWindowWhenEmpty:  testArmCacheWindowWhenEmptySecs,
			ResyncInterval:        testArmResyncIntervalSecs,
			ExistingSslCertSecret: testArmUseExistingSslCertSecret,
			Namespaces:            ns,
			CreateApiServer:       createApiServer,
			CreateMemcached:       createMemcached,
			CreateAuthentication:  createAuthentication,
		},
	}

	Expect(k8sClient.Create(ctx, &runner)).Should(Succeed())

	return &runner, nil
}

func testArmResults(ctx context.Context, expectedCreate []bool, test func(*appsv1.Deployment, *appsv1.Deployment, *keda.ClusterTriggerAuthentication) bool) {
	Eventually(func() bool {
		dep := appsv1.Deployment{}
		memCached := appsv1.Deployment{}
		auth := keda.ClusterTriggerAuthentication{}
		nsName := types.NamespacedName{Namespace: testArmNamespace, Name: testArmName}
		e1 := k8sClient.Get(ctx, nsName, &dep)
		e2 := k8sClient.Get(ctx, types.NamespacedName{Namespace: testArmNamespace, Name: fmt.Sprintf("%s-cache", testArmName)}, &memCached)
		e3 := k8sClient.Get(ctx, nsName, &auth)
		if e1 == nil != expectedCreate[0] || e2 == nil != expectedCreate[1] || e3 == nil != expectedCreate[2] {
			return false
		}
		return test(&dep, &memCached, &auth)
	}, time.Minute, time.Second).Should(BeTrue())
}
