# Oubliette Makefile
# Test targets and build automation

.PHONY: all build test test-cover test-race test-unit test-integration test-load test-chaos clean lint vet

# Default target
all: lint test build

# Build the server
build:
	go build -o bin/oubliette-server ./cmd/server

# Run all tests
test:
	go test ./... -timeout 60s

# Run tests with coverage report
test-cover:
	go test ./... -coverprofile=coverage.out -covermode=atomic -timeout 60s
	go tool cover -func=coverage.out | tail -n 1
	@echo "Coverage report: coverage.out"
	@echo "View HTML: go tool cover -html=coverage.out"

# Run tests with race detector (slower but catches concurrency bugs)
test-race:
	go test ./... -race -timeout 120s

# Run unit tests only (exclude integration, load, chaos)
test-unit:
	go test ./internal/... -timeout 60s

# Run integration tests
test-integration:
	go test ./test/... -timeout 120s

# Run load tests
test-load:
	go test ./test/load/... -timeout 300s

# Run chaos tests
test-chaos:
	go test ./test/chaos/... -race -timeout 180s

# Run all tests with race detector and coverage
test-full: lint vet
	go test ./... -race -coverprofile=coverage.out -covermode=atomic -timeout 180s
	go tool cover -func=coverage.out | tail -n 1

# Run linter
lint:
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run ./...; \
	else \
		echo "golangci-lint not installed, skipping lint"; \
	fi

# Run go vet
vet:
	go vet ./...

# Clean build artifacts
clean:
	rm -rf bin/ coverage.out

# Watch for changes and run tests (requires watchexec or entr)
watch:
	@if command -v watchexec >/dev/null 2>&1; then \
		watchexec -e go -- go test ./internal/...; \
	elif command -v entr >/dev/null 2>&1; then \
		find . -name '*.go' | entr -c go test ./internal/...; \
	else \
		echo "Install watchexec or entr for file watching"; \
		exit 1; \
	fi

# Generate test coverage HTML report
cover-html: test-cover
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report: coverage.html"

# Run security scanner
security:
	@if command -v gosec >/dev/null 2>&1; then \
		gosec -quiet ./...; \
	else \
		echo "gosec not installed, skipping security scan"; \
	fi

# Run vulnerability check
vuln:
	@if command -v govulncheck >/dev/null 2>&1; then \
		govulncheck ./...; \
	else \
		echo "govulncheck not installed, skipping vulnerability check"; \
	fi

# Pre-commit check (run before committing)
check: lint vet test-race
	@echo "All checks passed!"

# Help
help:
	@echo "Oubliette Makefile Targets:"
	@echo ""
	@echo "  make test          Run all tests"
	@echo "  make test-cover    Run tests with coverage report"
	@echo "  make test-race     Run tests with race detector"
	@echo "  make test-unit     Run unit tests only"
	@echo "  make test-full     Run all tests with race + coverage"
	@echo "  make lint          Run golangci-lint"
	@echo "  make vet           Run go vet"
	@echo "  make check         Pre-commit checks (lint, vet, race tests)"
	@echo "  make build         Build the server"
	@echo "  make clean         Remove build artifacts"
	@echo "  make security      Run security scanner"
	@echo "  make vuln          Run vulnerability check"
