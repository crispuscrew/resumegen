RUNTIME    := $(shell which podman 2>/dev/null || which docker)
BINARY     := resumegen
CMD        := ./cmd/resumegen
BUILD_DIR  := ./bin
IMAGE_DEV  := resumegen-dev
IMAGE_LINT := resumegen-lint
IMAGE_TEST := resumegen-test
RUN_DEV    := $(RUNTIME) run --rm
TEST_PKG   ?= ./...

.PHONY: build run lint tidy clean test rebuild help

_image-dev:
	@$(RUNTIME) image exists $(IMAGE_DEV) || \
		$(RUNTIME) build -f container/dev/Containerfile -t $(IMAGE_DEV) .

_image-lint: _image-dev
	@$(RUNTIME) image exists $(IMAGE_LINT) || \
		$(RUNTIME) build -f container/lint/Containerfile -t $(IMAGE_LINT) .

_image-test: _image-dev
	@$(RUNTIME) image exists $(IMAGE_TEST) || \
		$(RUNTIME) build -f container/test/Containerfile -t $(IMAGE_TEST) .

rebuild:
	$(RUNTIME) rmi -f $(IMAGE_DEV) $(IMAGE_LINT) $(IMAGE_TEST) 2>/dev/null ; \
	$(RUNTIME) build -f container/dev/Containerfile -t $(IMAGE_DEV) . && \
	$(RUNTIME) build -f container/lint/Containerfile -t $(IMAGE_LINT) . && \
	$(RUNTIME) build -f container/test/Containerfile -t $(IMAGE_TEST) .

tidy: _image-dev
	$(RUN_DEV) -v $(PWD):/app $(IMAGE_DEV) go mod tidy

build: _image-dev
	@mkdir -p $(BUILD_DIR)
	$(RUN_DEV) -v $(PWD):/app -v $(PWD)/$(BUILD_DIR):/out $(IMAGE_DEV) \
		go build -o /out/$(BINARY) $(CMD)

test: _image-test
	$(RUN_DEV) -v $(PWD):/app $(IMAGE_TEST) go test -v $(TEST_PKG)

lint: _image-lint
	$(RUN_DEV) -v $(PWD):/app $(IMAGE_LINT) golangci-lint run ./...

run:
	./$(BUILD_DIR)/$(BINARY)

clean:
	rm -rf $(BUILD_DIR)

help:
	@echo "Usage:"
	@echo "  make build    — build binary to ./bin/resumegen"
	@echo "  make run      — run built binary"
	@echo "  make test     — run tests"
	@echo "  make lint     — run golangci-lint"
	@echo "  make tidy     — go mod tidy"
	@echo "  make rebuild  — force rebuild all container images"
	@echo "  make clean    — remove build artifacts"
