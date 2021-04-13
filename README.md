# github-action-autoscaler

## Purpose

This operator allows you to create github action runners which scale up and down in response to demand. There are a few other projects which allow you to do the same thing, however this is geared towards running platforms that support thousands of individual Github repositories with their own discrete sets of permissions, cost codes etc. The main problems that this was created to solve are:

- Runners are created upfront during onboarding and then remain offline if unused. This means that we do not need to retain admin access to thousands of repositories.
- Runners are scaled to 0 replicas when not used.
- Runners are unique to each repository and can be associated with specific IAM roles, Azure Identities or even use a custom runner image.
- A repository can be associated with multiple runners using runner labels and other metadata.

## Design

The operator creates an API server which interfaces with Github and exposes custom metrics, this is backed by a memcached instance.
The operator is also responsible for translating ScaledActionRunner CRs in to StatefulSets and ScaledObjects.

![design](docs/design.png "Design")

## Installation & Usage

### Prerequisites

- [KEDA](https://github.com/kedacore/keda/)
- [cert-manager](https://github.com/jetstack/cert-manager)
- [Prometheus](https://prometheus.io/)

Because this essentially just functions as an API server that exposes custom metrics it doesn't technically require KEDA or any other dependencies (you can just point a HPA at the custom metrics.) However KEDA allows HPAs to scale to zero which is important from a cost saving perspective. You can also get away with not using cert-manager and just create the certificates yourself if you really want to, and Prometheus is for monitoring (github-action-autoscaler exposes detailed metrics.)

These dependencies can be installed like this:

```
helm repo add prometheus-community https://prometheus-community.github.io/helm-charts
helm repo add jetstack https://charts.jetstack.io
helm repo add kedacore https://kedacore.github.io/charts
helm repo update

helm install prometheus prometheus-community/kube-prometheus-stack -n monitoring --create-namespace
helm install keda kedacore/keda -n keda --set prometheus.operator.enabled=true --set prometheus.metricServer.enabled=true --create-namespace
helm install cert-manager jetstack/cert-manager -n cert-manager --create-namespace --set installCRDs=true
```

We can then install the operator (this will track the latest tag):

```
kubectl apply -f install.yaml
```

We are going to create an API server called 'metrics' in the namespace 'github' before doing this we need to create the namespace and a certificate for it to use:

```
kubectl create ns github
kubectl apply -f - <<EOF
apiVersion: cert-manager.io/v1alpha2
kind: Issuer
metadata:
  name: selfsigned-issuer
  namespace: cert-manager
spec:
  selfSigned: {}
---
apiVersion: cert-manager.io/v1alpha2
kind: Certificate
metadata:
  name: ca-certificate
  namespace: cert-manager
spec:
  secretName: ca-cert
  duration: 2880h # 120d
  renewBefore: 360h # 15d
  commonName: github-action-autoscaler-server-ca
  isCA: true
  keySize: 2048
  usages:
    - digital signature
    - key encipherment
  issuerRef:
    name: selfsigned-issuer
    kind: Issuer
    group: cert-manager.io
---
apiVersion: cert-manager.io/v1alpha2
kind: ClusterIssuer
metadata:
  name: ca-issuer
spec:
  ca:
    secretName: ca-cert
---
apiVersion: cert-manager.io/v1alpha2
kind: Certificate
metadata:
  name: github-action-autoscaler-server
  namespace: github
spec:
  secretName: tls-cert
  duration: 2160h
  renewBefore: 360h
  isCA: false
  keySize: 2048
  keyAlgorithm: rsa
  keyEncoding: pkcs1
  usages:
    - digital signature
    - key encipherment
    - server auth
  commonName: metrics.github.svc.cluster.local
  dnsNames:
    - "metrics.github.svc.cluster.local"
    - "metrics.github.svc"
    - "metrics"
  issuerRef:
    name: ca-issuer
    kind: ClusterIssuer
    group: cert-manager.io
---
apiVersion: cert-manager.io/v1alpha2
kind: Certificate
metadata:
  name: github-action-autoscaler-client
  namespace: keda
spec:
  secretName: tls-cert
  duration: 2160h # 90d
  renewBefore: 360h # 15d
  keySize: 2048
  keyAlgorithm: rsa
  keyEncoding: pkcs1
  usages:
    - digital signature
    - key encipherment
    - client auth
  commonName: Keda
  issuerRef:
    name: ca-issuer
    kind: ClusterIssuer
    group: cert-manager.io
EOF
```

This will create a self signed CA certificate in the cert-manager namespace and expose a cluster wide issuer. It then creates a certificate for the API server and a certificate for KEDA. These certificates aren't actually used for authentication, API servers are meant to use SecureServing and KEDA's external metrics scaler doesn't support self signed certificates.

Now we can create the ScaledActionRunnerCore CR. Because this CR has to create a cluster wide APIService called "v1beta1.custom.metrics.k8s.io" there can only be one ScaledActionRunnerCore object in the cluster. To ensure this is the case ScaledActionRunnerCore is a cluster wide object and has to be named 'core', if it isn't named core then it will be ignored.

```
kubectl apply -f - <<EOF
kind: ScaledActionRunnerCore
apiVersion: runner.devjoes.com/v1alpha1
metadata:
  name: core
spec:
  apiServerName: metrics
  apiServerNamespace: github
  sslCertSecret: tls-cert
  apiServerImage: "joeshearn/github-runner-autoscaler-apiserver:master"
  prometheusNamespace: monitoring
EOF
```

After a short while we should see some pods like this:

```
kubectl get po -n github
NAME                       READY   STATUS    RESTARTS   AGE
metrics-78cfb79876-cr9ft   1/1     Running   0          57s
metrics-78cfb79876-jgts5   1/1     Running   0          57s
metrics-cache-0            1/1     Running   0          56s
metrics-cache-1            1/1     Running   0          28s
```

We now need to onboard our first repo. We will create a very simple workflow and add it to a github repo.

```
(
cd `mktemp -d`
mkdir -p example-repo/.github/workflows
cd example-repo
git init
cat > .github/workflows/main.yml <<EOF
name: main
on:
  push:
jobs:
  main:
    runs-on: [self-hosted]
    steps:
      - run: sleep 2m; echo Finished
EOF
hub create
git add .
git commit -m "this build will fail"
git push --set-upstream origin master
)
```

This will create a repository which will build on push, the build will fail. To fix this we need to create a ScaledActionRunner and some secrets. You will need two PAT tokens first. One PAT token will be for an account that has admin access to the repo, the other will be for an account that has read only access. (For testing purposes you can just use the same admin PAT token.)

There is a node application which will handle the creation of the secrets and ScaledActionRunner. Update owner, read_token and admin_token

```
owner=
read_token=/path/to/file/containing/token
admin_token=/path/to/file/containing/token
node examples/add-runner.js -n "example-repo" -o "$owner" -r "example-repo" -m 4 -p "$read_token" -a "$admin_token" -f example-repo.yaml
```

This will have created a file called example-repo.yaml which will provision a runner called example-repo that can scale up to 4 replicas and deploy it:

```
kubectl create ns example-repo
kubectl apply -n example-repo -f example-repo.yaml
```

You should then see that several resources:

```
kubectl get statefulset,scaledobject,scaledactionrunner -n example-repo
NAME                            READY   AGE
statefulset.apps/example-repo   0/0     57s

NAME                                SCALETARGETKIND       SCALETARGETNAME   MIN   MAX   TRIGGERS      AUTHENTICATION   READY   ACTIVE    AGE
scaledobject.keda.sh/example-repo   apps/v1.StatefulSet   example-repo      0     4     metrics-api   metrics          False   Unknown   58s

NAME                                                 AGE
scaledactionrunner.runner.devjoes.com/example-repo   58s
```

If you now re-trigger the build that failed you should see the metrics and the stateful set should scale up:

```
kubectl get --raw '/apis/custom.metrics.k8s.io/v1beta1/namespaces/example-repo/Scaledactionrunners/example-repo/*' | jq .
{
  "kind": "MetricValueList",
  "apiVersion": "custom.metrics.k8s.io/v1beta1",
  "metadata": {
    "selfLink": "/apis/custom.metrics.k8s.io/v1beta1/namespaces/example-repo/Scaledactionrunners/example-repo/%2A"
  },
  "items": [
    {
      "describedObject": {
        "kind": "ScaledActionRunner",
        "namespace": "example-repo",
        "name": "example-repo",
        "apiVersion": "v1alpha1"
      },
      "metricName": "example-repo",
      "timestamp": "2021-04-13T10:14:35Z",
      "value": "2",
      "selector": {
        "matchLabels": {
          "cr_name": "example-repo",
          "cr_namespace": "example-repo",
          "cr_owner": "devjoes",
          "cr_repo": "example-repo",
          "wf_id": "7939907",
          "wf_name": "main",
          "wf_runs_on": "self-hosted",
          "wf_runs_on_self-hosted": "self-hosted"
        }
      }
    }
  ]
}
```

When the StatefulSet is first scaled up it will have to create persistant volumes which may take a while. But eventually you should see pods created:

```
kubectl get statefulset -n example-repo
NAME           READY   AGE
example-repo   2/2     3m4s
kubectl logs example-repo-0 -n example-repo
Starting Runner listener with startup type: service
Started listener process, pid: 16
Started running service

√ Connected to GitHub

2021-04-13 10:38:43Z: Listening for Jobs
2021-04-13 10:38:47Z: Running job: main
2021-04-13 10:40:51Z: Job main completed with result: Succeeded
```
