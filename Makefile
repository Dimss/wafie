build:
	go build \
      -ldflags="-X 'github.com/Dimss/cwaf/cmd/agent/discovery/cmd.Build=$$(git rev-parse --short HEAD)'" \
      -o bin/cwaf-discovery-agent cmd/agent/discovery/main.go


proto:
	cd api && protoc \
    --go_out=./proto/pb \
	--go_opt=paths=source_relative \
	--go-grpc_out=./proto/pb \
	--go-grpc_opt=paths=source_relative ./proto/google/api/*.proto \
	./proto/*.proto