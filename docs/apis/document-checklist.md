# API Documentation Checklist

Tracks documentation status for every REST endpoint group.  
One file per handler / feature area. Both ports are covered.

Legend:
- `[x]` Documented
- `[ ]` Not yet documented

---

## Port 8080 — Internal / Management

### Setup
- [x] `setup/setup.md` — `GET /api/v1/setup/status`, `POST /api/v1/setup/create_tenant`, `POST /api/v1/setup/create_admin`, `POST /api/v1/setup/create_profile`

### Authentication (Internal)
- [x] `auth/login.md` — `POST /api/v1/login`, `POST /api/v1/logout`
- [x] `auth/register.md` — `POST /api/v1/register`, `POST /api/v1/register/invite`
- [x] `auth/forgot-password.md` — `POST /api/v1/forgot-password`
- [x] `auth/reset-password.md` — `POST /api/v1/reset-password`
- [x] `auth/email-verification.md` — `POST /api/v1/email-verification/send`, `POST /api/v1/email-verification/verify`
- [x] `auth/magic-link.md` — `POST /api/v1/magic-link/send`, `POST /api/v1/magic-link/verify`

### Profile & User Settings (Shared — also on port 8081)
- [x] `profile/profile.md` — `GET/POST/PUT/DELETE /api/v1/profile`, `GET/POST /api/v1/profiles`, `GET/PUT/PATCH/DELETE /api/v1/profiles/{profile_uuid}`
- [x] `profile/user-settings.md` — `GET/POST/DELETE /api/v1/user-settings`

### Tenant Management
- [x] `tenant/tenant.md` — `GET/POST/PUT/DELETE /api/v1/tenants`, `GET /api/v1/tenants/{tenant_uuid}`, status/public toggles
- [x] `tenant/tenant-members.md` — `GET/POST /api/v1/tenants/{tenant_uuid}/members`, `PATCH/DELETE /api/v1/tenants/{tenant_uuid}/members/{tenant_member_uuid}`

### IAM — Services, APIs, Permissions, Policies
- [x] `iam/service.md` — `GET/POST/PUT/DELETE /api/v1/services`, `PUT .../status`, `POST/DELETE .../policies/{policy_uuid}`
- [x] `iam/api.md` — `GET/POST/PUT/DELETE /api/v1/apis`, `PUT .../status`
- [x] `iam/permission.md` — `GET/POST/PUT/DELETE /api/v1/permissions`, `PUT .../status`
- [x] `iam/policy.md` — `GET/POST/PUT/DELETE /api/v1/policies`, `PUT .../status`, `GET .../services`
- [x] `iam/authorization.md` — `GET /api/v1/services/me/policy-bundle`, `POST /api/v1/authorize/`

### Identity Providers
- [x] `idp/identity-provider.md` — `GET/POST/PUT/DELETE /api/v1/identity_providers`, `PUT .../status`

### Client Management
- [x] `client/client.md` — `GET/POST/PUT/DELETE /api/v1/clients`, `PUT .../status`
- [x] `client/client-uris.md` — `GET/POST/PUT/DELETE /api/v1/clients/{client_uuid}/uris`
- [x] `client/client-apis.md` — `GET/POST/DELETE /api/v1/clients/{client_uuid}/apis`, permissions sub-resource
- [x] `client/client-secret.md` — `GET /api/v1/clients/{client_uuid}/secret`, `GET .../config`

### Roles & Users
- [x] `rbac/role.md` — `GET/POST/PUT/DELETE /api/v1/roles`, `PUT .../status`, permissions sub-resource
- [x] `rbac/user.md` — `GET/POST/PUT/DELETE /api/v1/users`, status/verify/complete-account endpoints
- [x] `rbac/user-roles.md` — `GET/POST/DELETE /api/v1/users/{user_uuid}/roles`
- [x] `rbac/user-identities.md` — `GET /api/v1/users/{user_uuid}/identities`
- [x] `rbac/user-profiles.md` — Admin profile management under `/api/v1/users/{user_uuid}/profiles`

### Invites
- [x] `auth/invite.md` — `POST /api/v1/invite`

### API Keys
- [x] `api-keys/api-key.md` — `GET/POST/PUT/DELETE /api/v1/api_keys`, `PUT .../status`, `GET .../config`
- [x] `api-keys/api-key-apis.md` — `GET/POST/DELETE /api/v1/api_keys/{api_key_uuid}/apis`, permissions sub-resource

### Signup Flows
- [x] `signup/signup-flow.md` — `GET/POST/PUT/DELETE /api/v1/signup_flows`, status, roles sub-resource

### Security Settings
- [x] `settings/security-settings.md` — MFA, password, session, threat, lockout, registration, token config (`GET/PUT` per sub-resource)

### Tenant Settings & Config
- [x] `settings/tenant-settings.md` — Rate-limit, audit, maintenance, feature-flags config (`GET/PUT` per sub-resource)
- [x] `settings/email-config.md` — `GET/PUT /api/v1/email-config`
- [x] `settings/sms-config.md` — `GET/PUT /api/v1/sms-config`
- [x] `settings/branding.md` — `GET/PUT /api/v1/branding`

