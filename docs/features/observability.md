# Observability

> OpenTelemetry traces + logs (OTLP/gRPC, opt-in) and always-on Prometheus metrics, unified under one resource (`service.name`/`service.version`) and correlated to structured logs by `trace_id`.

| | |
|---|---|
| **Status** | Implemented. Traces + OTLP log export are opt-in (`OTEL_ENABLED=true`); Prometheus `/metrics` and the domain counters are **always on** regardless of that flag. |
| **Code** | `internal/platform/telemetry` (init + metric recorders); `cmd/server/{telemetry,logging,bootstrap}.go` (wiring); `internal/server/{rest,router,grpc}.go` (server instrumentation); `internal/platform/logging` (log fan-out + PII redaction); `internal/platform/middleware/logging_middleware.go` (trace↔log correlation) |
| **Endpoints** | `GET /metrics` (Prometheus text) on the **management** surface (`:8082` default). Traces/logs are **pushed** to an OTLP/gRPC collector (no scrape endpoint). |
| **Storage** | n/a — telemetry is exported to external systems (Prometheus/collector), not persisted in Postgres. (`auth_events` rows do carry the `trace_id`, but that table is the audit feature, not observability.) |
| **Config** | `OTEL_ENABLED`, `OTEL_SERVICE_NAME`, `OTEL_EXPORTER_OTLP_ENDPOINT` + all standard `OTEL_*` SDK vars; `MANAGEMENT_PORT`; `APP_VERSION`, `LOG_LEVEL`. No per-tenant settings. |

## Overview

Maintainerd Auth emits the three OpenTelemetry signals — **traces**, **metrics**, and **logs** — through the OTel SDK, all sharing one resource identity built from `service.name` + `service.version` (`buildResource`, `internal/platform/telemetry/telemetry.go:98`).

Two distinct export paths, with **different enable semantics**:

| Signal | Transport | Enable gate | Where it goes |
|---|---|---|---|
| Traces | OTLP/gRPC (push) | `OTEL_ENABLED=true` | OTel collector at `OTEL_EXPORTER_OTLP_ENDPOINT` |
| Logs | OTLP/gRPC (push) | `OTEL_ENABLED=true` | same collector (slog→OTel bridge) |
| Metrics | Prometheus (pull) | **always on** | `GET /metrics` on the management port |

When `OTEL_ENABLED` is not `true`, no-op Tracer/Logger providers are installed so every `otel.Tracer(...)` / bridge call is a zero-cost no-op with no branching at call sites (`telemetry.go:53`, `:122`). **Metrics do not branch on this flag** — `InitMetrics` always installs the Prometheus-backed `MeterProvider` (`telemetry.go:174`, no `Enabled()` guard), so `/metrics` and the domain counters are live in every deployment.

## How it works

### Startup wiring (`cmd/server`)

The bootstrap sequence (`cmd/server/bootstrap.go:run`) initializes telemetry early so startup failures are themselves observable:

1. `telemetry.InitLogs(ctx)` — installs the OTel `LoggerProvider` (real when enabled, no-op otherwise) **before** the logger is rebuilt, so the slog→OTLP bridge has a provider to emit through (`bootstrap.go:30`).
2. `initConfiguredLogger()` — builds the runtime logger: a JSON stdout handler, and when `OTEL_ENABLED=true` a **fan-out** to both stdout and the `otelslog` bridge, all wrapped by a **PII-redaction** handler at the top so both sinks receive already-sanitised records (`cmd/server/logging.go:28`).
3. `initTelemetry(ctx)` → `telemetry.Init` (traces) + `telemetry.InitMetrics` (metrics), returning one combined shutdown that flushes **metrics first, then traces** (`cmd/server/telemetry.go:26`).

Each `Init*` returns a `shutdown func(context.Context) error`; `run` defers them, and `shutdownWithTimeout` caps each final flush at 5 s so a slow/unavailable exporter cannot hang process exit (`cmd/server/telemetry.go:36`).

### Traces

