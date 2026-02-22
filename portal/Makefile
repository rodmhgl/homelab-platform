.PHONY: help install build test lint clean dev preview docker-build docker-push serve

# Build variables
APP_NAME=portal-ui
VERSION?=v0.1.0
GIT_COMMIT=$(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILD_DATE=$(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
DOCKER_REGISTRY?=homelabplatformacr.azurecr.io
DOCKER_IMAGE=${DOCKER_REGISTRY}/${APP_NAME}:${VERSION}
DOCKER_IMAGE_LATEST=${DOCKER_REGISTRY}/${APP_NAME}:latest

# Runtime configuration
VITE_API_URL?=

help: ## Show this help message
	@echo 'Usage: make [target]'
	@echo ''
	@echo 'Available targets:'
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  %-15s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

install: ## Install dependencies
	@echo "Installing dependencies..."
	npm ci

build: ## Build production bundle
	@echo "Building ${APP_NAME}..."
	@echo "Version: ${VERSION}"
	@echo "Git Commit: ${GIT_COMMIT}"
	@echo "Build Date: ${BUILD_DATE}"
	npm run build

test: ## Run tests (placeholder - add test framework later)
	@echo "Running tests..."
	@echo "No tests configured yet. Consider adding Vitest or Jest."

lint: ## Run ESLint
	@echo "Running linter..."
	npm run lint

lint-fix: ## Run ESLint with --fix
	@echo "Running linter with auto-fix..."
	npx eslint . --fix

type-check: ## Run TypeScript type checking
	@echo "Running TypeScript type checker..."
	npx tsc --noEmit

clean: ## Clean build artifacts and node_modules
	@echo "Cleaning..."
	rm -rf dist
	rm -rf node_modules
	rm -rf .vite

dev: ## Run development server
	@echo "Starting dev server..."
	npm run dev

preview: build ## Preview production build locally
	@echo "Starting preview server..."
	npm run preview

docker-build: ## Build Docker image
	@echo "Building Docker image..."
	docker build \
		--build-arg VITE_API_URL="${VITE_API_URL}" \
		--build-arg BUILD_VERSION="${VERSION}" \
		--build-arg BUILD_COMMIT="${GIT_COMMIT}" \
		--build-arg BUILD_DATE="${BUILD_DATE}" \
		-t ${DOCKER_IMAGE} \
		-t ${DOCKER_IMAGE_LATEST} \
		.
	@echo "Built: ${DOCKER_IMAGE}"

docker-push: docker-build ## Build and push Docker image to registry
	@echo "Authenticating with Docker registry..."
	az acr login -n ${DOCKER_REGISTRY}
	@echo "Pushing Docker image..."
	docker push ${DOCKER_IMAGE}
	docker push ${DOCKER_IMAGE_LATEST}
	@echo "Pushed: ${DOCKER_IMAGE}"

docker-run: ## Run Docker container locally
	@echo "Running Docker container..."
	docker run --rm -p 8080:8080 ${DOCKER_IMAGE}

serve: build ## Serve production build using simple HTTP server
	@echo "Serving production build on http://localhost:8080..."
	@if command -v python3 > /dev/null; then \
		cd dist && python3 -m http.server 8080; \
	elif command -v python > /dev/null; then \
		cd dist && python -m SimpleHTTPServer 8080; \
	else \
		echo "Python not found. Install python3 or use 'make preview' instead."; \
		exit 1; \
	fi

deps: install ## Alias for install

outdated: ## Check for outdated dependencies
	@echo "Checking for outdated packages..."
	npm outdated

update: ## Update dependencies (interactive)
	@echo "Updating dependencies..."
	npm update

check: lint type-check ## Run all checks (lint + type-check)
	@echo "All checks passed!"