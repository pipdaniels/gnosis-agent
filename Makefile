# Makefile for GeoAgent Platform

.PHONY: help build run test clean docker-build docker-up docker-down install-deps

# Variables
BINARY_NAME=geoagent
VERSION?=1.0.0
BUILD_DIR=bin
DOCKER_IMAGE=geoagent
DOCKER_TAG=latest

# Colors
GREEN=\033[0;32m
YELLOW=\033[1;33m
NC=\033[0m # No Color

help: ## Show this help message
	@echo 'Usage: make [target]'
	@echo ''
	@echo 'Available targets:'
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  ${GREEN}%-15s${NC} %s\n", $$1, $$2}' $(MAKEFILE_LIST)

install-deps: ## Install development dependencies
	@echo "${YELLOW}Installing dependencies...${NC}"
	@./scripts/install-deps.sh
	@echo "${GREEN}Dependencies installed${NC}"

generate: ## Generate Templ templates
	@echo "${YELLOW}Generating templates...${NC}"
	@templ generate
	@echo "${GREEN}Templates generated${NC}"

build: generate ## Build the application
	@echo "${YELLOW}Building application...${NC}"
	@./scripts/build.sh
	@echo "${GREEN}Build complete${NC}"

build-all: ## Build for all platforms
	@echo "${YELLOW}Building for all platforms...${NC}"
	@PLATFORMS="linux/amd64 linux/arm64 darwin/amd64 darwin/arm64 windows/amd64" ./scripts/build.sh
	@echo "${GREEN}All builds complete${NC}"

run: build## Run the application locally
	@echo "${YELLOW}Starting application...${NC}"
	@go run main.go start --config configs/org.example.yaml

test: ## Run tests
	@echo "${YELLOW}Running tests...${NC}"
	@go test -v ./...
	@echo "${GREEN}Tests complete${NC}"

test-integration: ## Run integration tests
	@echo "${YELLOW}Running integration tests...${NC}"
	@go test -v ./tests/integration/...
	@echo "${GREEN}Integration tests complete${NC}"

test-coverage: ## Run tests with coverage
	@echo "${YELLOW}Running tests with coverage...${NC}"
	@go test -coverprofile=coverage.out ./...
	@go tool cover -html=coverage.out -o coverage.html
	@echo "${GREEN}Coverage report: coverage.html${NC}"

lint: ## Run linters
	@echo "${YELLOW}Running linters...${NC}"
	@golangci-lint run
	@echo "${GREEN}Linting complete${NC}"

clean: ## Clean build artifacts
	@echo "${YELLOW}Cleaning...${NC}"
	@rm -rf $(BUILD_DIR)
	@rm -rf web/templates/*_templ.go
	@rm -f internal/ml/mlpack/*.o internal/ml/mlpack/*.a
	@rm -f coverage.out coverage.html
	@echo "${GREEN}Clean complete${NC}"

docker-build: ## Build Docker image
	@echo "${YELLOW}Building Docker image...${NC}"
	@docker build -t $(DOCKER_IMAGE):$(DOCKER_TAG) .
	@docker tag $(DOCKER_IMAGE):$(DOCKER_TAG) $(DOCKER_IMAGE):$(VERSION)
	@echo "${GREEN}Docker image built: $(DOCKER_IMAGE):$(DOCKER_TAG)${NC}"

docker-up: ## Start Docker Compose services
	@echo "${YELLOW}Starting services...${NC}"
	@docker-compose up -d
	@echo "${GREEN}Services started${NC}"
	@echo "GeoAgent: http://localhost:8080"
	@echo "MongoDB Admin: http://localhost:8081 (use --profile admin)"

docker-down: ## Stop Docker Compose services
	@echo "${YELLOW}Stopping services...${NC}"
	@docker-compose down
	@echo "${GREEN}Services stopped${NC}"

docker-logs: ## View Docker Compose logs
	@docker-compose logs -f

docker-clean: ## Clean Docker resources
	@echo "${YELLOW}Cleaning Docker resources...${NC}"
	@docker-compose down -v
	@docker rmi $(DOCKER_IMAGE):$(DOCKER_TAG) $(DOCKER_IMAGE):$(VERSION) || true
	@echo "${GREEN}Docker cleanup complete${NC}"

dev: ## Run in development mode with hot reload
	@echo "${YELLOW}Generating templates...${NC}"
	@templ generate
	@echo "${YELLOW}Starting development server...${NC}"
	@air || go run ./main.go start --config configs/org.example.yaml

format: ## Format code
	@echo "${YELLOW}Formatting code...${NC}"
	@go fmt ./...
	@templ fmt .
	@echo "${GREEN}Format complete${NC}"

mod-tidy: ## Tidy Go modules
	@echo "${YELLOW}Tidying modules...${NC}"
	@go mod tidy
	@echo "${GREEN}Modules tidied${NC}"

# Database management
db-migrate: ## Run database migrations (placeholder)
	@echo "${YELLOW}Running migrations...${NC}"
	@echo "${GREEN}Migrations complete${NC}"

db-seed: ## Seed database with sample data (placeholder)
	@echo "${YELLOW}Seeding database...${NC}"
	@echo "${GREEN}Database seeded${NC}"

# Release
release: clean build-all ## Create a release build
	@echo "${YELLOW}Creating release...${NC}"
	@mkdir -p release
	@cd $(BUILD_DIR) && tar -czf ../release/geoagent-$(VERSION)-linux-amd64.tar.gz geoagent-linux-amd64
	@cd $(BUILD_DIR) && tar -czf ../release/geoagent-$(VERSION)-darwin-amd64.tar.gz geoagent-darwin-amd64
	@cd $(BUILD_DIR) && tar -czf ../release/geoagent-$(VERSION)-darwin-arm64.tar.gz geoagent-darwin-arm64
	@cd $(BUILD_DIR) && zip ../release/geoagent-$(VERSION)-windows-amd64.zip geoagent-windows-amd64.exe
	@echo "${GREEN}Release created in release/${NC}"

.DEFAULT_GOAL := help
