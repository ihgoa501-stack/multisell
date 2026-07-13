FROM golang:1.25-alpine AS build
WORKDIR /src
ARG GOPROXY=https://proxy.golang.org,direct
COPY go.mod go.sum ./
RUN GOPROXY="$GOPROXY" go mod download
COPY cmd ./cmd
COPY internal ./internal
RUN CGO_ENABLED=0 go build -trimpath -o /out/image-service ./cmd/server && \
    CGO_ENABLED=0 go build -trimpath -o /out/image-service-migrate ./cmd/migrate

FROM alpine:3.22
RUN sed -i 's#https://dl-cdn.alpinelinux.org/alpine#https://mirrors.aliyun.com/alpine#g' /etc/apk/repositories && \
    apk add --no-cache ca-certificates wget && \
    addgroup -S app && adduser -S -G app app
WORKDIR /app
COPY --from=build /out/image-service /usr/local/bin/image-service
COPY --from=build /out/image-service-migrate /usr/local/bin/image-service-migrate
RUN mkdir /data && chown app:app /data
USER app
EXPOSE 8092
ENTRYPOINT ["image-service"]
