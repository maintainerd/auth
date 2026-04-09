# --- Stage 1: Build ---
FROM golang:1.26-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git

# Set working directory
WORKDIR /app

# Cache deps
COPY go.mod ./
RUN go mod download

# Copy the source
COPY . .

# Build the Go binary statically
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /auth ./cmd/server

# --- Stage 2: Production image ---
FROM alpine:3.21

# Install wget for health checks (tiny footprint)
RUN apk add --no-cache wget

# Copy the compiled Go binary
COPY --from=builder /auth /auth

# Health check against the internal readiness endpoint
HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3 \
  CMD wget --no-verbose --tries=1 --spider http://localhost:8080/ready || exit 1

# Expose port
EXPOSE 8080

# Set entrypoint
ENTRYPOINT ["/auth"]
