# Build stage
FROM golang:1.25-alpine AS builder

RUN apk update && apk upgrade && \
    apk add --no-cache git make ca-certificates tzdata
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN make build


# Runtime stage
FROM alpine:latest
RUN apk update && apk upgrade && \
    apk --no-cache add ca-certificates curl openssh-client

# Create non-root user
RUN addgroup -g 1001 -S dokku && \
    adduser -S dokku -u 1001 -G dokku

WORKDIR /app
COPY --from=builder /app/build/dokku-mcp .

ENV DOKKU_MCP_TRANSPORT_TYPE=sse \
    DOKKU_MCP_TRANSPORT_HOST=0.0.0.0 \
    DOKKU_MCP_TRANSPORT_PORT=8080 \
    DOKKU_MCP_LOG_LEVEL=info \
    DOKKU_MCP_EXPOSE_SERVER_LOGS=false

RUN mkdir -p /var/log/dokku-mcp && \
    chown -R dokku:dokku /app /var/log/dokku-mcp

USER dokku

EXPOSE 8080

# Healthcheck (non-fatal) - returns success if TCP port is accepting connections
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD sh -c 'curl -fsS http://localhost:8080/ || nc -z localhost 8080 || exit 1'

CMD ["./dokku-mcp"] 