package main

import (
	"flag"
	"net/http"
	"os"

	"github.com/kubernetes-sigs/custom-metrics-apiserver/pkg/apiserver"
	basecmd "github.com/kubernetes-sigs/custom-metrics-apiserver/pkg/cmd"
	"github.com/kubernetes-sigs/custom-metrics-apiserver/pkg/provider"
	generatedopenapi "github.com/kubernetes-sigs/custom-metrics-apiserver/test-adapter/generated/openapi"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"k8s.io/apimachinery/pkg/util/wait"
	openapinamer "k8s.io/apiserver/pkg/endpoints/openapi"
	genericapiserver "k8s.io/apiserver/pkg/server"
	"k8s.io/component-base/logs"
	"k8s.io/klog/v2"

	config "github.com/devjoes/github-runner-autoscaler/apiserver/pkg/config"
	"github.com/devjoes/github-runner-autoscaler/apiserver/pkg/health"
	host "github.com/devjoes/github-runner-autoscaler/apiserver/pkg/host"
	k8sProvider "github.com/devjoes/github-runner-autoscaler/apiserver/pkg/k8sprovider"
)

type WorkflowMetricsAdapter struct {
	basecmd.AdapterBase

	// the message printed on startup
	Message string
}

func main() {
	logs.InitLogs()
	defer logs.FlushLogs()

	cmd := &WorkflowMetricsAdapter{}

	cmd.OpenAPIConfig = genericapiserver.DefaultOpenAPIConfig(generatedopenapi.GetOpenAPIDefinitions, openapinamer.NewDefinitionNamer(apiserver.Scheme))
	cmd.OpenAPIConfig.Info.Title = "github-workflow-metrics-adapter"
	cmd.OpenAPIConfig.Info.Version = "1.0.0"

	cmd.Flags().StringVar(&cmd.Message, "msg", "starting adapter...", "startup message")
	conf := config.Config{}
	conf.AddFlags()
	cmd.Flags().AddGoFlagSet(flag.CommandLine)
	cmd.Flags().Parse(os.Args)

	err := conf.SetupConfig()
	if err != nil {
		klog.Fatalf("Error loading config: %v", err)
	}
	h, err := host.NewHost(conf)
	if err != nil {
		klog.Fatal(err)
	}

	go cmd.initHandlers(conf)
	testProvider := cmd.makeK8sProvider(h)
	cmd.Authorization.WithAlwaysAllowGroups("system:unauthenticated")
	cmd.WithCustomMetrics(testProvider)

	klog.Infof(cmd.Message)
	if err := cmd.Run(wait.NeverStop); err != nil {
		klog.Fatalf("unable to run custom metrics adapter: %v", err)
	}
}

func (a *WorkflowMetricsAdapter) makeK8sProvider(orchestrator *host.Host) provider.CustomMetricsProvider {
	return k8sProvider.NewProvider(orchestrator)
}

// func (a *WorkflowMetricsAdapter) makeKedaProvider(orchestrator *host.Host) {
// 	kedaProvider.NewKedaProvider(orchestrator)
// }

func (a *WorkflowMetricsAdapter) initHandlers(conf config.Config) {
	h := health.NewHealth(conf)
	http.HandleFunc("/readyz", h.Readyz())
	http.HandleFunc("/livez", h.Livez())
	http.Handle("/metrics", promhttp.Handler())
	http.ListenAndServe(":2112", nil)
}
