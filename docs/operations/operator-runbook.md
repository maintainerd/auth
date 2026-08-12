# Operator Runbook

> **Before deploying behind a reverse proxy / ingress, read
> [Production Edge & Security Hardening](edge-and-security-hardening.md).** It
> covers the two edge settings an auth service must get right — proxy header
> buffers (or login 502s on the `Set-Cookie` JWTs) and trusted forwarded headers
> — plus non-root secret/cert readability. Each is easy to miss and each breaks
> production on its own.

## Deployment

The all-in-one image serves four HTTP ports — `:8080` control API, `:8081` public
data API (OAuth2/OIDC), `:3000` console SPA, `:3001` identity SPA — plus `:8082`
metrics and `:50051` gRPC. It serves plain HTTP; terminate TLS at the edge, and
expose only the **public** ports (`:8081`, `:3001`) to the internet while keeping
the **private** ports (`:8080`, `:3000`, `:8082`) internal/VPN-only.

### Docker

```bash
docker build -t maintainerd-auth .
docker run -d \
  --name auth \
  -p 8080:8080 -p 8081:8081 -p 3000:3000 -p 3001:3001 -p 8082:8082 \
  --env-file .env \
  maintainerd-auth
```

The image runs as non-root (uid 65532); any bind-mounted secret/cert must be
readable by that uid — prefer `--env-file` / a secret manager over bind mounts.

### Docker Compose

```yaml
version: "3.9"
services:
  auth:
    build: .
    ports:
      - "8080:8080"
      - "8081:8081"
      - "3000:3000"
      - "3001:3001"
      - "8082:8082"
    env_file: .env
    depends_on:
      postgres:
        condition: service_healthy
      redis:
        condition: service_healthy
  postgres:
    image: postgres:16-alpine
    environment:
      POSTGRES_USER: maintainerd
      POSTGRES_PASSWORD: changeme
      POSTGRES_DB: maintainerd
    volumes:
      - pgdata:/var/lib/postgresql/data
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U maintainerd"]
      interval: 5s
  redis:
    image: redis:7-alpine
    healthcheck:
      test: ["CMD", "redis-cli", "ping"]
      interval: 5s
volumes:
  pgdata:
```

### Kubernetes

Key considerations:
- Use Secrets for `JWT_PRIVATE_KEY`, `JWT_PUBLIC_KEY`, `APP_ENCRYPTION_KEY`, `HMAC_SECRET_KEY`
- Set `SECRET_PROVIDER=env` for env-var-based secrets (or use external secret manager)
- Enable `DB_SSLMODE=require` and `REDIS_TLS=true` in production
- Use readiness probes: `GET /readyz` (checks DB + Redis + JWKS)
- Use liveness probes: `GET /livez` (process-level only)
- Set resource limits: recommend 256Mi memory, 0.5 CPU minimum
- **Run non-root with a hardened `securityContext`** (`runAsNonRoot: true`,
  `runAsUser`/`runAsGroup`/`fsGroup: 65532`, `readOnlyRootFilesystem: true` with a
  writable `/tmp` emptyDir, drop `ALL` capabilities, `seccompProfile:
  RuntimeDefault`). The image is already uid 65532.
- **Mounted Secret/ConfigMap volumes need `defaultMode: 0440`** so the non-root
  uid can read them — `fsGroup` alone does not (they are read-only projected
  volumes). Applies to any mounted gRPC/TLS certs or key files.
