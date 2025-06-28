build:
	go build \
      -ldflags="-X 'github.com/Dimss/wafie/cmd/agent/discovery/cmd.Build=$$(git rev-parse --short HEAD)'" \
      -o bin/discovery-agent cmd/agent/discovery/main.go

	go build \
      -ldflags="-X 'github.com/Dimss/wafie/cmd/apiserver/cmd.Build=$$(git rev-parse --short HEAD)'" \
      -o bin/api-server cmd/apiserver/main.go

	go build \
		-ldflags="-X 'github.com/Dimss/wafie/cmd/proxycontrolplane/cmd.Build=$$(git rev-parse --short HEAD)'" \
		-o bin/proxycontrolplane cmd/proxycontrolplane/main.go

docker-wafy:
	docker buildx build --push -t dimssss/wafy-core -f dockerfiles/Dockerfile_wafy .

docker-proxy:
	docker buildx build --push -t dimssss/wafy-proxy -f dockerfiles/Dockerfile_proxy .

.PHONY: proto
proto:
	cd api \
	&& buf dep update \
	&& buf export buf.build/googleapis/googleapis --output vendor \
	&& buf lint \
	&& buf generate


.PHONY: run-test-postgres stop-test-postgres

run-test-postgres:
	docker run --name test-postgres -e POSTGRES_USER=cwafpg -e POSTGRES_PASSWORD=cwafpg -e POSTGRES_DB=cwaf -p 5432:5432 -d postgres

stop-test-postgres:
	docker stop test-postgres
	docker rm test-postgres

.PHONY: chart
install:
	cd chart && helm upgrade -i wafy .
uninstall:
	cd chart && helm delete wafy && kubectl delete pvc data-wafy-postgresql-0

