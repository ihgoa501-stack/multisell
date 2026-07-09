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

# docgen — auto-generate module reference Markdown from Go source.
# Run after adding/modifying routes or models.
# Generated docs live in docs/auto/modules/ and are checked into git
# so CI can verify freshness with `make check-docs`.
docs:
	go run cmd/docgen/main.go

# check-docs — verify auto-generated docs are up to date with source.
# Returns non-zero if the generated output differs from what's committed.
check-docs:
	@tmpdir=$$(mktemp -d) && \
	go run cmd/docgen/main.go --out "$$tmpdir" && \
	diff -rq "$$tmpdir" docs/auto/modules/ 2>/dev/null || \
	(echo "ERROR: Auto-generated docs are stale. Run 'make docs' to regenerate." && exit 1)
	@echo "Auto-generated docs are up to date."
