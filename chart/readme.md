# Installations steps

### Create local Kind cluster 
```bash
cat <<EOF | kind create cluster --config=-
kind: Cluster
apiVersion: kind.x-k8s.io/v1alpha4
name: kind
nodes:
- role: control-plane
  extraPortMappings:
  - containerPort: 30980
    hostPort: 80
    protocol: TCP
  - containerPort: 30943
    hostPort: 443
    protocol: TCP
EOF
```

kubectl create -f [nginx-ingress.yaml](../ops/kind/nginx-ingress.yaml)
### Deploy Nginx Ingress Controller from [nginx-ingress.yaml](../ops/kind/nginx-ingress.yaml)
```bash
kubectl create -f ../ops/kind/nginx-ingress.yaml
```

Deploy sample application. 
Note the `ingress.hostname` show include your local IP address.
```bash 
helm install wp oci://registry-1.docker.io/bitnamicharts/wordpress \
  --set image.repository=bitnamilegacy/wordpress \
  --set mariadb.image.repository=bitnamilegacy/mariadb \
  --set global.security.allowInsecureImages=true \
  --set ingress.enabled=true \
  --set ingress.hostname=wp.<YOUR-LOCAL-IP-GOES-HERE>.nip.io \
  --set service.type=ClusterIP
```

Once deployed, try access http://wp.<YOUR-LOCAL-IP-GOES-HERE>.nip.io
You should get WordPress website. 

Deploy wafie helm chart 
```bash
helm repo add wafie https://charts.wafie.io
helm install wafie wafie/wafie
```

Check all wafie pods are running 
```bash
kubectl get pods -l 'app in (wafie-relay,appsecgw,wafie-control-plane)'
```

List all discovered applications and enable protections
```bash
# here we've to talk. I have no API docs yet, and it's 
# easiest to explain over a meeting about what's need to be done for 
# enabling application protections  
```
