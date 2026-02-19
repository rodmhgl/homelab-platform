.PHONY: build install test clean run help

# Build variables
BINARY_NAME=rdp
VERSION?=dev
GIT_COMMIT=$(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILD_DATE=$(shell date -u +%Y-%m-%dT%H:%M:%SZ)
LDFLAGS=-ldflags "\
	-X github.com/rodmhgl/homelab-platform/cli/cmd.Version=$(VERSION) \
	-X github.com/rodmhgl/homelab-platform/cli/cmd.GitCommit=$(GIT_COMMIT) \
	-X github.com/rodmhgl/homelab-platform/cli/cmd.BuildDate=$(BUILD_DATE)"

help: ## Display this help
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z_-]+:.*?##/ { printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)

build: ## Build the CLI binary
	go build $(LDFLAGS) -o $(BINARY_NAME) .

install: ## Install the CLI to $GOPATH/bin
	go install $(LDFLAGS) .

run: ## Run the CLI (pass args with ARGS="...")
	go run $(LDFLAGS) . $(ARGS)

test: ## Run tests
	go test -v ./...

clean: ## Remove build artifacts
	rm -f $(BINARY_NAME)
	go clean

fmt: ## Format code
	go fmt ./...

vet: ## Run go vet
	go vet ./...

deps: ## Download dependencies
	go mod download
	go mod tidy

.DEFAULT_GOAL := help
