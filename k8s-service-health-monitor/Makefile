.PHONY: help build build-controller build-dashboard test docker-build docker-push deploy install uninstall manifests generate fmt vet

# Image URL to use all building/pushing image targets
IMG ?= k8s-service-health-monitor:latest
CONTROLLER_IMG ?= $(IMG)
DASHBOARD_IMG ?= $(IMG)

# Get the currently used golang install path (in GOPATH/bin, unless GOBIN is set)
ifeq (,$(shell go env GOBIN))
GOBIN=$(shell go env GOPATH)/bin
else
GOBIN=$(shell go env GOBIN)
endif

# Setting SHELL to bash allows bash commands to be executed by recipes.
SHELL = /usr/bin/env bash -o pipefail
.SHELLFLAGS = -ec

##@ General

help: ## Display this help.
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z_0-9-]+:.*?##/ { printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)

##@ Development

fmt: ## Run go fmt against code.
	go fmt ./...

vet: ## Run go vet against code.
	go vet ./...

test: fmt vet ## Run tests.
	go test ./... -coverprofile cover.out

##@ Build

build: build-controller build-dashboard ## Build controller and dashboard binaries.

build-controller: fmt vet ## Build controller binary.
	go build -o bin/controller cmd/controller/main.go

build-dashboard: fmt vet ## Build dashboard binary.
	go build -o bin/dashboard cmd/dashboard/main.go

run-controller: fmt vet ## Run controller from your host.
	go run cmd/controller/main.go

run-dashboard: fmt vet ## Run dashboard from your host.
	go run cmd/dashboard/main.go

##@ Docker

docker-build: ## Build docker image.
	docker build -t ${IMG} .

docker-push: ## Push docker image.
	docker push ${IMG}

##@ Deployment

install: manifests ## Install CRDs into the K8s cluster.
	kubectl apply -f config/crd/

uninstall: manifests ## Uninstall CRDs from the K8s cluster.
	kubectl delete -f config/crd/

deploy: manifests ## Deploy controller and dashboard to the K8s cluster.
	kubectl apply -f config/rbac/
	kubectl apply -f config/manager/

undeploy: ## Undeploy controller and dashboard from the K8s cluster.
	kubectl delete -f config/manager/
	kubectl delete -f config/rbac/

##@ Code Generation

manifests: ## Generate CRD manifests.
	@echo "CRD manifests are manually maintained in config/crd/"
	@echo "Use controller-gen if you have it installed:"
	@echo "  controller-gen rbac:roleName=manager-role crd paths=\"./...\" output:crd:artifacts:config=config/crd"

generate: ## Generate code (DeepCopy, etc).
	@echo "Code generation requires controller-gen"
	@echo "  controller-gen object:headerFile=\"hack/boilerplate.go.txt\" paths=\"./...\""
