.PHONY: help build run test clean migrate-up migrate-down migrate-create docker-up docker-down deps

# Variables
BINARY_NAME=auth-service
MIGRATIONS_DIR=./migrations
DATABASE_URL=postgres://auth_service:auth_service_password@localhost:5432/auth_service_db?sslmode=disable

help: ## Show this help message
	@echo 'Usage: make [target]'
	@echo ''
	@echo 'Available targets:'
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  %-15s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

deps: ## Install dependencies
	go mod download
	go mod tidy

build: ## Build the application
	go build -o bin/$(BINARY_NAME) ./cmd/server

run: ## Run the application
	go run ./cmd/server

test: ## Run tests
	go test -v -race -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html

clean: ## Clean build artifacts
	rm -rf bin/
	rm -f coverage.out coverage.html

docker-up: ## Start Docker containers (PostgreSQL and Redis)
	docker-compose up -d

docker-down: ## Stop Docker containers
	docker-compose down

docker-logs: ## Show Docker container logs
	docker-compose logs -f

migrate-install: ## Install migrate tool
	@which migrate > /dev/null || (echo "Installing migrate tool..." && go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest)

migrate-up: migrate-install ## Run database migrations
	migrate -path $(MIGRATIONS_DIR) -database "$(DATABASE_URL)" up

migrate-down: migrate-install ## Rollback database migrations
	migrate -path $(MIGRATIONS_DIR) -database "$(DATABASE_URL)" down

migrate-create: ## Create a new migration (usage: make migrate-create NAME=create_table)
	@if [ -z "$(NAME)" ]; then echo "Usage: make migrate-create NAME=create_table"; exit 1; fi
	migrate create -ext sql -dir $(MIGRATIONS_DIR) -seq $(NAME)

migrate-force: migrate-install ## Force migration version (usage: make migrate-force VERSION=1)
	@if [ -z "$(VERSION)" ]; then echo "Usage: make migrate-force VERSION=1"; exit 1; fi
	migrate -path $(MIGRATIONS_DIR) -database "$(DATABASE_URL)" force $(VERSION)

lint: ## Run linter
	golangci-lint run

fmt: ## Format code
	go fmt ./...

vet: ## Run go vet
	go vet ./...

.DEFAULT_GOAL := help

