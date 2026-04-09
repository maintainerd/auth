# OpenTelemetry Instrumentation Checklist

This file tracks OpenTelemetry observability coverage across the codebase.
Only items that provide real value are listed — CPU-only functions, auto-instrumented
layers (handlers, repositories), and thin middleware are excluded to reduce noise.

- [x] = Instrumented
- [ ] = Not yet instrumented

---

## Infrastructure (Auto-Instrumentation)

- [x] HTTP REST server — `otelhttp.NewHandler()` wraps both `:8080` and `:8081`
- [x] gRPC server — `otelgrpc.NewServerHandler()` registered as StatsHandler
- [x] PostgreSQL — `otelgorm` plugin on `*gorm.DB`
- [x] Redis — `redisotel.InstrumentTracing()` on `*redis.Client`
- [x] OTEL TracerProvider initialization — `telemetry.Init()` in `main.go`
- [x] Noop fallback — `noop.NewTracerProvider()` when `OTEL_ENABLED != true`
- [x] Graceful shutdown — deferred `otelShutdown()` with 5s timeout

## Logging & Correlation

- [x] Trace ID / Span ID extraction — `telemetry.TraceIDFromContext()`
- [x] Structured log correlation — `LoggingMiddleware` seeds logger with `trace_id`, `span_id`

## Email (Manual Instrumentation)

- [x] `email.SendEmail` — `otel.Tracer("email").Start(ctx, "smtp.send")`

---

## Broken Trace Propagation (High Priority)

> These use `context.Background()` instead of request context, creating orphaned spans.

- [x] `service/invite.go` — email sending uses `context.Background()`
- [x] `service/forgot_password.go` — email sending uses `context.Background()`
- [x] `service/role.go` (×5) — cache invalidation uses `context.Background()`
- [x] `service/user.go` — cache invalidation uses `context.Background()`

---

## Service Layer — `internal/service/` (High Priority)

> The biggest observability gap. Service methods orchestrate business logic between the
> HTTP root span and the auto-traced DB/Redis leaf spans. Without these, the middle
> of every trace is a black box.

### service/api.go

- [x] `Get`
- [x] `GetByUUID`
- [x] `GetServiceIDByUUID`
- [x] `Create`
- [x] `Update`
- [x] `SetStatusByUUID`
- [x] `DeleteByUUID`

### service/api_key.go

- [x] `Get`
- [x] `GetByUUID`
- [x] `GetConfigByUUID`
- [x] `Create`
- [x] `Update`
- [x] `SetStatusByUUID`
- [x] `Delete`
- [x] `ValidateAPIKey`
- [x] `GetAPIKeyAPIs`
- [x] `AddAPIKeyAPIs`
- [x] `RemoveAPIKeyAPI`
- [x] `GetAPIKeyAPIPermissions`
- [x] `AddAPIKeyAPIPermissions`
- [x] `RemoveAPIKeyAPIPermission`

### service/client.go

- [x] `Get`
- [x] `GetByUUID`
- [x] `GetSecretByUUID`
- [x] `GetConfigByUUID`
- [x] `Create`
- [x] `Update`
- [x] `SetStatusByUUID`
- [x] `DeleteByUUID`
- [x] `CreateURI`
- [x] `UpdateURI`
- [x] `DeleteURI`
- [x] `GetClientAPIs`
- [x] `AddClientAPIs`
- [x] `RemoveClientAPI`
- [x] `GetClientAPIPermissions`
- [x] `AddClientAPIPermissions`
- [x] `RemoveClientAPIPermission`

### service/email_template.go

- [x] `GetAll`
- [x] `GetByUUID`
- [x] `Create`
- [x] `Update`
- [x] `UpdateStatus`
- [x] `Delete`

### service/forgot_password.go

- [x] `SendPasswordResetEmail`

### service/identity_provider.go

- [x] `Get`
- [x] `GetByUUID`
- [x] `Create`
- [x] `Update`
- [x] `SetStatusByUUID`
- [x] `DeleteByUUID`

### service/invite.go

- [x] `SendInvite`

### service/ip_restriction_rule.go

- [x] `GetAll`
- [x] `GetByUUID`
- [x] `Create`
- [x] `Update`
- [x] `UpdateStatus`
- [x] `Delete`

### service/login.go

- [x] `LoginPublic`
- [x] `Login`
- [x] `GetUserByEmail`

### service/login_template.go

- [x] `GetAll`
- [x] `GetByUUID`
- [x] `Create`
- [x] `Update`
- [x] `UpdateStatus`
- [x] `Delete`

### service/permission.go

- [x] `Get`
- [x] `GetByUUID`
- [x] `Create`
- [x] `Update`
- [x] `SetActiveStatusByUUID`
- [x] `SetStatus`
- [x] `DeleteByUUID`

### service/policy.go

- [x] `Get`
- [x] `GetByUUID`
- [x] `GetServicesByPolicyUUID`
- [x] `Create`
- [x] `Update`
- [x] `SetStatusByUUID`
- [x] `DeleteByUUID`

### service/profile.go

- [x] `CreateOrUpdateProfile`
- [x] `CreateOrUpdateSpecificProfile`
- [x] `GetByUUID`
- [x] `GetByUserUUID`
- [x] `GetAll`
- [x] `SetDefaultProfile`
- [x] `DeleteByUUID`