- **`telemetry.Init`** (when enabled) creates an `otlptracegrpc` exporter, registers a `BatchSpanProcessor` (`WithBatcher`), sets the global `TracerProvider`, and installs a composite **W3C `TraceContext` + `Baggage`** propagator (`telemetry.go:64`–`:78`). The OTLP endpoint, headers, TLS, and sampler are all read from standard `OTEL_*` env vars by the SDK — the code passes no endpoint explicitly (default `localhost:4317`).
- **Automatic spans** come from instrumentation wired at the edges:

| Layer | Instrumentation | Wiring site |
|---|---|---|
| Inbound HTTP | `otelhttp.NewHandler(router, name)` on **all three** REST servers — `internal`, `public`, `management` | `internal/server/rest.go:69`–`:76` |
| Inbound gRPC | `otelgrpc.NewServerHandler()` stats handler | `internal/server/grpc.go:271` |
| PostgreSQL | `otelgorm.NewPlugin()` GORM plugin | `internal/platform/config/db.go:54` |
| Redis | `redisotel.InstrumentTracing(rdb)` | `internal/platform/config/redis.go:53` |
| Outbound SMTP | Manual span `smtp.send` | `internal/platform/email/smtp.go:33` |

- **Manual service spans** are opened in domain services with `otel.Tracer("service").Start(ctx, "<op>")` (e.g. `auth_event.log`, `workloadIdentityFederation.create`), setting domain attributes and `codes.Ok`/`codes.Error` status. The SMTP span uses tracer name `"email"`; auth-event uses `"service"`.

### Metrics

`telemetry.InitMetrics` (`telemetry.go:174`) builds a Prometheus exporter (`promexporter.New()`), installs it as the global `MeterProvider`, and registers the app's own instruments. Because otelhttp/otelgorm/redisotel emit their metrics through the **global** MeterProvider, their HTTP/DB/Redis metrics also land in the Prometheus registry — independent of `OTEL_ENABLED`. The registry is served via `promhttp.Handler()` at `GET /metrics` on the management router (`internal/server/router.go:156`).

> **Correction to prior docs / code comment:** `/metrics` is served on the **management** surface (`MANAGEMENT_PORT`, default `:8082`) via `buildManagementRouter`, **not** the internal `:8080` port. The doc comment on `InitMetrics` (`telemetry.go:168`) saying "internal port" is stale; the route is registered only in `buildManagementRouter` (`router.go:146`,`:156`).

Instruments registered by `InitMetrics`:

| Metric | Type | Labels | Meaning | Recorder |
|---|---|---|---|---|
| `build_info` | Int64 observable gauge (value `1`) | `version`, `service`, `commit`, `build_date` | Pin dashboards to a deployed build. Commit/date read from Go build info (`vcs.revision`/`vcs.time`, `readBuildInfo`) | callback in `InitMetrics` |
| `auth_events_total` | Int64 counter | `category`, `event_type`, `result` | Every auth/authz event (login/token/lockout/oauth…) | `RecordAuthEvent` |
| `security_denials_total` | Int64 counter | `type` (`permission_denied` / `rate_limited` / `ip_blocked`) | Access denials at the middleware boundary; primary probing/brute-force signal | `RecordSecurityDenial` |
| `audit_write_failures_total` | Int64 counter | — | Management-audit writes that failed; the **only** reliable signal that the best-effort audit trail has gaps | `RecordAuditWriteFailure` |

Recorder call sites (`internal/platform/telemetry/metrics.go`):

- `RecordAuthEvent` — fired from the central auth-event `Log` path (`internal/authevent/service_event.go:142`), so every recorded auth event is metered even when `audit_config` skips persisting it.
- `RecordSecurityDenial` — `permission_middleware.go:35` (permission denied), `rate_limit.go:96`/`:168` (rate limited), `ip_restriction.go:129` (IP blocked).
- `RecordAuditWriteFailure` — `internal/auditlog/service_management_audit_log.go:95`.

