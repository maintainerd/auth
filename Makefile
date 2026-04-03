APP_NAME := auth
MAIN := cmd/server/main.go
PROTO_SRC := internal/contract
PROTO_OUT := internal/gen/go

.PHONY: run build clean proto proto-clean tidy test test-cover test-race

# Run the main application
run:
	go run $(MAIN)

# Build the binary
build:
	go build -o bin/$(APP_NAME) $(MAIN)

# Clean build artifacts
clean:
	rm -rf bin

# Generate Go code from .proto definitions
proto:
	@echo "Generating Go gRPC code from proto files..."
	@mkdir -p $(PROTO_OUT)
	protoc \
		--go_out=$(PROTO_OUT) \
		--go-grpc_out=$(PROTO_OUT) \
		--go_opt=paths=source_relative \
		--go-grpc_opt=paths=source_relative \
		-I $(PROTO_SRC) \
		$$(find $(PROTO_SRC) -name "*.proto")

# Clean generated proto files
proto-clean:
	@echo "Cleaning generated proto files..."
	@rm -rf $(PROTO_OUT)

# Tidy up dependencies
tidy:
	go mod tidy

# Run all unit tests
test:
	go test ./... -count=1

# Run all unit tests with race detector
test-race:
	go test ./... -count=1 -race

# Run unit tests and output an HTML coverage report (opens in browser)
test-cover:
	go test ./... -count=1 -coverprofile=coverage.out
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report written to coverage.html"