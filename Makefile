BINARY := lmstudio-exporter
VERSION ?= dev
COMMIT ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo unknown)
DATE ?= $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
LDFLAGS := -s -w \
	-X main.version=$(VERSION) \
	-X main.commit=$(COMMIT) \
	-X main.buildDate=$(DATE)

.PHONY: build test fmt vet check clean

build:
	mkdir -p bin
	go build -trimpath -ldflags "$(LDFLAGS)" -o bin/$(BINARY) ./cmd/lmstudio-exporter

test:
	go test ./...

fmt:
	gofmt -w $$(find . -name '*.go' -type f)

vet:
	go vet ./...

check: test vet
	@test -z "$$(gofmt -l $$(find . -name '*.go' -type f))" || \
		(echo "gofmt required:"; gofmt -l $$(find . -name '*.go' -type f); exit 1)

clean:
	rm -rf bin dist coverage.out
