.PHONY: help build test lint clean run docker-build

# Build variables
BINARY_NAME=platform-api
VERSION?=dev
GIT_COMMIT=$(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILD_DATE=$(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
LDFLAGS=-ldflags "-X main.Version=${VERSION} -X main.GitCommit=${GIT_COMMIT} -X main.BuildDate=${BUILD_DATE}"

help: ## Show this help message
	@echo 'Usage: make [target]'
	@echo ''
	@echo 'Available targets:'
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  %-15s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

build: ## Build the binary
	@echo "Building ${BINARY_NAME}..."
	go build ${LDFLAGS} -o ${BINARY_NAME} .

test: ## Run tests
	@echo "Running tests..."
	go test -v -race -coverprofile=coverage.out ./...

test-coverage: test ## Run tests and show coverage
	go tool cover -html=coverage.out

lint: ## Run linter
	@echo "Running linter..."
	@if command -v golangci-lint > /dev/null; then \
		golangci-lint run; \
	else \
		echo "golangci-lint not installed. Install with: go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest"; \
	fi

clean: ## Clean build artifacts
	@echo "Cleaning..."
	rm -f ${BINARY_NAME}
	rm -f coverage.out
	go clean

run: build ## Build and run the service
	./${BINARY_NAME}

docker-build: ## Build Docker image
	@echo "Building Docker image..."
	docker build -t homelab/platform-api:${VERSION} .

tidy: ## Tidy go modules
	go mod tidy

deps: ## Download dependencies
	go mod download

dev: ## Run in development mode (with live reload if air is installed)
	@if command -v air > /dev/null; then \
		air; \
	else \
		echo "air not installed. Install with: go install github.com/cosmtrek/air@latest"; \
		echo "Running without live reload..."; \
		go run ${LDFLAGS} .; \
	fi
