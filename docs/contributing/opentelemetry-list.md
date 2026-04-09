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

- [ ] `Get`
- [ ] `GetByUUID`
- [ ] `GetConfigByUUID`
- [ ] `Create`
- [ ] `Update`
- [ ] `SetStatusByUUID`
- [ ] `Delete`
- [ ] `ValidateAPIKey`
- [ ] `GetAPIKeyAPIs`
- [ ] `AddAPIKeyAPIs`
- [ ] `RemoveAPIKeyAPI`
- [ ] `GetAPIKeyAPIPermissions`
- [ ] `AddAPIKeyAPIPermissions`
- [ ] `RemoveAPIKeyAPIPermission`

### service/client.go

- [ ] `Get`
- [ ] `GetByUUID`
- [ ] `GetSecretByUUID`
- [ ] `GetConfigByUUID`
- [ ] `Create`
- [ ] `Update`
- [ ] `SetStatusByUUID`
- [ ] `DeleteByUUID`
- [ ] `CreateURI`
- [ ] `UpdateURI`
- [ ] `DeleteURI`
- [ ] `GetClientAPIs`
- [ ] `AddClientAPIs`
- [ ] `RemoveClientAPI`
- [ ] `GetClientAPIPermissions`
- [ ] `AddClientAPIPermissions`
- [ ] `RemoveClientAPIPermission`

### service/email_template.go

- [ ] `GetAll`
- [ ] `GetByUUID`
- [ ] `Create`
- [ ] `Update`
- [ ] `UpdateStatus`
- [ ] `Delete`

### service/forgot_password.go

- [ ] `SendPasswordResetEmail`

### service/identity_provider.go

- [ ] `Get`
- [ ] `GetByUUID`
- [ ] `Create`
- [ ] `Update`
- [ ] `SetStatusByUUID`
- [ ] `DeleteByUUID`

### service/invite.go

- [ ] `SendInvite`

### service/ip_restriction_rule.go

- [ ] `GetAll`
- [ ] `GetByUUID`
- [ ] `Create`
- [ ] `Update`
- [ ] `UpdateStatus`
- [ ] `Delete`

### service/login.go

- [ ] `LoginPublic`
- [ ] `Login`
- [ ] `GetUserByEmail`

### service/login_template.go

- [ ] `GetAll`
- [ ] `GetByUUID`
- [ ] `Create`
- [ ] `Update`
- [ ] `UpdateStatus`
- [ ] `Delete`

### service/permission.go

- [ ] `Get`
- [ ] `GetByUUID`
- [ ] `Create`
- [ ] `Update`
- [ ] `SetActiveStatusByUUID`
- [ ] `SetStatus`
- [ ] `DeleteByUUID`

### service/policy.go

- [ ] `Get`
- [ ] `GetByUUID`
- [ ] `GetServicesByPolicyUUID`
- [ ] `Create`
- [ ] `Update`
- [ ] `SetStatusByUUID`
- [ ] `DeleteByUUID`

### service/profile.go

- [ ] `CreateOrUpdateProfile`
- [ ] `CreateOrUpdateSpecificProfile`
- [ ] `GetByUUID`
- [ ] `GetByUserUUID`
- [ ] `GetAll`
- [ ] `SetDefaultProfile`
- [ ] `DeleteByUUID`

### service/register.go

- [ ] `RegisterPublic`
- [ ] `RegisterInvitePublic`
- [ ] `Register`
- [ ] `RegisterInvite`

### service/reset_password.go

- [ ] `ResetPassword`

### service/role.go

- [ ] `Get`
- [ ] `GetByUUID`
- [ ] `GetRolePermissions`
- [ ] `Create`
- [ ] `Update`
- [ ] `SetStatusByUUID`
- [ ] `DeleteByUUID`
- [ ] `AddRolePermissions`
- [ ] `RemoveRolePermissions`

### service/security_setting.go

- [ ] `GetByTenantID`
- [ ] `GetGeneralConfig`
- [ ] `GetPasswordConfig`
- [ ] `GetSessionConfig`
- [ ] `GetThreatConfig`
- [ ] `GetIPConfig`
- [ ] `UpdateGeneralConfig`
- [ ] `UpdatePasswordConfig`
- [ ] `UpdateSessionConfig`
- [ ] `UpdateThreatConfig`
- [ ] `UpdateIPConfig`

### service/service.go

