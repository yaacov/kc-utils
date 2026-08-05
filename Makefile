PROJECT := kc-utils
MODULE  := github.com/yaacov/kc-utils

# Key environment variables (all optional, shown with defaults):
#   REGISTRY=quay.io            Container image registry
#   REGISTRY_ORG=kubev2v        Registry organization / namespace
#   REGISTRY_TAG=devel          Image tag for builds and pushes
#   PLATFORM=linux/amd64        Container build platform (linux/amd64 or linux/arm64)
#   CONTAINER_RUNTIME=          Force docker or podman (auto-detected if empty)

# All binaries require GOOS=linux (pkg/guest is //go:build linux).
BINS := kc-prepare kc-finalize kc-v2v kc-copy kc-convert-linux kc-convert-windows
BIN_DIR    := bin

GO      := go
GOFLAGS := -trimpath
LDFLAGS := -s -w

GOPATH ?= $(shell go env GOPATH)
GOBIN  ?= $(GOPATH)/bin

GOLANGCI_LINT_VERSION ?= v1.64.2
GOLANGCI_LINT_BIN     ?= $(GOBIN)/golangci-lint
GOLANGCI_LINT_GOVER   := $(shell $(GO) env GOVERSION | tr ':' '_' | tr '/' '_')
GOLANGCI_LINT_STAMP   := $(GOBIN)/.golangci-lint-$(GOLANGCI_LINT_VERSION)-$(GOLANGCI_LINT_GOVER).stamp

CONTAINER_RUNTIME ?=

ifeq ($(CONTAINER_RUNTIME),)
CONTAINER_CMD ?= $(shell command -v docker 2>/dev/null)
ifeq ($(CONTAINER_CMD),)
CONTAINER_CMD := $(shell command -v podman 2>/dev/null)
endif
CONTAINER_RUNTIME=$(shell basename $(CONTAINER_CMD))
else
CONTAINER_CMD := $(shell command -v $(CONTAINER_RUNTIME) 2>/dev/null)
endif

PLATFORM ?= linux/amd64
ifneq ($(PLATFORM),)
PLATFORM_FLAG := --platform $(PLATFORM)
else
PLATFORM_FLAG :=
endif

ifeq ($(CONTAINER_RUNTIME),docker)
PLATFORM_FLAG += --provenance=false
endif

PLATFORM_ARCH ?= $(shell echo $(PLATFORM) | cut -d'/' -f2)
PLATFORM_SUFFIX := -$(PLATFORM_ARCH)

REGISTRY ?= quay.io
REGISTRY_ORG ?= kubev2v
REGISTRY_TAG ?= devel

KC_V2V_IMAGE ?= $(REGISTRY)/$(REGISTRY_ORG)/kc-v2v:$(REGISTRY_TAG)
TEST_IMAGE   := kc-utils-test

.PHONY: all build $(BINS) test test-race test-coverage lint lint-install \
        fmt vet clean cross-linux-amd64 cross-linux-arm64 cross-linux-ppc64le \
        cross-linux-s390x cross-all mod-tidy mod-verify check test-e2e \
        test-e2e-container test-e2e-disk test-e2e-disk-guestfs test-image \
        test-image-rebuild test-build check-all help build-kc-v2v \
        build-kc-copy prepare-windows-virtio-drivers build-kc-v2v-image push-kc-v2v-image check_container_runtime

all: build

## Show available targets
help:
	@printf "\n\033[1mKey environment variables (all optional, shown with defaults):\033[0m\n"
	@printf "  \033[36m%-28s\033[0m %s\n" "REGISTRY=quay.io"            "Container image registry"
	@printf "  \033[36m%-28s\033[0m %s\n" "REGISTRY_ORG=kubev2v"        "Registry organization / namespace"
	@printf "  \033[36m%-28s\033[0m %s\n" "REGISTRY_TAG=devel"          "Image tag for builds and pushes"
	@printf "  \033[36m%-28s\033[0m %s\n" "PLATFORM=linux/amd64"        "Container build platform (linux/amd64 or linux/arm64)"
	@printf "  \033[36m%-28s\033[0m %s\n" "CONTAINER_RUNTIME="          "Force docker or podman (auto-detected if empty)"
	@printf "\n\033[1mExample:\033[0m  REGISTRY_ORG=myuser make build-kc-v2v-image push-kc-v2v-image\n\n"
	@awk '/^## /{desc=substr($$0,4)} /^[a-zA-Z0-9_-]+:/ && desc{sub(/:.*/, "", $$1); printf "  \033[36m%-24s\033[0m %s\n", $$1, desc; desc=""}' $(MAKEFILE_LIST)