### service/register.go

- [x] `RegisterPublic`
- [x] `RegisterInvitePublic`
- [x] `Register`
- [x] `RegisterInvite`

### service/reset_password.go

- [x] `ResetPassword`

### service/role.go

- [x] `Get`
- [x] `GetByUUID`
- [x] `GetRolePermissions`
- [x] `Create`
- [x] `Update`
- [x] `SetStatusByUUID`
- [x] `DeleteByUUID`
- [x] `AddRolePermissions`
- [x] `RemoveRolePermissions`

### service/security_setting.go

- [x] `GetByTenantID`
- [x] `GetGeneralConfig`
- [x] `GetPasswordConfig`
- [x] `GetSessionConfig`
- [x] `GetThreatConfig`
- [x] `GetIPConfig`
- [x] `UpdateGeneralConfig`
- [x] `UpdatePasswordConfig`
- [x] `UpdateSessionConfig`
- [x] `UpdateThreatConfig`
- [x] `UpdateIPConfig`

### service/service.go

- [x] `Get`
- [x] `GetByUUID`
- [x] `Create`
- [x] `Update`
- [x] `SetStatusByUUID`
- [x] `DeleteByUUID`
- [x] `AssignPolicy`
- [x] `RemovePolicy`

### service/setup.go

- [x] `GetSetupStatus`
- [x] `CreateTenant`
- [x] `CreateAdmin`
- [x] `CreateProfile`

### service/signup_flow.go

- [x] `GetAll`
- [x] `GetByUUID`
- [x] `Create`
- [x] `Update`
- [x] `UpdateStatus`
- [x] `Delete`
- [x] `AssignRoles`
- [x] `GetRoles`
- [x] `RemoveRole`

### service/sms_template.go

- [x] `GetAll`
- [x] `GetByUUID`
- [x] `Create`
- [x] `Update`
- [x] `UpdateStatus`
- [x] `Delete`

### service/tenant.go

- [x] `Get`
- [x] `GetByUUID`
- [x] `GetDefault`
- [x] `GetByIdentifier`
- [x] `Create`
- [x] `Update`
- [x] `SetStatusByUUID`
- [x] `SetActivePublicByUUID`
- [x] `SetDefaultStatusByUUID`
- [x] `DeleteByUUID`

### service/tenant_access.go

- [x] `ValidateTenantAccess`
- [x] `ValidateTenantAccessByID`

### service/tenant_member.go

- [x] `Create`
- [x] `CreateByUserUUID`
- [x] `GetByUUID`
- [x] `GetByTenantAndUser`
- [x] `ListByTenant`
- [x] `ListByUser`
- [x] `UpdateRole`
- [x] `DeleteByUUID`
- [x] `IsUserInTenant`

### service/user.go

- [x] `Get`
- [x] `GetByUUID`
- [x] `Create`
- [x] `Update`
- [x] `SetStatus`
- [x] `VerifyEmail`
- [x] `VerifyPhone`
- [x] `CompleteAccount`
- [x] `DeleteByUUID`
- [x] `AssignUserRoles`
- [x] `RemoveUserRole`
- [x] `GetUserRoles`
- [x] `GetUserIdentities`
- [x] `FindBySubAndClientID`

### service/user_setting.go

- [x] `CreateOrUpdateUserSetting`
- [x] `GetByUUID`
- [x] `GetByUserUUID`
- [x] `DeleteByUUID`

---

## JWT — `internal/jwt/` (Medium Priority)

> Critical auth path. Helps diagnose token-related latency.

- [x] `GenerateAccessToken`
- [x] `GenerateIDToken`
- [x] `GenerateRefreshToken`
- [x] `ValidateToken`

---

## Security — `internal/security/` (Medium Priority)

> Only I/O-bound or intentionally slow functions. CPU-only validators excluded.

- [x] `HashPassword` — bcrypt, intentionally slow (100-300ms)
- [x] `CheckRateLimit` — hits Redis
- [x] `RecordFailedAttempt` — hits Redis
- [x] `ResetFailedAttempts` — hits Redis
- [x] `ValidateSessionLimit` — hits Redis

---

## Cache — `internal/cache/` (Low Priority)

> Redis commands are already auto-traced by `redisotel`. These wrapping spans add
> visibility into serialization/deserialization and cache logic.

- [x] `Cache.GetUserContext`
- [x] `Cache.SetUserContext`
- [x] `Cache.InvalidateUser`
- [x] `Cache.InvalidateUserAll`
- [x] `Cache.InvalidateAllUsers`

---

## Summary

| Layer | Total | Done | Remaining | Priority |
|-------|:-----:|:----:|:---------:|----------|
| Infrastructure (auto) | 7 | 7 | 0 | — |
| Logging & Correlation | 2 | 2 | 0 | — |
| Email (manual) | 1 | 1 | 0 | — |
| Broken context.Background() | 4 | 4 | 0 | High |
| Service Layer | 172 | 81 | 91 | High |
| JWT | 4 | 0 | 4 | Medium |
| Security (I/O funcs) | 5 | 0 | 5 | Medium |
| Cache | 5 | 0 | 5 | Low |
| **Total** | **200** | **95** | **105** | |