### Templates
- [x] `templates/email-template.md` — `GET/POST/PUT/DELETE /api/v1/email_templates`, `PATCH .../status`
- [x] `templates/sms-template.md` — `GET/POST/PUT/DELETE /api/v1/sms_templates`, `PATCH .../status`
- [x] `templates/login-template.md` — `GET/POST/PUT/DELETE /api/v1/login_templates`, `PATCH .../status`

### IP Restriction Rules
- [x] `settings/ip-restriction-rules.md` — `GET/POST/PUT/DELETE /api/v1/ip-restriction-rules`, `PATCH .../status`

### Webhooks
- [x] `webhooks/webhook-endpoint.md` — `GET/POST/PUT/DELETE /api/v1/webhook-endpoints`, `PATCH .../status`

### Audit
- [x] `audit/auth-events.md` — `GET /api/v1/auth-events`, `GET /api/v1/auth-events/count`, `GET /api/v1/auth-events/{uuid}`

### OAuth (Internal)
- [x] `oauth/introspect.md` — `POST /api/v1/oauth/introspect` (RFC 7662, management port only)

---

## Port 8081 — Public / Identity

### Discovery
- [x] `oauth/discovery.md` — `GET /.well-known/openid-configuration`, `GET /.well-known/oauth-authorization-server`, `GET /.well-known/jwks.json`

### Tenant (Public Read)
- [x] `tenant/tenant-public.md` — `GET /api/v1/tenant`, `GET /api/v1/tenant/{identifier}`

### Authentication (Public)
- [x] `auth/login-public.md` — `POST /api/v1/login` (with client_id/provider_id), `POST /api/v1/logout`
- [x] `auth/register-public.md` — `POST /api/v1/register`, `POST /api/v1/register/invite` (with client_id)
- [x] `auth/forgot-password.md` — `POST /api/v1/forgot-password` (both ports covered in same file)
- [x] `auth/reset-password.md` — `POST /api/v1/reset-password` (both ports covered in same file)
- [x] `auth/email-verification.md` — both ports covered in same file
- [x] `auth/magic-link.md` — both ports covered in same file

### OAuth 2.0 / OIDC
- [x] `oauth/oauth2.md` — General OAuth 2.0 overview (existing file)
- [x] `oauth/authorize.md` — `GET /api/v1/oauth/authorize` (RFC 6749 §4.1.1)
- [x] `oauth/token.md` — `POST /api/v1/oauth/token` (authorization_code, refresh_token, client_credentials)
- [x] `oauth/revoke.md` — `POST /api/v1/oauth/revoke` (RFC 7009)
- [x] `oauth/userinfo.md` — `GET /api/v1/oauth/userinfo` (OIDC Core §5.3)
- [x] `oauth/consent.md` — `GET /api/v1/oauth/consent/{challenge_id}`, `POST /api/v1/oauth/consent`
- [x] `oauth/consent-grants.md` — `GET /api/v1/oauth/consent/grants`, `DELETE /api/v1/oauth/consent/grants/{grant_uuid}`
- [x] `oauth/par.md` — `POST /api/v1/oauth/par` (RFC 9126)
- [x] `oauth/device.md` — `POST /api/v1/oauth/device_authorization`, `POST /api/v1/oauth/device`, `POST /api/v1/oauth/device/deny` (RFC 8628)
- [x] `oauth/token-exchange.md` — `POST /api/v1/oauth/token` with `grant_type=urn:ietf:params:oauth:grant-type:token-exchange` (RFC 8693)
- [x] `oauth/ciba.md` — `POST /api/v1/oauth/ciba`, `POST /api/v1/oauth/ciba/approve`, `POST /api/v1/oauth/ciba/deny` (CIBA Core)
- [x] `oauth/register.md` — `POST /api/v1/oauth/register` (RFC 7591 Dynamic Client Registration)
- [x] `oauth/end-session.md` — `GET/POST /api/v1/oauth/end_session` (OIDC Session Management 1.0)
- [x] `oauth/backchannel-logout.md` — `POST /api/v1/oauth/logout/backchannel` (OIDC Back-Channel Logout 1.0)

---

## Progress

| Area | Documented | Total |
|------|-----------|-------|
| Setup | 1 | 1 |
| Auth (internal) | 6 | 6 |
| Auth (public) | 6 | 6 |
| Profile & User Settings | 2 | 2 |
| Tenant | 3 | 3 |
| IAM (services/apis/permissions/policies/authorization) | 5 | 5 |
| Identity Providers | 1 | 1 |
| Client Management | 4 | 4 |
| Roles & Users | 5 | 5 |
| Invites | 1 | 1 |
| API Keys | 2 | 2 |
| Signup Flows | 1 | 1 |
| Security Settings | 1 | 1 |
| Tenant Settings & Config | 4 | 4 |
| Templates | 3 | 3 |
| IP Restriction Rules | 1 | 1 |
| Webhooks | 1 | 1 |
| Audit | 1 | 1 |
| OAuth (internal) | 1 | 1 |
| Discovery | 1 | 1 |
| OAuth (public) | 14 | 14 |
| **Total** | **64** | **64** |