## Build all binaries (linux)
build: $(BINS)

$(BINS):
	CGO_ENABLED=0 GOOS=linux $(GO) build $(GOFLAGS) -ldflags '$(LDFLAGS)' -o $(BIN_DIR)/$@ ./cmd/$@

## Run unit tests locally
test:
	$(GO) test ./...

## Run unit tests locally with race detector
test-race:
	$(GO) test -race ./...

## Run unit tests locally with coverage report
test-coverage:
	$(GO) test -coverprofile=coverage.out ./...
	$(GO) tool cover -func=coverage.out
	@echo "HTML report: go tool cover -html=coverage.out"

## Install golangci-lint (pinned version, into GOBIN)
lint-install:
	@echo "Installing golangci-lint $(GOLANGCI_LINT_VERSION)..."
	GOBIN=$(GOBIN) $(GO) install github.com/golangci/golangci-lint/cmd/golangci-lint@$(GOLANGCI_LINT_VERSION)
	@echo "golangci-lint installed successfully."

## Run golangci-lint (GOOS=linux: pkg/guest and related packages are linux-only)
lint: $(GOLANGCI_LINT_STAMP)
	@echo "Running golangci-lint..."
	GOOS=linux $(GOLANGCI_LINT_BIN) run ./pkg/... ./cmd/... ./internal/...

$(GOLANGCI_LINT_STAMP):
	$(MAKE) lint-install
	@touch $@

## Format Go source files
fmt:
	gofmt -l -w .

## Run go vet
vet:
	$(GO) vet ./...

## Remove build artifacts
clean:
	rm -rf $(BIN_DIR) coverage.out tests/bin

## Cross-compile for linux/amd64
cross-linux-amd64:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 $(GO) build $(GOFLAGS) -ldflags '$(LDFLAGS)' -o $(BIN_DIR)/linux-amd64/ ./cmd/...

## Cross-compile for linux/arm64
cross-linux-arm64:
	CGO_ENABLED=0 GOOS=linux GOARCH=arm64 $(GO) build $(GOFLAGS) -ldflags '$(LDFLAGS)' -o $(BIN_DIR)/linux-arm64/ ./cmd/...

## Cross-compile for linux/ppc64le
cross-linux-ppc64le:
	CGO_ENABLED=0 GOOS=linux GOARCH=ppc64le $(GO) build $(GOFLAGS) -ldflags '$(LDFLAGS)' -o $(BIN_DIR)/linux-ppc64le/ ./cmd/...

## Cross-compile for linux/s390x
cross-linux-s390x:
	CGO_ENABLED=0 GOOS=linux GOARCH=s390x $(GO) build $(GOFLAGS) -ldflags '$(LDFLAGS)' -o $(BIN_DIR)/linux-s390x/ ./cmd/...

## Cross-compile for all architectures
cross-all: cross-linux-amd64 cross-linux-arm64 cross-linux-ppc64le cross-linux-s390x

## Tidy go.mod and go.sum
mod-tidy:
	$(GO) mod tidy

## Verify module dependencies
mod-verify:
	$(GO) mod verify

## Run fmt + vet + lint + unit tests
check: fmt vet lint test

## Build test binaries into tests/bin/
test-build:
	@cd tests && bash build.sh

## Run e2e tests locally (Windows tests require hivexregedit)
test-e2e: test-build
	@cd tests && pass=0 fail=0 skip=0; \
	for t in test-linux-*.sh test-windows-*.sh test-root-*.sh test-kc-*.sh test-dynamicscripts-*.sh; do \
	    printf "=== %-40s " "$$t"; \
	    bash "$$t" >$$t.log 2>&1; rc=$$?; \
	    if [ $$rc -eq 0 ]; then echo "PASS"; pass=$$((pass+1)); \
	    elif [ $$rc -eq 77 ]; then echo "SKIP"; skip=$$((skip+1)); \
	    else echo "FAIL (see tests/$$t.log)"; fail=$$((fail+1)); fi; \
	done; \
	echo ""; echo "Results: $$pass passed, $$fail failed, $$skip skipped"; \
	[ $$fail -eq 0 ]

KC_UTILS_WORKSPACE_MOUNT := -v $(CURDIR):/workspace/kc-utils:Z

