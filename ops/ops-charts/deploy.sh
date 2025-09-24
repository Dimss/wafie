helm install cwaf-pg oci://registry-1.docker.io/bitnamicharts/postgresql \
  --set auth.postgresPassword=cwafpg \
  --set auth.username=cwafpg \
  --set auth.password=cwafpg \
  --set auth.database=cwaf


helm install wp oci://registry-1.docker.io/bitnamicharts/wordpress \
  --set ingress.enabled=true \
  --set ingress.hostname=wp.apps.user-rhos-01-01.servicemesh.rhqeaws.com \
  --set service.type=ClusterIP \
  --set volumePermissions.enabled=true \
  --set mariadb.volumePermissions.enabled=true




helm install wp oci://registry-1.docker.io/bitnamicharts/wordpress \
  --set ingress.enabled=true \
  --set ingress.hostname=wp.10.100.102.89.nip.io \
  --set service.type=ClusterIP


helm repo add runix https://helm.runix.net

helm install pgadmin4 runix/pgadmin4 \
  --set ingress.enabled=true \
  --set ingress.hosts[0].host=pgadmin.10.100.102.89.nip.io \
  --set ingress.hosts[0].paths[0].path="/" \
  --set ingress.hosts[0].paths[0].pathType="Prefix" \
  --set ingress.ingressClassName=nginx