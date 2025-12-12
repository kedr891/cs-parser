ifneq ($(wildcard .env),)
include .env
export
else
$(warning WARNING: .env file not found! Using .env.example)
include .env.example
export
endif

.PHONY: help
help: ## Display this help screen
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z_-]+:.*?##/ { printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)

##@ Development

.PHONY: deps
deps: ## Install dependencies
	go mod download && go mod verify && go mod tidy
	@echo "✓ Dependencies installed"

.PHONY: format
format: ## Format code
	go fmt ./...
	@echo "✓ Code formatted"

.PHONY: lint
lint: ## Run linter
	golangci-lint run ./...
	@echo "✓ Linting complete"

.PHONY: test
test: ## Run tests
	go test -v -race -covermode=atomic -coverprofile=coverage.txt ./...
	@echo "✓ Tests passed"

.PHONY: test-coverage
test-coverage: test ## Run tests with coverage report
	go tool cover -html=coverage.txt -o coverage.html
	@echo "✓ Coverage report: coverage.html"

##@ Docker Compose

.PHONY: infra-up
infra-up: ## Start infrastructure (Postgres, Redis, Kafka)
	docker compose up -d postgres redis zookeeper kafka
	@echo "✓ Infrastructure started"

.PHONY: infra-down
infra-down: ## Stop infrastructure
	docker compose down
	@echo "✓ Infrastructure stopped"

.PHONY: infra-logs
infra-logs: ## View infrastructure logs
	docker compose logs -f postgres redis kafka

.PHONY: up
up: ## Start all services
	docker compose up -d
	@echo "✓ All services started"

.PHONY: down
down: ## Stop all services
	docker compose down --remove-orphans
	@echo "✓ All services stopped"

.PHONY: logs
logs: ## View all service logs
	docker compose logs -f

.PHONY: restart
restart: down up ## Restart all services

.PHONY: clean
clean: down ## Clean all volumes and data
	docker compose down -v
	rm -f coverage.txt coverage.html
	@echo "✓ Cleaned"

##@ Database

.PHONY: migrate-create
migrate-create: ## Create new migration (usage: make migrate-create name=create_users_table)
	@if [ -z "$(name)" ]; then \
		echo "Error: name parameter is required"; \
		echo "Usage: make migrate-create name=create_users_table"; \
		exit 1; \
	fi
	migrate create -ext sql -dir migrations -seq $(name)
	@echo "✓ Migration created: migrations/*_$(name).sql"

.PHONY: migrate-up
migrate-up: ## Run migrations up
	migrate -path migrations -database "$(PG_URL)" up
	@echo "✓ Migrations applied"

.PHONY: migrate-down
migrate-down: ## Rollback last migration
	migrate -path migrations -database "$(PG_URL)" down 1
	@echo "✓ Migration rolled back"

.PHONY: migrate-force
migrate-force: ## Force migration version (usage: make migrate-force version=1)
	@if [ -z "$(version)" ]; then \
		echo "Error: version parameter is required"; \
		echo "Usage: make migrate-force version=1"; \
		exit 1; \
	fi
	migrate -path migrations -database "$(PG_URL)" force $(version)
	@echo "✓ Migration version forced to $(version)"

.PHONY: db-reset
db-reset: ## Reset database (drop and recreate)
	docker compose exec postgres psql -U cs2_user -d postgres -c "DROP DATABASE IF EXISTS cs2_skins;"
	docker compose exec postgres psql -U cs2_user -d postgres -c "CREATE DATABASE cs2_skins;"
	@echo "✓ Database reset"

.PHONY: db-console
db-console: ## Connect to database console
	docker compose exec postgres psql -U cs2_user -d cs2_skins

##@ Services

.PHONY: run-parser
run-parser: ## Run parser service locally
	go run cmd/parser/main.go

.PHONY: run-price-consumer
run-price-consumer: ## Run price consumer service locally
	go run cmd/price-consumer/main.go

.PHONY: run-api
run-api: ## Run API service locally
	go run cmd/api/main.go

.PHONY: run-notification
run-notification: ## Run notification service locally
	go run cmd/notification/main.go

##@ Build

.PHONY: build-parser
build-parser: ## Build parser binary
	CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o bin/parser cmd/parser/main.go
	@echo "✓ Parser built: bin/parser"

.PHONY: build-price-consumer
build-price-consumer: ## Build price-consumer binary
	CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o bin/price-consumer cmd/price-consumer/main.go
	@echo "✓ Price consumer built: bin/price-consumer"

.PHONY: build-api
build-api: ## Build API binary
	CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o bin/api cmd/api/main.go
	@echo "✓ API built: bin/api"

.PHONY: build-notification
build-notification: ## Build notification binary
	CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o bin/notification cmd/notification/main.go
	@echo "✓ Notification built: bin/notification"

.PHONY: build-all
build-all: build-parser build-price-consumer build-api build-notification ## Build all services

##@ Kafka

.PHONY: kafka-topics
kafka-topics: ## List Kafka topics
	docker compose exec kafka kafka-topics --bootstrap-server localhost:9092 --list

.PHONY: kafka-create-topics
kafka-create-topics: ## Create Kafka topics
	docker compose exec kafka kafka-topics --bootstrap-server localhost:9092 --create --topic skin.price.updated --partitions 3 --replication-factor 1 --if-not-exists
	docker compose exec kafka kafka-topics --bootstrap-server localhost:9092 --create --topic skin.discovered --partitions 3 --replication-factor 1 --if-not-exists
	docker compose exec kafka kafka-topics --bootstrap-server localhost:9092 --create --topic notification.price_alert --partitions 3 --replication-factor 1 --if-not-exists
	@echo "✓ Kafka topics created"

.PHONY: kafka-consume-prices
kafka-consume-prices: ## Consume price update messages
	docker compose exec kafka kafka-console-consumer --bootstrap-server localhost:9092 --topic skin.price.updated --from-beginning

##@ Redis

.PHONY: redis-cli
redis-cli: ## Connect to Redis CLI
	docker compose exec redis redis-cli

.PHONY: redis-flush
redis-flush: ## Flush all Redis data
	docker compose exec redis redis-cli FLUSHALL
	@echo "✓ Redis flushed"

##@ Tools

.PHONY: install-tools
install-tools: ## Install development tools
	go install github.com/golang-migrate/migrate/v4/cmd/migrate@latest
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	@echo "✓ Tools installed"

.PHONY: mock
mock: ## Generate mocks
	go generate ./...
	@echo "✓ Mocks generated"

##@ Quick Start

.PHONY: quick-start
quick-start: infra-up migrate-up kafka-create-topics ## Quick start (infra + migrations + topics)
	@echo "✓ Quick start complete!"
	@echo ""
	@echo "Services ready:"
	@echo "  PostgreSQL: localhost:5432"
	@echo "  Redis:      localhost:6379"
	@echo "  Kafka:      localhost:29092"
	@echo ""
	@echo "Next steps:"
	@echo "  make run-api          # Start API server"
	@echo "  make run-parser       # Start parser"
	@echo "  make run-price-consumer   # Start consumer"