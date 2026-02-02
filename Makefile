.PHONY: build run clean test deps

# Binary name
BINARY_NAME=llm-agent
BUILD_DIR=bin

# Go commands
GOCMD=go
GOBUILD=$(GOCMD) build
GORUN=$(GOCMD) run
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOMOD=$(GOCMD) mod

# Build the project
build:
	@mkdir -p $(BUILD_DIR)
	$(GOBUILD) -o $(BUILD_DIR)/$(BINARY_NAME) ./cmd/server

# Run the project
run:
	$(GORUN) ./cmd/server -config ./configs/config.yaml

# Run with hot reload (requires air)
dev:
	air

# Clean build artifacts
clean:
	@rm -rf $(BUILD_DIR)
	@rm -f $(BINARY_NAME)

# Run tests
test:
	$(GOTEST) -v ./...

# Install dependencies
deps:
	$(GOMOD) download
	$(GOMOD) tidy

# Generate encryption key
genkey:
	@$(GORUN) -e 'package main; import ("crypto/rand"; "encoding/base64"; "fmt"); func main() { k := make([]byte, 32); rand.Read(k); fmt.Println(base64.StdEncoding.EncodeToString(k)) }'

# Docker build
docker-build:
	docker build -t $(BINARY_NAME) .

# Docker run
docker-run:
	docker run -p 8080:8080 $(BINARY_NAME)
