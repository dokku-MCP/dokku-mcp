# Build stage
FROM golang:1.24-alpine AS builder

RUN apk add --no-cache git make ca-certificates tzdata
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN make build


# Runtime stage
FROM alpine:latest
RUN apk --no-cache add ca-certificates curl

# Create non-root user
RUN addgroup -g 1001 -S dokku && \
    adduser -S dokku -u 1001 -G dokku

WORKDIR /app
COPY --from=builder /app/build/dokku-mcp .
COPY --from=builder /app/config.yaml.example ./config.yaml

RUN mkdir -p /var/log/dokku-mcp && \
    chown -R dokku:dokku /app /var/log/dokku-mcp

USER dokku

EXPOSE 8080

HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD curl -f http://localhost:8080/health || exit 1

CMD ["./dokku-mcp"] 