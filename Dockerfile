# Build stage
FROM golang:1.21-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git ca-certificates tzdata

# Set working directory
WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags='-w -s -extldflags "-static"' \
    -a -installsuffix cgo \
    -o dokku-mcp \
    src/interface/cmd/main.go

# Runtime stage
FROM alpine:latest

# Install runtime dependencies
RUN apk --no-cache add ca-certificates curl

# Create non-root user
RUN addgroup -g 1001 -S dokku && \
    adduser -S dokku -u 1001 -G dokku

# Set working directory
WORKDIR /app

# Copy binary from builder stage
COPY --from=builder /app/dokku-mcp .

# Copy configuration example
COPY --from=builder /app/config.yaml.example ./config.yaml.example

# Create necessary directories
RUN mkdir -p /var/log/dokku-mcp && \
    chown -R dokku:dokku /app /var/log/dokku-mcp

# Switch to non-root user
USER dokku

# Expose port
EXPOSE 8080

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD curl -f http://localhost:8080/health || exit 1

# Set default command
CMD ["./dokku-mcp"] 