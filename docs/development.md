# Development

[Back to README](../README.md)

The Makefile uses [Podman](https://podman.io) (rootless, recommended) or Docker to run
Go tools in containers. Go is optional on the host with these targets. A native
build needs [Go 1.25+](https://go.dev/dl/).

```sh
make build     # -> ./bin/resumegen
make lint      # golangci-lint
make test      # all tests
make coverage  # per-function coverage on domain + usecase
make coverage-gate
make tidy      # go mod tidy
make rebuild   # force rebuild of tool images
make clean     # remove build artifacts
make help      # all targets, discovered from Makefile comments
```

Run a subset using `make test TEST_PKG=./internal/adapter/render/host`.
The PDF integration tests use host `typst`; scanner extraction tests also require
`pdftotext` from Poppler. They skip when these tools are unavailable. A Go-only test
run therefore does not prove that PDF text extraction works.

With native tools already installed, the CI-equivalent checks are:

```sh
go vet ./...
go vet -tags notui ./...
go build ./...
go build -tags notui ./...
go test ./...
go test -tags notui ./...
golangci-lint run ./...
```

## Preview assets

Render the bundled default profile in a temporary workspace, then update
`assets/default.pdf` and `assets/default.png` together. The PNG is a preview;
the PDF is the actual text-based resume. Validate extraction before replacing assets:

```sh
pdftotext -layout assets/default.pdf -
pdfinfo assets/default.pdf
```

## Third-party components

Linked Go module licenses and full texts are in [THIRD_PARTY_NOTICES.md](../THIRD_PARTY_NOTICES.md).
Typst is invoked as a separate program. Its PDF fonts include New Computer Modern
(GUST Font License), Libertinus Serif (SIL Open Font License 1.1), and DejaVu Sans Mono.
These font licenses do not place conditions on generated resumes.
