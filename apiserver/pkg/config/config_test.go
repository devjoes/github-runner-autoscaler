package config

import (
	"context"
	"testing"
	"time"

	"github.com/devjoes/github-runner-autoscaler/apiserver/pkg/runnerclient"
	runnerv1alpha1 "github.com/devjoes/github-runner-autoscaler/operator/api/v1alpha1"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes/fake"
)

func createConfig(runnerNSsStr string, allNs bool, kubeconfig string, inCluster bool, resyncInterval time.Duration, params ...interface{}) (Config, error) {
	flagRunnerNSs := &ArrayFlags{}
	flagRunnerNSs.Set(runnerNSsStr)
	rs := resyncInterval.String()
	empty := ""
	config := Config{
		flagRunnerNSs:         flagRunnerNSs,
		flagAllNs:             &allNs,
		flagKubeconfig:        &kubeconfig,
		flagInClusterConfig:   &inCluster,
		flagResyncIntervalStr: &rs,
		flagMemcachedServers:  &empty,
		flagMemcachedUser:     &empty,
		flagMemcachedPass:     &empty,
	}
	err := config.SetupConfig(params...)
	return config, err
}

func TestErrorsOnInvalidArgs(t *testing.T) {
	var err error
	_, err = createConfig("", false, "", false, time.Second)
	assert.NotNil(t, err)
	_, err = createConfig("a,b", true, "", false, time.Second)
	assert.NotNil(t, err)
}

const (
	namespace    = "wfNamespace"
	name         = "wfName"
	wfOwner      = "wfOwner"
	wfRepo       = "wfRepo"
	wfSecretName = "wfSecretName"
	wfToken      = "wfToken"
	foo          = "foo"
)

var secret corev1.Secret
var fakeclient *fake.Clientset
var fakeRunnerClient *runnerclient.FakeRunnersV1Alpha1Client
var fakeRunnerClientWatch *watch.FakeWatcher
var runner runnerv1alpha1.ScaledActionRunner = runnerv1alpha1.ScaledActionRunner{
	ObjectMeta: metav1.ObjectMeta{
		Name:      name,
		Namespace: namespace,
	},
	TypeMeta: metav1.TypeMeta{Kind: "ScaledActionRunner"},
	Spec: runnerv1alpha1.ScaledActionRunnerSpec{
		Owner:             wfOwner,
		Repo:              wfRepo,
		GithubTokenSecret: wfSecretName,
	},
}

func setup() {
	secret = corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      wfSecretName,
			Namespace: namespace,
		},
		Data: map[string][]byte{
			"token": []byte(wfToken),
		},
	}
	fakeclient = fake.NewSimpleClientset(&secret)

	fakeRunnerClient, fakeRunnerClientWatch = runnerclient.NewFakeRunnersV1Alpha1Client(
		[]runnerv1alpha1.ScaledActionRunner{runner})
}

func TestLoadsWorkflowFromRunners(t *testing.T) {
	setup()
	config, err := createConfig(namespace, false, "", false, time.Hour, fakeclient, fakeRunnerClient)
	assert.Nil(t, err)
	wfs := config.GetAllWorkflows()
	assert.Len(t, wfs, 1)
	assert.Equal(t, wfToken, wfs[0].Token)
	assert.Equal(t, wfOwner, wfs[0].Owner)
	assert.Equal(t, wfRepo, wfs[0].Repository)
}

func TestGetsWorkflowByName(t *testing.T) {
	setup()
	config, err := createConfig(namespace, false, "", false, time.Hour, fakeclient, fakeRunnerClient)
	assert.Nil(t, err)
	key := name
	wf, err := config.GetWorkflow(key)
	assert.Nil(t, err)
	assert.NotNil(t, wf)
	assert.Equal(t, name, wf.Name)
	assert.Equal(t, namespace, wf.Namespace)
	assert.Equal(t, wfToken, wf.Token)
	assert.Equal(t, wfOwner, wf.Owner)
	assert.Equal(t, wfRepo, wf.Repository)
}

func TestReSyncsWorkflowFromRunners(t *testing.T) {
	setup()
	config, err := createConfig(namespace, false, "", false, time.Second, fakeclient, fakeRunnerClient)
	assert.Nil(t, err)
	wfs := config.GetAllWorkflows()

	(*fakeRunnerClient.Runners)[0].ObjectMeta.Name = foo
	secret.Data["token"] = []byte(foo)
	fakeclient.CoreV1().Secrets(secret.Namespace).Update(context.TODO(), &secret, metav1.UpdateOptions{})
	assert.NotEqual(t, foo, wfs[0].Name)
	assert.NotEqual(t, foo, wfs[0].Token)
	time.Sleep(time.Millisecond * 1200)
	wfs = config.GetAllWorkflows()
	assert.Equal(t, foo, wfs[0].Name)
	assert.Equal(t, foo, wfs[0].Token)
}

func TestWatcherUpdatesWorkflowOnChange(t *testing.T) {
	setup()
	config, err := createConfig(namespace, false, "", false, time.Hour, fakeclient, fakeRunnerClient)
	assert.Nil(t, err)
	wfs := config.GetAllWorkflows()

	assert.NotEqual(t, foo, wfs[0].Owner)

	modRunner := runner.DeepCopyObject().(*runnerv1alpha1.ScaledActionRunner)
	modRunner.Spec.Owner = foo
	fakeRunnerClientWatch.Modify(modRunner)
	time.Sleep(time.Second)
	assert.Equal(t, foo, config.GetAllWorkflows()[0].Owner)

	fakeRunnerClientWatch.Delete(modRunner)
	time.Sleep(time.Second)
	assert.Empty(t, config.GetAllWorkflows())

	fakeRunnerClientWatch.Add(&runner)
	time.Sleep(time.Second)
	assert.Equal(t, wfOwner, config.GetAllWorkflows()[0].Owner)
}
