(
	set -x
	cd `mktemp -d`
	domain="external-metrics-apiserver.runners.svc.cluster.local"
	alt_name="external-metrics-apiserver.runners"
	openssl genrsa -out ca.key 2048 
	openssl req -x509 -new -nodes -key ca.key -sha256 -days 3650 -out ca.pem -subj "/C=GB/O=K8s/CN=cluster.local"

cat <<EOF > "$domain.cnf"
[req]
distinguished_name = req_distinguished_name
x509_extensions = v3_req
prompt = no
ca = ca.pem
ca_key = ca.key
[req_distinguished_name]
C = GB
L = SomeCity
O = MyCompany
OU = MyDivision
CN = DOMAIN
[v3_req]
keyUsage = keyEncipherment, dataEncipherment
extendedKeyUsage = serverAuth
subjectAltName = @alt_names
[alt_names]
DNS.1 = DOMAIN
EOF
	set -e
	sed -i "s/DOMAIN/$domain/g" "$domain.cnf"
	openssl req -nodes -days 3650 -newkey rsa:2048 -keyout "$domain.key" -out "$domain.csr" -config "$domain.cnf" -extensions 'v3_req'
	openssl x509 -req -extensions v3_req -extfile "$domain.cnf" -in "$domain.csr" -CA ca.pem -CAkey ca.key -CAcreateserial -out "$domain.pem" -days 3650 -sha256 
	cat "$domain.pem"  | openssl x509 -text
	
	mkdir k8s
	cd k8s
	cp "../$domain.pem" tls.crt
	cp "../$domain.key" tls.key
	cp "../ca.pem" ca.crt

	kubectl delete secret cert -n runners || true
	kubectl delete secret cert -n keda 	|| true
	
	kubectl create secret generic cert -n runners --from-file=tls.crt  --from-file=tls.key --from-file=ca.crt
	kubectl create secret generic cert -n keda --from-file=tls.crt  --from-file=tls.key --from-file=ca.crt

	kubectl delete po --all -n runners
	kubectl delete po --all -n keda
)