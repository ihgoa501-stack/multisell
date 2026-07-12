# ============================
#  Builder stage
# ============================
FROM golang:1.25.12-alpine AS builder

ENV GOPROXY=https://goproxy.cn,direct

RUN sed -i 's#https://dl-cdn.alpinelinux.org/alpine#https://mirrors.aliyun.com/alpine#g' /etc/apk/repositories \
    && apk add --no-cache git ca-certificates

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

RUN sed -i 's#https://dl-cdn.alpinelinux.org/alpine#https://mirrors.aliyun.com/alpine#g' /etc/apk/repositories \
    && apk add --no-cache \
    ca-certificates \
    wget

# Create a non-root user.
RUN addgroup -S appgroup && adduser -S appuser -G appgroup

WORKDIR /app
COPY --from=builder --chown=appuser:appgroup /app/server .
COPY --from=builder --chown=appuser:appgroup /build/configs ./configs
# Ship the synthetic carrier-rate YAML files so the logistics module can
# load shipping rates at startup. Override the path at runtime via
# CARRIER_RATES_DIR env var if mounting a different volume.
COPY --chown=appuser:appgroup carrier-rates ./carrier-rates

USER appuser
EXPOSE 8080

HEALTHCHECK --interval=30s --timeout=10s --start-period=10s --retries=3 \
  CMD wget -qO- http://localhost:8080/api/health >/dev/null || exit 1

ENTRYPOINT ["/app/server"]
