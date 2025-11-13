build:
	go build -o tm main.go

image:
	podman buildx build --build-arg ARCH=arm64 \
         -t docker.io/dimssss/wafie-traffic-mirror:latest \
         --platform linux/arm64 \
          -f Containerfile ../../
	podman push docker.io/dimssss/wafie-traffic-mirror:latest

helm:
	helm package chart
	scp wafie-traffic-mirror-0.0.1.tgz root@charts.wafie.io:/var/www/charts
	rm wafie-traffic-mirror-0.0.1.tgz
	ssh root@charts.wafie.io "helm repo index /var/www/charts"