# Architecture

This document describes the internal architecture of **Maintainerd Auth** — a multi-tenant authentication and authorization service built in Go.
If you are new to the codebase, read this first to understand how the pieces fit together before diving into any specific package.

---

## Table of Contents

- [High-Level Overview](#high-level-overview)
- [Architecture Diagram](#architecture-diagram)
- [Request Lifecycle](#request-lifecycle)
- [Entry Point & Bootstrap](#entry-point--bootstrap)
- [Dependency Injection](#dependency-injection)
- [Layer-by-Layer Breakdown](#layer-by-layer-breakdown)
  - [Configuration](#configuration)
  - [Models](#models)
  - [Repositories](#repositories)
  - [Services](#services)
  - [DTOs & Validation](#dtos--validation)
  - [REST Handlers & Routes](#rest-handlers--routes)
  - [gRPC](#grpc)
  - [Middleware](#middleware)
  - [JWT & Tokens](#jwt--tokens)
  - [Security](#security)
- [Multi-Tenancy](#multi-tenancy)
- [Database & Migrations](#database--migrations)
- [Caching](#caching)
- [Key Design Patterns](#key-design-patterns)
- [Technology Stack](#technology-stack)

---

## High-Level Overview

The service follows **clean architecture** principles with strict one-way dependency flow:

```
Handlers (REST / gRPC)
       ↓
   Services  (business logic)
       ↓
  Repositories  (data access)
       ↓
   Database / Cache
```

Each layer only depends on the layer directly below it.
All cross-layer communication happens through **interfaces**, making every layer independently testable.

---

## Architecture Diagram

```
┌──────────────────────────────────────────────────────────────┐
│                     cmd/server/main.go                       │
│  config → DB → Redis → migrations → seeders → DI → servers  │
└────────────┬─────────────────────────────────┬───────────────┘
             │                                 │
  ┌──────────▼───────────┐          ┌──────────▼──────────┐
  │  REST Server :8080   │          │  REST Server :8081   │
  │  (Internal / Admin)  │          │  (Public / Auth)     │
  │  VPN-only access     │          │  Public-facing       │
  └──────────┬───────────┘          └──────────┬───────────┘
             │                                 │
             └────────────────┬────────────────┘
                              │
  ┌───────────────────────────▼─────────────────────────────┐
  │                    Middleware Stack                      │
  │                                                         │
  │  Security Headers → Request Size Limit → Timeout        │
  │  → Security Context → JWT Auth → User Context           │
  │  → Permission Check                                     │
  └───────────────────────────┬─────────────────────────────┘
                              │
  ┌───────────────────────────▼─────────────────────────────┐
  │                   REST Handlers (29)                     │
  │                                                         │
  │  Parse request → Validate DTO → Call service            │
  │  → Map result to response DTO → Write JSON response     │
  └───────────────────────────┬─────────────────────────────┘
                              │
  ┌───────────────────────────▼─────────────────────────────┐
  │                Service Interfaces (26)                   │
  │                                                         │
  │  Business rules, transaction management,                │
  │  tenant isolation, audit logging                        │
  └──────────┬────────────────────────────────┬─────────────┘
             │                                │
  ┌──────────▼───────────┐         ┌──────────▼──────────┐
  │  Repository Layer    │         │      Redis           │
  │  (39 interfaces)     │         │  (cache + rate       │
  │                      │         │   limiting)          │
  │  Generic CRUD base   │         └─────────────────────┘
  │  + domain queries    │
  │  + tenant scoping    │
  └──────────┬───────────┘
             │
  ┌──────────▼───────────┐         ┌─────────────────────┐
  │    PostgreSQL        │         │  gRPC Server :50051  │
  │    (GORM + JSONB)    │         │  (SeederService)     │
  └──────────────────────┘         └─────────────────────┘
```

---

## Request Lifecycle

A typical authenticated API request flows through these steps:

1. **Nginx** proxy receives the request and forwards it to the appropriate port.
2. **Security Headers Middleware** sets CSP, HSTS, X-Frame-Options, etc.
3. **Request Size Limit Middleware** rejects oversized payloads.
4. **Timeout Middleware** applies a deadline to the request context.
5. **Security Context Middleware** extracts client IP, user-agent, and generates a `X-Request-ID`.
6. **JWT Auth Middleware** extracts and validates the Bearer token (or `access_token` cookie). Populates context with `sub`, `scope`, `aud`, `iss`, `jti`, `client_id`, `provider_id`.
7. **User Context Middleware** looks up the full user (with roles, permissions, tenant, client) from Redis cache or the database. Populates context.
8. **Permission Middleware** checks if the user has at least one of the required permissions for the endpoint.
9. **Handler** parses the request body/query, validates the DTO, calls the service, maps the result, and writes the response.
10. **Service** runs business logic inside a database transaction.
11. **Repository** executes tenant-scoped queries via GORM.

---

## Entry Point & Bootstrap

**File:** `cmd/server/main.go`

The boot sequence is strictly sequential — each step must succeed before the next begins:

```
1. Configure structured JSON logging
2. Load environment variables and validate
3. Parse and validate RSA JWT key pair
4. Connect to PostgreSQL
5. Connect to Redis and validate the connection
6. Run database migrations (with advisory lock)
7. Run database seeders
8. Wire the dependency graph  →  app.NewApp(db, redisClient)
9. Start gRPC server in a background goroutine
10. Start two REST servers (internal + public)
11. Wait for OS signal (SIGINT / SIGTERM)
12. Graceful shutdown: drain REST (30s) → cancel gRPC context → exit
```

If any step from 1–8 fails, the process exits immediately with a non-zero code.

---

## Dependency Injection

**Files:** `internal/app/app.go`, `internal/app/repositories.go`, `internal/app/services.go`

DI is **manual and constructor-based** — no frameworks, no reflection, no globals.
The wiring happens in three phases:

```
Phase 1: initRepos(db)           → creates 39 repository instances
Phase 2: initServices(db, repos) → creates 26 service instances, injecting repos
Phase 3: NewApp(db, redis)       → returns App struct exposing services
```

The `App` struct is then passed to both REST and gRPC server initializers, which create their handlers from the service interfaces.

```go
// Simplified view of the App struct
type App struct {
    DB              *gorm.DB
    RedisClient     *redis.Client
    UserService     service.UserService       // interface
    LoginService    service.LoginService      // interface
    RegisterService service.RegisterService   // interface
    RoleService     service.RoleService       // interface
    // ... 19 more service interfaces
}
```

All fields are **interfaces**, never concrete types. This makes every component swappable for testing.

---

## Layer-by-Layer Breakdown

### Configuration

**Package:** `internal/config/`

Follows the **12-Factor App** model — all configuration comes from environment variables.

| File | Responsibility |
|---|---|
| `config.go` | Central config loader. Reads env vars, resolves secrets via a pluggable `SecretManager`, returns a config struct or an error. |
| `env.go` | `GetEnv()` (required, returns error) and `GetEnvOrDefault()` (with fallback). |
| `db.go` | Opens a GORM PostgreSQL connection. |
| `redis.go` | Creates and validates a Redis client. |
| `secret_manager.go` | `SecretManager` interface with implementations for: environment variables, Docker secrets (file-based), AWS SSM, AWS Secrets Manager, HashiCorp Vault, Azure Key Vault. |

---

### Models

**Package:** `internal/model/`

GORM model structs representing database tables. Key entities:

| Model | Description |
|---|---|
| `User` | Core user entity with credentials, status, metadata (JSONB). |
| `Tenant` | An isolated tenant. Users belong to tenants via junction tables. |
| `Role` | Tenant-scoped role containing permissions. |
| `Permission` | An action scoped to an API (e.g., `user:read`, `user:create`). |
| `Client` | OAuth/OIDC application with redirect URIs, secrets, and config. |
| `IdentityProvider` | Authentication provider (internal, Google, GitHub, etc.). |
| `API` | Represents a registered API (REST, gRPC, GraphQL, etc.). |
| `Service` | A top-level service containing APIs. |
| `UserIdentity` | Links a user to a tenant + client + provider (multi-tenant identity). |
| `UserRole` | Junction table: user ↔ role assignment within a tenant. |
| `UserToken` | Tokens for email verification, password reset, etc. |
| `OAuthAuthorizationCode` | Short-lived authorization code for the OAuth 2.0 authorization code grant (RFC 6749 §4.1). Stored as SHA-256 hash. |
| `OAuthRefreshToken` | Long-lived opaque refresh token with family-based rotation and reuse detection. Stored as SHA-256 hash. |
| `OAuthConsentGrant` | Persisted record of which scopes a user has approved for a client. Unique per user-client pair. |
| `OAuthConsentChallenge` | Short-lived pending consent challenge created during the authorization flow when user approval is required. |

**Common patterns across all models:**
- UUID auto-generation in `BeforeCreate` hook.
- `CreatedAt` / `UpdatedAt` managed automatically by GORM.
- JSONB `Metadata` field for extensibility.
- Status constants defined in `internal/model/constants.go`.

---

### Repositories

**Package:** `internal/repository/`

Every repository is defined as an **interface** with a concrete implementation.

**Generic base:**

```go
type BaseRepository[T any] struct {
    db            *gorm.DB
    uuidFieldName string
    idFieldName   string
}
```

The `BaseRepositoryMethods[T]` interface provides: `Create`, `CreateOrUpdate`, `FindAll`, `FindByUUID`, `FindByUUIDs`, `FindByID`, `UpdateByUUID`, `UpdateByID`, `DeleteByUUID`, `DeleteByID`, `Paginate`.

**Domain repositories** embed the base and add entity-specific queries (e.g., `FindByEmail`, `FindBySubAndClientID`).

**Key safety patterns:**
- All queries use parameterized placeholders (`?`) — no string concatenation.
- `ORDER BY` columns are validated against an allowlist via `sanitizeOrder()`.
- `WithTx(tx *gorm.DB)` creates a new repository instance bound to a transaction.
- `FindByUUID` returns `(nil, nil)` when not found (not an error), letting the service layer decide the response.

---

### Services

**Package:** `internal/service/`

Each service is an **interface** with a private struct implementation. Services receive repositories (and the `*gorm.DB` for transactions) via constructors.

**Responsibilities:**
- Business logic and validation.
- Transaction management via `db.Transaction()`.
- Tenant-scoped data access (every method receives `tenantID`).
- Mapping between model entities and service result structs.

**Transaction pattern:**

```go
err := s.db.Transaction(func(tx *gorm.DB) error {
    txUserRepo := s.userRepo.WithTx(tx)
    txRoleRepo := s.roleRepo.WithTx(tx)
    // All operations within this block use the same transaction.
    // Return nil to commit, return error to rollback.
    return nil
})
```

**Core services:**

| Service | Responsibility |
|---|---|
| `UserService` | CRUD, status changes, role assignment, email/phone verification. |
| `LoginService` | Authentication, rate limiting, token generation. |
| `RegisterService` | User registration, invite-based registration. |
| `RoleService` | Role CRUD, permission assignment. |
| `PermissionService` | Permission CRUD scoped to APIs. |
| `TenantService` | Tenant management. |
| `ClientService` | OAuth/OIDC client management. |
| `IdentityProviderService` | IDP configuration. |
| `SetupService` | Initial tenant bootstrap (creates default users, roles, permissions). |
| `PolicyService` | Authorization policy management. |
| `APIKeyService` | API key generation and validation. |
| `SecuritySettingService` | IP restriction rules, security policies. |
| `ForgotPasswordService` | Password reset flow initiation. |
| `ResetPasswordService` | Password reset execution. |
| `SignupFlowService` | Configurable signup flow management. |
| `EmailTemplateService` | Email template management. |
| `SMSTemplateService` | SMS template management. |
| `LoginTemplateService` | Login page template management. |
| `OAuthAuthorizeService` | OAuth 2.0 authorization endpoint logic: validates clients, redirect URIs, PKCE; issues authorization codes or creates consent challenges. |
| `OAuthTokenService` | OAuth 2.0 token endpoint logic: authorization code exchange, refresh token rotation with reuse detection, client credentials grant, token revocation (RFC 7009), and introspection (RFC 7662). |
| `OAuthConsentService` | Manages user consent grants: lists and revokes persisted consent records. |

---

### DTOs & Validation

**Package:** `internal/dto/`

DTOs sit between the handler and service layers. They define the shape of API requests and responses.

**Every request DTO has a `Validate()` method:**

```go
func (r *LoginRequestDTO) Validate() error {
    r.Username = security.SanitizeInput(r.Username)  // sanitize first
    r.Password = security.SanitizeInput(r.Password)

    return validation.ValidateStruct(r,
        validation.Field(&r.Username, validation.Required, validation.Length(1, 255)),
        validation.Field(&r.Password, validation.Required, validation.Length(1, 128)),
    )
}
```

Validation uses `github.com/go-ozzo/ozzo-validation/v4` with declarative rules. Input is sanitized *before* validation.

**Pagination is standardized:**

```go
type PaginationRequestDTO struct {
    Page, Limit         int
    SortBy, SortOrder   string
}

type PaginatedResponseDTO[T any] struct {
    Rows       []T
    Total      int64
    Page       int
    Limit      int
    TotalPages int
}
```

---

### REST Handlers & Routes

**Packages:** `internal/rest/handler/`, `internal/rest/route/`, `internal/rest/server/`

**Two separate HTTP servers run on different ports:**

| Server | Port | Purpose | Access |
|---|---|---|---|
| Internal | `:8080` | Admin and management endpoints, token introspection | VPN / private network only |
| Public | `:8081` | Login, register, password reset, OAuth 2.0 flows, OIDC discovery | Public internet |

This enforces a hard trust boundary at the network level.

**Handler pattern:**

```
1. Extract tenant from context (set by middleware)
2. Parse and validate the request DTO
3. Call the service method with tenant context
4. Map the service result to a response DTO
5. Write the JSON response
```

**Route registration** uses `chi.Router` with middleware applied per route group:

```go
r.Route("/users", func(r chi.Router) {
    r.Use(middleware.JWTAuthMiddleware)
    r.Use(middleware.UserContextMiddleware(userRepo, redisClient))

    r.With(middleware.PermissionMiddleware([]string{"user:read"})).
        Get("/", userHandler.GetUsers)

    r.With(middleware.PermissionMiddleware([]string{"user:create"})).
        Post("/", userHandler.CreateUser)
})
```

**Standardized JSON response format:**

```json
{
    "success": true,
    "data": { },
    "message": "Users fetched successfully"
}
```

Error responses follow the same shape with `success: false` and an `error` field.

---

### gRPC

**Packages:** `internal/grpc/`, `proto/maintainerd/`, `internal/gen/go/`

- Proto definitions live in `proto/maintainerd/`.
- Generated Go code is output to `internal/gen/go/` (do not edit manually).
- Regenerate with `make proto`.
- The gRPC server runs on `:50051` in a background goroutine and shuts down via context cancellation.

Currently only `SeederService` is implemented. The infrastructure is ready for expansion.

---

### Middleware

**Package:** `internal/middleware/`

Middleware is applied as a composable stack using chi's `r.Use()`:

| Middleware | Responsibility |
|---|---|
| **SecurityHeadersMiddleware** | Sets `X-Content-Type-Options`, `X-Frame-Options`, `CSP`, `HSTS` (production), `Referrer-Policy`, `Permissions-Policy`. |
| **RequestSizeLimitMiddleware** | Rejects request bodies exceeding a configurable limit. |
| **TimeoutMiddleware** | Applies a deadline to the request context. |
| **SecurityContextMiddleware** | Extracts client IP, user-agent, generates `X-Request-ID`, logs security events. |
| **JWTAuthMiddleware** | Validates Bearer token or `access_token` cookie. Populates context with JWT claims (`sub`, `scope`, `aud`, `iss`, `jti`, `client_id`, `provider_id`). |
| **UserContextMiddleware** | Resolves the full user object (with roles, permissions, tenant, client) from Redis cache or DB. Populates context. |
| **PermissionMiddleware** | Checks the user's roles/permissions against the required set. Returns 403 if insufficient. |

The stack is ordered so that **cheap checks run first** (headers, size limits) and **expensive checks run last** (DB lookups, permission evaluation).

---

### JWT & Tokens

**Package:** `internal/jwt/`

| Token Type | TTL | Purpose |
|---|---|---|
| Access Token | 15 minutes | API authentication |
| ID Token | 1 hour | User identity claims |
| Refresh Token | 7 days | Obtain new access tokens |
| Authorization Code | 10 minutes | OAuth 2.0 authorization code grant (stored as SHA-256 hash, single-use) |
| OAuth Refresh Token | 7 days (configurable per client) | Opaque token with family-based rotation and reuse detection |

**Security properties:**
- RSA-256 signing algorithm.
- Minimum 2048-bit key size enforced at startup.
- Cryptographically secure JTI (JWT ID) for uniqueness.
- Key pair validation on boot (private and public keys must match).
- PKCE (S256 only) required for all authorization code grants.
- Refresh token rotation with family-based reuse detection — replaying a revoked token revokes the entire family.
- JWKS endpoint exposes the public RSA key for external token verification.

---

### Security

**Package:** `internal/security/`

| Feature | Details |
|---|---|
| **Rate limiting** | Max 5 login attempts per 15-minute window per identifier. |
| **Account lockout** | 30-minute lockout after exceeding attempts. |
| **Password strength** | Minimum 12 characters. Requires uppercase, lowercase, digit, and special character. |
| **Password hashing** | bcrypt with default cost. |
| **Input sanitization** | Applied in DTOs before validation. |
| **Signed URLs** | HMAC-SHA256 signed, time-limited URLs for email verification, password reset, invites. |
| **Secure cookies** | `HttpOnly`, `Secure`, `SameSite=Strict`. Refresh token scoped to `/auth/refresh` path. |
| **Security event logging** | Structured logs for login success/failure, rate limiting, lockout — aligned with SOC2 CC7.2. |
| **PKCE** | S256-only code challenge enforcement for all OAuth authorization code grants. |
| **Redirect URI validation** | Exact-match validation against pre-registered client URIs. |
| **Token rotation** | Family-based refresh token rotation with reuse detection (compromised tokens revoke the entire family). |

---

## Multi-Tenancy

Multi-tenancy is a first-class concern built into every layer:

```
Tenant
  ├── Users (via tenant_users junction)
  ├── Roles (tenant-scoped)
  │     └── Permissions
  ├── Clients (OAuth apps)
  ├── Identity Providers
  ├── Services & APIs
  └── Security Settings & IP Rules
```

**How isolation is enforced:**

1. **Middleware** extracts the tenant from the authenticated user's context.
2. **Handlers** pass `tenant.TenantID` to every service call.
3. **Services** include `tenantID` in every repository query.
4. **Repositories** use `JOIN tenant_users` and `WHERE tenant_id = ?` to scope all data access.

A user can belong to **multiple tenants** via the `UserIdentity` model, each with separate roles and permissions.

---

## Database & Migrations

**Engine:** PostgreSQL with GORM.

**Migration runner:** `internal/runner/migration.go`

- Migrations are versioned functions registered in an ordered slice.
- A PostgreSQL **advisory lock** (`SELECT pg_advisory_lock(7316949)`) prevents concurrent migration execution across multiple pods.
- Applied migrations are tracked in a `schema_migrations` table.
- **Rule:** Never reorder or delete existing migrations. Only append new ones.

**Seeder runner:** `internal/runner/seeder.go`

Seeds default data required for the service to function (default tenant, roles, permissions, system clients, etc.).

---

## Caching

**Engine:** Redis

| Use Case | Key Pattern | TTL |
|---|---|---|
| User context (middleware) | `user:{sub}:{client_id}` | 5 minutes |
| Rate limiting | Identifier-based counters | 15 minutes |

> **Note:** If a user's roles or permissions change, the cached context may be stale for up to 5 minutes.

---

## Key Design Patterns

| Pattern | Where | Why |
|---|---|---|
| **Constructor-based DI** | `internal/app/` | Explicit wiring, no magic, easy to trace. |
| **Generic repository** | `internal/repository/` | Eliminates boilerplate CRUD across 35 entities. |
| **Interface-driven layers** | Services, Repositories | Every dependency is mockable for unit tests. |
| **Transaction propagation** | Services → `WithTx()` | Ensures atomicity across multi-entity operations. |
| **DTO boundary** | `internal/dto/` | Decouples API shape from internal models. Sanitizes and validates at the edge. |
| **Middleware composition** | `internal/middleware/` | Separates cross-cutting concerns (auth, logging, security) from business logic. |
| **Dual-server split** | `internal/rest/server/` | Hard network-level separation between admin and public APIs. |
| **Function variable injection** | `email.SendEmail`, `crypto.GenerateIdentifier`, `security.HashPassword` | Allows test doubles without interfaces for simple leaf dependencies. |
| **Advisory-locked migrations** | `internal/runner/` | Safe for multi-pod deployments — only one pod migrates at a time. |

---

## Technology Stack

| Category | Technology |
|---|---|
| Language | Go 1.26 |
| Database | PostgreSQL 12+ |
| Cache | Redis 6.0+ |
| ORM | GORM (`gorm.io/gorm`) |
| HTTP Router | chi (`github.com/go-chi/chi/v5`) |
| JWT | `github.com/golang-jwt/jwt/v5` |
| Validation | ozzo-validation (`github.com/go-ozzo/ozzo-validation/v4`) |
| Crypto | `golang.org/x/crypto` (bcrypt) |
| UUID | `github.com/google/uuid` |
| Email | gomail (`gopkg.in/gomail.v2`) |
| gRPC | `google.golang.org/grpc` + protobuf |
| Testing | testify (`github.com/stretchr/testify`), go-sqlmock |
| Container | Docker (multi-stage, `alpine` production image) |
| Proxy | Nginx (local development) |