- [ ] `Get`
- [ ] `GetByUUID`
- [ ] `Create`
- [ ] `Update`
- [ ] `SetStatusByUUID`
- [ ] `DeleteByUUID`
- [ ] `AssignPolicy`
- [ ] `RemovePolicy`

### service/setup.go

- [ ] `GetSetupStatus`
- [ ] `CreateTenant`
- [ ] `CreateAdmin`
- [ ] `CreateProfile`

### service/signup_flow.go

- [ ] `GetAll`
- [ ] `GetByUUID`
- [ ] `Create`
- [ ] `Update`
- [ ] `UpdateStatus`
- [ ] `Delete`
- [ ] `AssignRoles`
- [ ] `GetRoles`
- [ ] `RemoveRole`

### service/sms_template.go

- [ ] `GetAll`
- [ ] `GetByUUID`
- [ ] `Create`
- [ ] `Update`
- [ ] `UpdateStatus`
- [ ] `Delete`

### service/tenant.go

- [ ] `Get`
- [ ] `GetByUUID`
- [ ] `GetDefault`
- [ ] `GetByIdentifier`
- [ ] `Create`
- [ ] `Update`
- [ ] `SetStatusByUUID`
- [ ] `SetActivePublicByUUID`
- [ ] `SetDefaultStatusByUUID`
- [ ] `DeleteByUUID`

### service/tenant_access.go

- [ ] `ValidateTenantAccess`
- [ ] `ValidateTenantAccessByID`

### service/tenant_member.go

- [ ] `Create`
- [ ] `CreateByUserUUID`
- [ ] `GetByUUID`
- [ ] `GetByTenantAndUser`
- [ ] `ListByTenant`
- [ ] `ListByUser`
- [ ] `UpdateRole`
- [ ] `DeleteByUUID`
- [ ] `IsUserInTenant`

### service/user.go

- [ ] `Get`
- [ ] `GetByUUID`
- [ ] `Create`
- [ ] `Update`
- [ ] `SetStatus`
- [ ] `VerifyEmail`
- [ ] `VerifyPhone`
- [ ] `CompleteAccount`
- [ ] `DeleteByUUID`
- [ ] `AssignUserRoles`
- [ ] `RemoveUserRole`
- [ ] `GetUserRoles`
- [ ] `GetUserIdentities`
- [ ] `FindBySubAndClientID`

### service/user_setting.go

- [ ] `CreateOrUpdateUserSetting`
- [ ] `GetByUUID`
- [ ] `GetByUserUUID`
- [ ] `DeleteByUUID`

---

## JWT — `internal/jwt/` (Medium Priority)

> Critical auth path. Helps diagnose token-related latency.

- [ ] `GenerateAccessToken`
- [ ] `GenerateIDToken`
- [ ] `GenerateRefreshToken`
- [ ] `ValidateToken`

---

## Security — `internal/security/` (Medium Priority)

> Only I/O-bound or intentionally slow functions. CPU-only validators excluded.

- [ ] `HashPassword` — bcrypt, intentionally slow (100-300ms)
- [ ] `CheckRateLimit` — hits Redis
- [ ] `RecordFailedAttempt` — hits Redis
- [ ] `ResetFailedAttempts` — hits Redis
- [ ] `ValidateSessionLimit` — hits Redis

---

## Cache — `internal/cache/` (Low Priority)

> Redis commands are already auto-traced by `redisotel`. These wrapping spans add
> visibility into serialization/deserialization and cache logic.

- [ ] `Cache.GetUserContext`
- [ ] `Cache.SetUserContext`
- [ ] `Cache.InvalidateUser`
- [ ] `Cache.InvalidateUserAll`
- [ ] `Cache.InvalidateAllUsers`

---

## Summary

| Layer | Total | Done | Remaining | Priority |
|-------|:-----:|:----:|:---------:|----------|
| Infrastructure (auto) | 7 | 7 | 0 | — |
| Logging & Correlation | 2 | 2 | 0 | — |
| Email (manual) | 1 | 1 | 0 | — |
| Broken context.Background() | 4 | 4 | 0 | High |
| Service Layer | 172 | 7 | 165 | High |
| JWT | 4 | 0 | 4 | Medium |
| Security (I/O funcs) | 5 | 0 | 5 | Medium |
| Cache | 5 | 0 | 5 | Low |
| **Total** | **200** | **21** | **179** | |