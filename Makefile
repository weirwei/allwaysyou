.PHONY: build run test clean docker-build docker-up docker-down deps

# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GORUN=$(GOCMD) run
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOMOD=$(GOCMD) mod
BINARY_NAME=llm-agent

# Build the application
build:
	$(GOBUILD) -o $(BINARY_NAME) ./cmd/server

# Run the application
run:
	$(GORUN) ./cmd/server

# Run tests
test:
	$(GOTEST) -v ./...

# Clean build artifacts
clean:
	rm -f $(BINARY_NAME)

# Download dependencies
deps:
	$(GOMOD) download
	$(GOMOD) tidy

# Docker commands
docker-build:
	docker-compose build

docker-up:
	docker-compose up -d

docker-down:
	docker-compose down

docker-logs:
	docker-compose logs -f app

# Development with hot reload (requires air)
dev:
	air

# Generate swagger docs (requires swag)
swagger:
	swag init -g cmd/server/main.go -o docs
