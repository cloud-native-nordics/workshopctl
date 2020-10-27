VERSION=v2.1.0
all: build
build:
	docker build --pull -t luxas/k8s-web-ide:$(VERSION) .

push: build
	docker push luxas/k8s-web-ide:$(VERSION)

run: build
	# Sample usage locally
	docker run -it \
		-p 127.0.0.1:8080:8080 \
		-e TUTORIALS_REPO=https://github.com/cloud-native-nordics/workshopctl \
		-e TUTORIALS_DIR=tutorials \
		-e PASSWORD=test1234 \
		-v /var/run/docker.sock:/var/run/docker.sock \
		luxas/k8s-web-ide:$(VERSION)
