# OpenTelemetry — Production Operations Guide

This document covers the production observability setup for **Maintainerd Auth** using OpenTelemetry tracing.

> **Looking for environment variable reference?**
> See the [OpenTelemetry section in environment-variables.md](environment-variables.md#opentelemetry-tracing).
>
> **Looking for developer instrumentation details?**
> See [`docs/contributing/opentelemetry.md`](../contributing/opentelemetry.md).

---

## Table of Contents

- [Overview](#overview)
- [What Is Traced](#what-is-traced)
- [Span Reference](#span-reference)
- [Log Correlation](#log-correlation)
- [Collector Architecture](#collector-architecture)
- [Sampling Strategy](#sampling-strategy)
- [Backend Recommendations](#backend-recommendations)
- [Dashboards & Alerts](#dashboards--alerts)
- [Troubleshooting](#troubleshooting)

---

## Overview

Maintainerd Auth emits distributed traces via the [OpenTelemetry](https://opentelemetry.io/) SDK. Traces are exported over **OTLP/gRPC** to an OpenTelemetry Collector (or compatible backend).

When disabled (`OTEL_ENABLED != "true"`), a no-op tracer is installed with **zero runtime overhead**.

---

## What Is Traced

| Layer | Library | Scope |
|---|---|---|
| **Inbound HTTP** (REST) | `otelhttp` | Every request to both internal (`:8080`) and public (`:8081`) servers |
| **Inbound gRPC** | `otelgrpc` | Every RPC to the gRPC server |
| **PostgreSQL** | `otelgorm` | Every SQL query and transaction |
| **Redis** | `redisotel` | Every Redis command (GET, SET, DEL, EXPIRE, etc.) |
| **Outbound SMTP** | Manual span | Every email send (invites, password resets) |
| **Logs** | Correlation | `trace_id` and `span_id` injected into JSON log output |

### Not traced (by design)

| Area | Reason |
|---|---|
| Secret manager calls | Startup-only; runs before TracerProvider is initialized |
| JWT signing / verification | CPU-only, sub-millisecond |
| Template rendering | CPU-only |
| Database migrations | Startup-only |

---

## Span Reference

### HTTP spans

Created automatically by `otelhttp` for every inbound REST request.

| Attribute | Description | Example |
|---|---|---|
| `http.method` | HTTP method | `POST` |
| `http.route` | Matched route pattern | `/v1/login` |
| `http.status_code` | Response status | `200` |
| `http.target` | Full request path + query | `/v1/users?page=2` |
| `http.scheme` | Protocol | `http` |
| `net.host.name` | Server hostname | `auth.yourdomain.com` |

### gRPC spans

Created automatically by `otelgrpc` for every inbound RPC.

| Attribute | Description | Example |
|---|---|---|
| `rpc.system` | RPC framework | `grpc` |
| `rpc.service` | Service name | `maintainerd.auth.v1.AuthService` |
| `rpc.method` | Method name | `ValidateToken` |
| `rpc.grpc.status_code` | gRPC status | `0` |

### Database spans

Created automatically by `otelgorm` for every GORM operation.

| Attribute | Description | Example |
|---|---|---|
| `db.system` | Database type | `postgresql` |
| `db.statement` | SQL query | `SELECT * FROM md_users WHERE …` |
| `db.operation` | Operation type | `SELECT` |
| `db.sql.table` | Table name | `md_users` |

### Redis spans

Created automatically by `redisotel` for every Redis command.

| Attribute | Description | Example |
|---|---|---|
| `db.system` | Store type | `redis` |
| `db.statement` | Command + key | `GET session:abc` |
| `db.operation` | Command | `GET` |
| `net.peer.name` | Redis host | `redis-db` |
| `net.peer.port` | Redis port | `6379` |

### SMTP spans

Created manually in `internal/email/email.go` for every email send.

| Attribute | Description | Example |
|---|---|---|
| `smtp.host` | SMTP server | `smtp.sendgrid.net` |
| `smtp.port` | SMTP port | `587` |
| `email.to` | Recipient | `user@example.com` |
| `email.subject` | Subject line | `You've been invited` |

**Span name:** `smtp.send`
**Error recording:** On failure, `span.RecordError()` is called and status is set to `Error`.

---

## Log Correlation

Every structured JSON log line includes:

```json
{
  "request_id": "a1b2c3d4-...",
  "trace_id": "4bf92f3577b34da6a3ce929d0e0e4736",
  "span_id": "00f067aa0ba902b7",
  "method": "POST",
  "path": "/v1/login",
  "status": 200,
  "latency_ms": 42
}
```

Use `trace_id` to jump from a log line directly to the corresponding trace in your tracing backend. When OTel is disabled, `trace_id` and `span_id` are omitted.

---

## Collector Architecture

Deploy an [OpenTelemetry Collector](https://opentelemetry.io/docs/collector/) between the service and your tracing backend:

```
┌──────────────────┐     OTLP/gRPC      ┌──────────────────┐     Export      ┌─────────────┐
│  Maintainerd Auth │ ──────────────────▶│  OTel Collector   │ ─────────────▶│   Backend    │
│  (OTLP exporter)  │     :4317         │  (sidecar/daemon) │               │ (Tempo/Jaeger│
└──────────────────┘                     └──────────────────┘               │  /Honeycomb) │
                                                                            └─────────────┘
```

### Sidecar pattern (Kubernetes)

Add the collector as a sidecar container in the same pod. The service connects to `localhost:4317`.

```env
OTEL_EXPORTER_OTLP_ENDPOINT="localhost:4317"
```

### DaemonSet pattern (Kubernetes)

Run one collector per node. The service connects to the node-local collector via the downward API.

```env
OTEL_EXPORTER_OTLP_ENDPOINT="${NODE_IP}:4317"
```

### Gateway pattern (centralized)

Route all traffic through a central collector cluster with load balancing and retry.

```env
OTEL_EXPORTER_OTLP_ENDPOINT="otel-gateway.internal:4317"
```

---

## Sampling Strategy

In high-traffic production environments, tracing 100% of requests is not practical. Configure sampling using standard OTel environment variables:

| Variable | Recommended Value | Description |
|---|---|---|
| `OTEL_TRACES_SAMPLER` | `parentbased_traceidratio` | Respects parent context; samples a percentage of root spans |
| `OTEL_TRACES_SAMPLER_ARG` | `0.1` | Sample 10% of traces (adjust based on traffic volume) |

**Example:**

```env
OTEL_TRACES_SAMPLER=parentbased_traceidratio
OTEL_TRACES_SAMPLER_ARG=0.1
```

> Start with 10–25% sampling and adjust based on storage costs and trace volume. Use tail-based sampling in the Collector for error-biased retention.

---

## Backend Recommendations

| Backend | Type | Best For |
|---|---|---|
| [Grafana Tempo](https://grafana.com/oss/tempo/) | Open-source | Cost-effective, pairs with Grafana dashboards |
| [Jaeger](https://www.jaegertracing.io/) | Open-source | Simple self-hosted setup |
| [Honeycomb](https://www.honeycomb.io/) | Managed | Powerful query engine, excellent debugging UX |
| [Datadog](https://www.datadoghq.com/) | Managed | Full-stack APM with logs + traces + metrics |
| [AWS X-Ray](https://aws.amazon.com/xray/) | Managed | Native AWS ECS/Lambda integration |
| [GCP Cloud Trace](https://cloud.google.com/trace) | Managed | Native GKE/Cloud Run integration |

---

## Dashboards & Alerts

### Recommended alert rules

| Alert | Condition | Severity |
|---|---|---|
| High HTTP error rate | `http.status_code >= 500` rate > 1% over 5 min | Critical |
| Slow DB queries | `otelgorm` span duration > 1 s | Warning |
| SMTP send failures | `smtp.send` span status = Error | High |
| Redis latency spike | `redisotel` span duration > 100 ms | Warning |
| gRPC error rate | `rpc.grpc.status_code != 0` rate > 0.5% | High |

### Key metrics to derive from traces

| Metric | How |
|---|---|
| **P50/P95/P99 latency** per endpoint | Histogram of HTTP span durations grouped by `http.route` |
| **Error rate** per endpoint | Count of `http.status_code >= 500` / total spans |
| **DB query count** per request | Count of child `otelgorm` spans per trace |
| **Email send success rate** | `smtp.send` spans with `Ok` status / total |
| **Downstream dependency health** | Span error rates for DB, Redis, SMTP |

---

## Troubleshooting

### No traces appearing

1. Verify `OTEL_ENABLED=true` is set.
2. Check that `OTEL_EXPORTER_OTLP_ENDPOINT` is reachable from the pod/container.
3. Look for `"OpenTelemetry tracing enabled"` in startup logs.
4. Ensure the Collector is running and its OTLP gRPC receiver is on port 4317.

### Traces are incomplete (missing child spans)

1. Verify `context.Context` is being passed through the call chain — child spans need the parent's context.
2. Check that middleware order is correct: `otelhttp` must wrap the router before any handler runs.

### High memory usage from tracing

1. Lower the sampling rate via `OTEL_TRACES_SAMPLER_ARG`.
2. Check that `BatchSpanProcessor` is exporting successfully (failed exports cause span buffering).
3. Reduce `OTEL_BSP_MAX_QUEUE_SIZE` (default: 2048) if needed.

### Spans have no `trace_id` in logs

1. Confirm `LoggingMiddleware` is registered **after** `otelhttp` in the middleware chain.
2. The `otelhttp` handler creates the root span; `LoggingMiddleware` then reads it from context.
