build:
	podman buildx build -t docker.io/dimssss/wafie-devpod --platform linux/arm64 .
	podman push docker.io/dimssss/wafie-devpod

apply:
	kubectl apply -f devpod.yaml