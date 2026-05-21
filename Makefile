VERSION := v1.3.0-rc0

# Name of this service/application
SERVICE_NAME := redis-operator

# Docker image name for this project
IMAGE_NAME := spotahome/$(SERVICE_NAME)

# Repository url for this project
REPOSITORY := quay.io/$(IMAGE_NAME)

# Shell to use for running scripts
SHELL := $(shell which bash)

# Get docker path or an empty string
DOCKER := $(shell command -v docker)

# Get the main unix group for the user running make (to be used by docker-compose later)
GID := $(shell id -g)

# Get the unix user id for the user running make (to be used by docker-compose later)
UID := $(shell id -u)

# Commit hash from git
COMMIT=$(shell git rev-parse HEAD)
GITTAG_COMMIT := $(shell git rev-list --tags --max-count=1)
GITTAG := $(shell git describe --abbrev=0 --tags ${GITTAG_COMMIT} 2>/dev/null || true)

# Branch from git
BRANCH=$(shell git rev-parse --abbrev-ref HEAD)

TAG := $(GITTAG)
ifneq ($(COMMIT), $(GITTAG_COMMIT))
    TAG := $(COMMIT)
endif

ifneq ($(shell git status --porcelain),)
    TAG := $(TAG)-dirty
endif


PROJECT_PACKAGE := github.com/spotahome/redis-operator
CODEGEN_IMAGE := ghcr.io/slok/kube-code-generator:v0.9.0
PORT := 9710

# CMDs
UNIT_TEST_CMD := go test `go list ./... | grep -v /vendor/` -v
GO_GENERATE_CMD := go generate `go list ./... | grep -v /vendor/`
GO_INTEGRATION_TEST_CMD := go test `go list ./... | grep test/integration` -v -tags='integration'
GET_DEPS_CMD := dep ensure
UPDATE_DEPS_CMD := dep ensure
MOCKS_CMD := go generate ./mocks

# environment dirs
DEV_DIR := docker/development
APP_DIR := docker/app

# workdir
WORKDIR := /go/src/github.com/spotahome/redis-operator

# The default action of this Makefile is to build the development docker image
.PHONY: default
default: build

# Run the development environment in non-daemonized mode (foreground)
.PHONY: docker-build
docker-build: deps-development
	docker build \
		--build-arg uid=$(UID) \
		-t $(REPOSITORY)-dev:latest \
		-t $(REPOSITORY)-dev:$(COMMIT) \
		-f $(DEV_DIR)/Containerfile \
		.

# Run a shell into the development docker image
.PHONY: shell
shell: docker-build
	docker run -ti --rm -v ~/.kube:/.kube:ro -v $(PWD):$(WORKDIR) -u $(UID):$(UID) --name $(SERVICE_NAME) -p $(PORT):$(PORT) $(REPOSITORY)-dev /bin/bash

# Build redis-failover executable file
.PHONY: build
build: docker-build
	docker run -ti --rm -v $(PWD):$(WORKDIR) -u $(UID):$(UID) --name $(SERVICE_NAME) $(REPOSITORY)-dev ./scripts/build.sh

# Run the development environment in the background
.PHONY: run
run: docker-build
	docker run -ti --rm -v ~/.kube:/.kube:ro -v $(PWD):$(WORKDIR) -u $(UID):$(UID) --name $(SERVICE_NAME) -p $(PORT):$(PORT) $(REPOSITORY)-dev ./scripts/run.sh

# Build the production image based on the public one
.PHONY: image
image: deps-development
	docker build \
	-t $(SERVICE_NAME) \
	-t $(REPOSITORY):latest \
	-t $(REPOSITORY):$(COMMIT) \
	-t $(REPOSITORY):$(BRANCH) \
	-f $(APP_DIR)/Containerfile \
	.

.PHONY: image-release
image-release:
	docker buildx build \
	--platform linux/amd64,linux/arm64,linux/arm/v7 \
	--push \
	--build-arg VERSION=$(TAG) \
	-t $(REPOSITORY):latest \
	-t $(REPOSITORY):$(COMMIT) \
	-t $(REPOSITORY):$(TAG) \
	-f $(APP_DIR)/Containerfile \
	.

