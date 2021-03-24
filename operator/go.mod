module github.com/devjoes/github-runner-autoscaler/operator

go 1.15

require (
	github.com/go-logr/logr v0.4.0
	github.com/kedacore/keda/v2 v2.2.0
	github.com/onsi/ginkgo v1.15.2
	github.com/onsi/gomega v1.11.0
	github.com/pingcap/errors v0.11.4
	github.com/stretchr/testify v1.7.0
	k8s.io/api v0.20.5
	k8s.io/apimachinery v0.20.5
	k8s.io/client-go v11.0.1-0.20190805182717-6502b5e7b1b5+incompatible
	knative.dev/pkg v0.0.0-20210318052054-dfeeb1817679 // indirect
	sigs.k8s.io/controller-runtime v0.8.3
)

replace k8s.io/client-go => k8s.io/client-go v0.20.5
