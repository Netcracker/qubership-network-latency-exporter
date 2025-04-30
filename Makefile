SHELL=/usr/bin/env bash -o pipefail

# Get the currently used golang install path (in GOPATH/bin, unless GOBIN is set)
ifeq (,$(shell go env GOBIN))
	GOBIN=$(shell go env GOPATH)/bin
else
	GOBIN=$(shell go env GOBIN)
endif

#############
# Constants #
#############

# Directories and files
BUILD_DIR=build

# Set build version
ARTIFACT_NAME="qubership-network-latency-exporter"
VERSION?=2.7.0

# Detect the build environment, local or Jenkins builder
BUILD_DATE=$(shell date +"%Y%m%d-%T")
ifndef JENKINS_URL
	BUILD_USER?=$(USER)
	BUILD_BRANCH?=$(shell git branch --show-current)
	BUILD_REVISION?=$(shell git rev-parse --short HEAD)
else
	BUILD_USER=$(BUILD_USER)
	BUILD_BRANCH=$(LOCATION:refs/heads/%=%)
	BUILD_REVISION=$(REPO_HASH)
endif

# The import path
NETWORK_LATENCY_EXPORTER_PKG=github.com/Netcracker/network-latency-exporter

# The ldflags for the go build process to set the version related data.
GO_BUILD_LDFLAGS=\
	-s \
	-X $(NETWORK_LATENCY_EXPORTER_PKG)/version.Revision=$(BUILD_REVISION) \
	-X $(NETWORK_LATENCY_EXPORTER_PKG)/version.BuildUser=$(BUILD_USER) \
	-X $(NETWORK_LATENCY_EXPORTER_PKG)/version.BuildDate=$(BUILD_DATE) \
	-X $(NETWORK_LATENCY_EXPORTER_PKG)/version.Branch=$(BUILD_BRANCH) \
	-X $(NETWORK_LATENCY_EXPORTER_PKG)/version.Version=$(VERSION)

# Go build flags
GO_BUILD_RECIPE=\
	GOOS=$(GOOS) \
	GOARCH=$(GOARCH) \
	CGO_ENABLED=0 \
	go build -ldflags="$(GO_BUILD_LDFLAGS)"

# Default test arguments
TEST_RUN_ARGS=-vet=off --shuffle=on

# List of packages
pkgs = $(shell go list ./...)

# Container name
CONTAINER_CLI?=docker
CONTAINER_NAME="qubership-network-latency-exporter"
DOCKERFILE=Dockerfile

###########
# Generic #
###########

# Default run without arguments
.PHONY: all
all: test lint build-binary image

# Run only build
.PHONY: build
build: build-binary image

# Run only build inside the Dockerfile
.PHONY: build-image
build-image: image

# Remove all files and directories ignored by git
.PHONY: clean
clean:
	echo "=> Cleanup repository ..."
	git clean -Xfd .

#########
# Build #
#########

# Build binary
.PHONY: build-binary
build-binary: fmt vet
	echo "=> Build binary ..."
	rm -rf ${BUILD_DIR}
	mkdir -p ${BUILD_DIR}
	$(GO_BUILD_RECIPE) -a -o ${BUILD_DIR}/network-latency-exporter ./cmd/

# Run go fmt against code
.PHONY: fmt
fmt:
	echo "=> Formatting Golang code ..."
	go fmt ./...

# Run go vet against code
.PHONY: vet
vet:
	echo "=> Examines Golang code ..."
	go vet ./...

###############
# Build image #
###############

.PHONY: image
image:
	echo "=> Build image ..."
	docker build --pull -t $(CONTAINER_NAME) -f $(DOCKERFILE) .

	# Set image tag if build inside the Jenkins
	for id in $(DOCKER_NAMES) ; do \
		docker tag $(CONTAINER_NAME) "$$id"; \
	done

###########
# Testing #
###########

.PHONY: test
test: unit-test

# Run unit tests in all packages
.PHONY: unit-test
unit-test:
	echo "=> Run Golang unit-tests ..."
	go test -race $(TEST_RUN_ARGS) $(pkgs) -count=1 -v

##########################
# Running linter locally #
##########################

.PHONY: lint
lint:
	echo "=> Run linter ..."
	docker run \
		-e RUN_LOCAL=true \
		-e DEFAULT_BRANCH=$(shell git rev-parse --abbrev-ref HEAD) \
		--env-file .github/super-linter.env \
		-v ${PWD}:/tmp/lint \
		--rm \
		ghcr.io/super-linter/super-linter:slim-v7.3.0

###################
# Running locally #
###################

# Run against the configured Kubernetes cluster in ~/.kube/config
.PHONY: run
run: fmt vet
	echo "=> Run ..."
	go run ./cmd/main.go