All recorders are **nil-safe**: the package-level counters stay `nil` until `InitMetrics` runs, and each recorder no-ops when nil (safe to call before init and in unit tests) — `metrics.go:24`,`:51`,`:68`. Denial labels are kept deliberately **low-cardinality** (no per-tenant/per-IP labels) to avoid Prometheus cardinality blow-up (`metrics.go:41`).

### Logs & correlation

- `telemetry.InitLogs` installs the OTel `LoggerProvider` with a `BatchProcessor` over an `otlploggrpc` exporter when enabled, else a no-op (`telemetry.go:118`).
- `logging.FanoutHandler` clones each record to every sink under a single level gate; `logging.NewPIIRedactHandler` sits above it so redaction happens once for all sinks (`cmd/server/logging.go:36`,`:39`).
- **Trace↔log correlation:** `LoggingMiddleware` extracts the active span's IDs via `telemetry.TraceIDFromContext` and seeds a request-scoped logger with `request_id` and (when a span exists) `trace_id` + `span_id`, so every access-log line and downstream error log carries the same IDs (`internal/platform/middleware/logging_middleware.go:44`). It must run **after** `otelhttp` (which creates the root span) and after `SecurityContextMiddleware` (which sets `request_id`). Ordering is fixed in `mountCommonMiddleware` (`router.go:326`).
- `TraceIDFromContext` returns empty strings for no/no-op spans, so `trace_id`/`span_id` are simply omitted when tracing is off (`telemetry.go:154`).

## Implementation

Key files and symbols:

| File | Symbol | Role |
|---|---|---|
| `internal/platform/telemetry/telemetry.go` | `Init`, `InitLogs`, `InitMetrics`, `Enabled`, `buildResource`, `TraceIDFromContext`, `readBuildInfo` | Signal bootstrap + resource + correlation helper |
| `internal/platform/telemetry/metrics.go` | `RecordAuthEvent`, `RecordSecurityDenial`, `RecordAuditWriteFailure`; `Denial*` label consts | Domain metric recorders (nil-safe) |
| `cmd/server/telemetry.go` | `initTelemetry`, `shutdownWithTimeout` | Combined start/stop of traces+metrics |
| `cmd/server/logging.go` | `initConfiguredLogger`, `parseSlogLevel` | Fan-out + PII-redact + OTel bridge logger |
| `cmd/server/bootstrap.go` | `run` | Ordered startup (logs → logger → traces/metrics) |
| `internal/server/rest.go` | `StartRESTServer` | `otelhttp` wrap of internal/public/management servers |
| `internal/server/router.go` | `buildManagementRouter` | Registers `GET /metrics` (`promhttp.Handler()`) |
| `internal/server/grpc.go` | `grpcServerOptions` | `otelgrpc` stats handler |
| `internal/platform/config/db.go` | `InitDB` | `otelgorm` plugin |
| `internal/platform/config/redis.go` | `NewRedisClient` | `redisotel.InstrumentTracing` |
| `internal/platform/email/smtp.go` | `smtpProvider.Send` | Manual `smtp.send` span |
| `internal/platform/logging/fanout_handler.go` | `FanoutHandler` | Multi-sink slog handler |

**Notes on span attributes (verified in code):**
- SMTP span `smtp.send` sets `smtp.host`, `smtp.port`, `email.to`, `email.subject`; on failure calls `span.RecordError` + `codes.Error` (`smtp.go:35`–`:66`). (`email.to`/`email.subject` are PII in the *span*; the redaction handler applies to **logs**, not span attributes — treat trace backends accordingly.)
- HTTP/gRPC/DB/Redis span attributes are produced by the upstream instrumentation libraries (semconv), not hand-set here.

**Instrumentation gaps (by design / not wired):** Redis is instrumented for **tracing only** — `redisotel.InstrumentMetrics` is **not** called, so there are no Redis client metrics. Startup-only work (secret-manager fetch, migrations, JWT key load) runs before/around provider init and is not traced. JWT sign/verify and template rendering are CPU-only and uninstrumented.

