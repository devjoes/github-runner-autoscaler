package config

import (
	"context"
	"testing"
	"time"

	runnerv1alpha1 "github.com/devjoes/github-runner-autoscaler/operator/api/v1alpha1"
	"github.com/devjoes/github-runner-autoscaler/apiserver/pkg/runnerclient"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes/fake"
)

// func getWorkflowConfig() (WorkflowConfig, []byte, map[string]interface{}, []byte, rsa.PrivateKey, []byte) {
// 	wfc := WorkflowConfig{
// 		AgentId:    json.Number("46"),
// 		AgentName:  "test",
// 		PoolId:     json.Number("1"),
// 		PoolName:   "Default",
// 		ServerUrl:  "https://pipelines.actions.githubusercontent.com/flkkjdlkjsdfkljsdfkl",
// 		GitHubUrl:  "https://github.com/foo/bar",
// 		WorkFolder: "/_work",
// 	}
// 	runner, _ := json.Marshal(wfc)

// 	creds := map[string]interface{}{
// 		"authorizationUrl": "https://vstoken.actions.githubusercontent.com/_apis/oauth2/token/e54afd9c-12b0-459d-9e6b-19ea67e10ad9",
// 		"clientId":         "123afd9c-12b0-459d-9e6b-19ea67e10ad9",
// 	}
// 	credentials := []byte("{\"data\":{\"authorizationUrl\":\"https://vstoken.actions.githubusercontent.com/_apis/oauth2/token/e54afd9c-12b0-459d-9e6b-19ea67e10ad9\",\"clientId\":\"123afd9c-12b0-459d-9e6b-19ea67e10ad9\"}}")
// 	keyBytes, key := getTestPrivateKey()
// 	return wfc, runner, creds, credentials, key, keyBytes
// }

func createConfig(runnerNSsStr string, allNs bool, kubeconfig string, inCluster bool, resyncInterval time.Duration, params ...interface{}) (Config, error) {
	flagRunnerNSs := &ArrayFlags{}
	flagRunnerNSs.Set(runnerNSsStr)
	rs := resyncInterval.String()
	config := Config{
		flagRunnerNSs:         flagRunnerNSs,
		flagAllNs:             &allNs,
		flagKubeconfig:        &kubeconfig,
		flagInClusterConfig:   &inCluster,
		flagResyncIntervalStr: &rs,
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
	namespace    = "namespace"
	name         = "name"
	wfName       = "wfName"
	wfOwner      = "wfOwner"
	wfRepo       = "wfRepo"
	wfNamespace  = "wfNamespace"
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
		Name:              wfName,
		Namespace:         wfNamespace,
		Owner:             wfOwner,
		Repo:              wfRepo,
		GithubTokenSecret: wfSecretName,
	},
}

func setup() {
	secret = corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      wfSecretName,
			Namespace: wfNamespace,
		},
		Data: map[string][]byte{
			"token": []byte(wfToken),
		},
	}
	fakeclient = fake.NewSimpleClientset(&secret)

	fakeRunnerClient, fakeRunnerClientWatch = runnerclient.NewFakeRunnersV1Alpha1Client([]runnerv1alpha1.ScaledActionRunner{runner})
}

func TestLoadsWorkflowFromRunners(t *testing.T) {
	setup()
	config, err := createConfig(namespace, false, "", false, time.Hour, fakeclient, fakeRunnerClient)
	assert.Nil(t, err)
	wfs := config.GetAllWorkflows()
	assert.Len(t, wfs, 1)
	assert.Equal(t, wfName, wfs[0].Name)
	assert.Equal(t, wfNamespace, wfs[0].Namespace)
	assert.Equal(t, wfToken, wfs[0].Token)
	assert.Equal(t, wfOwner, wfs[0].Owner)
	assert.Equal(t, wfRepo, wfs[0].Repository)
}

func TestGetsWorkflowByName(t *testing.T) {
	setup()
	config, err := createConfig(namespace, false, "", false, time.Hour, fakeclient, fakeRunnerClient)
	assert.Nil(t, err)
	key := wfName
	wf, err := config.GetWorkflow(key)
	assert.Nil(t, err)
	assert.NotNil(t, wf)
	assert.Equal(t, wfName, wf.Name)
	assert.Equal(t, wfNamespace, wf.Namespace)
	assert.Equal(t, wfToken, wf.Token)
	assert.Equal(t, wfOwner, wf.Owner)
	assert.Equal(t, wfRepo, wf.Repository)
}

func TestReSyncsWorkflowFromRunners(t *testing.T) {
	setup()
	config, err := createConfig(namespace, false, "", false, time.Second, fakeclient, fakeRunnerClient)
	assert.Nil(t, err)
	wfs := config.GetAllWorkflows()

	(*fakeRunnerClient.Runners)[0].Spec.Name = foo
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

// func TestLoadsSecrets(t *testing.T) {
// 	test := func(runnerNSs []string, allNs bool, labelSelector string) {
// 		wfc, runner, creds, credentials, key, keyBytes := getWorkflowConfig()
// 		fakeclient := fake.NewSimpleClientset(&corev1.Secret{
// 			ObjectMeta: metav1.ObjectMeta{
// 				Name:      "testsecret",
// 				Namespace: "test",
// 				Labels: map[string]string{
// 					"action-secret-type": "autoscaler",
// 				}},
// 			Data: map[string][]byte{
// 				".runner":      runner,
// 				".credentials": credentials,
// 				"private.pem":  keyBytes,
// 			}},
// 			&corev1.Secret{
// 				ObjectMeta: metav1.ObjectMeta{
// 					Name:      "runner-secret",
// 					Namespace: "test",
// 					Labels: map[string]string{
// 						"action-secret-type": "runner",
// 					}},
// 				Data: map[string][]byte{
// 					".runner":      runner,
// 					".credentials": credentials,
// 					"private.pem":  keyBytes,
// 				}},
// 			&corev1.Secret{
// 				ObjectMeta: metav1.ObjectMeta{Name: "unrelated", Namespace: "test"},
// 				Data: map[string][]byte{
// 					foo: []byte("bar"),
// 				}})

// 		config, err := createConfig(runnerNSs, allNs, labelSelector, "", false, fakeclient)

// 		assert.Nil(t, err)
// 		assert.Equal(t, wfc.AgentId, config.Workflows[0].AgentId)
// 		assert.Equal(t, wfc.AgentName, config.Workflows[0].AgentName)
// 		assert.Equal(t, wfc.PoolId, config.Workflows[0].PoolId)
// 		assert.Equal(t, wfc.PoolName, config.Workflows[0].PoolName)
// 		assert.Equal(t, wfc.ServerUrl, config.Workflows[0].ServerUrl)
// 		assert.Equal(t, wfc.GitHubUrl, config.Workflows[0].GitHubUrl)
// 		assert.Equal(t, wfc.WorkFolder, config.Workflows[0].WorkFolder)
// 		assert.Equal(t, creds["authorizationUrl"], config.Workflows[0].Auth.AuthorizationUrl)
// 		assert.Equal(t, creds["clientId"], config.Workflows[0].Auth.ClientId)
// 		assert.Equal(t, key, *config.Workflows[0].Auth.Token)
// 	}
// 	test([]string{"test"}, false, "action-secret-type=autoscaler")
// 	test([]string{}, true, "action-secret-type=autoscaler")
// 	test([]string{"test"}, false, "")
// 	test([]string{}, true, "")
// }
