PROJECT=github.com/cloud-native-nordics/workshopctl
GO_VERSION=1.16
BINARIES=workshopctl
CACHE_DIR = $(shell pwd)/bin/cache

all: build
build: $(BINARIES)

.PHONY: $(BINARIES) 
$(BINARIES):
	make shell COMMAND="make bin/$@"

generated: /go/bin/go-bindata
	# This autogenerates the file ./pkg/charts/charts.go from manifests in the ./charts directory
	# The package name of ./pkg/charts is charts, and the "charts" prefix is stripped from the beginning
	# of the file name path within the application.
	# The modification time is hardcoded to 2020-01-01 00:00:00
	go-bindata \
		-pkg=charts \
		-o=pkg/charts/charts.go \
		-modtime=1577836800 \
		-prefix=charts \
		charts/...

.PHONY: bin/workshopctl
bin/workshopctl: bin/%: generated
	$(MAKE) tidy
	CGO_ENABLED=0 go build -ldflags "$(shell ./hack/ldflags.sh)" -o bin/$* ./cmd/$*

shell:
	mkdir -p $(CACHE_DIR)/go $(CACHE_DIR)/cache
	docker run -it --rm \
		-v $(CACHE_DIR)/go:/go \
		-v $(CACHE_DIR)/cache:/.cache/go-build \
		-v $(shell pwd):/go/src/${PROJECT} \
		-w /go/src/${PROJECT} \
		-u $(shell id -u):$(shell id -g) \
		golang:$(GO_VERSION) \
		$(COMMAND)

node_modules:
	docker run -it -v $(pwd):/project -w /project node:slim npm install

tidy: /go/bin/goimports
	go mod tidy
	gofmt -s -w pkg cmd
	goimports -w pkg cmd
	go run hack/cobra.go

/go/bin/go-bindata:
	go get -u github.com/go-bindata/go-bindata/...

/go/bin/goimports:
	go get golang.org/x/tools/cmd/goimports
