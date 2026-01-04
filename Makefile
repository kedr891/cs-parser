.PHONY: generate
generate:
	buf generate

.PHONY: lint
lint:
	buf lint

.PHONY: build-api
build-api:
	go build -o bin/api.exe ./cmd/api

.PHONY: build-parser
build-parser:
	go build -o bin/parser.exe ./cmd/parser

.PHONY: run-api
run-api:
	configPath=config.yaml swaggerPath=./internal/pb/swagger/skins_api/skins.swagger.json go run ./cmd/api

.PHONY: run-parser
run-parser:
	configPath=config.yaml go run ./cmd/parser

.PHONY: docker-build
docker-build:
	docker-compose build

.PHONY: docker-up
docker-up:
	docker-compose --profile sharding up -d

.PHONY: docker-down
docker-down:
	docker-compose --profile sharding down

.PHONY: docker-clean
docker-clean:
	docker-compose --profile sharding down -v
	docker system prune -f

.PHONY: docker-logs
docker-logs:
	docker-compose --profile sharding logs -f api

.PHONY: test
test:
	go test -v ./...

.PHONY: clean
clean:
	rm -rf bin/
	rm -rf internal/pb/

.PHONY: migrate
migrate:
	./scripts/migrate.sh

.PHONY: tidy
tidy:
	go mod tidy