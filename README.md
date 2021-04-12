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

We can then install the operator like this:

```
cd operator
make install deploy
```

TODO: Come up with a proper way of packaging the operator

We are going to create an API service called 'metrics' in the namespace 'github' before doing this we need to create the namespace and a certificate for it to use like this:

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
  secretName: cert
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
