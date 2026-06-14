# Local observability with SigNoz (OpenTelemetry)

`maintainerd-auth` is **vendor-neutral**: it emits OpenTelemetry **traces and logs
over OTLP** (and Prometheus metrics on `/metrics`), configured entirely by the
standard `OTEL_*` environment variables. Nothing in the code is tied to SigNoz -
point the same OTLP endpoint at Datadog, Grafana/Tempo+Loki, Honeycomb, etc. and
it just works. SigNoz is simply a convenient open-source backend for **local**
testing.

```
auth --OTLP(:4317)--> SigNoz OTel Collector --> SigNoz (ClickHouse + UI :8080)
   traces + logs
```

## 1. Run the local stack

The root `docker-compose.yml` includes the pinned SigNoz stack under
`deploy/signoz/docker`, so your normal local command starts the app and SigNoz
together:

```bash
docker compose up --build -d
```

This publishes on your host:

- **OTLP gRPC** `:4317` and **OTLP HTTP** `:4318` (the collector)
- **SigNoz UI** http://localhost:8080
- **Prometheus** http://localhost:9090
- **Grafana** http://localhost:3001 (`admin` / `admin`)

> Pin to whatever SigNoz release you like; `v0.128.0` is known-good at time of
> writing. The only thing this project depends on is the OTLP collector port.

## 2. Turn on telemetry for auth

Set these in your `.env` (see `.env.example` for the block):

```dotenv
OTEL_ENABLED=true
OTEL_SERVICE_NAME=maintainerd-auth
OTEL_EXPORTER_OTLP_ENDPOINT=http://otel-collector:4317
OTEL_EXPORTER_OTLP_PROTOCOL=grpc
OTEL_EXPORTER_OTLP_INSECURE=true
```

`docker-compose.yml` also puts `auth` on SigNoz's Compose network, so
`otel-collector:4317` resolves directly to the bundled collector. No SigNoz
credentials are needed locally (plaintext OTLP). For a TLS/cloud backend later, set
`OTEL_EXPORTER_OTLP_INSECURE=false` and add `OTEL_EXPORTER_OTLP_HEADERS`
(e.g. an API key) - still no code change.

If you run the Go binary directly on your host instead of inside Compose, use
`OTEL_EXPORTER_OTLP_ENDPOINT=http://localhost:4317`.

## 3. Verify

In the SigNoz UI (http://localhost:8080):

- **Services / Traces** → you should see `maintainerd-auth`. Hit an endpoint
  (e.g. login/register) and watch spans appear; failures show as error spans.
- **Logs** → your `slog` output is shipped over OTLP and **correlated to traces**
  by `trace_id`/`span_id` (PII is redacted before export). This is how you
  confirm OpenTelemetry logging is working end-to-end.

In Prometheus (http://localhost:9090):

- **Status → Targets** should show `maintainerd-auth` as **UP**.
- Query `up{job="maintainerd-auth"}` and expect `1`.
- Query `go_goroutines{job="maintainerd-auth"}` to confirm app runtime metrics.

In Grafana (http://localhost:3001):

- Log in with `admin` / `admin`.
- The `Prometheus` datasource is provisioned automatically.
- Use **Explore** with the `Prometheus` datasource and query
  `up{job="maintainerd-auth"}` or `go_goroutines{job="maintainerd-auth"}`.

## Switching backends (e.g. Datadog)

Run the [OpenTelemetry Collector](https://opentelemetry.io/docs/collector/) (or
the Datadog Agent's OTLP intake), point `OTEL_EXPORTER_OTLP_ENDPOINT` at it, and
configure that collector's exporter for your vendor. The app is unchanged.

## What each signal uses

| Signal  | Transport               | Where to look |
|---------|-------------------------|---------------|
| Traces  | OTLP/gRPC (`:4317`)     | SigNoz → Traces |
| Logs    | OTLP/gRPC (`:4317`)     | SigNoz → Logs |
| Metrics | Prometheus pull (`auth:8082/metrics`) | Prometheus → Grafana |
