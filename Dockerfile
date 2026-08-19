# syntax=docker/dockerfile:1
FROM golang:1.26-alpine AS builder

WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/snipvault ./cmd/api

FROM alpine:3.24
RUN apk add --no-cache ca-certificates && \
    addgroup -S snipvault && adduser -S -G snipvault snipvault
WORKDIR /app
COPY --from=builder /out/snipvault /app/snipvault
USER snipvault
EXPOSE 8090
HEALTHCHECK --interval=15s --timeout=3s --start-period=10s --retries=3 \
  CMD wget -qO- http://127.0.0.1:8090/health >/dev/null || exit 1
ENTRYPOINT ["/app/snipvault"]
