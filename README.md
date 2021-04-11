# github-action-autoscaler

# notes

## on new cluster

helm install prometheus prometheus-community/kube-prometheus-stack
helm install keda kedacore/keda -n keda --set prometheus.operator.enabled=true --set prometheus.metricServer.enabled=true --create-namespace
kubectl create ns runners
./cert.sh
kubectl apply -f runners-creds.secret.yaml
(
cd operator
make install deploy;kubectl delete deployment -n operator-system operator-controller-manager
kubectl apply -f ../crs.yaml
make run
)

## Known issues

- The authentication isnt actually authentication
- Runner labels are essentially backwards

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

- TODO: Cert auth is just used as a way of fudging TLS
- TODO: Make runners close session on shutdown
- TODO: Have option to not output labels - could be sensitive

* TODO: Get GIT_TOKEN

# Cleanup

kubectl delete -f ../crs.yaml
sleep 3s
make undeploy
(echo runners;kubectl get ns | grep -Po 'test-repo\S+') | xargs kubectl delete ns

for node in `kubectl get nodes -o yaml | grep -oP '10\.99\.\d+\.\d+' | sort | uniq`; do
ssh setup@$node "sudo bash -c \"docker image ls | grep joeshearn | grep -Po '^\S+\s+\S+' | sed -E 's/\s+/:/g' | xargs docker image rm -f \""
done

kubectl create ns runners
make deploy IMG=joeshearn/github-runner-autoscaler-operator:master
kubectl apply -f ../crs.yaml
