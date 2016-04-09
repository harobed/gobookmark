export GOPATH := $(PWD)
export GOBIN := $(PWD)/bin
export PATH := $(PWD):$(PATH)
export GO15VENDOREXPERIMENT=1

BINARY=gobookmark

VERSION=0.1.0
BUILD_TIME=`date +%FT%T%z`

build_bindata = bin/go-bindata $(1) -o src/gobookmark/bindata.go -prefix src/gobookmark/ src/gobookmark/public/... src/gobookmark/templates/... src/gobookmark/migrations/...
build_gobookmark = go build -v -o bin/${BINARY} gobookmark

.PHONY: all
all: install serve

.PHONY: install
install:
	go get -u github.com/Masterminds/glide/...
	go get -u github.com/jteeuwen/go-bindata/...
	cd src/gobookmark/; $(PWD)/bin/glide install; $(PWD)/bin/glide rebuild
	$(call build_bindata,-debug)
	$(call build_gobookmark)

.PHONY: build
build:
	$(call build_gobookmark)

.PHONY: serve
serve:
	$(call build_gobookmark)
	bin/${BINARY}


.PHONY: assets
assets:
	$(call build_bindata,-debug)

.PHONY: release
release:
	mkdir -p releases/darwin_amd64/
	mkdir -p releases/linux_amd64/
	$(call build_bindata)
	go build -o releases/darwin_amd64/${BINARY} gobookmark
	docker run -it --rm -v $(PWD):/usr/src/myapp -w /usr/src/myapp golang:1.6 bash -c "export CGO_ENABLED=1; export GOPATH=/usr/src/myapp/; go build -ldflags '-s' -o releases/linux_amd64/${BINARY} gobookmark"
	$(call build_bindata,-debug)
	cd releases/darwin_amd64/; tar czf ../gobookmark_darwin_amd64.tar.gz gobookmark
	cd releases/linux_amd64/; tar czf ../gobookmark_linux_amd64.tar.gz gobookmark

.PHONY: test
test:
	go test gobookmark -v

.PHONY: clean
clean:
	rm -rf bin/ releases/ pkg/ src/gobookmark/vendor/ src/gopkg.in/ src/github.com/

.PHONY: build-docker
build-docker: release
	docker build -t gobookmark .

.PHONY: push-docker
push-docker: build-docker 
	docker tag gobookmark:latest docker.santa-maria.io/stephane/gobookmark
	docker push docker.santa-maria.io/stephane/gobookmark
