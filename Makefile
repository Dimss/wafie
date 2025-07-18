build:
	go build \
      -ldflags="-X 'github.com/Dimss/wafie/cmd/agent/discovery/cmd.Build=$$(git rev-parse --short HEAD)'" \
      -o bin/discovery-agent cmd/agent/discovery/main.go

	go build \
      -ldflags="-X 'github.com/Dimss/wafie/cmd/apiserver/cmd.Build=$$(git rev-parse --short HEAD)'" \
      -o bin/api-server cmd/apiserver/main.go

	go build \
		-ldflags="-X 'github.com/Dimss/wafie/cmd/gwctrl/cmd.Build=$$(git rev-parse --short HEAD)'" \
		-o bin/gwctrl cmd/gwctrl/main.go

	go build \
		-ldflags="-X 'github.com/Dimss/wafie/cmd/gwsupervisor/cmd.Build=$$(git rev-parse --short HEAD)'" \
		-o bin/gwsupervisor cmd/gwsupervisor/main.go

docker-wafie-control-plane:
	docker buildx build --push -t dimssss/wafie-control-plane --platform linux/amd64/v2 -f dockerfiles/Dockerfile_wafie_control_plane .

docker-wafie-gateway:
	docker buildx build --push -t dimssss/wafie-gateway --platform linux/amd64/v2 -f dockerfiles/Dockerfile_wafie_gateway .

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
	cd chart && helm upgrade -i wafie .
uninstall:
	cd chart && helm delete wafie && kubectl delete pvc data-wafie-postgresql-0

