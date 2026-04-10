# Makefile for rustdesk-api
#
# Environment variables (scoped to each invocation — not persisted via `go env -w`):
#   GOOS     target OS       (default: host)
#   GOARCH   target arch     (default: host)
#   CGO_ENABLED              (default: 1, required by the sqlite driver)
#   GOPROXY                  (default: current go env)
#   DOCS     set empty to skip swagger generation (default: true)

GOOS        ?= $(shell go env GOOS)
GOARCH      ?= $(shell go env GOARCH)
CGO_ENABLED ?= 1
GOPROXY     ?= $(shell go env GOPROXY)
DOCS        ?= true

RELEASE_DIR := release
BINARY_NAME := apimain
ifeq ($(GOOS),windows)
	BINARY_NAME := apimain.exe
endif

GO_ENV := GO111MODULE=on \
	GOPROXY=$(GOPROXY) \
	CGO_ENABLED=$(CGO_ENABLED) \
	GOOS=$(GOOS) \
	GOARCH=$(GOARCH)

.PHONY: all build clean docs resources dirs help generate-keypair

all: build

help:
	@echo "Targets:"
	@echo "  build      Clean, generate docs, compile binary, and stage resources (default)"
	@echo "  docs       Generate Swagger documentation"
	@echo "  clean      Remove the release directory"
	@echo ""
	@echo "Override variables on the command line, e.g.:"
	@echo "  make build GOOS=linux GOARCH=arm64"
	@echo "  make build DOCS="

clean:
	rm -rf $(RELEASE_DIR)

docs:
ifeq ($(DOCS),)
	@echo "Skipping Swagger documentation generation (DOCS is empty)."
else
	@if ! command -v swag >/dev/null 2>&1; then \
		echo "swag command not found. Install it with:"; \
		echo "  go install github.com/swaggo/swag/cmd/swag@latest"; \
		exit 1; \
	fi
	@echo "Generating Swagger documentation..."
	swag init -g cmd/apimain.go --output docs/api   --instanceName api   --exclude internal/http/controller/admin
	swag init -g cmd/apimain.go --output docs/admin --instanceName admin --exclude internal/http/controller/api
endif

build: clean docs
	@echo "Building $(BINARY_NAME) for $(GOOS)/$(GOARCH)..."
	$(GO_ENV) go build -o $(RELEASE_DIR)/$(BINARY_NAME) cmd/apimain.go
	@$(MAKE) resources

resources: dirs
	cp -a resources $(RELEASE_DIR)/
	cp -a docs      $(RELEASE_DIR)/
	cp -a conf      $(RELEASE_DIR)/

dirs:
	mkdir -p $(RELEASE_DIR)/data
	mkdir -p $(RELEASE_DIR)/runtime

generate-keypair:
	@go run scripts/generate_keypair.go
