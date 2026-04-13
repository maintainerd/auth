# Frontend Initialization

## Scope

This document covers how **auth-console** (admin UI) and **auth-identity** (login page) boot
themselves before any user authentication happens. It does not cover OIDC third-party provider
compatibility — see `docs/v1-features/oidc-provider.md` for that.

## How auth providers handle this

Every major auth provider splits initialization into two concerns:

| Concern | What it answers |
|---|---|
| **Tenant validation** | Is this URL/subdomain pointing to a real, active tenant? |
| **UI configuration** | What flows are available? What does the login page look like? |

Providers like Keycloak and Auth0 serve both from a single composite endpoint (e.g.
`GET /realms/{realm}` on Keycloak). SuperTokens and Firebase bake all config into the
JS bundle at build time — that doesn't work for a subdomain-per-tenant model.

## Our URL pattern

This is a **single-tenant-per-container** system. The container already knows who it is.
The subdomain (`tenant-name` in `https://tenant-name.auth.maintainerd.com`) is handled by
the reverse proxy routing traffic to the right container.

**The identifier in the URL is used for client-side validation only** — confirming the
subdomain the user landed on matches the tenant before rendering a login form.

### auth-console boot

```
GET :8080/tenant/       → returns system tenant (name, status, branding)
```

### auth-identity boot

```
1. Extract "tenant-name" from subdomain
2. GET :8081/tenant/{identifier}/config   → validate + get all init data in one call
3. Render login page
```

---

## Checklist

### Phase 1 — Tenant Config Endpoint (Composite Init Call)

This is the single call the login frontend makes on boot. It returns everything needed to render the login page.

- [ ] Create `GET /tenant/{identifier}/config` on port **8081** (public, no auth)
- [ ] Add `TenantConfigResponseDTO` in `internal/dto/tenant.go`
  ```json
  {
    "tenant": {
      "tenant_id": "uuid",
      "name": "Acme Corp",
      "display_name": "Acme",
      "identifier": "acme",
      "status": "active",
      "is_public": true
    },
    "identity_providers": [
      { "provider": "google", "client_id": "...", "enabled": true },
      { "provider": "github", "client_id": "...", "enabled": true }
    ],
    "security": {
      "password_min_length": 8,
      "password_require_uppercase": true,
      "password_require_number": true,
      "password_require_symbol": false,
      "mfa_enforced": false,
      "mfa_methods": ["totp", "sms"]
    },
    "login_template": {
      "logo_url": "https://...",
      "primary_color": "#0077FF",
      "background_color": "#FFFFFF",
      "custom_css": ""
    },
    "signup": {
      "enabled": true,
      "required_fields": ["email", "fullname"],
      "allowed_email_domains": []
    }
  }
  ```
- [ ] Add `GetConfig(ctx, identifier)` method to `TenantService` interface
  - Calls `tenantRepo.FindByIdentifier(identifier)`
  - If tenant `status != "active"` → return `apperror.NewNotFound` (treat as not found for security)
  - Loads identity providers for the tenant (only enabled ones, strip secrets — expose only `client_id` and `provider` name)
  - Loads security settings
  - Loads login template
  - Loads signup flow config (registration enabled/disabled, required fields)
- [ ] Add `GetConfig` handler method to `TenantHandler`
- [ ] Add route in `TenantPublicRoute`
- [ ] Unit tests for `GetConfig` service method (table-driven, 100% coverage)
  - Case: tenant not found
  - Case: tenant found but status is `inactive` → 404
  - Case: tenant found, active, no IDPs configured
  - Case: tenant found, active, with IDPs, security settings, and login template
- [ ] Ensure IDP OAuth2 `client_secret` is **never** included in this response

### Phase 2 — CORS for Public Port

The login frontend runs on a different origin (`https://tenant-name.auth.maintainerd.com`).
The backend public port (`:8081`) must allow cross-origin requests from it.

- [ ] Add CORS middleware to the public server (`internal/rest/server/server.go` port 8081)
  - Use `github.com/go-chi/cors` or `github.com/rs/cors`
  - Allowed origins: configurable via env var `CORS_ALLOWED_ORIGINS` (comma-separated)
  - Allowed methods: `GET, POST, OPTIONS`
  - Allowed headers: `Content-Type, Authorization`
  - `AllowCredentials: true` (needed for cookie-based refresh tokens on public flows)
- [ ] Add `CORS_ALLOWED_ORIGINS` to environment variable documentation
  - `docs/contributing/environment-variables.md`
  - `docs/deployment/environment-variables.md`
- [ ] Unit test: CORS preflight to `/tenant/{identifier}/config` returns correct headers

### Phase 3 — Caching

- [ ] Cache `GET /tenant/{identifier}/config` response in Redis
  - Key: `tenant_config:{identifier}`
  - TTL: 5 minutes (`300s`)
  - **Invalidate** when any of the following change for this tenant:
    - Tenant updated (name, status, display settings)
    - Identity provider added/removed/enabled/disabled
    - Security settings updated
    - Login template updated
    - Signup flow updated
- [ ] Wire `cache.Invalidator` calls into the relevant service update methods
- [ ] Unit tests for cache invalidation paths

### Phase 4 — Health & Status Endpoints (Prerequisite / Related)

These are needed by load balancers and Kubernetes before any tenant request reaches the app.

- [ ] `GET /healthz` on **both** port 8080 and 8081
  - Returns `200 OK` with `{ "status": "ok" }` when the process is running
  - No DB check (liveness probe — just "am I alive?")
- [ ] `GET /readyz` on **both** ports
  - Checks DB connectivity (`db.Ping()`) and Redis connectivity
  - Returns `200 OK` if both pass, `503 Service Unavailable` if either fails
  - Response body: `{ "status": "ready", "db": "ok", "redis": "ok" }` or failure details
- [ ] Unit tests and integration tests for both endpoints

---

## Implementation order

1. Phase 4 first — health checks unblock deployment validation
2. Phase 2 — CORS unblocks any frontend testing from day one
3. Phase 1 — Tenant config endpoint is the core deliverable
4. Phase 3 — Caching last (can ship without it, add after correctness is verified)
