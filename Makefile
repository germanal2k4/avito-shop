.PHONY: generate build run test migrate

generate:
	@echo "Generating OpenAPI code..."
	@oapi-codegen -generate types,server -o internal/api/openapi.gen.go -package api openapi.json

build: generate
	@echo "Building project..."
	@go build -o build ./cmd

run: build
	@echo "Running project..."
	@./build

test:
	@go test ./...

migrate:
	@goose -dir=migrations postgres "$(DATABASE_URL)" up
