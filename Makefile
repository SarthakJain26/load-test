# Makefile for Load Manager Control Plane

# Variables
BINARY_NAME=controlplane
BINARY_PATH=bin/$(BINARY_NAME)
MAIN_PATH=cmd/controlplane/main.go
CONFIG_PATH=config/config.yaml

# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOMOD=$(GOCMD) mod
GOFMT=$(GOCMD) fmt

.PHONY: all build clean test run deps fmt help

# Default target
all: deps fmt test build

# Build the application
build:
	@echo "Building $(BINARY_NAME)..."
	@mkdir -p bin
	$(GOBUILD) -o $(BINARY_PATH) $(MAIN_PATH)
	@echo "Build complete: $(BINARY_PATH)"

# Run the application
run: build
	@echo "Running $(BINARY_NAME)..."
	./$(BINARY_PATH) -config $(CONFIG_PATH)

# Run without building (using go run)
run-dev:
	@echo "Running in development mode..."
	$(GOCMD) run $(MAIN_PATH) -config $(CONFIG_PATH)

# Clean build artifacts
clean:
	@echo "Cleaning..."
	$(GOCLEAN)
	rm -rf bin/
	@echo "Clean complete"

# Run tests
test:
	@echo "Running tests..."
	$(GOTEST) -v ./...

# Run tests with coverage
test-coverage:
	@echo "Running tests with coverage..."
	$(GOTEST) -v -coverprofile=coverage.out ./...
	$(GOCMD) tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

# Install/update dependencies
deps:
	@echo "Installing dependencies..."
	$(GOMOD) download
	$(GOMOD) tidy
	@echo "Dependencies installed"

# Format code
fmt:
	@echo "Formatting code..."
	$(GOFMT) ./...
	@echo "Formatting complete"

# Lint code (requires golangci-lint)
lint:
	@echo "Linting code..."
	@which golangci-lint > /dev/null || (echo "golangci-lint not found. Install from https://golangci-lint.run/usage/install/" && exit 1)
	golangci-lint run ./...
	@echo "Linting complete"

# Build Docker image for control plane
docker-build:
	@echo "Building Docker image..."
	docker build -t load-manager-control-plane:latest .
	@echo "Docker image built: load-manager-control-plane:latest"

# Build Locust Docker image
docker-build-locust:
	@echo "Building Locust Docker image..."
	cd locust && docker build -t locust-with-hooks:latest .
	@echo "Locust Docker image built: locust-with-hooks:latest"

# Start Locust cluster with Docker Compose
locust-up:
	@echo "Starting Locust cluster..."
	cd locust && docker-compose up -d
	@echo "Locust cluster started. Web UI: http://localhost:8089"

# Stop Locust cluster
locust-down:
	@echo "Stopping Locust cluster..."
	cd locust && docker-compose down
	@echo "Locust cluster stopped"

# View Locust logs
locust-logs:
	cd locust && docker-compose logs -f

# Install Python dependencies for Locust (local development)
locust-deps:
	@echo "Installing Locust Python dependencies..."
	cd locust && pip install -r requirements.txt
	@echo "Locust dependencies installed"

# Help
help:
	@echo "Available targets:"
	@echo "  make build            - Build the control plane binary"
	@echo "  make run              - Build and run the control plane"
	@echo "  make run-dev          - Run in development mode (go run)"
	@echo "  make clean            - Remove build artifacts"
	@echo "  make test             - Run tests"
	@echo "  make test-coverage    - Run tests with coverage report"
	@echo "  make deps             - Install/update Go dependencies"
	@echo "  make fmt              - Format Go code"
	@echo "  make lint             - Lint Go code (requires golangci-lint)"
	@echo "  make docker-build     - Build control plane Docker image"
	@echo "  make docker-build-locust - Build Locust Docker image"
	@echo "  make locust-up        - Start Locust cluster (Docker Compose)"
	@echo "  make locust-down      - Stop Locust cluster"
	@echo "  make locust-logs      - View Locust cluster logs"
	@echo "  make locust-deps      - Install Locust Python dependencies"
	@echo "  make help             - Show this help message"
