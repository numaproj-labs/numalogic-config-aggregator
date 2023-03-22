CURRENT_DIR=$(shell pwd)
DIST_DIR=${CURRENT_DIR}/dist

BINARY_NAME:=numalogic-config-aggregator

# docker image publishing options
DOCKER_PUSH?=false
IMAGE_NAMESPACE?=quay.io/numaio
IMAGE_TAG?=latest

DOCKERFILE:=Dockerfile

VERSION=$(shell cat ${CURRENT_DIR}/VERSION)
BUILD_DATE=$(shell date -u +'%Y-%m-%dT%H:%M:%SZ')
GIT_COMMIT=$(shell git rev-parse HEAD)
GIT_TAG=$(shell if [ -z "`git status --porcelain`" ]; then git describe --exact-match --tags HEAD 2>/dev/null; fi)
GIT_TREE_STATE=$(shell if [ -z "`git status --porcelain`" ]; then echo "clean" ; else echo "dirty"; fi)

override LDFLAGS += \
  -X ${PACKAGE}.version=${VERSION} \
  -X ${PACKAGE}.buildDate=${BUILD_DATE} \
  -X ${PACKAGE}.gitCommit=${GIT_COMMIT} \
  -X ${PACKAGE}.gitTreeState=${GIT_TREE_STATE}

ifeq (${DOCKER_PUSH},true)
ifndef IMAGE_NAMESPACE
$(error IMAGE_NAMESPACE must be set to push images (e.g. IMAGE_NAMESPACE=quay.io/numaio))
endif
endif

ifneq (${GIT_TAG},)
IMAGE_TAG=${GIT_TAG}
override LDFLAGS += -X ${PACKAGE}.gitTag=${GIT_TAG}
endif

.PHONY: test
test:
	go test $(shell go list ./... | grep -v /vendor/) -race -short -v

.PHONY: build
build: $(DIST_DIR)/$(BINARY_NAME)-linux-amd64

${DIST_DIR}/$(BINARY_NAME)-linux-amd64: GOARGS = GOOS=linux GOARCH=amd64

${DIST_DIR}/$(BINARY_NAME)-%:
	CGO_ENABLED=0 $(GOARGS) go build -v -ldflags '${LDFLAGS}' -o ${DIST_DIR}/$(BINARY_NAME) ./main.go

image: $(DIST_DIR)/$(BINARY_NAME)-linux-amd64
	docker build -t $(IMAGE_NAMESPACE)/$(BINARY_NAME):$(IMAGE_TAG)  -f $(DOCKERFILE) .
	@if [ "$(DOCKER_PUSH)" = "true" ] ; then  docker push $(IMAGE_NAMESPACE)/$(BINARY_NAME):$(IMAGE_TAG) ; fi

clean:
	-rm -rf ${CURRENT_DIR}/dist


.PHONY: manifests
manifests:
	kustomize build manifests/install > manifests/install.yaml

