THIS_FILE := $(lastword $(MAKEFILE_LIST))
HOSTOS := $(shell uname -s)
export GOOS ?= $(shell go env GOOS)
export GOARCH ?= $(shell go env GOARCH)
export CGO_ENABLED ?= 0
export GOPATH := $(shell go env GOPATH)
PLATFORMS ?= linux/amd64,linux/arm64
GOMINVERSION := $(shell go version | cut -f3 -d\ | cut -f2 -d.)
GOGETCMD := $(shell test $(GOMINVERSION) -gt 15 && echo install || echo get)
OUTPUT := build/bin
CONTROLLER_NAME := pelagia-ceph
CONTROLLER_CMD := ./cmd/controller
DISK_DAEMON_NAME := pelagia-disk-daemon
DISK_DAEMON_CMD := ./cmd/disk-daemon
CONNECTOR_NAME := pelagia-connector
CONNECTOR_CMD := ./cmd/connector
CEPH_E2E_NAME := pelagia-e2e
SKIP_SNAPSHOT_CONTROLLER ?= ""
SKIP_ROOK_CRDS ?= ""
CURRENT_RELEASE_VERSION := "2.0.0"
CODE_VERSION := $(shell build/scripts/get_version.sh)
DEV_VERSION ?= "dev-$(CODE_VERSION)"
VERSION := $(shell build/scripts/get_version.sh $(CURRENT_RELEASE_VERSION) $(DEV_VERSION))
E2E_TESTLIST_LOCAL ?= $(shell ls ./test/e2e/ | grep _test.go | grep -v entrypoint_test | xargs printf "./test/e2e/%s " $1)
LDFLAGS := "-X 'github.com/Mirantis/pelagia/version.Version=${VERSION}'"
IMAGE_NAME ?= localdocker:5000/$(CONTROLLER_NAME)
E2E_IMAGE_NAME ?= localdocker:5000/$(CEPH_E2E_NAME)
IMAGE_TAG ?= $(VERSION)
PUSH_ON_BUILD ?= true
IMAGE_MULTIBUILD_OUTPUT ?= type=image,name=$(IMAGE_NAME):$(IMAGE_TAG),push=$(PUSH_ON_BUILD)
IMAGE_E2E_MULTIBUILD_OUTPUT ?= type=image,name=$(E2E_IMAGE_NAME):$(IMAGE_TAG),push=$(PUSH_ON_BUILD)
HELM_REGISTRY ?= oci://localhost/pelagia/pelagia-ceph

#============#
# Helm stuff #
#============#

.PHONY: pelagia-ceph-chart rook-crds-chart snapshot-controller-chart publish-chart
pelagia-ceph-chart: snapshot-controller-chart rook-crds-chart ## Build helm package
	@printf "\n=== PACKAGING PELAGIA-CEPH CHART ===\n"
	@cp charts/pelagia-ceph/Chart.yaml charts/pelagia-ceph/.Chart.yaml.bckp
	@if [ -n $(SKIP_SNAPSHOT_CONTROLLER) ]; then \
		printf "\n=== REMOVING SNAPSHOT-CONTROLLER DEPENDENCY ===\n"; \
		sed -i '/- name: snapshot-controller/,+2d' charts/pelagia-ceph/Chart.yaml ; \
	fi
	@if [ -n $(SKIP_ROOK_CRDS) ]; then \
		printf "\n=== REMOVING ROOK-CRDS DEPENDENCY ===\n"; \
		sed -i '/- name: rook-crds/,+2d' charts/pelagia-ceph/Chart.yaml ; \
	fi
	@if [ -z $(SKIP_SNAPSHOT_CONTROLLER) -o -z $(SKIP_ROOK_CRDS) ]; then \
		sed -i 's/^  version:.*$$/  version: $(VERSION)/g' charts/pelagia-ceph/Chart.yaml ; \
	else \
		sed -i '/^dependencies:/,$$d' charts/pelagia-ceph/Chart.yaml ; \
	fi
	helm lint charts/pelagia-ceph
	helm package charts/pelagia-ceph --version $(VERSION) --app-version $(VERSION)
	@mv charts/pelagia-ceph/.Chart.yaml.bckp charts/pelagia-ceph/Chart.yaml

