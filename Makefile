.PHONY: help build docker-build docker-run clean test lint

BINARY_NAME=packwiz-pull-serve
DOCKER_IMAGE=packwiz-pull-serve:latest
GO_FLAGS=-v

help:
	@echo "packwiz-pull-serve - Make targets:"
	@echo ""
	@echo "  make build          - Build the Go application locally"
	@echo "  make docker-build   - Build Docker image"
	@echo "  make docker-run     - Run Docker container (requires .env)"
	@echo "  make docker-compose - Run with docker-compose (requires .env)"
	@echo "  make clean          - Clean build artifacts"
	@echo "  make test           - Run tests (placeholder)"
	@echo "  make lint           - Run linting (requires golangci-lint)"
	@echo "  make help           - Show this help message"

build:
	@echo "Building $(BINARY_NAME)..."
	go build $(GO_FLAGS) -o $(BINARY_NAME) .
	@echo "✓ Build complete: ./$(BINARY_NAME)"

docker-build:
	@echo "Building Docker image $(DOCKER_IMAGE)..."
	docker build -t $(DOCKER_IMAGE) .
	@echo "✓ Docker image built successfully"

docker-run: docker-build
	@echo "Running Docker container (ensure .env is configured)..."
	docker run --rm \
		-p 8080:8080 \
		-p 8081:8081 \
		-v packwiz-repo:/tmp/packwiz-repo \
		--env-file .env \
		$(DOCKER_IMAGE)

docker-compose:
	@echo "Starting with docker-compose (ensure .env is configured)..."
	docker-compose up

docker-compose-down:
	docker-compose down

clean:
	@echo "Cleaning build artifacts..."
	rm -f $(BINARY_NAME)
	go clean
	@echo "✓ Cleaned"

test:
	@echo "Running tests..."
	go test -v ./...

lint:
	@echo "Running linter..."
	golangci-lint run ./...

deps:
	@echo "Downloading dependencies..."
	go mod download
	go mod tidy
	@echo "✓ Dependencies updated"

run: build
	@echo "Running application (ensure environment variables set)..."
	./$(BINARY_NAME)

fmt:
	@echo "Formatting code..."
	go fmt ./...
	@echo "✓ Formatted"

vet:
	@echo "Running go vet..."
	go vet ./...
