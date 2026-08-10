# Environment Variables

This is the **single source of truth** for every environment variable **maintainerd-auth**
reads. It is generated from the actual `config.Init()` and the `os.Getenv` / secret-provider
call sites in the code, so it stays in step with the binary.

> **Security notice** — never commit a `.env` with real credentials. `.env`, `.env.local`
> and the per-app SPA `.env` files are gitignored and must stay that way. Secrets
> (`APP_ENCRYPTION_KEY`, `HMAC_SECRET_KEY`, `JWT_*`, `DB_PASSWORD`, `SETUP_BOOTSTRAP_TOKEN`)
> should be delivered through a secret manager in production — see **Secret management** below.

Legend: **Required** = startup fails if unset. **Default** = value used when unset.
A blank Default means "unset ⇒ feature off / host-only".

---

## Required

The process refuses to start (with a clear error) if any of these is missing.

| Variable | Description |
|---|---|
| `APP_PUBLIC_HOSTNAME` | Public data-plane base URL — this is the **OIDC issuer** (`iss`) and must match the `/.well-known/openid-configuration` value. e.g. `https://identity-api.auth.example.com` |
| `APP_PRIVATE_HOSTNAME` | Private control-plane base URL (management API). e.g. `https://console-api.auth.example.com` |
| `APP_FRONTEND_IDENTITY_HOSTNAME` | Hosted identity (login) SPA host. e.g. `https://identity.auth.example.com` |
| `APP_FRONTEND_CONSOLE_HOSTNAME` | Admin console SPA host. e.g. `https://console.auth.example.com` |
| `DB_HOST` / `DB_PORT` / `DB_USER` / `DB_NAME` | PostgreSQL connection. |
| `DB_PASSWORD` | PostgreSQL password (loaded via the secret provider). |
| `JWT_PRIVATE_KEY` / `JWT_PUBLIC_KEY` | RS256 token-signing keypair, PEM (loaded via the secret provider). |
| `APP_ENCRYPTION_KEY` | AES-256 key for encryption-at-rest — **must be exactly 32 bytes** (loaded via the secret provider). |
| `HMAC_SECRET_KEY` | HMAC key for signed URLs (loaded via the secret provider). |

## Application & networking

| Variable | Default | Description |
|---|---|---|
| `APP_ENV` | `development` | `development` or `production`. In `production`, `DB_SSLMODE=disable` is rejected at startup. |
| `APP_VERSION` | build-injected → `dev` | Overrides the version baked in via `-ldflags`. Not usually set by hand. |
| `MANAGEMENT_PORT` | `8082` | Management listener (health + `/metrics`). Keep INTERNAL. |
| `APP_CONSOLE_PORT` | `3000` | Console SPA server port. |
| `APP_IDENTITY_PORT` | `3001` | Identity SPA server port. |
| `CORS_ALLOWED_ORIGINS` | (empty) | Comma-separated extra allowed origins. Tenant/client origins are always allowed; wildcard is never combined with credentials. |
| `TRUST_ALL_PROXIES` | (unset) | `true` trusts `X-Forwarded-For` from any hop. Prefer `TRUSTED_PROXY_CIDRS`. |
| `TRUSTED_PROXY_CIDRS` | (unset) | Comma-separated CIDRs whose `X-Forwarded-For` is trusted for client-IP resolution. |
| `WEBAUTHN_RP_ID` | (empty) | WebAuthn Relying Party ID (registrable domain). Required only if you enable WebAuthn/passkeys. |
| `WEBAUTHN_EXTRA_ORIGINS` | (empty) | Additional allowed WebAuthn origins (comma-separated). |
| `LOG_LEVEL` | `info` | `debug` / `info` / `warn` / `error`. |

## Cookies

| Variable | Default | Description |
|---|---|---|
| `COOKIE_SECURE` | `true` | Set `false` only for local http dev. |
| `COOKIE_SAMESITE` | `lax` | `lax` / `strict` / `none`. **Use `lax`** — as an IdP the session cookie must ride the cross-site redirects back from upstream providers; `strict` breaks federated SSO. |
| `COOKIE_DOMAIN` | (empty = host-only) | Set to a parent domain you fully control to share one session across sibling first-party surfaces. Setting it trades the `__Host-` prefix for `__Secure-`. |

## Database pool

| Variable | Default | Description |
|---|---|---|
| `DB_SSLMODE` | `disable` | Postgres sslmode. **Rejected when `APP_ENV=production`** — set `require`/`verify-full` in prod. |
| `DB_MAX_OPEN_CONNS` | `25` | Max open connections. |
| `DB_MAX_IDLE_CONNS` | `10` | Max idle connections. |
| `DB_CONN_MAX_LIFETIME_SEC` | `300` | Connection max lifetime (seconds). |
| `DB_STATEMENT_TIMEOUT_MS` | `30000` | Per-statement timeout (ms). |

## Redis (sessions, rate-limit, replay stores)

| Variable | Default | Description |
|---|---|---|
| `REDIS_ADDR` | `redis-db:6379` | Redis `host:port`. |
| `REDIS_PASSWORD` | (via secret provider) | Redis password; optional. |
| `REDIS_TLS` | `false` | `true` to dial Redis over TLS. |

## Event bus (optional)

| Variable | Default | Description |
|---|---|---|
| `RABBITMQ_URL` | (unset ⇒ AMQP disabled) | AMQP connection URL for outbound integration events. |

## CAPTCHA (optional)

| Variable | Default | Description |
|---|---|---|
| `CAPTCHA_SECRET` | (unset ⇒ CAPTCHA off) | Provider secret (reCAPTCHA/hCaptcha/Turnstile). Secret-bearing. |
| `CAPTCHA_VERIFY_URL` | `https://www.google.com/recaptcha/api/siteverify` | Verification endpoint. |
| `CAPTCHA_MIN_SCORE` | `0.5` | Minimum score (reCAPTCHA v3). |

