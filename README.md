# github-action-autoscaler

# notes

## on new cluster

helm install prometheus prometheus-community/kube-prometheus-stack
helm install keda kedacore/keda -n keda --set prometheus.operator.enabled=true --set prometheus.metricServer.enabled=true --create-namespace
./cert.sh
kubectl create ns runners
kubectl apply -f runners-creds.secret.yaml
(
cd operator
make install deploy
kubectl delete deployment -n operator-system operator-controller-manager
kubectl apply -f ../crs.yaml
make run
)

# This is only required for my home cluster

for i in {0..3}; do
echo '{"kind":"PersistentVolume","apiVersion":"v1","metadata":{"name":"hp-pv-000","labels":{"type":"local"}},"spec":{"storageClassName":"manual","capacity":{"storage":"5Gi"},"accessModes":["ReadWriteOnce"],"hostPath":{"path":"/tmp/data00"}}}' |
sed "s/00\"/0$i\"/g" | kubectl apply -f -
done

# temp - pending fix

kubectl apply -f - <<EOF
kind: TriggerAuthentication
apiVersion: keda.sh/v1alpha1
metadata:
name: test123
namespace: runners
spec:
secretTargetRef: - key: cert
name: cert
parameter: cert - key: ca
name: cert
parameter: ca - key: key
name: cert
parameter: key
EOF

# known issues

Cert auth is just used as a way of fudging TLS
Deploy resources in corect order?
Make runners close session on shutdown