## Build test container image (skips if already cached)
test-image: check_container_runtime
	@if ! $(CONTAINER_CMD) image inspect $(TEST_IMAGE) >/dev/null 2>&1; then \
	    $(CONTAINER_CMD) build -t $(TEST_IMAGE) -f tests/Containerfile .; \
	else \
	    echo "$(TEST_IMAGE) image already exists (use 'make test-image-rebuild' to force)"; \
	fi

## Force-rebuild the test container image
test-image-rebuild: check_container_runtime
	$(CONTAINER_CMD) build --no-cache -t $(TEST_IMAGE) -f tests/Containerfile .

## [container] Run e2e tests in a Fedora container (all tests, incl. Windows)
test-e2e-container: test-image
	$(CONTAINER_CMD) run --rm --privileged \
	    $(KC_UTILS_WORKSPACE_MOUNT) \
	    $(TEST_IMAGE) bash -c 'losetup -D 2>/dev/null || true; make test-e2e'

## [container] Run disk-image e2e tests (--privileged, requires guestfs)
test-e2e-disk: test-image
	$(CONTAINER_CMD) run --rm --privileged \
	    $(KC_UTILS_WORKSPACE_MOUNT) \
	    $(TEST_IMAGE) bash -c 'export LIBGUESTFS_BACKEND=direct && \
	    losetup -D 2>/dev/null || true && \
	    cd /workspace/kc-utils && make test-build && cd tests && \
	    pass=0 fail=0 skip=0; \
	    for t in test-disk-*.sh; do \
	        printf "=== %-40s " "$$t"; \
	        bash "$$t" >$$t.log 2>&1; rc=$$?; \
	        if [ $$rc -eq 0 ]; then echo "PASS"; pass=$$((pass+1)); \
	        elif [ $$rc -eq 77 ]; then echo "SKIP"; skip=$$((skip+1)); \
	        else echo "FAIL (see tests/$$t.log)"; fail=$$((fail+1)); fi; \
	    done; \
	    echo ""; echo "Disk tests: $$pass passed, $$fail failed, $$skip skipped"; \
	    [ $$fail -eq 0 ]'

## [container] Run disk-image e2e tests via guestfs (no --privileged / no FUSE)
test-e2e-disk-guestfs: test-image
	$(CONTAINER_CMD) run --rm \
	    $(KC_UTILS_WORKSPACE_MOUNT) \
	    $(TEST_IMAGE) bash -c 'export LIBGUESTFS_BACKEND=direct && \
	    cd /workspace/kc-utils && make test-build && cd tests && \
	    pass=0 fail=0 skip=0; \
	    for t in test-disk-*-guestfs.sh; do \
	        printf "=== %-40s " "$$t"; \
	        bash "$$t" >$$t.log 2>&1; rc=$$?; \
	        if [ $$rc -eq 0 ]; then echo "PASS"; pass=$$((pass+1)); \
	        elif [ $$rc -eq 77 ]; then echo "SKIP"; skip=$$((skip+1)); \
	        else echo "FAIL (see tests/$$t.log)"; fail=$$((fail+1)); fi; \
	    done; \
	    echo ""; echo "Guestfs disk tests: $$pass passed, $$fail failed, $$skip skipped"; \
	    [ $$fail -eq 0 ]'

## Full check locally: unit tests + lint + e2e
check-all: check test-e2e

## Build kc-v2v entrypoint binary
build-kc-v2v: kc-v2v

## Build kc-copy binary (pipeline stage + standalone CLI)
build-kc-copy: kc-copy

## Stage per-version Windows virtio-win vendor files (see build/kc-v2v/vendor/README.md)
prepare-windows-virtio-drivers:
	CONTAINER_CMD="$(CONTAINER_CMD)" \
		bash build/kc-v2v/prepare-windows-virtio-drivers.sh

## Build kc-v2v container image
build-kc-v2v-image: check_container_runtime build
	$(CONTAINER_CMD) build $(PLATFORM_FLAG) -t $(KC_V2V_IMAGE)$(PLATFORM_SUFFIX) -f build/kc-v2v/Containerfile .

## Push kc-v2v container image
push-kc-v2v-image: build-kc-v2v-image
	$(CONTAINER_CMD) push $(KC_V2V_IMAGE)$(PLATFORM_SUFFIX)

check_container_runtime:
	@if [ ! -x "$(CONTAINER_CMD)" ]; then \
		echo "Container runtime was not automatically detected"; \
		echo "Please install podman or docker and make sure it's available in PATH"; \
		exit 1; \
	fi
