# Build stage
FROM golang:1.21-alpine AS builder

# Install git and ca-certificates (needed for downloading dependencies)
RUN apk --no-cache add git ca-certificates tzdata

# Set working directory
WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the binary
RUN CGO_ENABLED=0 GOOS=linux go build \
    -v -trimpath \
    -ldflags="-s -w -X main.version=$(date -u +%Y%m%d-%H%M%S)" \
    -o singbox_sub \
    ./src/github.com/sixproxy/sub.go

# Final stage
FROM alpine:latest

# Install ca-certificates for HTTPS requests
RUN apk --no-cache add ca-certificates tzdata

# Create non-root user
RUN addgroup -g 1001 -S appgroup && \
    adduser -u 1001 -S appuser -G appgroup

# Set working directory
WORKDIR /app

# Copy binary from builder stage
COPY --from=builder /app/singbox_sub /app/singbox_sub

# Copy config template if exists
COPY --from=builder /app/src/github.com/sixproxy/config/ /app/config/

# Change ownership to non-root user
RUN chown -R appuser:appgroup /app

# Switch to non-root user
USER appuser

# Expose port if needed (adjust based on your needs)
# EXPOSE 8080

# Set entrypoint
ENTRYPOINT ["/app/singbox_sub"]

# Add healthcheck (optional)
# HEALTHCHECK --interval=30s --timeout=5s --start-period=5s --retries=3 \
#   CMD ["/app/singbox_sub", "--health-check"] || exit 1

# Labels for better maintainability
LABEL maintainer="your-email@example.com"
LABEL description="Sing-box subscription configuration generator"
LABEL version="latest"