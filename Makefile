ifndef VERBOSE
	MAKEFLAGS += --silent
endif

PKGS=$(shell go list ./... | grep -v /vendor)
SHELL_IMAGE=golang:1.10
PWD=$(shell pwd)
GOFILES=$(shell find . -type f -name '*.go' -not -path "./vendor/*")
GOBUILD=go build -o bin/etcdproxy-controller
GIT_REF = $(shell git rev-parse --short=8 --verify HEAD)

TARGET = etcdproxy-controller
GOTARGET = github.com/xmudrii/$(TARGET)
REGISTRY ?= xmudrii
IMAGE = $(REGISTRY)/$(TARGET)
DOCKER ?= docker

default: authorsfile compile ## Create etcdproxy-controller executable in the ./bin directory and the AUTHORS file.

all: default install # Create etcdproxy-controller executable in the ./bin and $GOPATH/bin directories and the AUTHORS file.

compile: ## Create the etcdproxy-controller executable in the ./bin directory.
	CGO_ENABLED=0 ${GOBUILD} .

install: ## Create the etcdproxy-controller executable in $GOPATH/bin directory.
	install -m 0755 bin/etcdproxy-controller ${GOPATH}/bin/etcdproxy-controller

authorsfile: ## Update the AUTHORS file from the git logs
	git log --all --format='%aN <%cE>' | sort -u | egrep -v "noreply|mailchimp|@Kris" > AUTHORS

clean: ## Clean the project tree from binary files
	rm -rf bin/*

gofmt: install-tools ## Go fmt your code
	echo "Fixing format of go files..."; \
	for file in $(GOFILES); \
	do \
		gofmt -l -w $$file ; \
		goimports -l -w $$file ; \
	done

golint: install-tools ## check for style mistakes all Go files using golint
	golint $(PKGS)

govet: ## apply go vet to all the Go files
	@go vet $(PKGS)

push: ## push to the docker registry
	$(DOCKER) push $(REGISTRY)/$(TARGET):$(GIT_REF)
	$(DOCKER) push $(REGISTRY)/$(TARGET):latest

build-docker: ## build docker images
	$(DOCKER) build -t $(REGISTRY)/$(TARGET):$(GIT_REF) .
    $(DOCKER) build -t $(REGISTRY)/$(TARGET):latest .

build: authors clean build-linux-amd64 build-darwin-amd64 build-windows-amd64 ## Build etcdproxy-controller executable for Linux, macOS and Windows.

# Because of https://github.com/golang/go/issues/6376 We actually have to build this in a container
build-linux-amd64: ## Create the etcdproxy-controller executable for Linux 64-bit OS in the ./bin directory. Requires Docker.
	mkdir -p bin
	docker run \
	-u $$(id -u):$$(id -g) \
	-it \
	-w /go/src/github.com/xmudrii/etcdproxy-controller \
	-v ${PWD}:/go/src/github.com/xmudrii/etcdproxy-controller \
	-e GOPATH=/go \
	--rm ${SHELL_IMAGE} make docker-build-linux-amd64

docker-build-linux-amd64:
	${GOBUILD} -o bin/linux-amd64

build-darwin-amd64: ## Create the etcdproxy-controller executable for Darwin (osX) 64-bit OS in the ./bin directory. Requires Docker.
	GOOS=darwin GOARCH=amd64 ${GOBUILD} -o bin/darwin-amd64 &

build-windows-amd64: ## Create the etcdproxy-controller executable for Windows 64-bit OS in the ./bin directory. Requires Docker.
	GOOS=windows GOARCH=amd64${GOBUILD} -o bin/windows-amd64 &

.PHONY: test
test: ## Run tests.
	go test -timeout 20m -v $(PKGS)

.PHONY: verify-ci
verify-ci: install-tools ## Run code checks
	PKGS="${GOFILES}" GOFMT="gofmt" ./hack/verify-ci.sh

.PHONY: install-tools
install-tools:
	GOIMPORTS_CMD=$(shell command -v goimports 2> /dev/null)
ifndef GOIMPORTS_CMD
	go get golang.org/x/tools/cmd/goimports
endif
	GOLINT_CMD=$(shell command -v golint 2> /dev/null)
ifndef GOLINT_CMD
	go get github.com/golang/lint/golint
endif

.PHONY: help
help:  ## Show help messages for make targets
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[32m%-30s\033[0m %s\n", $$1, $$2}'
