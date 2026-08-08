# --- Stage 1: Build ---
FROM golang:1.26-alpine@sha256:0178a641fbb4858c5f1b48e34bdaabe0350a330a1b1149aabd498d0699ff5fb2 AS builder

RUN apk add --no-cache git ca-certificates

WORKDIR /app

ARG TARGETOS
ARG TARGETARCH
ARG VERSION=dev

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH go build \
    -trimpath -ldflags="-s -w -X github.com/maintainerd/maintainerd-auth/internal/platform/config.AppVersion=$VERSION" \
    -o /auth ./cmd/server

# --- Stage 2: Runtime ---
FROM alpine:3.24@sha256:28bd5fe8b56d1bd048e5babf5b10710ebe0bae67db86916198a6eec434943f8b

RUN apk add --no-cache ca-certificates && \
    adduser -D -u 65532 -g 65532 m9d

COPY --from=builder /auth /auth

EXPOSE 8080 8081

HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8080/readyz || exit 1

USER m9d

ENTRYPOINT ["/auth"]