rook-crds-chart: ## Build helm package with rook-crds deps
	@if [ -z $(SKIP_ROOK_CRDS) ]; then \
		printf "\n=== PACKAGING ROOK-CRDS CHART ===\n"; \
		helm lint charts/rook-crds; \
		helm package charts/rook-crds --version $(VERSION) -d charts/pelagia-ceph/charts; \
	else \
		printf "\n=== PACKAGING ROOK-CRDS CHART HAS BEEN SKIPPED ===\n"; \
	fi

snapshot-controller-chart: ## Build helm package with snapshot-controller deps
	@if [ -z $(SKIP_SNAPSHOT_CONTROLLER) ]; then \
		printf "\n=== PACKAGING SNAPSHOT-CONTROLLER CHART ===\n"; \
		helm lint charts/snapshot-controller; \
		helm package charts/snapshot-controller --version $(VERSION) -d charts/pelagia-ceph/charts; \
	else \
		printf "\n=== PACKAGING SNAPSHOT-CONTROLLER CHART HAS BEEN SKIPPED ===\n"; \
	fi

publish-chart: ## Push chart to helm registry
	helm push pelagia-ceph-$(VERSION).tgz $(HELM_REGISTRY)

#================#
# Go build stuff #
#================#
comma := ,

.PHONY: go.build.all go.build
go.build/controller.%: vendor ## Build 'controller' binary for platform
	env GOARCH=$(word 2, $(subst /, ,$*)) GOOS=$(word 1, $(subst /, ,$*)) go build -o $(OUTPUT)/$*/$(CONTROLLER_NAME) -trimpath -ldflags $(LDFLAGS) -mod=vendor $(CONTROLLER_CMD)

go.build/disk-daemon.%: vendor ## Build 'disk-daemon' binary for platform
	env GOARCH=$(word 2, $(subst /, ,$*)) GOOS=$(word 1, $(subst /, ,$*)) go build -o $(OUTPUT)/$*/$(DISK_DAEMON_NAME) -trimpath -ldflags $(LDFLAGS) -mod=vendor $(DISK_DAEMON_CMD)

go.build/connector.%: vendor ## Build 'connector' binary for platform
	env GOARCH=$(word 2, $(subst /, ,$*)) GOOS=$(word 1, $(subst /, ,$*)) go build -o $(OUTPUT)/$*/$(CONNECTOR_NAME) -trimpath -ldflags $(LDFLAGS) -mod=vendor $(CONNECTOR_CMD)

go.build/e2e.%: vendor ## Build 'e2e' binary for platform 
	@go clean -testcache
	env GOOS=linux GOARCH=$(word 2, $(subst /, ,$*)) go test -test.v -timeout 0 -ldflags $(LDFLAGS) $(E2E_TESTLIST_LOCAL) -c -o $(OUTPUT)/$*/$(CEPH_E2E_NAME)

go.build/%: ## Build binary (controller, disk-daemon, e2e, connector) for all specified platforms
	@$(MAKE) $(foreach plat,$(subst $(comma), ,$(PLATFORMS)), go.build/$*.$(plat))

go.build: go.build/controller go.build/disk-daemon go.build/connector ## Build Pelagia (controller, disk-daemon, connector) binaries for all specified platforms

go.build/all: go.build go.build/e2e ## Build all binaries for all specified platforms

#===================#
# Doker build stuff #
#===================#

.PHONY: docker.build docker.build/controller docker.build/e2e docker.publish/controller docker.publish/e2e
docker.build/controller.%: go.build/controller.% go.build/disk-daemon.% go.build/connector.% ## Build Pelagia docker image for platform
	docker image build --build-arg REF=$(CODE_VERSION) --platform $* -t $(IMAGE_NAME):$(IMAGE_TAG) -f controller.Dockerfile .

docker.build/controller: go.build ## Build Pelagia docker image for all specified platforms
	docker buildx build --platform $(PLATFORMS) --build-arg REF=$(CODE_VERSION) \
	    $(foreach output,$(IMAGE_MULTIBUILD_OUTPUT), --output $(output)) -f controller.Dockerfile .

docker.build/e2e.%: go.build/e2e.% ## Build Pelagia E2E docker image for platform
	docker image build --build-arg REF=$(CODE_VERSION) --platform $* -t $(E2E_IMAGE_NAME):$(IMAGE_TAG) -f e2e.Dockerfile .

