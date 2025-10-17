SHELL := /usr/bin/env bash

shell:
	@$(RUN) /bin/bash



build:
	go build \
      -ldflags="-X 'github.com/Dimss/wafie/cmd/agent/discovery/cmd.Build=$$(git rev-parse --short HEAD)'" \
      -o bin/discovery-agent cmd/agent/discovery/main.go

	go build \
      -ldflags="-X 'github.com/Dimss/wafie/cmd/apiserver/cmd.Build=$$(git rev-parse --short HEAD)'" \
      -o bin/api-server cmd/apiserver/main.go

	go build \
		-ldflags="-X 'github.com/Dimss/wafie/appsecgw/cmd.Build=$$(git rev-parse --short HEAD)'" \
		-o bin/appsecgw appsecgw/cmd/main.go

#	go build \
#		-ldflags="-X 'github.com/Dimss/wafie/cmd/gwsupervisor/cmd.Build=$$(git rev-parse --short HEAD)'" \
#		-o bin/gwsupervisor cmd/gwsupervisor/main.go

build-api:
	go build \
      -ldflags="-X 'github.com/Dimss/wafie/cmd/apiserver/cmd.Build=$$(git rev-parse --short HEAD)'" \
      -o bin/api-server cmd/apiserver/main.go


build-discovery:
	go build \
      -ldflags="-X 'github.com/Dimss/wafie/cmd/agent/discovery/cmd.Build=$$(git rev-parse --short HEAD)'" \
      -o bin/discovery-agent cmd/agent/discovery/main.go


build-cni:
	go build -o bin/wafie-cni cni/cmd/wafie-cni/main.go

build-relay:
	go build -o bin/wafie-relay relay/cmd/main.go

docker-wafie-control-plane:
	podman buildx build -t docker.io/dimssss/wafie-control-plane --platform linux/arm64 -f dockerfiles/Dockerfile_wafie_control_plane .
	podman push docker.io/dimssss/wafie-control-plane

docker-appsecgw:
	podman buildx build --build-arg ARCH=arm64 --push -t dimssss/wafie-appsecgw --platform linux/arm64 -f dockerfiles/appsecgw/Dockerfile .

docker-relay:
	podman buildx build --build-arg ARCH=arm64 -t docker.io/dimssss/wafie-relay --platform linux/arm64 -f dockerfiles/relay/Dockerfile .

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



foo:
	@echo "The current shell is: $(SHELL)"
	@echo "Bash version is: $$BASH_VERSION"
	@echo "$(shell pwd)"
