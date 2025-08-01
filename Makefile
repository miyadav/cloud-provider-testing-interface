# Makefile for cloud-provider-testing-interface

.PHONY: help test test-verbose build clean fmt lint vet coverage

# Default target
help:
	@echo "Available targets:"
	@echo "  test         - Run tests"
	@echo "  test-verbose - Run tests with verbose output"
	@echo "  build        - Build the package"
	@echo "  clean        - Clean build artifacts"
	@echo "  fmt          - Format code"
	@echo "  lint         - Run linter"
	@echo "  vet          - Run go vet"
	@echo "  coverage     - Run tests with coverage"

# Run tests
test:
	go test ./...

# Run tests with verbose output
test-verbose:
	go test -v ./...

# Build the package
build:
	go build ./...

# Clean build artifacts
clean:
	go clean ./...
	rm -rf coverage.out

# Format code
fmt:
	go fmt ./...

# Run linter
lint:
	golangci-lint run

# Run go vet
vet:
	go vet ./...

# Run tests with coverage
coverage:
	go test -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

# Install dependencies
deps:
	go mod download
	go mod tidy

# Run all checks
check: fmt vet test
	@echo "All checks passed!"

# Run integration tests (if any)
integration-test:
	go test -tags=integration ./...

# Run benchmarks
bench:
	go test -bench=. ./...

# Generate documentation
docs:
	godoc -http=:6060 &
	@echo "Documentation server started at http://localhost:6060"
	@echo "Press Ctrl+C to stop"

# Install development tools
install-tools:
	go install golang.org/x/lint/golint@latest
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	go install golang.org/x/tools/cmd/goimports@latest

# Update dependencies
update-deps:
	go get -u ./...
	go mod tidy

# Show test coverage summary
coverage-summary:
	go test -cover ./...

# Run race detector
race:
	go test -race ./...

# Run tests with timeout
test-timeout:
	go test -timeout=30s ./...

# Generate mocks (if using mockery)
mocks:
	mockery --all --output=./mocks

# Run specific test
test-specific:
	@read -p "Enter test name: " test_name; \
	go test -v -run $$test_name ./...

# Run tests in parallel
test-parallel:
	go test -parallel=4 ./...

# Show test dependencies
test-deps:
	go list -f '{{.TestImports}}' ./...

# Run tests with specific tags
test-tags:
	@read -p "Enter tags (e.g., integration): " tags; \
	go test -tags=$$tags ./...

# Generate test binary
test-binary:
	go test -c ./...

# Run tests with memory profiling
test-memprofile:
	go test -memprofile=mem.prof ./...
	go tool pprof mem.prof

# Run tests with CPU profiling
test-cpuprofile:
	go test -cpuprofile=cpu.prof ./...
	go tool pprof cpu.prof

# Show package information
info:
	go list -f '{{.Name}} {{.ImportPath}} {{.Dir}}' ./...

# Show module information
module-info:
	go mod graph
	go mod why

# Clean and rebuild
rebuild: clean build

# Full development cycle
dev-cycle: deps fmt vet test build
	@echo "Development cycle completed successfully!"

# CI/CD pipeline
ci: deps fmt vet test-verbose coverage
	@echo "CI pipeline completed successfully!"

# Local development setup
setup: install-tools deps
	@echo "Development environment setup completed!" 