.PHONY: testing
testing: image
	docker push $(REPOSITORY):$(BRANCH)

.PHONY: tag
tag:
	git tag $(VERSION)

.PHONY: publish
publish:
	@COMMIT_VERSION="$$(git rev-list -n 1 $(VERSION))"; \
	docker tag $(REPOSITORY):"$$COMMIT_VERSION" $(REPOSITORY):$(VERSION)
	docker push $(REPOSITORY):$(VERSION)
	docker push $(REPOSITORY):latest

.PHONY: release
release: tag image-release

# Test stuff in dev
.PHONY: unit-test
unit-test: docker-build
	docker run -ti --rm -v $(PWD):$(WORKDIR) -u $(UID):$(UID) --name $(SERVICE_NAME) $(REPOSITORY)-dev /bin/sh -c '$(UNIT_TEST_CMD)'

.PHONY: ci-unit-test
ci-unit-test:
	$(UNIT_TEST_CMD)

.PHONY: ci-integration-test
ci-integration-test:
	$(GO_INTEGRATION_TEST_CMD)

.PHONY: integration-test
integration-test:
	./scripts/integration-tests.sh

.PHONY: helm-test
helm-test:
	./scripts/helm-tests.sh

# Run all tests
.PHONY: test
test: ci-unit-test ci-integration-test helm-test

.PHONY: go-generate
go-generate: docker-build
	docker run -ti --rm -v $(PWD):$(WORKDIR) -u $(UID):$(UID) --name $(SERVICE_NAME) $(REPOSITORY)-dev /bin/sh -c '$(GO_GENERATE_CMD)'

.PHONY: generate
generate: go-generate

.PHONY: get-deps
get-deps: docker-build
	docker run -ti --rm -v $(PWD):$(WORKDIR) -u $(UID):$(UID) --name $(SERVICE_NAME) $(REPOSITORY)-dev /bin/sh -c '$(GET_DEPS_CMD)'

.PHONY: update-deps
update-deps: docker-build
	docker run -ti --rm -v $(PWD):$(WORKDIR) -u $(UID):$(UID) --name $(SERVICE_NAME) $(REPOSITORY)-dev /bin/sh -c '$(UPDATE_DEPS_CMD)'

.PHONY: mocks
mocks: docker-build
	docker run -ti --rm -v $(PWD):$(WORKDIR) -u $(UID):$(UID) --name $(SERVICE_NAME) $(REPOSITORY)-dev /bin/sh -c '$(MOCKS_CMD)'

.PHONY: deps-development
# Test if the dependencies we need to run this Makefile are installed
deps-development:
ifndef DOCKER
	@echo "Docker is not available. Please install docker"
	@exit 1
endif

# Generate Kubernetes deepcopy and client code.
#
# slok/kube-code-generator v0.x rewrote its CLI: the older update-*.sh entry
# scripts are gone, replaced by a single Go binary configured with flags.
# The image ships an older Go toolchain than our go.mod requires, so we
# set GOTOOLCHAIN=auto to let it download a matching one on demand.
.PHONY: update-codegen
update-codegen:
	@echo ">> Generating Kubernetes deepcopy and client code..."
	docker run --rm \
		--env GOTOOLCHAIN=auto \
		--volume $(PWD):/go/src/$(PROJECT_PACKAGE) \
		--workdir /go/src/$(PROJECT_PACKAGE) \
		$(CODEGEN_IMAGE) \
		--apis-in=./api \
		--go-gen-out=./client/k8s \
		--client-package="$(PROJECT_PACKAGE)/client/k8s"

# Generate the CRD manifest from the +kubebuilder markers in ./api.
.PHONY: generate-crd
generate-crd:
	@echo ">> Generating CRD manifest..."
	docker run --rm \
		--env GOTOOLCHAIN=auto \
		--volume $(PWD):/go/src/$(PROJECT_PACKAGE) \
		--workdir /go/src/$(PROJECT_PACKAGE) \
		$(CODEGEN_IMAGE) \
		--apis-in=./api \
		--crd-gen-out=./manifests
	cp -f manifests/databases.spotahome.com_redisfailovers.yaml manifests/kustomize/base/
	cp -f manifests/databases.spotahome.com_redisfailovers.yaml charts/redisoperator/crds/
