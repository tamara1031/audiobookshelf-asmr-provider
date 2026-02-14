.PHONY: build run test test-unit test-integration lint fmt vet tidy clean docker-build help

# ── Variables ───────────────────────────────────────────────
APP_NAME     := audiobookshelf-asmr-provider
BINARY_NAME  := server
CMD_DIR      := ./cmd/server
DOCKER_IMAGE := $(APP_NAME)
GOBIN        := $(shell go env GOPATH)/bin

# Build flags
LDFLAGS := -s -w
CGO_ENABLED ?= 0

# ── Help (default) ──────────────────────────────────────────
help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | \
		awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}'

# ── Build & Run ─────────────────────────────────────────────
build: ## Build the application binary
	CGO_ENABLED=$(CGO_ENABLED) go build -ldflags "$(LDFLAGS)" -o $(BINARY_NAME) $(CMD_DIR)/main.go

run: ## Run the application
	go run $(CMD_DIR)/main.go

# ── Testing ─────────────────────────────────────────────────
test: ## Run all tests (unit + integration)
	go test -v -race -count=1 ./...

test-unit: ## Run unit tests only (exclude integration)
	go test -v -race -count=1 -short $$(go list ./... | grep -v /test/)

test-integration: ## Run integration tests only
	go test -v -race -count=1 ./test/...

test-cover: ## Run tests with coverage report
	go test -v -race -count=1 -coverprofile=coverage.out ./...
	go tool cover -func=coverage.out
	@rm -f coverage.out

# ── Code Quality ────────────────────────────────────────────
lint: ## Run golangci-lint
	golangci-lint run ./...

fmt: ## Format code with gofmt and goimports
	gofmt -s -w .
	$(GOBIN)/goimports -w -local $(APP_NAME) .

vet: ## Run go vet
	go vet ./...

check: lint vet ## Run all static analysis (lint + vet)

# ── Dependencies ────────────────────────────────────────────
tidy: ## Tidy and verify go modules
	go mod tidy
	go mod verify

# ── Docker ──────────────────────────────────────────────────
docker-build: ## Build docker image
	docker build -t $(DOCKER_IMAGE) .

# ── Cleanup ─────────────────────────────────────────────────
clean: ## Remove build artifacts
	rm -f $(BINARY_NAME)
	rm -f coverage.out
