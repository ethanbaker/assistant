# Makefile for Personal AI Assistant

.PHONY: build run test clean setup docker-up docker-down

# Build the application
build:
	go build -o bin/assistant cmd/main.go

# Run the application
run: build
	./bin/assistant

# Run tests
test:
	go test -v ./...

# Clean build artifacts
clean:
	rm -rf bin/

# Setup development environment
setup:
	./scripts/setup.sh

# Start Docker services (MySQL)
docker-up:
	docker-compose up -d

# Stop Docker services
docker-down:
	docker-compose down

# Install dependencies
deps:
	go mod download
	go mod tidy

# Format code
fmt:
	go fmt ./...

# Lint code
lint:
	golangci-lint run

# Build for production
build-prod:
	CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o bin/assistant cmd/main.go

# Run in dry-run mode (all agents will mock their operations)
run-dry:
	DRY_RUN=true ./bin/assistant
