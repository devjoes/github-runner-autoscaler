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
)

const (
	testNamespace                   = "testns"
	testName                        = "testname"
	testArmReplicas                 = 3
	testArmCacheWindowSecs          = 123
	testArmCacheWindowWhenEmptySecs = 234
	testArmResyncIntervalSecs       = 345
	testArmUseExistingSslCertSecret = ""
	testNamespaces                  = ""
)

var _ = Describe("ActionRunnerMetrics controller", func() {
	Context("ActionRunnerMetrics CRD", func() {
		ctx := context.Background()
		var arm *runnerv1alpha1.ActionRunnerMetrics
		var version string
		It("Should set up dependant resources when created", func() {
			Expect(k8sClient.Create(ctx, &corev1.Namespace{ObjectMeta: v1.ObjectMeta{Name: testNamespace}})).Should(Succeed())
			arm = getRunner(true, true, true)
			Expect(k8sClient.Create(ctx, arm)).Should(Succeed())
			testArmResults(ctx, []bool{true, true, true}, func(dep, mc *appsv1.Deployment, auth *keda.ClusterTriggerAuthentication) bool {
				version = dep.ResourceVersion
				Expect(dep.Name).To(Equal(arm.Spec.Name))
				//Expect(mc.Name).To(Equal(arm.ObjectMeta.Name + "-cache"))
				//Expect(auth.Name).To(Equal(arm.ObjectMeta.Name))
				Expect(dep.Namespace).To(Equal(arm.Spec.Namespace))
				//Expect(mc.Namespace).To(Equal(arm.ObjectMeta.Namespace))
				//Expect(auth.Namespace).To(Equal(arm.ObjectMeta.Namespace))
				return true
			})
		})
		It("Should delete and recreate dependant resources when updated", func() {
			Expect(k8sClient.Update(ctx, arm)).Should(Succeed())
			secs := 0
			testArmResults(ctx, []bool{true, true, true}, func(dep, mc *appsv1.Deployment, auth *keda.ClusterTriggerAuthentication) bool {
				Expect(dep.ResourceVersion).To(Equal(version))
				secs++
				return secs >= 5
			})
			arm.Spec.Image = fmt.Sprintf("different_%s", arm.Spec.Image)
			Expect(k8sClient.Update(ctx, arm)).Should(Succeed())
			testArmResults(ctx, []bool{true, true, true}, func(dep, mc *appsv1.Deployment, auth *keda.ClusterTriggerAuthentication) bool {
				return dep.ResourceVersion != version
			})
		})
	})
})

func getRunner(createApiServer bool, createMemcached bool, createAuthentication bool) *runnerv1alpha1.ActionRunnerMetrics {
	ns := make([]string, 0)
	if testNamespaces != "" {
		ns = strings.Split(testNamespaces, ",")
	}

	return &runnerv1alpha1.ActionRunnerMetrics{
		ObjectMeta: v1.ObjectMeta{
			Name:      "main",
			Namespace: "ignored",
		},
		Spec: runnerv1alpha1.ActionRunnerMetricsSpec{
			Namespace:             testNamespace,
			Name:                  testName,
			Image:                 testImage,
			Replicas:              testArmReplicas,
			CacheWindow:           testArmCacheWindowSecs,
			CacheWindowWhenEmpty:  testArmCacheWindowWhenEmptySecs,
			ResyncInterval:        testArmResyncIntervalSecs,
			ExistingSslCertSecret: testArmUseExistingSslCertSecret,
			Namespaces:            ns,
			CreateApiServer:       &createApiServer,
			CreateMemcached:       &createMemcached,
			CreateAuthentication:  &createAuthentication,
		},
	}
}

func testArmResults(ctx context.Context, expectedCreate []bool, test func(*appsv1.Deployment, *appsv1.Deployment, *keda.ClusterTriggerAuthentication) bool) {
	Eventually(func() bool {
		dep := appsv1.Deployment{}
		memCached := appsv1.Deployment{}
		auth := keda.ClusterTriggerAuthentication{}
		nsName := types.NamespacedName{Name: "main"}
		e1 := k8sClient.Get(ctx, nsName, &dep)
		//TODO: more tests
		//e2 := k8sClient.Get(ctx, types.NamespacedName{Namespace: "main"space, Name: fmt.Sprintf("%s-cache", "main")}, &memCached)
		//e3 := k8sClient.Get(ctx, nsName, &auth)
		if e1 == nil != expectedCreate[0] {
			// || e2 == nil != expectedCreate[1] || e3 == nil != expectedCreate[2] {
			return false
		}
		return test(&dep, &memCached, &auth)
	}, time.Minute, time.Second).Should(BeTrue())
}
