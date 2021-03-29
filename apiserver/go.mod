module github.com/devjoes/github-runner-autoscaler/apiserver

go 1.15

require (
	github.com/devjoes/github-runner-autoscaler/operator v0.0.0-20210328184102-78147cd553f6 // indirect
	github.com/golang/protobuf v1.5.1
	github.com/google/go-github/v33 v33.0.0
	github.com/kubernetes-sigs/custom-metrics-apiserver v0.0.0-20210311094424-0ca2b1909cdc
	github.com/memcachier/mc v2.0.1+incompatible // indirect
	github.com/namsral/flag v1.7.4-pre // indirect
	github.com/pkg/errors v0.9.1
	github.com/prometheus/client_golang v1.10.0 // indirect
	github.com/stretchr/testify v1.7.0
	golang.org/x/oauth2 v0.0.0-20210313182246-cd4f82c27b84
	golang.org/x/sys v0.0.0-20210319071255-635bc2c9138d // indirect
	google.golang.org/appengine v1.6.7 // indirect
	google.golang.org/grpc v1.36.0
	k8s.io/api v0.20.5
	k8s.io/apimachinery v0.20.5
	k8s.io/apiserver v0.20.5
	k8s.io/client-go v11.0.1-0.20190805182717-6502b5e7b1b5+incompatible
	k8s.io/component-base v0.20.5
	k8s.io/klog/v2 v2.8.0
	k8s.io/metrics v0.20.5
)

replace k8s.io/client-go => k8s.io/client-go v0.20.5

replace google.golang.org/grpc => google.golang.org/grpc v1.26.0

replace github.com/googleapis/gnostic => github.com/googleapis/gnostic v0.4.1
