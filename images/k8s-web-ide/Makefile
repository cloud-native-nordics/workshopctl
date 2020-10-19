VERSION=v2.0.0
all: build
build:
	docker build --pull -t luxas/k8s-web-ide:$(VERSION) .

push: build
	docker push luxas/k8s-web-ide:$(VERSION)

run:
	# Sample usage locally
	docker run -it \
		-p 127.0.0.1:8080:8080 \
		-e GIT_REPO=https://github.com/luxas/workshopctl \
		-v /var/run/docker.sock:/var/run/docker.sock \
		-e PASSWORD=test1234 \
		luxas/k8s-web-ide:$(VERSION)
