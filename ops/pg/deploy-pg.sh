helm install cwaf-pg oci://registry-1.docker.io/bitnamicharts/postgresql \
  --set auth.postgresPassword=cwafpg \
  --set auth.username=cwafpg \
  --set auth.password=cwafpg \
  --set auth.database=cwaf 