# Go ONVIF Library Makefile

.PHONY: all build test clean install deps lint fmt vet check examples cli docker

# Configuration
BINARY_DIR := bin
GOPATH := $(shell go env GOPATH)
GOOS := $(shell go env GOOS)
GOARCH := $(shell go env GOARCH)

# Binaries
CLI_BINARY := $(BINARY_DIR)/onvif-cli
QUICK_BINARY := $(BINARY_DIR)/onvif-quick

# Build all targets
all: deps check test build

# Build all binaries
build: $(CLI_BINARY) $(QUICK_BINARY)

# Build CLI tool (comprehensive)
$(CLI_BINARY):
	@echo "🔨 Building ONVIF CLI..."
	@mkdir -p $(BINARY_DIR)
	CGO_ENABLED=0 go build -o $(CLI_BINARY) ./cmd/onvif-cli

# Build quick tool (simple)
$(QUICK_BINARY):
	@echo "🔨 Building ONVIF Quick Tool..."
	@mkdir -p $(BINARY_DIR)
	CGO_ENABLED=0 go build -o $(QUICK_BINARY) ./cmd/onvif-quick

# Install binaries to GOPATH
install: build
	@echo "📦 Installing binaries..."
	cp $(CLI_BINARY) $(GOPATH)/bin/
	cp $(QUICK_BINARY) $(GOPATH)/bin/

# Download dependencies
deps:
	@echo "📥 Downloading dependencies..."
	go mod download
	go mod tidy

# Run tests
test:
	@echo "🧪 Running tests..."
	go test -v -race -coverprofile=coverage.out ./...

# Run tests with coverage report
test-coverage: test
	@echo "📊 Generating coverage report..."
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report: coverage.html"

# Run benchmarks
bench:
	@echo "⚡ Running benchmarks..."
	go test -bench=. -benchmem ./...

# Lint code
lint:
	@echo "🔍 Linting code..."
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run ./...; \
	else \
		echo "⚠️  golangci-lint not installed. Install with: go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest"; \
	fi

# Format code
fmt:
	@echo "🎨 Formatting code..."
	go fmt ./...

# Vet code
vet:
	@echo "🔬 Vetting code..."
	go vet ./...

# Run all checks
check: fmt vet lint

# Clean build artifacts
clean:
	@echo "🧹 Cleaning..."
	rm -rf $(BINARY_DIR)
	rm -f coverage.out coverage.html

# Build examples
examples:
	@echo "📚 Building examples..."
	@mkdir -p $(BINARY_DIR)/examples
	go build -o $(BINARY_DIR)/examples/discovery ./examples/discovery
	go build -o $(BINARY_DIR)/examples/device_info ./examples/device_info
	go build -o $(BINARY_DIR)/examples/media ./examples/media
	go build -o $(BINARY_DIR)/examples/ptz ./examples/ptz

# Build for multiple platforms
build-all:
	@echo "🌍 Building for multiple platforms..."
	@mkdir -p $(BINARY_DIR)
	
	# Linux AMD64
	GOOS=linux GOARCH=amd64 go build -o $(BINARY_DIR)/onvif-cli-linux-amd64 ./cmd/onvif-cli
	GOOS=linux GOARCH=amd64 go build -o $(BINARY_DIR)/onvif-quick-linux-amd64 ./cmd/onvif-quick
	
	# Linux ARM64
	GOOS=linux GOARCH=arm64 go build -o $(BINARY_DIR)/onvif-cli-linux-arm64 ./cmd/onvif-cli
	GOOS=linux GOARCH=arm64 go build -o $(BINARY_DIR)/onvif-quick-linux-arm64 ./cmd/onvif-quick
	
	# Windows AMD64
	GOOS=windows GOARCH=amd64 go build -o $(BINARY_DIR)/onvif-cli-windows-amd64.exe ./cmd/onvif-cli
	GOOS=windows GOARCH=amd64 go build -o $(BINARY_DIR)/onvif-quick-windows-amd64.exe ./cmd/onvif-quick
	
	# macOS AMD64
	GOOS=darwin GOARCH=amd64 go build -o $(BINARY_DIR)/onvif-cli-darwin-amd64 ./cmd/onvif-cli
	GOOS=darwin GOARCH=amd64 go build -o $(BINARY_DIR)/onvif-quick-darwin-amd64 ./cmd/onvif-quick
	
	# macOS ARM64 (Apple Silicon)
	GOOS=darwin GOARCH=arm64 go build -o $(BINARY_DIR)/onvif-cli-darwin-arm64 ./cmd/onvif-cli
	GOOS=darwin GOARCH=arm64 go build -o $(BINARY_DIR)/onvif-quick-darwin-arm64 ./cmd/onvif-quick

# Create Docker image
docker:
	@echo "🐳 Building Docker image..."
	docker build -t go-onvif:latest .

# Development setup
dev-setup:
	@echo "🛠️  Setting up development environment..."
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	go install golang.org/x/tools/cmd/goimports@latest
	go mod download

# Run quick tool
run-quick:
	@if [ ! -f $(QUICK_BINARY) ]; then $(MAKE) $(QUICK_BINARY); fi
	$(QUICK_BINARY)

# Run CLI tool
run-cli:
	@if [ ! -f $(CLI_BINARY) ]; then $(MAKE) $(CLI_BINARY); fi
	$(CLI_BINARY)

# Show help
help:
	@echo "📖 Available targets:"
	@echo "  all          - Build, test, and check everything"
	@echo "  build        - Build both CLI tools"
	@echo "  test         - Run tests"
	@echo "  test-coverage- Run tests with coverage report"
	@echo "  bench        - Run benchmarks"
	@echo "  check        - Run fmt, vet, and lint"
	@echo "  clean        - Clean build artifacts"
	@echo "  install      - Install binaries to GOPATH"
	@echo "  examples     - Build example programs"
	@echo "  build-all    - Build for multiple platforms"
	@echo "  docker       - Build Docker image"
	@echo "  dev-setup    - Set up development environment"
	@echo "  run-quick    - Run the quick tool"
	@echo "  run-cli      - Run the comprehensive CLI"
	@echo "  help         - Show this help"