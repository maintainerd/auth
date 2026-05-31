# Architecture Flowchart

This document shows how the application connects from the executable entrypoint
down into domain packages and platform infrastructure.

Use this as a navigation map before editing cross-package behavior. The diagram
shows ownership boundaries, not every function call.

## End-to-End Flow

```mermaid
flowchart TD
  main["cmd/server/main.go<br/>process entrypoint"]
  run["cmd/server/bootstrap.go<br/>run(ctx) startup sequence"]

  main --> run

  subgraph cmd["cmd/server: executable bootstrap"]
    run --> bootlog["logging.go<br/>bootstrap JSON logger"]
    run --> config["platform/config<br/>load env and secrets"]
    config --> cfglog["logging.go<br/>configured PII-redacting logger"]
    run --> telemetry["telemetry.go<br/>OpenTelemetry tracing + metrics"]
    run --> jwtkeys["platform/jwt<br/>RSA key parsing + token config"]
    run --> db["platform/config.InitDB<br/>PostgreSQL connection"]
    run --> redis["platform/config.NewRedisClient<br/>Redis connection"]
    redis --> ratelimit["platform/security<br/>Redis-backed rate limiter"]
    db --> migrations["platform/runner<br/>database migrations"]
    run --> appnew["internal/app.NewApp<br/>composition root"]
    run --> workers["workers.go<br/>retention + gRPC workers"]
    run --> reststart["server.StartRESTServer<br/>foreground REST lifecycle"]
  end

  subgraph app["internal/app: dependency composition"]
    appnew --> repos["repositories.go<br/>construct domain repositories"]
    repos --> services["services.go<br/>construct domain services"]
    services --> adapters["adapters_*<br/>cross-domain interface adapters"]
    services --> appbundle["App<br/>service bundle + DB + Redis + cache"]
    appbundle --> serverapp["application.go<br/>server.Application adapter"]
  end

  subgraph server["internal/server: transport runtime"]
    serverapp --> handlers["handlers.go<br/>construct REST handlers"]
    serverapp --> grpc["grpc.go<br/>SeederService on :50051"]
    handlers --> router["router.go<br/>internal/public routers"]
    router --> middleware["platform/middleware<br/>security, logging, limits, auth context"]
    router --> health["health.go<br/>/health + /ready"]
    router --> openapi["openapi.go<br/>/openapi.json"]
    router --> restports["rest.go<br/>REST :8080 internal + :8081 public"]
  end

  subgraph domains["internal/<domain>: feature/domain packages"]
    handler["handler_<name>.go<br/>parse request + call service"]
    routes["routes.go<br/>mount endpoints"]
    dto["types.go<br/>request/response DTOs"]
    validation["validation_<name>.go<br/>DTO validation"]
    service["service_<name>.go<br/>business rules + transactions"]
    deps["deps.go<br/>consumer-side dependency contracts"]
    repo["repository_<name>.go<br/>persistence interface + implementation"]
    model["model_<name>.go<br/>GORM model + hooks"]

    routes --> handler
    handler --> dto
    handler --> validation
    handler --> service
    service --> deps
    service --> repo
    repo --> model
  end

  subgraph platform["internal/platform: reusable infrastructure"]
    pdb["database<br/>GORM base repository + migrations"]
    pcache["cache<br/>Redis auth/session cache"]
    pjwt["jwt / dpop / signedurl<br/>token and proof utilities"]
    psec["security / middleware<br/>headers, rate limits, auth middleware"]
    pobs["logging / telemetry<br/>logs, traces, metrics"]
    pio["email / sms / templates<br/>notification infrastructure"]
    putil["apperror / response / pagination / ptr / valid / jsonutil / crypto<br/>generic helpers"]
  end

  restports --> router
  router --> routes
  middleware --> pcache
  middleware --> pjwt
  middleware --> psec
  handlers --> handler
  services --> service
  repo --> pdb
  service --> pcache
  service --> pjwt
  service --> pio
  handler --> putil
  service --> putil
  model --> pdb
  telemetry --> pobs
  bootlog --> pobs
```