## Secret management

Credentials are read through the configured provider, so the same binary works whether secrets
live in env vars or a vault. With the default `env` provider these are plain environment variables.

| Variable | Default | Description |
|---|---|---|
| `SECRET_PROVIDER` | `env` | `env` / `aws_ssm` / `aws_secrets` / `vault` / `azure_kv` / `file`. |
| `SECRET_PREFIX` | `maintainerd/auth` | Name prefix for secrets in external providers. |
| `SECRET_FILE_PATH` | `/run/secrets` | Directory for the `file` provider (Docker/K8s secrets). |
| `SECRET_STRICT` | `false` | `true` fails closed if a secret cannot be loaded from the provider. |
| `SECRET_REFRESH_PERIOD_SECONDS` | `300` | How often refreshable secrets are re-read. |
| `AWS_REGION` | `us-east-1` | For the AWS providers. |
| `GCP_PROJECT_ID` | (empty) | For GCP Secret Manager. |
| `AZURE_KEYVAULT_URL` | (empty) | For Azure Key Vault. |
| `VAULT_ADDR` | `http://localhost:8200` | Vault address. |
| `VAULT_TOKEN` / `VAULT_ROLE_ID` / `VAULT_SECRET_ID` | (empty) | Vault auth (token, or AppRole). |
| `VAULT_MOUNT` | `secret` | Vault KV mount. |
| `VAULT_SECRET_FIELD` | `value` | Field within a Vault secret. |

## Keys & rotation

| Variable | Default | Description |
|---|---|---|
| `APP_ENCRYPTION_KEYS_PREVIOUS` | (empty) | Comma-separated **decrypt-only** retired 32-byte keys, kept during a rotation until every row is re-encrypted. |
| `JWT_KEY_ID` | `maintainerd-auth-key-1` | `kid` advertised in JWKS. |
| `JWT_KEY_ROTATION_PERIOD_SECONDS` | `86400` | Signing-key rotation period. |

## Control plane (orchestrated deployments only — OFF by default)

The default deployment is **standalone**; leave these unset unless an orchestrator (core) drives this instance.

| Variable | Default | Description |
|---|---|---|
| `CONTROL_PLANE_ENABLED` | `false` | Enables the gRPC control plane. Turning it on **forces mTLS on** and implies `GRPC_ENABLED`. |
| `GRPC_ENABLED` | `false` | Enables the gRPC listener (implied by `CONTROL_PLANE_ENABLED`). |
| `GRPC_REQUIRE_MTLS` | `false` | mTLS on the gRPC listener. Forced `true` whenever the control plane is enabled. |
| `GRPC_TLS_CERT_FILE` / `GRPC_TLS_KEY_FILE` / `GRPC_CLIENT_CA_FILE` | (empty) | gRPC server cert/key and client CA (required when mTLS is on). |
| `INSTANCE_ROLE` | `system` | This instance's role. |
| `SETUP_BOOTSTRAP_TOKEN` | (unset ⇒ gRPC setup disabled) | Per-instance bootstrap credential gating the gRPC SetupService. **Never log.** Standalone instances bootstrap via the REST setup wizard instead. |
| `SETUP_WINDOW_TTL` | `30m` | How long the orchestrated setup surface stays open after boot. Must be a positive Go duration. |

## Observability

| Variable | Default | Description |
|---|---|---|
| `OTEL_ENABLED` | `false` | Enables OpenTelemetry logs/traces/metrics. |
| `OTEL_SERVICE_NAME` | `maintainerd-auth` | Service name in telemetry. |
| `OTEL_EXPORTER_OTLP_ENDPOINT` / `OTEL_EXPORTER_OTLP_PROTOCOL` / `OTEL_EXPORTER_OTLP_INSECURE` | (SDK defaults) | Standard OTLP exporter settings, read directly by the OTel SDK. Consulted only when `OTEL_ENABLED=true`. |

## Miscellaneous / operational

| Variable | Default | Description |
|---|---|---|
| `INVITE_TTL_HOURS` | code default | Invitation link TTL (hours). |
| `GEOIP_DB_PATH` | (empty ⇒ GeoIP off) | Path to a MaxMind GeoIP DB for auth-event geolocation. |
| `MIGCHECK_DSN` / `MIGCHECK_FROM_APP_CONFIG` | (unset) | Migration-check tooling only; not needed at runtime. |
| `MAINTAINERD_DEV_LOG_OTP` | (unset) | **Dev only** — logs OTP codes. Never set in production. |

---

### Deprecated / removed — do NOT use

These older names are **no longer read** by the code. If you see them in an old `.env`, delete them.

| Old variable | Replacement |
|---|---|
| `ENV` | `APP_ENV` (legacy alias; prefer `APP_ENV` exclusively) |
| `ACCOUNT_HOSTNAME`, `AUTH_HOSTNAME` | `APP_FRONTEND_IDENTITY_HOSTNAME`, `APP_FRONTEND_CONSOLE_HOSTNAME` |
| `REDIS_HOST`, `REDIS_PORT`, `REDIS_CONNECTION_STRING` | `REDIS_ADDR` (+ `REDIS_TLS`) |
| `DB_TABLE_PREFIX` | (removed — no table prefixing) |
| `SMTP_HOST` / `SMTP_PORT` / `SMTP_USER` / `SMTP_PASS` / `SMTP_FROM_EMAIL` / `SMTP_FROM_NAME`, `EMAIL_LOGO_URL` | Email is now **per-tenant** DB config (`email_config`), not env. |
