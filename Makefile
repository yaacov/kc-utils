PROJECT := kc-utils
MODULE  := github.com/yaacov/kc-utils

# Key environment variables (all optional, shown with defaults):
#   REGISTRY=quay.io            Container image registry
#   REGISTRY_ORG=kubev2v        Registry organization / namespace
#   REGISTRY_TAG=devel          Image tag for builds and pushes
#   GIT_TAGS=                   Extra image tags to push (auto-detected from git when empty)
#   PLATFORM=linux/amd64        Container build platform (linux/amd64 or linux/arm64)
#   CONTAINER_RUNTIME=          Force docker or podman (auto-detected if empty)

# Release binaries default to GOOS=linux. Override for a native Mac build:
#   GOOS=darwin make build-kc-copy
BINS := kc-prepare kc-finalize kc-v2v kc-copy kc-convert-linux kc-convert-windows
BIN_DIR    := bin

GO      := go
GOOS    ?= linux
GOARCH  ?= $(shell $(GO) env GOARCH)
GOFLAGS := -trimpath
LDFLAGS := -s -w

GOPATH ?= $(shell go env GOPATH)
GOBIN  ?= $(GOPATH)/bin

GOLANGCI_LINT_VERSION ?= v1.64.2
GOLANGCI_LINT_BIN     ?= $(GOBIN)/golangci-lint
GOLANGCI_LINT_GOVER   := $(shell $(GO) env GOVERSION | tr ':' '_' | tr '/' '_')
GOLANGCI_LINT_STAMP   := $(GOBIN)/.golangci-lint-$(GOLANGCI_LINT_VERSION)-$(GOLANGCI_LINT_GOVER).stamp

DEADCODE_VERSION ?= v0.49.0
DEADCODE_BIN     ?= $(GOBIN)/deadcode
DEADCODE_GOVER   := $(shell $(GO) env GOVERSION | tr ':' '_' | tr '/' '_')
DEADCODE_STAMP   := $(GOBIN)/.deadcode-$(DEADCODE_VERSION)-$(DEADCODE_GOVER).stamp

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

# Optional override; empty = auto-detect tags pointing at HEAD
GIT_TAGS ?=
GIT_TAGS := $(if $(GIT_TAGS),$(GIT_TAGS),$(shell git tag --points-at HEAD 2>/dev/null))

KC_V2V_IMAGE_BASE := $(REGISTRY)/$(REGISTRY_ORG)/kc-v2v
KC_V2V_IMAGE ?= $(KC_V2V_IMAGE_BASE):$(REGISTRY_TAG)
KC_V2V_IMAGE_LOCAL := $(KC_V2V_IMAGE)$(PLATFORM_SUFFIX)
TEST_IMAGE   := kc-utils-test

.PHONY: all build $(BINS) test test-race test-coverage test-container \
        lint lint-install deadcode deadcode-install fmt vet clean cross-linux-amd64 cross-linux-arm64 \
        cross-linux-ppc64le cross-linux-s390x cross-all mod-tidy mod-verify \
        check test-e2e test-e2e-container test-e2e-disk test-e2e-disk-guestfs \
        test-image test-image-rebuild test-build check-all help build-kc-v2v \
        build-kc-copy cache-virtio-win build-kc-v2v-image build-appliance \
        push-kc-v2v-image test-kc-v2v-image check_container_runtime

all: build

## Show available targets
help:
	@printf "\n\033[1mKey environment variables (all optional, shown with defaults):\033[0m\n"
	@printf "  \033[36m%-28s\033[0m %s\n" "REGISTRY=quay.io"            "Container image registry"
	@printf "  \033[36m%-28s\033[0m %s\n" "REGISTRY_ORG=kubev2v"        "Registry organization / namespace"
	@printf "  \033[36m%-28s\033[0m %s\n" "REGISTRY_TAG=devel"          "Image tag for builds and pushes"
	@printf "  \033[36m%-28s\033[0m %s\n" "GIT_TAGS="                  "Extra image tags to push (auto-detected from git when empty)"
	@printf "  \033[36m%-28s\033[0m %s\n" "PLATFORM=linux/amd64"        "Container build platform (linux/amd64 or linux/arm64)"
	@printf "  \033[36m%-28s\033[0m %s\n" "GOOS=linux"                 "Target OS for make build (e.g. GOOS=darwin)"
	@printf "  \033[36m%-28s\033[0m %s\n" "GOARCH=$(GOARCH)"             "Target arch for make build (host default)"
	@printf "\n\033[1mExample:\033[0m  REGISTRY_ORG=myuser make build-kc-v2v-image push-kc-v2v-image\n"
	@printf "\033[1mRelease:\033[0m     git tag v0.1.0 && make push-kc-v2v-image  # pushes devel-amd64 + v0.1.0-amd64\n\n"
	@awk '/^## /{desc=substr($$0,4)} /^[a-zA-Z0-9_-]+:/ && desc{sub(/:.*/, "", $$1); printf "  \033[36m%-24s\033[0m %s\n", $$1, desc; desc=""}' $(MAKEFILE_LIST)

## Build all binaries (linux; set GOOS/GOARCH to override)
build: $(BINS)

$(BINS):
	CGO_ENABLED=0 GOOS=$(GOOS) GOARCH=$(GOARCH) $(GO) build $(GOFLAGS) -ldflags '$(LDFLAGS)' -o $(BIN_DIR)/$@ ./cmd/$@

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

