THIS_FILE := $(lastword $(MAKEFILE_LIST))
HOSTOS := $(shell uname -s)
export GOOS ?= $(shell go env GOOS)
export GOARCH ?= $(shell go env GOARCH)
export CGO_ENABLED ?= 0
export GOPATH := $(shell go env GOPATH)
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
IMAGE_NAME ?= localdocker:5000/$(CONTROLLER_NAME)
E2E_IMAGE_NAME ?= localdocker:5000/$(CEPH_E2E_NAME)
IMAGE_TAG ?= latest
GERRIT_USER_NAME ?= mcp-ci-gerrit
SKIP_SNAPSHOT_CONTROLLER ?= ""
SKIP_ROOK_CRDS ?= ""
CHART_VERSION := $(shell build/scripts/get_version.sh chart)
CODE_VERSION := $(shell build/scripts/get_version.sh app)
LDFLAGS := "-X 'github.com/Mirantis/pelagia/version.Version=${CODE_VERSION}'"
GITHUB_USERNAME ?= infra-ci-user
E2E_TESTLIST_LOCAL ?= $(shell ls ./test/e2e/ | grep _test.go | grep -v entrypoint_test | xargs printf "./test/e2e/%s " $1)

# TODO(prazumovsky): add envvar for different products: kubevirt, k0rdent, MOSK, upstream. It will allow us to build different
# charts for different products. Then manage this envvar in helm build and other commands.

# (degorenko): VERSION is var for CI, used for backward compatibility
ifdef VERSION
	CODE_VERSION=$(VERSION)
	CHART_VERSION=$(VERSION)
endif

pelagia-ceph: snapshot-controller rook-crds ## Build helm package.
	@printf "\n=== PACKAGING PELAGIA-CEPH CHART ===\n"
	cp charts/pelagia-ceph/Chart.yaml charts/pelagia-ceph/.Chart.yaml.bckp
	@if [ -n $(SKIP_SNAPSHOT_CONTROLLER) ]; then \
		printf "\n=== REMOVING SNAPSHOT-CONTROLLER DEPENDENCY ===\n"; \
		sed -i '/- name: snapshot-controller/,+2d' charts/pelagia-ceph/Chart.yaml ; \
	fi
	@if [ -n $(SKIP_ROOK_CRDS) ]; then \
		printf "\n=== REMOVING ROOK-CRDS DEPENDENCY ===\n"; \
		sed -i '/- name: rook-crds/,+2d' charts/pelagia-ceph/Chart.yaml ; \
	fi
	@if [ -z $(SKIP_SNAPSHOT_CONTROLLER) -o -z $(SKIP_ROOK_CRDS) ]; then \
		sed -i 's/^  version:.*$$/  version: $(CHART_VERSION)/g' charts/pelagia-ceph/Chart.yaml ; \
	else \
		sed -i '/^dependencies:/,$$d' charts/pelagia-ceph/Chart.yaml ; \
	fi
	helm lint charts/pelagia-ceph
	helm package charts/pelagia-ceph --version $(CHART_VERSION)
	mv charts/pelagia-ceph/.Chart.yaml.bckp charts/pelagia-ceph/Chart.yaml

rook-crds:
	@if [ -z $(SKIP_ROOK_CRDS) ]; then \
		printf "\n=== PACKAGING ROOK-CRDS CHART ===\n"; \
		helm lint charts/rook-crds; \
		helm package charts/rook-crds --version $(CHART_VERSION) -d charts/pelagia-ceph/charts; \
	else \
		printf "\n=== PACKAGING ROOK-CRDS CHART HAS BEEN SKIPPED ===\n"; \
	fi

snapshot-controller:
	@if [ -z $(SKIP_SNAPSHOT_CONTROLLER) ]; then \
		printf "\n=== PACKAGING SNAPSHOT-CONTROLLER CHART ===\n"; \
		helm lint charts/snapshot-controller; \
		helm package charts/snapshot-controller --version $(CHART_VERSION) -d charts/pelagia-ceph/charts; \
	else \
		printf "\n=== PACKAGING SNAPSHOT-CONTROLLER CHART HAS BEEN SKIPPED ===\n"; \
	fi

$(OUTPUT)/$(CONTROLLER_NAME): vendor ## Build controller binary
	go build -o $@ -trimpath -ldflags $(LDFLAGS) -mod=vendor $(CONTROLLER_CMD)

$(OUTPUT)/$(DISK_DAEMON_NAME): vendor
	go build -o $@ -trimpath -ldflags $(LDFLAGS) -mod=vendor $(DISK_DAEMON_CMD)

$(OUTPUT)/$(CONNECTOR_NAME): vendor
	go build -o $@ -trimpath -ldflags $(LDFLAGS) -mod=vendor $(CONNECTOR_CMD)

.docker: $(OUTPUT)/$(CONTROLLER_NAME) $(OUTPUT)/$(DISK_DAEMON_NAME) $(OUTPUT)/$(CONNECTOR_NAME)
	@touch $@
	docker image build --platform linux/$(GOARCH) -t $(IMAGE_NAME):$(IMAGE_TAG) -f controller.Dockerfile .

.docker-e2e: build-e2e
	docker image build --platform linux/$(GOARCH) -t $(E2E_IMAGE_NAME):$(IMAGE_TAG) -f e2e.Dockerfile .

.PHONY: get-version get-code-version get-chart-version
get-version: get-chart-version
get-code-version:
	@printf $(CODE_VERSION)

