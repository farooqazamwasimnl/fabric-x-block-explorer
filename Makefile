# Copyright IBM Corp. All Rights Reserved.
# SPDX-License-Identifier: Apache-2.0

.PHONY: help build test test-no-db test-requires-db test-all coverage clean

help: ## Display this help message
	@echo "Fabric X Block Explorer - Makefile targets"
	@echo ""
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2}'

build: ## Build the explorer binary
	@echo "Building explorer..."
	go build -o explorer cmd/explorer/main.go
	@echo "Build complete: ./explorer"

test-no-db: ## Run tests that don't require database
	@echo "Running tests without database requirement..."
	go test -v -count=1 \
		./pkg/config/... \
		./pkg/parser/... \
		./pkg/types/... \
		./pkg/workerpool/... \
		./pkg/contracts/... \
		./pkg/sidecarstream/...
	@echo "Running blockpipeline tests (excluding writer tests that need DB)..."
	go test -v -count=1 ./pkg/blockpipeline/... \
		-run="^Test(NewBackoff|BackoffProgression|BackoffReset|BackoffNoStop|ConsumeBlocks|BlockReceiverReconnect|BlockReceiverContextCancellation|BlockProcessor|ProcessBlock)"

test-requires-db: ## Run tests that require database (set DB_DEPLOYMENT=local)
	@echo "Running tests that require database..."
	@echo "Note: Requires DB_DEPLOYMENT=local and postgres-local container running"
	DB_DEPLOYMENT=local go test -v -count=1 \
		./pkg/db/... \
		./pkg/api/...

test-all: ## Run all tests (requires database)
	@echo "Running all tests..."
	@echo "Note: Database tests require DB_DEPLOYMENT=local and postgres-local container"
	DB_DEPLOYMENT=local go test -v -count=1 ./pkg/...

test: test-all ## Alias for test-all

coverage: ## Generate test coverage report
	@echo "Generating coverage report..."
	@mkdir -p coverage
	DB_DEPLOYMENT=local go test -coverprofile=coverage/coverage.out ./pkg/...
	@echo "Filtering coverage data..."
	@./scripts/filter-coverage.sh coverage/coverage.out coverage/coverage-filtered.out
	go tool cover -html=coverage/coverage-filtered.out -o coverage/coverage.html
	go tool cover -func=coverage/coverage-filtered.out
	@echo ""
	@echo "Coverage report generated:"
	@echo "  - coverage/coverage.html (open in browser)"
	@echo "  - coverage/coverage-filtered.out (filtered data)"

clean: ## Remove build artifacts and coverage reports
	@echo "Cleaning build artifacts..."
	rm -f explorer
	rm -rf coverage/
	@echo "Clean complete"

.DEFAULT_GOAL := help
