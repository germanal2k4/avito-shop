.PHONY: generate build run test migrate all db-up db-down

# Используем Docker для конвертации Swagger 2.0 в OpenAPI 3.0
SWAGGER2OPENAPI_CMD := docker run --rm -v "$(PWD)":/workspace node:16-alpine npx swagger2openapi -p -o /workspace/openapi3.json /workspace/openapi.json

# Используем локальный образ oapi-codegen (обязательно предварительно собрать образ)
OAPI_CODEGEN_CMD := docker run --rm -v "$(PWD)":/local local/oapi-codegen -generate types,server -o /local/internal/api/openapi.gen.go -package api /local/openapi3.json

generate:
	@echo "Converting Swagger 2.0 to OpenAPI 3.0..."
	@$(SWAGGER2OPENAPI_CMD)
	@echo "Generating OpenAPI code..."
	@$(OAPI_CODEGEN_CMD)

build: generate
	@echo "Building project..."
	@go build -o build ./cmd

migrate:
	@echo "Running DB migrations..."
	@goose -dir=migrations postgres "postgres://postgres:password@127.0.0.1:5432/shop?sslmode=disable" up
reset-db:
	@echo "Resetting database schema..."
	@docker exec -it postgres psql -U postgres -d shop -c "DROP SCHEMA public CASCADE; CREATE SCHEMA public;"
db-status:
	@echo "Checking migration status..."
	DATABASE_URL="postgres://postgres:password@localhost:5432/shop?sslmode=disable" \
	goose -dir=migrations postgres "$$DATABASE_URL" status

run: build
	@echo "Running project..."
	@echo $DATABASE_URL
	@./build

test:
	@echo "Running tests..."
	@go test ./...

all: generate migrate build run

db-up:
	docker run --rm --name postgres \
	  -e POSTGRES_USER=postgres \
	  -e POSTGRES_PASSWORD=password \
	  -e POSTGRES_DB=shop \
	  -p 5432:5432 \
	  -d postgres:13

db-down:
	docker stop postgres