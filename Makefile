.PHONY: build test lint vet fmt tidy clean test-all

# Build all packages
build:
	go build ./...

# Run tests for root module
test:
	go test ./... -count=1

# Run tests for all modules (root + contrib)
test-all: test
	cd contrib/schedule && go test ./... -count=1
	cd contrib/auth/redis && go build ./...

# Run golangci-lint
lint:
	golangci-lint run ./...

# Run go vet
vet:
	go vet ./...

# Format code
fmt:
	gofmt -w .
	goimports -w -local github.com/bizjs/kratoscarf .

# Tidy all go.mod files
tidy:
	go mod tidy
	cd contrib/auth/redis && go mod tidy
	cd contrib/schedule && go mod tidy
	cd examples && go mod tidy

# Run all checks (CI equivalent)
ci: fmt vet lint test-all

# Clean build cache
clean:
	go clean -cache -testcache