## Configuration

All configuration is environment-variable driven; there are **no per-tenant observability settings**.

| Env var | Default | Effect |
|---|---|---|
| `OTEL_ENABLED` | `false` | Master switch for **traces + OTLP log export**. `true` installs real providers; anything else installs no-ops. Does **not** affect Prometheus metrics. (`telemetry.go:91`) |
| `OTEL_SERVICE_NAME` | `maintainerd-auth` | `service.name` on the shared resource (traces/metrics/logs) and the `otelslog` bridge name (`telemetry.go:50`, `logging.go:34`) |
| `OTEL_EXPORTER_OTLP_ENDPOINT` | `localhost:4317` | OTLP/gRPC collector endpoint. Read by the SDK, not the app code. |
| standard `OTEL_*` | SDK defaults | Sampler (`OTEL_TRACES_SAMPLER`, `OTEL_TRACES_SAMPLER_ARG`), headers (`OTEL_EXPORTER_OTLP_HEADERS`), TLS, batch tuning (`OTEL_BSP_*`) — all honoured automatically by the exporters. |
| `MANAGEMENT_PORT` | `:8082` | Port serving `GET /metrics` (plus health probes and `openapi.json`) via the management router. |
| `APP_VERSION` | `dev` | `service.version` on the resource and the `version` label of `build_info`. Precedence: `APP_VERSION` env > `-ldflags -X ...config.AppVersion` build value > `dev` (`config/config.go:148`). |
| `LOG_LEVEL` | `info` | slog level gate applied at the fan-out handler; unknown values fall back to `info` (`logging.go:44`). |

Sampling guidance for high-traffic deployments: set `OTEL_TRACES_SAMPLER=parentbased_traceidratio` with `OTEL_TRACES_SAMPLER_ARG=0.1` (10%) and prefer tail-based, error-biased sampling in the collector.

## Security considerations

- **Metrics are unauthenticated.** `GET /metrics` has no auth (only the common middleware chain). Bind `MANAGEMENT_PORT` to a private/VPN network or scrape it in-cluster; do not expose it publicly. The counters are intentionally low-cardinality and carry no per-tenant/per-IP labels, so `/metrics` does not leak tenant identity or client IPs.
- **PII redaction covers logs, not spans.** The `PIIRedactHandler` wraps both log sinks (stdout + OTLP), so exported logs are sanitised. Span attributes are **not** run through it — the SMTP span records `email.to`/`email.subject`, and GORM's `db.statement` can contain query values. Anyone with access to the trace backend can see these; scope trace-backend access accordingly.
- **`security_denials_total`** is the intended alerting signal for probing / brute-force / policy-violation activity (permission-denied, rate-limited, IP-blocked). Alert on abnormal rates.
- **`audit_write_failures_total`** must alert on **any** non-zero rate: management-audit writes are best-effort and never block the business action, so this counter is the only reliable indicator that the audit trail has gaps.
- **`auth_events_total`** meters every auth/authz event even when `audit_config` chooses not to persist it, so operational dashboards reflect true rates independent of per-tenant audit retention.
- **Fail-open telemetry:** shutdown flushes are bounded (5 s) and recorders/no-op providers never panic, so an unreachable collector degrades observability but never the auth service.
- **Trace context propagation** is limited to W3C `TraceContext` + `Baggage`; no vendor-specific propagators are installed.

## Related

- [./events.md](./events.md) — auth-event audit trail; its `Log` path is where `auth_events_total` is metered and each row carries `trace_id`.
- [./security-settings.md](./security-settings.md) — rate limiting & IP restriction, the sources of `security_denials_total`.
- [./email-and-sms.md](./email-and-sms.md) — SMTP delivery, source of the manual `smtp.send` span.
- [./multi-tenancy.md](./multi-tenancy.md) — tenant boundaries (deliberately absent from metric labels).
- [./authentication.md](./authentication.md) — the flows whose spans and auth-event metrics dominate traffic.