docker.build/e2e: go.build/e2e ## Build Pelagia E2E docker image for all specified platforms
	docker buildx build --platform $(PLATFORMS) --build-arg REF=$(CODE_VERSION) \
		$(foreach output,$(IMAGE_E2E_MULTIBUILD_OUTPUT), --output $(output)) -f e2e.Dockerfile .

docker.build: docker.build/controller docker.build/e2e ## Build all docker images for all platforms

docker.publish/controller: ## Publish Pelagia single docker image to its registry.
	docker push $(IMAGE_NAME):$(IMAGE_TAG)
docker.publish/e2e: ## Publish Pelagia E2E single docker image to its registry.
	docker push $(E2E_IMAGE_NAME):$(IMAGE_TAG)

docker.copy.%: ## Copy existing multiarch image to new registry
	@if [ -z $(NEW_IMAGE_NAME) ]; then \
		printf "\n=== Failed to create new manifest, NEW_IMAGE_NAME var is not specified ===\n"; \
		exit 1 ; \
	fi
	@if [ "$*" = "controller" ]; then \
		docker buildx imagetools create -t $(NEW_IMAGE_NAME):$(IMAGE_TAG) $(IMAGE_NAME):$(IMAGE_TAG) ; \
	else \
		docker buildx imagetools create -t $(NEW_IMAGE_NAME):$(IMAGE_TAG) $(E2E_IMAGE_NAME):$(IMAGE_TAG) ; \
	fi

#===============#
# Cleanup stuff #
#===============#

.PHONY: clean clean-all clean-docker
clean: ## Clean built objects and vendor libraries.
	@rm -rf $(OUTPUT)
	@rm -rf vendor
	@rm -f pelagia-ceph-*.tgz
	@rm -rf charts/pelagia-ceph/charts/
clean-docker: ## Clean built docker images.
	docker image ls --filter label=org.opencontainers.image.source="https://github.com/Mirantis/pelagia" -q | xargs docker image rm
clean-all: clean clean-docker ## Clean everything.

#====================#
# CI and tests stuff #
#====================#

.PHONY: get-version
get-version:
	@printf $(VERSION)

golangci-lint-install:
ifeq (,$(shell golangci-lint version 2>/dev/null))
	@printf "\n=== <INSTALL GO-LINT> ===\n"
	@echo Missing golangci-lint. Going to install if for $(HOSTOS).
