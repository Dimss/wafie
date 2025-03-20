build:
	go build \
      -ldflags="-X 'github.com/Dimss/cwaf/cmd/agent/discovery/cmd.Build=$$(git rev-parse --short HEAD)'" \
      -o bin/cwaf-discovery-agent cmd/agent/discovery/main.go


buf:
	cd api \
	&& buf dep update \
	&& buf export buf.build/googleapis/googleapis --output vendor \
	&& buf generate
