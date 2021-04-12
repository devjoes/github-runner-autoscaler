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
	testName                        = "testname"
	testArmImage                    = "foo"
	testArmReplicas                 = 3
	testArmCacheWindowSecs          = 123
	testArmCacheWindowWhenEmptySecs = 234
	testArmResyncIntervalSecs       = 345
	testArmUseExistingSslCertSecret = ""
	testNamespaces                  = ""
)

var _ = Describe("ScaledActionRunnerCore controller", func() {
	Context("ScaledActionRunnerCore CRD", func() {
		ctx := context.Background()
		var arm *runnerv1alpha1.ScaledActionRunnerCore
		var version string
		ns := "testns"
		testCreation := func(createApiServer, createMemcached, createTriggerAuth bool, testNamespace string, msg string) {
			It("Should set up dependant resources when created"+msg, func() {
				Expect(k8sClient.Create(ctx, &corev1.Namespace{ObjectMeta: v1.ObjectMeta{Name: testNamespace}})).Should(Succeed())
				arm = getRunner(createApiServer, createMemcached, createTriggerAuth, testNamespace)
				k8sClient.DeleteAllOf(ctx, &runnerv1alpha1.ScaledActionRunnerCore{})
				k8sClient.DeleteAllOf(ctx, &keda.ClusterTriggerAuthentication{})
				Expect(k8sClient.Create(ctx, arm)).Should(Succeed())
				testArmResults(ctx, []bool{createApiServer, createMemcached, createTriggerAuth}, testNamespace, func(dep *appsv1.Deployment, mc *appsv1.StatefulSet, auth *keda.ClusterTriggerAuthentication) bool {
					version = dep.ResourceVersion
					if createApiServer {
						Expect(dep.Name).To(Equal(arm.Spec.ApiServerName))
						Expect(dep.Namespace).To(Equal(arm.Spec.ApiServerNamespace))
					}
					if createMemcached {
						Expect(mc.Name).To(Equal(arm.Spec.ApiServerName + "-cache"))
						Expect(mc.Namespace).To(Equal(arm.Spec.ApiServerNamespace))
					}
					if createTriggerAuth {
						Expect(auth.Name).To(Equal(arm.Spec.ApiServerName))
						Expect(auth.Namespace).To(Equal(""))
					}
					return true
				})
			})
		}
		testCreation(true, true, true, ns, "")
		It("Should delete and recreate dependant resources when updated", func() {
			Expect(k8sClient.Update(ctx, arm)).Should(Succeed())
			secs := 0
			testArmResults(ctx, []bool{true, true, true}, ns, func(dep *appsv1.Deployment, mc *appsv1.StatefulSet, auth *keda.ClusterTriggerAuthentication) bool {
				Expect(dep.ResourceVersion).To(Equal(version))
				secs++
				return secs >= 5
			})
			arm.Spec.ApiServerImage = fmt.Sprintf("different_%s", arm.Spec.ApiServerImage)
			Expect(k8sClient.Update(ctx, arm)).Should(Succeed())
			testArmResults(ctx, []bool{true, true, true}, ns, func(dep *appsv1.Deployment, mc *appsv1.StatefulSet, auth *keda.ClusterTriggerAuthentication) bool {
				return dep.ResourceVersion != version
			})
		})
		testCreation(true, false, false, "testns1", " (only api server)")
		testCreation(false, true, false, "testns2", " (only memcached)")
		testCreation(false, false, true, "testns3", " (only trigger auth)")
	})
})

func getRunner(createApiServer bool, createMemcached bool, createAuthentication bool, testNamespace string) *runnerv1alpha1.ScaledActionRunnerCore {
	ns := make([]string, 0)
	if testNamespaces != "" {
		ns = strings.Split(testNamespaces, ",")
	}

	return &runnerv1alpha1.ScaledActionRunnerCore{
		ObjectMeta: v1.ObjectMeta{
			Name:      "main",
			Namespace: "ignored",
		},
		Spec: runnerv1alpha1.ScaledActionRunnerCoreSpec{
			ApiServerNamespace:   testNamespace,
			ApiServerName:        testName,
			ApiServerImage:       testArmImage,
			ApiServerReplicas:    testArmReplicas,
			CacheWindow:          testArmCacheWindowSecs,
			CacheWindowWhenEmpty: testArmCacheWindowWhenEmptySecs,
			ResyncInterval:       testArmResyncIntervalSecs,
			SslCertSecret:        testArmUseExistingSslCertSecret,
			Namespaces:           ns,
			CreateApiServer:      &createApiServer,
			CreateMemcached:      &createMemcached,
			CreateAuthentication: &createAuthentication,
		},
	}
}

func testArmResults(ctx context.Context, expectedCreate []bool, testNamespace string, test func(*appsv1.Deployment, *appsv1.StatefulSet, *keda.ClusterTriggerAuthentication) bool) {
	Eventually(func() bool {
		dep := appsv1.Deployment{}
		memCached := appsv1.StatefulSet{}
		auth := keda.ClusterTriggerAuthentication{}
		nsName := types.NamespacedName{Name: testName, Namespace: testNamespace}
		e := k8sClient.Get(ctx, nsName, &dep)
		if e == nil != expectedCreate[0] {
			return false
		}
		e = k8sClient.Get(ctx, types.NamespacedName{Namespace: testNamespace, Name: fmt.Sprintf("%s-cache", testName)}, &memCached)
		if e == nil != expectedCreate[1] {
			return false
		}
		e = k8sClient.Get(ctx, types.NamespacedName{Namespace: "keda", Name: testName}, &auth)
		if e == nil != expectedCreate[2] {
			return false
		}

		return test(&dep, &memCached, &auth)
	}, testTimeoutSecs*time.Second, time.Second).Should(BeTrue())
}