## Startup Flow

```mermaid
sequenceDiagram
  participant Main as cmd/server main
  participant Bootstrap as run(ctx)
  participant Platform as internal/platform
  participant App as internal/app
  participant Server as internal/server
  participant Domain as internal/<domain>

  Main->>Bootstrap: run(context.Background())
  Bootstrap->>Platform: init bootstrap logger
  Bootstrap->>Platform: config.Init()
  Bootstrap->>Platform: init configured logger
  Bootstrap->>Platform: telemetry.Init() + InitMetrics()
  Bootstrap->>Platform: jwt.InitJWTKeys()
  Bootstrap->>Platform: InitDB() + NewRedisClient()
  Bootstrap->>Platform: security.InitRateLimiter(redis)
  Bootstrap->>Platform: runner.RunMigrations(db)
  Bootstrap->>App: app.NewApp(db, redis)
  App->>Domain: construct repositories
  App->>Domain: construct services with repositories/adapters
  App-->>Bootstrap: App service bundle
  Bootstrap->>Server: application.ServerApplication()
  Bootstrap->>Server: StartGRPCServer(ctx, app) in background
  Bootstrap->>Domain: authevent.StartRetentionRunner(ctx, service)
  Bootstrap->>Server: StartRESTServer(app)
  Server->>Domain: construct handlers and mount routes
```

## Request Flow

```mermaid
flowchart LR
  client["HTTP client"] --> port{"REST port"}
  port --> internal[":8080 internal API"]
  port --> public[":8081 public API"]

  internal --> common["common middleware<br/>recover, security headers, request ID, logging,<br/>size limit, timeout, CORS, JSON content-type"]
  public --> common
  public --> iplimit["public IP rate limit"]

  common --> auth["route auth middleware<br/>JWT/cookie validation, user context, permissions"]
  iplimit --> auth
  auth --> route["domain routes.go"]
  route --> handler["handler_<name>.go"]
  handler --> validation["types.go + validation_<name>.go"]
  validation --> service["service_<name>.go"]
  service --> repo["repository_<name>.go"]
  repo --> model["model_<name>.go"]
  model --> pg[("PostgreSQL")]

  auth --> redis[("Redis cache/session/rate limit")]
  service --> redis
  service --> events["authevent service"]
  events --> webhook["webhook dispatcher"]
```

## Domain Package Shape

Most feature packages use the role-first structure below. A package should only
include files for roles it actually owns.

```text
internal/<domain>/
  model_<name>.go          database model, table names, hooks
  repository_<name>.go     persistence contract and implementation
  service_<name>.go        business behavior and transactions
  handler_<name>.go        HTTP handler logic
  validation_<name>.go     DTO validation
  types.go                 API request/response DTOs
  deps.go                  consumer-side upstream contracts
  foundation.go            local platform aliases/wrappers
  routes.go                route mounting
```

Current top-level domain packages:

```text
authevent
authn
branding
client
iam
idp
invite
mfa
notifier
oauth
secpolicy
setup
tenant
user
webhook
```

## Platform Boundary

`internal/platform` is for reusable infrastructure that does not import domain
packages. Domain packages may import platform helpers; platform must not import
domains.

```mermaid
flowchart TD
  domain["internal/<domain>"] --> platform["internal/platform/*"]
  app["internal/app"] --> domain
  server["internal/server"] --> domain
  cmd["cmd/server"] --> app
  cmd --> server
  platform -. forbidden .-> domain
```

Common platform responsibilities:

- `config`, `database`, `runner`: environment, storage, migrations.
- `cache`, `security`, `middleware`: Redis-backed cache, rate limits, HTTP middleware.
- `jwt`, `dpop`, `signedurl`, `crypto`: token, proof, and cryptographic helpers.
- `logging`, `telemetry`: structured logs, traces, metrics.
- `email`, `sms`, `templates`: notification infrastructure.
- `apperror`, `response`, `pagination`, `ptr`, `valid`, `jsonutil`: generic helpers.
