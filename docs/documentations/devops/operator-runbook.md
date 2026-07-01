# Operator Runbook

## Deployment

### Docker

```bash
docker build -t maintainerd-auth .
docker run -d \
  --name auth \
  -p 8080:8080 -p 8081:8081 -p 8082:8082 \
  --env-file .env \
  maintainerd-auth
```

### Docker Compose

```yaml
version: "3.9"
services:
  auth:
    build: .
    ports:
      - "8080:8080"
      - "8081:8081"
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
