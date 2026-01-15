.PHONY: build run-api run-worker test test-verbose clean tidy

# Build binaries
build:
	go build -o bin/api ./cmd/api
	go build -o bin/worker ./cmd/worker

# Run API server
run-api:
	go run ./cmd/api

# Run worker
run-worker:
	go run ./cmd/worker

# Run tests
test:
	go test ./...

# Run tests with verbose output
test-verbose:
	go test -v ./...

# Clean build artifacts
clean:
	rm -rf bin/

# Tidy dependencies
tidy:
	go mod tidy

# Run both API and worker (requires goreman or similar)
dev:
	@echo "Run 'make run-api' and 'make run-worker' in separate terminals"
