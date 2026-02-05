.PHONY: build run clean test deps desktop-deps desktop-dev desktop-build desktop-build-arm desktop-build-intel

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

# Desktop app (Wails)
desktop-deps:
	cd desktop && go mod tidy

desktop-dev:
	cd desktop && wails dev

desktop-build:
	cd desktop && wails build -platform darwin/universal

desktop-build-arm:
	cd desktop && wails build -platform darwin/arm64

desktop-build-intel:
	cd desktop && wails build -platform darwin/amd64
