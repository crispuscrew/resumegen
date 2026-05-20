RUNTIME    := $(shell which podman 2>/dev/null || which docker)
BINARY     := resumegen
CMD        := ./cmd/resumegen
BUILD_DIR  := ./bin
IMAGE_DEV  := resumegen-dev
IMAGE_LINT := resumegen-lint
IMAGE_TEST := resumegen-test
RUN_DEV    := $(RUNTIME) run --rm
TEST_PKG   ?= ./...
GATE       ?= 50

.PHONY: build run lint tidy clean test coverage coverage-gate rebuild help

.DEFAULT_GOAL := help

_image-dev:
	@$(RUNTIME) image exists $(IMAGE_DEV) || \
		$(RUNTIME) build -f container/dev/Containerfile -t $(IMAGE_DEV) .

_image-lint: _image-dev
	@$(RUNTIME) image exists $(IMAGE_LINT) || \
		$(RUNTIME) build -f container/lint/Containerfile -t $(IMAGE_LINT) .

_image-test: _image-dev
	@$(RUNTIME) image exists $(IMAGE_TEST) || \
		$(RUNTIME) build -f container/test/Containerfile -t $(IMAGE_TEST) .

build: _image-dev  ## build binary to ./bin/resumegen
	@mkdir -p $(BUILD_DIR)
	$(RUN_DEV) -v $(PWD):/app -v $(PWD)/$(BUILD_DIR):/out $(IMAGE_DEV) \
		go build -o /out/$(BINARY) $(CMD)

run:  ## run the built binary
	./$(BUILD_DIR)/$(BINARY)

test: _image-test  ## run all tests with verbose output
	$(RUN_DEV) -v $(PWD):/app $(IMAGE_TEST) go test -v $(TEST_PKG)

coverage: _image-test  ## print per-function coverage on domain + usecase
	$(RUN_DEV) -v $(PWD):/app $(IMAGE_TEST) sh -c \
		'go test -coverprofile=coverage.out ./internal/domain/... ./internal/usecase/... && \
		 go tool cover -func=coverage.out'

coverage-gate: _image-test  ## fail if coverage on domain+usecase < $GATE (default 50)
	$(RUN_DEV) -v $(PWD):/app -e GATE=$(GATE) $(IMAGE_TEST) sh scripts/coverage-gate.sh

lint: _image-lint  ## run golangci-lint
	$(RUN_DEV) -v $(PWD):/app $(IMAGE_LINT) golangci-lint run ./...

tidy: _image-dev  ## run go mod tidy
	$(RUN_DEV) -v $(PWD):/app $(IMAGE_DEV) go mod tidy

rebuild:  ## force rebuild all container images
	$(RUNTIME) rmi -f $(IMAGE_DEV) $(IMAGE_LINT) $(IMAGE_TEST) 2>/dev/null ; \
	$(RUNTIME) build -f container/dev/Containerfile -t $(IMAGE_DEV) . && \
	$(RUNTIME) build -f container/lint/Containerfile -t $(IMAGE_LINT) . && \
	$(RUNTIME) build -f container/test/Containerfile -t $(IMAGE_TEST) .

clean:  ## remove build artifacts
	rm -rf $(BUILD_DIR)

help:  ## show this help, auto-discovered from comments after each target
	@awk 'BEGIN { FS = ":[^#]*## "; printf "Usage:\n" } \
		/^[a-zA-Z0-9_-]+:[^#]*## / { printf "  make %-15s - %s\n", $$1, $$2 }' $(MAKEFILE_LIST)