ifeq ("Linux","$(HOSTOS)")
	$(shell wget -O- -nv https://raw.githubusercontent.com/golangci/golangci-lint/v1.62.0/install.sh | bash -s -- -b $(GOPATH))
endif
ifeq ("Darwin","$(HOSTOS)")
	brew install golangci/tap/golangci-lint
	brew upgrade golangci/tap/golangci-lint
endif
endif

.PHONY: lint golangci-lint-install
lint: golangci-lint-install vendor ## Run go lint
	@printf "\n=== <PROCESS GOLANGCI-LINT> ===\n"
	$(GOPATH)/golangci-lint run -v $(CURDIR)/...

.PHONY: vendor
vendor: ## Update vendor libraries.
	@printf "\n=== <PROCESS GO MOD> ===\n"
	@go mod tidy
	@go mod vendor

.PHONY: unit coverage job-coverage
unit: ## Run go unit
	@printf "\n=== <PROCESS GO UNIT> ===\n"
	@go clean -testcache
	go test -test.v -timeout 0 "./pkg/..."

.PHONY: coverage
coverage: ## Run go coverage
	@printf "\n=== <PROCESS GO COVERAGE> ===\n"
	go test -coverprofile=coverage.out "./pkg/..."
	go tool cover -html=coverage.out
	@rm coverage.out

.PHONY: job-coverage
job-coverage: ## Run go coverage and export to html file
	@printf "\n=== <PROCESS GO JOB COVERAGE> ===\n"
	go test -coverprofile=coverage.out "./pkg/..."
	go tool cover -html=coverage.out -o coverage.html
	@rm coverage.out

.PHONY: fmt check fix
fmt: vendor ## Run go fmt
	@printf "\n=== <PROCESS GO FMT> ===\n"
	go fmt ./pkg/... ./cmd/... ./test/...

check: fmt generate lint ## Run git diff check

.PHONY: check-diff
check-diff:
	@printf "\n=== <PROCESS GIT DIFF> ===\n"
	git diff --exit-code ":(exclude)github-password"

.PHONY: fix
fix: golangci-lint-install vendor ## Run go fix
	@printf "\n=== <PROCESS GO FIX> ===\n"
	$(GOPATH)/golangci-lint run -v --fix ./...

include e2e.mk
.PHONY: e2e-code
e2e-code: ## Run e2e tests
	@printf "\n=== <PROCESS E2E TESTS> ===\n"
	@go clean -testcache
	@echo "Using ${E2E_TESTCONFIG} configset.."
	go test -test.v -timeout 0 ${E2E_TESTLIST_LOCAL}

#
### Go generators stuff
#

.PHONY: install-controller-gen install-client-gen install-lister-gen install-informer-gen
install-controller-gen:
	go $(GOGETCMD) sigs.k8s.io/controller-tools/cmd/controller-gen@v0.16.1

install-client-gen:
	go $(GOGETCMD) k8s.io/code-generator/cmd/client-gen@v0.27.7

install-lister-gen:
	go $(GOGETCMD) k8s.io/code-generator/cmd/lister-gen@v0.27.7

install-informer-gen:
	go $(GOGETCMD) k8s.io/code-generator/cmd/informer-gen@v0.27.7

.PHONY: generate client-gen controller-go-generate client-go-generate lister-go-generate informer-go-generate copy-client-gen-output go-generate
generate: go-generate install-controller-gen controller-go-generate ## Generate API and CRDs

client-gen: install-client-gen install-lister-gen install-informer-gen client-go-generate lister-go-generate informer-go-generate ## Generate clients code

controller-go-generate: vendor
	@printf "\n=== <PROCESS CONTROLLER-GEN> ===\n"
	$(GOPATH)/bin/controller-gen object +paths=./pkg/apis/... +paths=./cmd/...
	$(GOPATH)/bin/controller-gen crd:allowDangerousTypes=true +paths=./pkg/apis/ceph.pelagia.lcm/... +output:dir=charts/pelagia-ceph/templates/crds

client-go-generate: vendor
	@printf "\n=== <PROCESS CLIENT-GEN> ===\n"
	$(GOPATH)/bin/client-gen --clientset-name versioned --input-base "" \
	   --input "github.com/Mirantis/pelagia/pkg/apis/ceph.pelagia.lcm/v1alpha1" \
	   --output-package "github.com/Mirantis/pelagia/pkg/client/clientset" \
	   -h boilerplate.go.txt

lister-go-generate: vendor
	@printf "\n=== <PROCESS LISTER-GEN> ===\n"
	$(GOPATH)/bin/lister-gen \
		--input-dirs "github.com/Mirantis/pelagia/pkg/apis/ceph.pelagia.lcm/v1alpha1" \
		--output-package "github.com/Mirantis/pelagia/pkg/client/listers" \
		-h boilerplate.go.txt

informer-go-generate: vendor
	@printf "\n=== <PROCESS INFORMER-GEN> ===\n"
	$(GOPATH)/bin/informer-gen \
		--input-dirs "github.com/Mirantis/pelagia/pkg/apis/ceph.pelagia.lcm/v1alpha1" \
		--versioned-clientset-package "github.com/Mirantis/pelagia/pkg/client/clientset/versioned" \
		--listers-package "github.com/Mirantis/pelagia/pkg/client/listers" \
		--output-package "github.com/Mirantis/pelagia/pkg/client/informers" \
		-h boilerplate.go.txt

go-generate: vendor
	@printf "\n=== <PROCESS GO GENERATE> ===\n"
	go generate ./pkg/... ./cmd/...

#
# Docs stuff
#

.PHONY: docs-build
docs-build: ## Run docs buld
	@python3 -m venv build/venv; \
	 . build/venv/bin/activate; \
	 pip3 install -U -r docs/requirements.txt; \
	 mkdocs build -d build/htmldocs; \
	 (deactivate || true)

.PHONY: docs-serve
docs-serve: ## Serve built docs
	@python3 -m venv build/venv
	. build/venv/bin/activate; \
	pip install -U -r docs/requirements.txt; \
	mkdocs serve

.PHONY: help
help: ## This help.
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z\/0-9_-.%]+:.*?## / {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}' $(MAKEFILE_LIST)

# vim: set ts=4 sw=4 tw=0 noet :
