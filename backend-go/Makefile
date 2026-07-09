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

# docgen — auto-generate docs from Go source (routes, models, events, agents, frontend pages).
# Run after adding/modifying routes, models, agents, or event subscriptions.
# Generated docs live in docs/auto/ and are checked into git.
# CI verifies freshness with `make check-docs`.
docs:
	go run cmd/docgen/main.go

# check-docs — verify all auto-generated docs are up to date with source.
# Compares entire docs/auto/ output against a fresh generation.
# Returns non-zero if any file differs (blocks CI).
check-docs:
	@tmpdir=$$(mktemp -d) && \
	go run cmd/docgen/main.go --out "$$tmpdir" && \
	diff -rq "$$tmpdir" docs/auto/ 2>/dev/null || \
	(echo "ERROR: Auto-generated docs are stale. Run 'make docs' to regenerate." && exit 1)
	@echo "Auto-generated docs are up to date."
