# =============================================================================
# maintainerd-auth — all-in-one production image
#
# Bundles the whole AUTH product in one image: the Go backend plus the two SPAs
# (admin console + hosted identity), each served on its own port by an in-image
# nginx that proxies same-origin API calls to the backend. One pull runs the
# entire auth stack.
#
#   :8080  backend control plane (management API)
#   :8081  backend data plane    (OAuth2/OIDC, public API)
#   :3000  admin console SPA
#   :3001  hosted identity SPA
#
# Databases (Postgres/Redis/RabbitMQ) are NOT in this image — provide them via
# docker-compose (see docker-compose.yml) or your platform.
# =============================================================================

# --- Stage 1: build the Go backend ---
# Build on the native BUILDPLATFORM and cross-compile via GOOS/GOARCH so multi-arch
# builds never run this stage under QEMU emulation.
FROM --platform=$BUILDPLATFORM golang:1.26-alpine@sha256:0178a641fbb4858c5f1b48e34bdaabe0350a330a1b1149aabd498d0699ff5fb2 AS backend

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

# --- Stage 2: build the admin console SPA ---
# SPA output is architecture-independent, so build on BUILDPLATFORM (never under
# QEMU emulation for arm64 — npm/vite under emulation is slow and OOM-prone).
FROM --platform=$BUILDPLATFORM node:22-alpine AS console
WORKDIR /app
COPY web/console/package*.json ./
RUN npm ci
COPY web/console/ ./
RUN npm run build

# --- Stage 3: build the hosted identity SPA ---
FROM --platform=$BUILDPLATFORM node:22-alpine AS identity
WORKDIR /app
COPY web/identity/package*.json ./
RUN npm ci
COPY web/identity/ ./
RUN npm run build

# --- Stage 4: runtime (backend + nginx + both SPAs under supervisord) ---
FROM alpine:3.24@sha256:28bd5fe8b56d1bd048e5babf5b10710ebe0bae67db86916198a6eec434943f8b

ARG VERSION=dev
LABEL org.opencontainers.image.title="maintainerd-auth" \
      org.opencontainers.image.description="All-in-one maintainerd auth stack (backend + console + identity)" \
      org.opencontainers.image.version="$VERSION" \
      org.opencontainers.image.source="https://github.com/maintainerd/maintainerd-auth"

RUN apk add --no-cache nginx supervisor ca-certificates curl tini \
    && addgroup -g 65532 m9d \
    && adduser -D -u 65532 -G m9d m9d \
    && mkdir -p /srv/console /srv/identity /run/nginx /tmp/nginx \
    && chown -R m9d:m9d /srv /run/nginx /tmp/nginx /var/lib/nginx /var/log/nginx

# Backend binary + built SPAs.
COPY --from=backend  /auth       /usr/local/bin/auth
COPY --from=console  /app/dist   /srv/console
COPY --from=identity /app/dist   /srv/identity

# Runtime config (nginx + supervisor + entrypoint).
COPY deploy/nginx.conf       /etc/nginx/nginx.conf
COPY deploy/supervisord.conf /etc/supervisord.conf
COPY deploy/entrypoint.sh    /entrypoint.sh
RUN chmod +x /entrypoint.sh && chown -R m9d:m9d /srv/console /srv/identity

# 3000 (console) + 3001 (identity) are the browser-facing ports to publish.
# 8081 (public data plane) is published where the OIDC issuer must resolve.
# 8080 (control/management plane) should stay INTERNAL — firewall it; the console
# reaches it same-origin through nginx, so it never needs public exposure.
EXPOSE 8080 8081 3000 3001

# Generous start-period: the backend runs schema migrations in-process at boot.
HEALTHCHECK --interval=30s --timeout=5s --start-period=60s --retries=5 \
    CMD curl -fsS http://localhost:8080/readyz >/dev/null \
     && curl -fsS http://localhost:8081/readyz >/dev/null \
     && curl -fsS http://localhost:3000/ >/dev/null \
     && curl -fsS http://localhost:3001/ >/dev/null || exit 1

USER m9d

# tini is PID 1 (reaps zombies, forwards signals); entrypoint injects runtime
# config then execs supervisord, which runs the backend and nginx.
ENTRYPOINT ["/sbin/tini", "--", "/entrypoint.sh"]
CMD ["supervisord", "-c", "/etc/supervisord.conf"]
