build:
	go build \
      -ldflags="-X 'github.com/Dimss/cwaf/cmd/agent/discovery/cmd.Build=$$(git rev-parse --short HEAD)'" \
      -o bin/cwaf-discovery-agent cmd/agent/discovery/main.go

	go build \
      -ldflags="-X 'github.com/Dimss/cwaf/cmd/apiserver/cmd.Build=$$(git rev-parse --short HEAD)'" \
      -o bin/api-server cmd/apiserver/main.go


proto:
	cd api \
	&& buf dep update \
	&& buf export buf.build/googleapis/googleapis --output vendor \
	&& buf generate
