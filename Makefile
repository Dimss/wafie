SHELL := /usr/bin/env bash

shell:
	@$(RUN) /bin/bash

build:
	# build api server
	go build \
      -ldflags="-X 'github.com/Dimss/wafie/apisrv/cmd/apiserver/cmd.Build=$$(git rev-parse --short HEAD)'" \
      -o .bin/api-server apisrv/cmd/apiserver/main.go
	# build discovery agent
	go build \
      -ldflags="-X 'github.com/Dimss/wafie/discovery/cmd/cmd.Build=$$(git rev-parse --short HEAD)'" \
      -o .bin/discovery-agent discovery/cmd/discovery/main.go


build.appsecgw:
	go build \
		-ldflags="-X 'github.com/Dimss/wafie/appsecgw/cmd.Build=$$(git rev-parse --short HEAD)'" \
		-o .bin/appsecgw appsecgw/cmd/main.go

build.appsecgw.image:
	podman buildx build --build-arg ARCH=arm64 -t docker.io/dimssss/wafie-appsecgw --platform linux/arm64 -f appsecgw/Dockerfile .
	podman push docker.io/dimssss/wafie-appsecgw

build.appsecgw.image.dev:
	podman buildx build --build-arg ARCH=arm64 -t docker.io/dimssss/wafie-appsecgw-dev --platform linux/arm64 -f appsecgw/Dockerfile.dev .
	podman push docker.io/dimssss/wafie-appsecgw-dev

build-api:
	go build \
      -ldflags="-X 'github.com/Dimss/wafie/cmd/apiserver/cmd.Build=$$(git rev-parse --short HEAD)'" \
      -o .bin/api-server cmd/apiserver/main.go


build.discovery:
	go build \
      -ldflags="-X 'github.com/Dimss/wafie/discovery/cmd/discovery/cmd.Build=$$(git rev-parse --short HEAD)'" \
      -o .bin/discovery-agent discovery/cmd/discovery/main.go


build-cni:
	go build -o .bin/wafie-cni cni/cmd/wafie-cni/main.go

build.relay:
	go build -o .bin/wafie-relay relay/cmd/main.go

build.relay.image:
	podman buildx build -t docker.io/dimssss/wafie-relay --platform linux/arm64 -f relay/Containerfile .
	podman push docker.io/dimssss/wafie-relay

helm:
	helm package chart
	scp wafie-0.0.1.tgz root@charts.wafie.io:/var/www/charts
	rm wafie-0.0.1.tgz

docker.controlplane:
	podman buildx build -t docker.io/dimssss/wafie-control-plane --platform linux/arm64 -f dockerfiles/controlplane/Dockerfile .
	podman push docker.io/dimssss/wafie-control-plane



docker-relay:
	podman buildx build --build-arg ARCH=arm64 -t docker.io/dimssss/wafie-relay --platform linux/arm64 -f dockerfiles/relay/Dockerfile .

.PHONY: proto
proto:
	cd api \
	&& buf dep update \
	&& buf export buf.build/googleapis/googleapis --output vendor \
	&& buf export buf.build/bufbuild/protovalidate --output vendor \
	&& buf lint \
	&& buf generate


.PHONY: run-test-postgres stop-test-postgres

run-test-postgres:
	docker run --name test-postgres -e POSTGRES_USER=cwafpg -e POSTGRES_PASSWORD=cwafpg -e POSTGRES_DB=cwaf -p 5432:5432 -d postgres

stop-test-postgres:
	docker stop test-postgres
	docker rm test-postgres

.PHONY: chart
install.controlplane:
	cd chart && helm upgrade -i wafie .
uninstall:
	cd chart && helm delete wafie && kubectl delete pvc data-wafie-postgresql-0



foo:
	@echo "The current shell is: $(SHELL)"
	@echo "Bash version is: $$BASH_VERSION"
	@echo "$(shell pwd)"
