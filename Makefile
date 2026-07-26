RUNTIME      := $(shell which podman 2>/dev/null || which docker)
BINARY       := resumegen
CMD          := ./cmd/resumegen
BUILD_DIR    := ./bin
IMAGE_DEV    := resumegen-dev
IMAGE_LINT   := resumegen-lint
IMAGE_TEST   := resumegen-test
RENDER_VER   ?= dev
IMAGE_RENDER := localhost/resumegen-render:$(RENDER_VER)
RUN_DEV      := $(RUNTIME) run --rm
# Extra flags for image builds. Normally empty: the default network works once
# container DNS is sane (a machine-level dns_servers override in
# ~/.config/containers/containers.conf pointing at a dead VPN resolver was the
# 2026-07 root cause; fix the machine, not the Makefile). Escape hatch:
#   make rebuild BUILD_NET="--dns=1.1.1.1"    or    BUILD_NET="--network=host"
BUILD_NET    ?=
BUILD        := $(RUNTIME) build $(BUILD_NET)
TEST_PKG     ?= ./...
GATE         ?= 65

# golangci-lint is prefetched on the HOST (container downloads proved flaky on
# VPN links) and COPY'd into the lint image. Pin + sha256 match ci.yml's
# golangci-lint-action version; bump both together.
LINT_VER     := 2.12.2
LINT_SHA256  := 8df580d2670fed8fa984aac0507099af8df275e665215f5c7a2ae3943893a553
LINT_BLOB    := container/lint/blobs/golangci-lint
LINT_URL     := https://github.com/golangci/golangci-lint/releases/download/v$(LINT_VER)/golangci-lint-$(LINT_VER)-linux-amd64.tar.gz

.PHONY: build run tui lint tidy clean test coverage coverage-gate rebuild help container-image _lint-blob

.DEFAULT_GOAL := help

_image-dev:
	@$(RUNTIME) image inspect $(IMAGE_DEV) >/dev/null 2>&1 || \
		$(BUILD) -f container/dev/Containerfile -t $(IMAGE_DEV) .

_lint-blob:
	@test -f $(LINT_BLOB) || { \
		echo "fetching golangci-lint v$(LINT_VER) (host network)"; \
		mkdir -p $(dir $(LINT_BLOB)); \
		curl -fsSL --retry 5 --retry-all-errors -o $(LINT_BLOB).tgz $(LINT_URL); \
		echo "$(LINT_SHA256)  $(LINT_BLOB).tgz" | sha256sum -c - >/dev/null; \
		tar -xzf $(LINT_BLOB).tgz -C $(dir $(LINT_BLOB)) --strip-components=1 \
			golangci-lint-$(LINT_VER)-linux-amd64/golangci-lint; \
		rm -f $(LINT_BLOB).tgz; \
	}

_image-lint: _image-dev _lint-blob
	@$(RUNTIME) image inspect $(IMAGE_LINT) >/dev/null 2>&1 || \
		$(BUILD) -f container/lint/Containerfile -t $(IMAGE_LINT) .

_image-test: _image-dev
	@$(RUNTIME) image inspect $(IMAGE_TEST) >/dev/null 2>&1 || \
		$(BUILD) -f container/test/Containerfile -t $(IMAGE_TEST) .

build: _image-dev  ## build binary to ./bin/resumegen
	@mkdir -p $(BUILD_DIR)
	$(RUN_DEV) -v $(PWD):/app -v $(PWD)/$(BUILD_DIR):/out $(IMAGE_DEV) \
		go build -o /out/$(BINARY) $(CMD)

run:  ## run the built binary
	./$(BUILD_DIR)/$(BINARY)

# Host appdir mounted into the container. Rootless podman maps container root to
# your user, so files stay yours; with ROOTFUL docker they'd be root-owned - use
# rootless podman for this target. Clipboard (`y`) and a custom $EDITOR are not
# available inside the container (no wl-copy/xclip, busybox vi only).
APPDIR ?= $(or $(XDG_CONFIG_HOME),$(HOME)/.config)/resumegen

tui: _image-dev  ## run the interactive TUI in the dev container (allocates a TTY, mounts your appdir)
	@mkdir -p $(APPDIR)
	$(RUN_DEV) -it -e TERM \
		-v $(PWD):/app \
		-v $(APPDIR):/root/.config/resumegen \
		$(IMAGE_DEV) go run $(CMD) tui

test: _image-test  ## run all tests with verbose output
	$(RUN_DEV) -v $(PWD):/app $(IMAGE_TEST) go test -v $(TEST_PKG)

coverage: _image-test  ## print per-function coverage on domain + usecase
	$(RUN_DEV) -v $(PWD):/app $(IMAGE_TEST) sh -c \
		'go test -coverprofile=coverage.out ./internal/domain/... ./internal/usecase/... && \
		 go tool cover -func=coverage.out'

coverage-gate: _image-test  ## fail if coverage on domain+usecase < $GATE (default 65)
	$(RUN_DEV) -v $(PWD):/app -e GATE=$(GATE) $(IMAGE_TEST) sh scripts/coverage-gate.sh

lint: _image-lint  ## run golangci-lint
	$(RUN_DEV) -v $(PWD):/app $(IMAGE_LINT) golangci-lint run ./...

tidy: _image-dev  ## run go mod tidy
	$(RUN_DEV) -v $(PWD):/app $(IMAGE_DEV) go mod tidy

rebuild: _lint-blob  ## force rebuild all container images
	$(RUNTIME) rmi -f $(IMAGE_DEV) $(IMAGE_LINT) $(IMAGE_TEST) 2>/dev/null ; \
	$(BUILD) -f container/dev/Containerfile -t $(IMAGE_DEV) . && \
	$(BUILD) -f container/lint/Containerfile -t $(IMAGE_LINT) . && \
	$(BUILD) -f container/test/Containerfile -t $(IMAGE_TEST) .

container-image:  ## build the local render image (slice 4, opt-in render backend)
	$(BUILD) -f container/render/Containerfile -t $(IMAGE_RENDER) container/render/

clean:  ## remove build artifacts
	rm -rf $(BUILD_DIR)

help:  ## show this help, auto-discovered from comments after each target
	@awk 'BEGIN { FS = ":[^#]*## "; printf "Usage:\n" } \
		/^[a-zA-Z0-9_-]+:[^#]*## / { printf "  make %-15s - %s\n", $$1, $$2 }' $(MAKEFILE_LIST)
