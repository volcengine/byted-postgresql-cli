# Volcengine PostgreSQL CLI build and release targets.

BINARY := byted-postgresql-cli
PKG := github.com/volcengine/byted-postgresql-cli
MAIN := .
BIN_DIR := bin
DIST_DIR := dist
DARWIN_ARCH ?= $(shell go env GOARCH)

VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
LDFLAGS := -s -w -X $(PKG)/cmd.Version=$(VERSION)
GOFLAGS := -trimpath -ldflags "$(LDFLAGS)"

.PHONY: all build install test test-evals test-evals-dry test-e2e test-e2e-replay test-e2e-live vet fmt lint tidy clean release test-npm release-dry npm-release

all: build

build:
	@mkdir -p $(BIN_DIR) $(DIST_DIR)
	CGO_ENABLED=0 go build $(GOFLAGS) -o $(BIN_DIR)/$(BINARY) $(MAIN)
	CGO_ENABLED=0 GOOS=darwin GOARCH=$(DARWIN_ARCH) go build $(GOFLAGS) -o $(DIST_DIR)/$(BINARY)-darwin-$(DARWIN_ARCH) $(MAIN)
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build $(GOFLAGS) -o $(DIST_DIR)/$(BINARY)-linux-amd64 $(MAIN)

install:
	CGO_ENABLED=0 go install $(GOFLAGS) $(MAIN)

test:
	go test ./...

test-e2e:
	go test ./test/e2e/... -run 'Test.*' -count=1

test-e2e-replay:
	VOLCENGINE_E2E_MODE=replay go test ./test/e2e/... -run 'Test' -count=1

test-e2e-live:
	VOLCENGINE_E2E_MODE=live go test ./test/e2e/... -run 'TestLive' -count=1

vet:
	go vet ./...

fmt:
	gofmt -w .

lint: vet
	@test -z "$$(gofmt -l .)" || { echo "gofmt issues found"; gofmt -l .; exit 1; }

tidy:
	go mod tidy

clean:
	rm -rf $(BIN_DIR) $(DIST_DIR)

release:
	@mkdir -p $(DIST_DIR)
	@set -e; for platform in darwin/amd64 darwin/arm64 linux/amd64 linux/arm64 windows/amd64 windows/arm64; do \
		os=$${platform%/*}; arch=$${platform#*/}; ext=""; \
		if [ "$$os" = "windows" ]; then ext=".exe"; fi; \
		CGO_ENABLED=0 GOOS=$$os GOARCH=$$arch go build $(GOFLAGS) -o "$(DIST_DIR)/$(BINARY)-$$os-$$arch$$ext" $(MAIN); \
	done

test-npm:
	node --test scripts/install.test.mjs

release-dry:
	node scripts/release.mjs --dry-run

npm-release:
	node scripts/release.mjs
