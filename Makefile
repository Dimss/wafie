build:
	go build \
      -ldflags="-X 'github.com/Dimss/cwaf/cmd/agent/discovery/cmd.Build=$$(git rev-parse --short HEAD)'" \
      -o bin/discovery-agent cmd/agent/discovery/main.go

	go build \
      -ldflags="-X 'github.com/Dimss/cwaf/cmd/agent/control/cmd.Build=$$(git rev-parse --short HEAD)'" \
      -o bin/control-agent cmd/agent/control/main.go

	go build \
      -ldflags="-X 'github.com/Dimss/cwaf/cmd/apiserver/cmd.Build=$$(git rev-parse --short HEAD)'" \
      -o bin/api-server cmd/apiserver/main.go

docker:
	docker buildx build --push -t dimssss/cwaf . -f Dockerfile_cwaf_agent

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