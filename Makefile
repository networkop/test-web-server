GOBIN ?= ${GOPATH}/bin
BINARY ?= $(shell basename $$PWD)
REGISTRY ?= networkop/echo-server
DOCKER_IMAGE ?= $(REGISTRY)/$(BINARY)
SOURCES := $(shell find . -name '*.go')

.DEFAULT_GOAL := $(BINARY)

$(BINARY): $(SOURCES)
	CGO_ENABLED=0 go build -ldflags "-X main.version=$(VERSION) -extldflags -static" .

docker_build: 
	docker build \
		--build-arg BUILD_DATE=`date -u +"%Y-%m-%dT%H:%M:%SZ"` \
		--build-arg VERSION=$(VERSION) \
		--build-arg VCS_REF=$(GIT_COMMIT) \
		-t $(DOCKER_IMAGE)  .

docker_push:
	docker push $(DOCKER_IMAGE):latest

docker_push_tagged:
	docker tag $(DOCKER_IMAGE) $(DOCKER_IMAGE):$(DOCKER_TAG)
	docker push $(DOCKER_IMAGE):$(DOCKER_TAG)

install:
	GOBIN=$(GOBIN) CGO_ENABLED=0 go install -ldflags "-X main.version=$(VERSION) -extldflags -static" .

clean:
	rm echo-server
	go get -u ./...

.PHONY: download
download:
	curl -o download http://localhost:8080/download?sizeMB=100

upload:
	curl -F 'test-download=@./download' http://localhost:8080/upload