get-chart-version:
	@printf $(CHART_VERSION)

.PHONY: build build-e2e image publish image-e2e publish-e2e
build: $(OUTPUT)/$(CONTROLLER_NAME) $(OUTPUT)/$(DISK_DAEMON_NAME) $(OUTPUT)/$(CONNECTOR_NAME) ## Build all binaries
build-e2e: vendor
	go clean -testcache
	env GOOS=linux go test -test.v -timeout 0 -ldflags $(LDFLAGS) $(E2E_TESTLIST_LOCAL) -c -o $(OUTPUT)/$(CEPH_E2E_NAME)
	go clean -testcache
image: .docker  ## Build docker image.
image-e2e: .docker-e2e  ## Build e2e docker image.
publish: image ## Publish docker image to its repostory.
	docker push $(IMAGE_NAME):$(IMAGE_TAG)
publish-e2e: image-e2e ## Publish docker image to its repostory.
	docker push $(E2E_IMAGE_NAME):$(IMAGE_TAG)

.PHONY: clean clean-all clean-docker
clean: ## Clean built object and vendor libraries.
	rm -f $(OUTPUT)/$(DISK_DAEMON_NAME)
	rm -f $(OUTPUT)/$(CONTROLLER_NAME)
	rm -f $(OUTPUT)/$(CONNECTOR_NAME)
	rm -f $(OUTPUT)/$(CEPH_E2E_NAME)
	rm -rf vendor
	rm -f pelagia-ceph-*.tgz
	rm -fr repository
	rm -f repositories.*
	rm -rf charts/pelagia-ceph/charts/
clean-docker: ## Clean built docker images.
	@docker images $(IMAGE_NAME):$(IMAGE_TAG) -q | xargs docker image rm
	@docker images $(E2E_IMAGE_NAME):$(IMAGE_TAG) -q | xargs docker image rm
	@rm -f .docker
clean-all: clean clean-docker ## Clean everything.

vendor: github-config ## Update vendor libraries.
	@printf "\n=== <PROCESS GO MOD> ===\n"
	@go mod tidy
	@go mod vendor

.PHONY: github-config
github-config:
	@printf "\n=== <PROCESS GITHUB CONFIG> ===\n"
	@set -ex
	@if [ -z $(shell git config --global credential."https://github.com".username) ]; then \
		git config --global credential."https://github.com".username "${GITHUB_USERNAME}"; \
	fi
	@if [ -z "$(shell git config --global credential."https://github.com".helper)" ]; then \
		git config --global credential."https://github.com".helper "! f() { echo \"password=$(shell head -1 ./github-password)\"; }; f"; \
	fi
	@if [ "$(shell git config --global safe.directory)" != "$(CURDIR)" ]; then \
		git config --global --add safe.directory "$(CURDIR)"; \
  	fi

include e2e.mk
.PHONY: e2e-code
e2e-code: github-config ## Run e2e tests
	@printf "\n=== <PROCESS E2E TESTS> ===\n"
	go clean -testcache
	echo "Using ${E2E_TESTCONFIG} configset.."
	go test -test.v -timeout 0 ${E2E_TESTLIST_LOCAL}

.PHONY: unit coverage job-coverage
unit: github-config ## Run go unit
	@printf "\n=== <PROCESS GO UNIT> ===\n"
	go test -test.v -timeout 0 "./pkg/..."

coverage: github-config ## Run go coverage
	@printf "\n=== <PROCESS GO COVERAGE> ===\n"
	go test -coverprofile=coverage.out "./pkg/..."
	go tool cover -html=coverage.out
	rm coverage.out

job-coverage: github-config ## Run go coverage and export to html file
	@printf "\n=== <PROCESS GO JOB COVERAGE> ===\n"
	go test -coverprofile=coverage.out "./pkg/..."
	go tool cover -html=coverage.out -o coverage.html
	rm coverage.out

.PHONY: lint golangci-lint-install
lint: golangci-lint-install vendor ## Run go lint
	@printf "\n=== <PROCESS GOLANGCI-LINT> ===\n"
	$(GOPATH)/golangci-lint run -v $(CURDIR)/...

golangci-lint-install:
ifeq (,$(shell $(GOPATH)/golangci-lint version 2>/dev/null))
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

.PHONY: fmt check fix
fmt: vendor ## Run go fmt
	@printf "\n=== <PROCESS GO FMT> ===\n"
	go fmt ./pkg/... ./cmd/... ./test/...

check: github-config fmt generate lint ## Run git diff check

.PHONY: check-diff
check-diff:
	@printf "\n=== <PROCESS GIT DIFF> ===\n"
	git diff --exit-code ":(exclude)github-password"

fix: golangci-lint-install vendor ## Run go fix
	@printf "\n=== <PROCESS GO FIX> ===\n"
	$(GOPATH)/golangci-lint run -v --fix ./...

.PHONY: docs-build
docs-build:
	@python3 -m venv build/venv; \
	 . build/venv/bin/activate; \
	 pip3 install -U -r docs/requirements.txt; \
	 mkdocs build -d build/htmldocs; \
	 (deactivate || true)

.PHONY: docs-serve
docs-serve:
	@python3 -m venv build/venv
	. build/venv/bin/activate; \
	pip install -U -r docs/requirements.txt; \
	mkdocs serve

.PHONY: help
help: ## This help.
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z\/_-]+:.*?## / {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}' $(MAKEFILE_LIST)

# vim: set ts=4 sw=4 tw=0 noet :
