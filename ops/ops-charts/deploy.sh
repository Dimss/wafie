helm install cwaf-pg oci://registry-1.docker.io/bitnamicharts/postgresql \
  --set auth.postgresPassword=cwafpg \
  --set auth.username=cwafpg \
  --set auth.password=cwafpg \
  --set auth.database=cwaf


helm install wp oci://registry-1.docker.io/bitnamicharts/wordpress \
  --set ingress.enabled=true \
  --set ingress.hostname=wp.10.100.102.84.nip.io \
  --set service.type=ClusterIP
