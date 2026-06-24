.PHONY: run build test migrate-up migrate-down lint

run:
	go run cmd/server/main.go

build:
	go build -o bin/server cmd/server/main.go

test:
	go test ./... -v

migrate-up:
	migrate -path migrations -database "postgresql://$(DB_URL)" up

migrate-down:
	migrate -path migrations -database "postgresql://$(DB_URL)" down 1

lint:
	golangci-lint run ./...