- **Ingress must set `nginx.ingress.kubernetes.io/proxy-buffer-size: "16k"`** (or
  raise your edge's response-header buffer) — login/refresh set the JWTs as
  `Set-Cookie` headers that overflow the default buffer and 502 otherwise.
- **Trust forwarded headers from the edge only** — set `TRUSTED_PROXY_CIDRS` to
  the ingress/LB range; never `TRUST_ALL_PROXIES=true` in production.

See **[Production Edge & Security Hardening](edge-and-security-hardening.md)** for
the full securityContext, Secret `defaultMode`, ingress annotations, and a sample
hardened edge config.

## Key Rotation

JWT signing keys auto-rotate based on `JWT_KEY_ROTATION_INTERVAL` (default: 30 days). To force rotation:

```bash
# Restart the key rotation runner (sends SIGUSR1 to trigger immediate rotation)
docker exec auth kill -SIGUSR1 1
```

The service maintains two key pairs during rotation:
- Active key (used for signing new tokens)
- Retiring key (used for verifying old tokens)

Both keys are served in `/.well-known/jwks.json`. Old tokens signed with the retiring key remain valid until expiry.

## Scaling

The service is stateless — all state is in Postgres and Redis. Horizontal scaling:

```bash
docker compose up -d --scale auth=3
```

Or in Kubernetes: `kubectl scale deployment auth --replicas=3`

### Connection pooling with PgBouncer

For production deployments with multiple app instances, front Postgres with
PgBouncer in transaction mode to reduce connection overhead. Size per-instance
`DB_MAX_OPEN_CONNS` as:

```
DB_MAX_OPEN_CONNS = (max_connections − reserved) / instance_count
```

- `max_connections`: your Postgres server's `max_connections` setting
- `reserved`: connections for maintenance, monitoring, and admin (typically 5–10)
- `instance_count`: number of auth service replicas

Example for 3 replicas with Postgres `max_connections = 100` and `reserved = 10`:

```
DB_MAX_OPEN_CONNS = (100 − 10) / 3 = 30
```

### Session stickiness

Cookie-based sessions (`__Host-access_token`) don't require stickiness — the JWT is self-contained. If using reference tokens, all instances share the same Redis denylist.

## Recovery

### Database failure

1. Restore Postgres from backup
2. Restart auth service — it will re-establish connections automatically

### Redis failure

1. Restart Redis
2. Auth service will reconnect on next request
3. Rate-limit counters and JTI denylist will be cold-started (acceptable degradation)

### Lost JWT keys

If JWT signing keys are lost:
1. Generate new keys: `./scripts/generate-jwt-keys.sh`
2. Set `JWT_PRIVATE_KEY` and `JWT_PUBLIC_KEY` environment variables
3. Restart the service
4. All existing tokens become invalid — users must re-authenticate

### Lost encryption key

If `APP_ENCRYPTION_KEY` is lost:
1. Encrypted data at rest (TOTP secrets, WebAuthn credentials, provider configs) is **unrecoverable**
2. Generate a new key and restart
3. Users will need to re-enroll MFA and providers will need reconfiguration

### First-run setup

If the database is fresh:
1. `GET /setup/status` — check setup step
2. `POST /setup/create_tenant` — create initial tenant (one-time)
3. `POST /setup/create_admin` — create super-admin user
4. `POST /setup/create_profile` — create admin profile

## Monitoring

### Metrics

Prometheus endpoint: `GET /metrics` on the dedicated management listener, port 8082 by default (`MANAGEMENT_PORT`).

Key metrics:
- `build_info` — version, commit, build date
- `http_server_request_duration_seconds` — request latency
- `http_server_requests_total` — request count by method/status

### Health endpoints

| Endpoint | Purpose | Port |
|----------|---------|------|
| `/healthz` | Liveness — process running | 8080, 8081 |
| `/livez` | Liveness — process + version | 8080, 8081 |
| `/readyz` | Readiness — DB + Redis + JWKS | 8080, 8081 |

### Logs

JSON structured logs via stdout. Key fields: `level`, `msg`, `trace_id`, `span_id`, `request_id`.

```bash
# Filter errors
docker logs auth 2>&1 | jq 'select(.level == "ERROR")'
```

## Backups

Required backup targets:
1. **Postgres database** — full pg_dump daily
2. **Redis** — optional, only if using Redis for persistent state beyond cache

Recommended RPO: 24 hours (daily backup). RTO: 1 hour (restore + verify).

## Upgrading

Maintainerd Auth follows semantic versioning; images are published to Docker Hub as `:<version>` and `:latest`.

1. **Read the release notes** in [`CHANGELOG.md`](../../CHANGELOG.md) for the target version and note any `BREAKING` entries.
2. **Back up the database** (see Backups above) before any upgrade.
3. **Pull the new image** by its immutable version tag (never rely on `:latest` in production), e.g. `docker pull <org>/maintainerd-auth:<version>`.
4. **Migrations run automatically at startup** (ordered registry in `internal/platform/runner/migration.go`, guarded by a Postgres advisory lock and the `schema_migrations` table, so concurrent replicas are safe). Migrations are create-only and forward-compatible within a minor series. For a large deployment, run one instance first to apply migrations, then roll the rest.
5. **Roll instances** one at a time behind the load balancer; readiness is gated by `/readyz`, and graceful shutdown drains in-flight requests, so rolling restarts drop no traffic.
6. **Verify** `/livez`, `/readyz`, and the reported version (`APP_VERSION` or the build-injected value) after the roll.
7. **Rollback:** redeploy the previous image tag. Because migrations are create-only and additive, the prior binary continues to run against the newer schema; restore from backup only if a release explicitly documents a non-additive migration.
