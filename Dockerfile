# --- Stage 1: Build ---
FROM golang:1.24.3-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git

# Set working directory
WORKDIR /app

# Cache deps
COPY go.mod go.sum ./
RUN go mod download

# Copy the source
COPY . .

# Build the Go binary statically
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /auth ./cmd/server

# --- Stage 2: Production image ---
FROM scratch

# Copy the compiled Go binary
COPY --from=builder /auth /auth

# Expose port
EXPOSE 8080

# Set entrypoint
ENTRYPOINT ["/auth"]
