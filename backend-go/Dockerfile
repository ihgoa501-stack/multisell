# ============================
#  Builder stage
# ============================
FROM golang:1.25-alpine AS builder

RUN apk add --no-cache git ca-certificates

WORKDIR /build

# Dependency caching layer: invalidated only when go.mod/go.sum change.
COPY go.mod go.sum ./
RUN go mod download

# Copy source and build a fully static binary.
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build \
    -trimpath \
    -ldflags="-s -w" \
    -o /app/server \
    ./cmd/server/main.go

# ============================
#  Runtime stage
# ============================
FROM alpine:3.21 AS runtime

RUN apk add --no-cache \
    ca-certificates \
    wget

# Create a non-root user.
RUN addgroup -S appgroup && adduser -S appuser -G appgroup

WORKDIR /app
COPY --from=builder --chown=appuser:appgroup /app/server .

USER appuser
EXPOSE 8080

HEALTHCHECK --interval=30s --timeout=10s --start-period=10s --retries=3 \
  CMD wget -qO- http://localhost:8080/api/health >/dev/null || exit 1

ENTRYPOINT ["/app/server"]
