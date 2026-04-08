# OpenTelemetry — Developer Reference

This document describes the current state of OpenTelemetry (OTel) tracing in **Maintainerd Auth**.
Use it to understand what is already instrumented, how the pieces connect, and to identify gaps that still need coverage.

> **Looking for environment variable configuration?**
> See the [OpenTelemetry section in environment-variables.md](environment-variables.md#opentelemetry-tracing).

---

## Table of Contents

- [Architecture Overview](#architecture-overview)
- [Initialization](#initialization)
- [Instrumented Layers](#instrumented-layers)
  - [HTTP (REST)](#http-rest)
  - [gRPC](#grpc)
  - [PostgreSQL (GORM)](#postgresql-gorm)
  - [Redis](#redis)
  - [SMTP (Email)](#smtp-email)
  - [Structured Logging (Correlation)](#structured-logging-correlation)
- [Span Inventory](#span-inventory)
- [Not Yet Instrumented](#not-yet-instrumented)
- [Adding New Instrumentation](#adding-new-instrumentation)
- [Local Development with Jaeger](#local-development-with-jaeger)

---

## Architecture Overview

```
┌──────────────┐         ┌──────────────────┐
│   Incoming   │ ──W3C──▶│  otelhttp wrap   │──▶ REST handlers
│   Request    │  trace  │  (internal/public)│
└──────────────┘  ctx    └──────────────────┘
                                │
                  ┌─────────────┼──────────────┐
                  ▼             ▼              ▼
            ┌──────────┐ ┌───────────┐ ┌────────────┐
            │ otelgorm │ │ redisotel │ │ email span │
            │ (Postgres)│ │  (Redis)  │ │  (SMTP)    │
            └──────────┘ └───────────┘ └────────────┘
                  │             │              │
                  └─────────────┼──────────────┘
                                ▼
                     ┌─────────────────┐
                     │ BatchSpanProcessor │
                     │  → OTLP/gRPC     │
                     └────────┬────────┘
                              ▼
                     ┌───────────────┐
                     │  OTel Collector│
                     │  (or Jaeger)   │
                     └───────────────┘
```

The OTel SDK is initialized once in `main()`. When disabled (`OTEL_ENABLED != "true"`), a **no-op TracerProvider** is installed so all `otel.Tracer()` calls are safe without branching.

---

## Initialization

| File | Function | Purpose |
|---|---|---|
| `internal/telemetry/telemetry.go` | `Init(ctx)` | Bootstraps TracerProvider, exporter, propagators |
| `cmd/server/main.go` | — | Calls `telemetry.Init()` at startup, defers shutdown |

**What `Init()` does when enabled:**

1. Creates an OTLP/gRPC exporter (auto-configured via `OTEL_EXPORTER_OTLP_ENDPOINT`).
2. Builds a `resource.Resource` with `service.name` and `service.version` attributes.
3. Creates a `BatchSpanProcessor` and registers a `TracerProvider`.
4. Installs W3C `TraceContext` + `Baggage` propagators globally.
5. Returns a `shutdown` function that flushes buffered spans (5 s timeout).

**What `Init()` does when disabled:**

1. Installs `noop.NewTracerProvider()` — all tracing calls become zero-cost no-ops.

---

## Instrumented Layers

### HTTP (REST)

| Item | Value |
|---|---|
| **Package** | `go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp` |
| **File** | `internal/rest/server/server.go` |
| **Integration** | `otelhttp.NewHandler(router, serverName)` wraps each HTTP server |
| **Servers** | Internal (port 8080, span prefix `"internal"`), Public (port 8081, span prefix `"public"`) |

**Auto-captured attributes:**

| Attribute | Example |
|---|---|
| `http.method` | `GET` |
| `http.route` | `/v1/users` |
| `http.status_code` | `200` |
| `http.target` | `/v1/users?page=1` |
| `http.scheme` | `http` |
| `net.host.name` | `localhost` |

**Context propagation:** Incoming `traceparent` / `tracestate` headers are parsed automatically. Child spans created in handlers inherit the request's trace context.

---

### gRPC

| Item | Value |
|---|---|
| **Package** | `go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc` |
| **File** | `internal/grpc/server/server.go` |
| **Integration** | `grpc.StatsHandler(otelgrpc.NewServerHandler())` passed to `grpc.NewServer()` |

**Auto-captured attributes:**

| Attribute | Example |
|---|---|
| `rpc.system` | `grpc` |
| `rpc.service` | `maintainerd.auth.v1.AuthService` |
| `rpc.method` | `ValidateToken` |
| `rpc.grpc.status_code` | `0` (OK) |

---

### PostgreSQL (GORM)

| Item | Value |
|---|---|
| **Package** | `github.com/uptrace/opentelemetry-go-extra/otelgorm` |
| **File** | `internal/config/db.go` |
| **Integration** | `db.Use(otelgorm.NewPlugin())` after `gorm.Open()` |

**Auto-captured attributes:**

| Attribute | Example |
|---|---|
| `db.system` | `postgresql` |
| `db.statement` | `SELECT * FROM md_users WHERE …` |
| `db.operation` | `SELECT` / `INSERT` / `UPDATE` / `DELETE` |
| `db.sql.table` | `md_users` |

> **Note:** `otelgorm` hooks into GORM callbacks, so every query and transaction is traced automatically — including those inside repository methods.

---

### Redis

| Item | Value |
|---|---|
| **Package** | `github.com/redis/go-redis/extra/redisotel/v9` |
| **File** | `internal/config/redis.go` |
| **Integration** | `redisotel.InstrumentTracing(rdb)` after client creation and ping |

**Auto-captured attributes:**

| Attribute | Example |
|---|---|
| `db.system` | `redis` |
| `db.statement` | `GET session:abc123` |
| `db.operation` | `GET` / `SET` / `DEL` / `EXPIRE` |
| `net.peer.name` | `redis-db` |
| `net.peer.port` | `6379` |

---

### SMTP (Email)

| Item | Value |
|---|---|
| **Package** | `go.opentelemetry.io/otel` (manual instrumentation) |
| **File** | `internal/email/email.go` |
| **Tracer name** | `"email"` |
| **Span name** | `"smtp.send"` |

**Manually-set attributes:**

| Attribute | Source |
|---|---|
| `smtp.host` | `config.SMTPHost` |
| `smtp.port` | `config.SMTPPort` |
| `email.to` | `params.To` |
| `email.subject` | `params.Subject` |

**Error handling:** On SMTP failure, `span.RecordError(err)` is called and span status is set to `codes.Error`.

> `gopkg.in/gomail.v2` does not accept `context.Context`, so the span wraps the entire `DialAndSend()` call. Context propagation stops at the SMTP boundary.

---

### Structured Logging (Correlation)

| Item | Value |
|---|---|
| **File** | `internal/middleware/logging_middleware.go` |
| **Integration** | `telemetry.TraceIDFromContext(r.Context())` extracts IDs from active span |

**Injected log fields:**

| Field | Condition | Example |
|---|---|---|
| `request_id` | Always | `a1b2c3d4-…` |
| `trace_id` | When OTel span is active | `4bf92f3577b34da6a3ce929d0e0e4736` |
| `span_id` | When OTel span is active | `00f067aa0ba902b7` |

The logger with these fields is stored in the request context via `response.WithLogger()`, making it available to all downstream handlers.

---

## Span Inventory

Complete list of spans produced by a typical request:

| # | Span Name | Source | Type |
|---|---|---|---|
| 1 | `HTTP {METHOD} {route}` | `otelhttp` | Auto |
| 2 | `{rpc.service}/{rpc.method}` | `otelgrpc` | Auto |
| 3 | `{db.operation} {db.sql.table}` | `otelgorm` | Auto |
| 4 | `{redis.command}` | `redisotel` | Auto |
| 5 | `smtp.send` | `internal/email` | Manual |

> Rows 1–4 are created automatically by instrumentation libraries. Row 5 is created by our code.

---

## Not Yet Instrumented

The following are **not** currently covered by tracing. They are either startup-only operations or lower-priority paths:

| Area | Reason | Priority |
|---|---|---|
| Secret manager calls (AWS, Vault, GCP, Azure) | Runs once at startup, before TracerProvider is initialized | Low |
| Signed URL generation (`internal/signedurl`) | CPU-only, no I/O | Low |
| Template rendering | CPU-only, no I/O | Low |
| JWT signing / verification | CPU-only, very fast | Low |
| OTP generation (`internal/crypto`) | CPU-only | Low |
| Database migrations / seeders | Runs at startup | Low |

> If any of these become performance bottlenecks, add a manual span following the pattern in `internal/email/email.go`.

---

## Adding New Instrumentation

### Manual span (for outbound I/O)

```go
import (
    "go.opentelemetry.io/otel"
    "go.opentelemetry.io/otel/attribute"
    "go.opentelemetry.io/otel/codes"
)

func doSomething(ctx context.Context) error {
    ctx, span := otel.Tracer("mypackage").Start(ctx, "operation.name")
    defer span.End()

    span.SetAttributes(
        attribute.String("key", "value"),
    )

    if err := actualWork(ctx); err != nil {
        span.RecordError(err)
        span.SetStatus(codes.Error, "short description")
        return err
    }

    span.SetStatus(codes.Ok, "")
    return nil
}
```

### Conventions

- **Tracer name:** Use the package name (e.g. `"email"`, `"signedurl"`).
- **Span name:** Use `noun.verb` format (e.g. `"smtp.send"`, `"vault.get_secret"`).
- **Attributes:** Follow [OpenTelemetry Semantic Conventions](https://opentelemetry.io/docs/specs/semconv/) when a matching convention exists.
- **Errors:** Always call `span.RecordError(err)` and `span.SetStatus(codes.Error, msg)` on failure.
- **Context threading:** Accept `context.Context` as the first parameter and pass it through. This ensures spans are correctly parented.

---

## Local Development with Jaeger

```bash
# Start Jaeger all-in-one (receives OTLP on port 4317, UI on 16686)
docker run -d --name jaeger \
  -p 4317:4317 \
  -p 16686:16686 \
  jaegertracing/all-in-one:latest
```

```env
OTEL_ENABLED="true"
OTEL_EXPORTER_OTLP_ENDPOINT="localhost:4317"
OTEL_SERVICE_NAME="maintainerd-auth"
```

Open <http://localhost:16686> → select **maintainerd-auth** → browse traces.

Each trace shows the full request lifecycle: HTTP entry → handler → DB queries → Redis commands → email sends (if applicable).
