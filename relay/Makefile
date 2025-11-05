build:
	go build -o ../.bin/wafie-relay cmd/main.go

image:
	podman buildx build -t docker.io/dimssss/wafie-relay --platform linux/arm64 .
	podman push docker.io/dimssss/wafie-relay