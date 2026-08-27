# =============================================================================
# maintainerd-auth — all-in-one production image (single binary)
#
# Bundles the whole AUTH product in one image: the Go backend with the two SPAs
# (admin console + hosted identity) COMPILED INTO the binary via go:embed. The
# Go process serves each SPA on its own port with its API mounted same-origin
# in-process — no nginx, no process supervisor. One pull, one process.
#
#   :8080  backend control plane (management API)
#   :8081  backend data plane    (OAuth2/OIDC, public API)
#   :3000  admin console SPA      (control plane at /api, data plane at /public-api)
#   :3001  hosted identity SPA    (data plane at /api + /.well-known)
#
# Databases (Postgres/Redis/RabbitMQ) are NOT in this image — provide them via
# your platform. Local development does NOT use this image: it runs the three
# apps in hot-reload mode via the maintainerd-dev repo.
# =============================================================================

# --- Stage 1: build the admin console SPA ---
# SPA output is architecture-independent, so build on BUILDPLATFORM (never under
# QEMU emulation for arm64 — npm/vite under emulation is slow and OOM-prone).
FROM --platform=$BUILDPLATFORM node:22-alpine@sha256:c610fcdfb1d5b4740dd70c284ed3cb16bb857e0f7166196e36a5501df7a3aa32 AS console
WORKDIR /app
COPY web/console/package*.json ./
RUN npm ci
COPY web/console/ ./
RUN npm run build

# --- Stage 2: build the hosted identity SPA ---
FROM --platform=$BUILDPLATFORM node:22-alpine@sha256:c610fcdfb1d5b4740dd70c284ed3cb16bb857e0f7166196e36a5501df7a3aa32 AS identity
WORKDIR /app
COPY web/identity/package*.json ./
RUN npm ci
COPY web/identity/ ./
RUN npm run build

# --- Stage 3: build the Go backend with the SPAs embedded ---
# Build on the native BUILDPLATFORM and cross-compile via GOOS/GOARCH so multi-arch
# builds never run this stage under QEMU emulation. The `embedassets` build tag
# turns on the go:embed of the two dist trees copied in below.
FROM --platform=$BUILDPLATFORM golang:1.27-alpine@sha256:4c9fe60190a2a3350ddc51de80d0224b8a6698d12bdfc999fee45ea9d6c46dbc AS backend
ENV GOTOOLCHAIN=auto

RUN apk add --no-cache git ca-certificates
WORKDIR /app
ARG TARGETOS
ARG TARGETARCH
ARG VERSION=dev

COPY go.mod go.sum ./
RUN go mod download
COPY . .

# Bake each SPA's built output into the embed dirs the `embedassets` tag reads.
COPY --from=console  /app/dist  ./internal/webui/console
COPY --from=identity /app/dist  ./internal/webui/identity

RUN CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH go build \
    -tags embedassets \
    -trimpath -ldflags="-s -w -X github.com/maintainerd/maintainerd-auth/internal/platform/config.AppVersion=$VERSION" \
    -o /auth ./cmd/server

# --- Stage 4: runtime (just the single binary) ---
FROM alpine:3.24@sha256:28bd5fe8b56d1bd048e5babf5b10710ebe0bae67db86916198a6eec434943f8b

ARG VERSION=dev
LABEL org.opencontainers.image.title="maintainerd-auth" \
      org.opencontainers.image.description="All-in-one maintainerd auth stack (backend + console + identity), single binary" \
      org.opencontainers.image.version="$VERSION" \
      org.opencontainers.image.source="https://github.com/maintainerd/maintainerd-auth"

RUN apk add --no-cache ca-certificates curl tini \
    && addgroup -g 65532 m9d \
    && adduser -D -u 65532 -G m9d m9d

COPY --from=backend /auth /usr/local/bin/auth

# 3000 (console) + 3001 (identity) are the browser-facing ports to publish.
# 8081 (public data plane) is published where the OIDC issuer must resolve.
# 8080 (control plane) and 8082 (management: /metrics + health) should stay
# INTERNAL — firewall them; the console reaches the control plane same-origin
# through the console server, so neither needs public exposure.
EXPOSE 8080 8081 8082 3000 3001

# Generous start-period: the backend runs schema migrations in-process at boot.
HEALTHCHECK --interval=30s --timeout=5s --start-period=60s --retries=5 \
    CMD curl -fsS http://localhost:8080/readyz >/dev/null \
     && curl -fsS http://localhost:8081/readyz >/dev/null \
     && curl -fsS http://localhost:3000/ >/dev/null \
     && curl -fsS http://localhost:3001/ >/dev/null || exit 1

USER m9d

# tini is PID 1: reaps zombies and forwards SIGTERM to the backend, which drains
# all four HTTP servers gracefully (30s) before exiting.
ENTRYPOINT ["/sbin/tini", "--", "/usr/local/bin/auth"]