## Run golangci-lint
lint: $(GOLANGCI_LINT_STAMP)
	@echo "Running golangci-lint..."
	$(GOLANGCI_LINT_BIN) run ./pkg/... ./cmd/...

$(GOLANGCI_LINT_STAMP):
	$(MAKE) lint-install
	@touch $@

## Install deadcode (pinned version, into GOBIN)
deadcode-install:
	@echo "Installing deadcode $(DEADCODE_VERSION)..."
	GOBIN=$(GOBIN) $(GO) install golang.org/x/tools/cmd/deadcode@$(DEADCODE_VERSION)
	@echo "deadcode installed successfully."

## Fail if unreachable functions exist (-test includes test-only entry points)
deadcode: $(DEADCODE_STAMP)
	@echo "Running deadcode..."
	@out="$$($(DEADCODE_BIN) -test ./...)"; \
	status=$$?; \
	if [ $$status -ne 0 ]; then exit $$status; fi; \
	if [ -n "$$out" ]; then \
		printf '%s\n' "$$out"; \
		echo "deadcode: unreachable functions found"; \
		exit 1; \
	fi

$(DEADCODE_STAMP):
	$(MAKE) deadcode-install
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

## [container] Run unit tests in Linux (integration / kernel-specific tests)
test-container: test-image
	$(CONTAINER_CMD) run --rm \
	    $(KC_UTILS_WORKSPACE_MOUNT) \
	    $(TEST_IMAGE) make test

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

## Build the qemu-backend appliance (kernel + initramfs) into bin/appliance/<arch>
## Arches via APPLIANCE_ARCHES (default: arm64 amd64). Cross builds need binfmt.
APPLIANCE_ARCHES ?= arm64 amd64
build-appliance: check_container_runtime
	CONTAINER_RUNTIME=$(CONTAINER_RUNTIME) build/kc-appliance/build.sh $(APPLIANCE_ARCHES)

## Cache virtio-win ISOs locally (avoids re-downloading on each image build)
cache-virtio-win:
	@mkdir -p build/kc-v2v/cache
	@for url in \
		"https://fedorapeople.org/groups/virt/virtio-win/direct-downloads/archive-virtio/virtio-win-0.1.285-1/virtio-win-0.1.285.iso" \
		"https://fedorapeople.org/groups/virt/virtio-win/direct-downloads/archive-virtio/virtio-win-0.1.160-1/virtio-win-0.1.160.iso"; do \
		f="build/kc-v2v/cache/$$(basename "$$url")"; \
		if [ -f "$$f" ]; then echo "Cached: $$f"; \
		else echo "Downloading $$(basename "$$url")..."; curl -fL --retry 3 --retry-delay 5 -o "$$f" "$$url"; fi; \
	done

## Build kc-v2v container image (guestfs backend; the qemu appliance is built
## separately with `make build-appliance` and is not shipped in this image)
build-kc-v2v-image: check_container_runtime build
	$(CONTAINER_CMD) build $(PLATFORM_FLAG) -t $(KC_V2V_IMAGE_LOCAL) -f build/kc-v2v/Containerfile .

## Push kc-v2v container image
push-kc-v2v-image: build-kc-v2v-image
	@set -e; \
	img="$(KC_V2V_IMAGE_LOCAL)"; \
	echo "Pushing $$img"; \
	$(CONTAINER_CMD) push "$$img"; \
	for tag in $(GIT_TAGS); do \
	  extra="$(KC_V2V_IMAGE_BASE):$$tag$(PLATFORM_SUFFIX)"; \
	  echo "Tagging and pushing $$extra"; \
	  $(CONTAINER_CMD) tag "$$img" "$$extra"; \
	  $(CONTAINER_CMD) push "$$extra"; \
	done

## Smoke-test the built kc-v2v image (RPMs, NTFS, guestfish ext4/xfs/btrfs/ntfs)
## Uses $(KC_V2V_IMAGE_LOCAL). Pass REQUIRE_GUESTFS=1 to require /dev/kvm + FS mounts.
## REQUIRE_CLEVIS defaults to 1 (guestfish clevisluks must be yes).
REQUIRE_CLEVIS ?= 1
test-kc-v2v-image: check_container_runtime
	@if ! $(CONTAINER_CMD) image inspect $(KC_V2V_IMAGE_LOCAL) >/dev/null 2>&1; then \
		echo "Image $(KC_V2V_IMAGE_LOCAL) not found — build with: make build-kc-v2v-image"; \
		exit 1; \
	fi
	$(CONTAINER_CMD) run --rm \
	    $(if $(wildcard /dev/kvm),--device /dev/kvm,) \
	    --entrypoint bash \
	    -e HOME=/var/tmp \
	    -e LIBGUESTFS_BACKEND=direct \
	    -e REQUIRE_GUESTFS=$(REQUIRE_GUESTFS) \
	    -e REQUIRE_NTFS=$(REQUIRE_NTFS) \
	    -e REQUIRE_CLEVIS=$(REQUIRE_CLEVIS) \
	    -v $(CURDIR)/tests/test-kc-v2v-image.sh:/tmp/test-kc-v2v-image.sh:ro,Z \
	    $(KC_V2V_IMAGE_LOCAL) \
	    /tmp/test-kc-v2v-image.sh

check_container_runtime:
	@if [ ! -x "$(CONTAINER_CMD)" ]; then \
		echo "Container runtime was not automatically detected"; \
		echo "Please install podman or docker and make sure it's available in PATH"; \
		exit 1; \
	fi
