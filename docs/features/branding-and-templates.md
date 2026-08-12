# Branding & Templates

> Per-tenant white-label theming for the admin console and hosted identity UI, plus tenant-scoped, DB-backed email and SMS message templates.

| | |
|---|---|
| **Status** | Implemented (branding themes, email templates, SMS templates). Partial: email/SMS `Create`+`Delete` exist in the service/handler layer but are **not routed**; `login_templates` has DTOs/constants only — **no table, model, service, handler, or route**. |
| **Code** | `internal/branding` (models, services, handlers, routes, client resolver); `internal/platform/templates/emailtemplate` (seed HTML bodies); `internal/platform/email` + `internal/platform/sms` (`template.go` renderers); `internal/shared/branding_theme_defaults.go` (seeded theme palettes) |
| **Endpoints** | Control API (`:8080`, VPN-only, base `/api/v1`): `GET/POST /branding`, `PUT/PATCH/DELETE /branding/{branding_uuid}...`, `GET/PUT/PATCH /email_templates...`, `GET/PUT/PATCH /sms_templates...`. Public API (`:8081`, base `/api/v1`, no auth): `GET /public/branding`, `GET /public/branding/{branding_id}/logo` |
| **Storage** | Tables `branding` (migration `002_create_branding_table.go`), `email_templates` (`058_...`), `sms_templates` (`059_...`). Redis for logo + rendered-template caches |
| **Config** | No dedicated env vars. Redis (`RedisClient` in the `email`/`sms` packages) enables template caching when present; degrades to direct DB reads when nil. All theming/template data is **per-tenant rows**, seeded on tenant creation |

## Overview

This feature covers three related, tenant-scoped resources:

1. **Branding themes** — the visual identity (colors, fonts, component tokens, logo, favicon, legal/support URLs, hosted-login layout) applied to both the admin console (`auth-console`) and the hosted identity/login UI (`auth-identity`). A tenant owns **many** branding rows: three undeletable seeded system themes (`default`, `light`, `dark`) plus any number of custom ones, with **exactly one active** at a time.
2. **Email templates** — named, per-tenant HTML/plain-text transactional email bodies (invite, password reset, MFA, device approval, CIBA, email-change, …) rendered with Go `html/template`.
3. **SMS templates** — named, per-tenant SMS message bodies (login OTP, MFA enroll/step-up) rendered with Go `text/template`.

The former singleton `branding` model no longer matches the code: branding is now **multi-theme**, theme tokens live in a `metadata` JSONB (not dedicated color/font columns), and **custom CSS is unsupported** (`002_create_branding_table.go:15`).

## How it works

### Branding resolution (read path)

1. **Admin console** reads themes through the authenticated `GET /branding` list and edits them via the control-plane endpoints.
2. **Hosted login page** fetches non-sensitive branding from the public API. `GetPublic` → `brandingService.Get` → `brandingRepo.FindActive(tenantID)` (`repository_branding.go:101`):
   - `tenant_id = 0` → global system default (`is_system=true AND is_active=true`).
   - Otherwise the tenant's `is_active` row; on miss, falls back to that tenant's `is_system` row.
3. **Per-client theming** for OAuth login flows goes through `ClientBrandingResolver.ResolveForClient(brandingID, tenantID)` (`client_branding_resolver.go:56`), consumed by the OAuth connections service (`internal/oauth/service_connections.go:73`): resolve by the client's attached `branding_id` → the tenant's active theme → system fallback (`{}` if none). It returns colors + `layout` + legal URLs + raw `metadata` for the login UI.
4. **Layout** (`centered` / `full_page` / `split`) and logo-label preferences are **not columns** — they are read from `metadata` and defaulted (`layout` → `centered`) so the connections endpoint and the branding API agree (`service_branding.go:149`, `client_branding_resolver.go:120`).

### Logo storage & serving

- On create/update, the request may carry `logo_data` (base64) + `logo_content_type`. `storeLogoUpload` decodes and calls `SetLogoData`, which enforces **PNG/JPEG/WebP only** and **< 256 KB** (`service_branding.go:377`), stores the bytes in the `logo_data` BYTEA column, and rewrites `logo_url` to `/public/branding/{uuid}/logo`.
- `GET /public/branding/{branding_id}/logo` (`ServeLogo`) streams the bytes with `Cache-Control: public, max-age=3600` and an `ETag` of the branding UUID. Bytes are additionally cached in Redis (`branding:logo:{uuid}`, 1h TTL); the cache is invalidated on update/restore/delete.
- Theming reads deliberately **exclude** the logo bytes (`FindPublicByID`, `brandingPublicColumns`) — the browser fetches the image separately so login-page renders don't pull up to 256 KB per request.

### Branding lifecycle rules

| Action | System theme (`is_system`) | Custom theme |
|---|---|---|
| Create (`POST /branding`) | n/a (only seeded) | Allowed; never active/system on create; `layout` validated + defaulted to `centered` |
| Update colors (`PUT /branding/{uuid}`) | Allowed (name is immutable) | Allowed |
| Legacy upsert (`brandingService.Update`) | **Rejected** — "create a custom branding instead" | Allowed |
| Activate (`PATCH /{uuid}/activate`) | Allowed (light/dark are switchable) | Allowed — `DeactivateAll` then activate, so exactly one active |
| Restore to seeded defaults (`PATCH /{uuid}/restore`) | Allowed (resets metadata + clears logo/URLs) | **Rejected** — system-only |
| Delete (`DELETE /{uuid}`) | **Rejected** — undeletable | Allowed (soft delete) |

### Email/SMS template rendering (send path)

Feature flows resolve a template by a **name key** at send time:

1. Caller invokes `email.RenderTemplate(db, name, tenantID, data)` or `sms.RenderTemplate(...)` with a name like `user:invite`, `user:password:reset`, `user:mfa:stepup`, `sms:login:otp`, `sms:mfa:enroll` (callers: `internal/mfa`, `internal/authn`, `internal/user`, `internal/oauth`).
2. `fetchTemplate` looks up Redis (`email_tpl:{tenant}:{name}` / `sms_tpl:{tenant}:{name}`, 15m TTL), else queries the table `WHERE name = ? AND tenant_id = ? AND status = 'active' AND deleted_at IS NULL`.
3. The body is parsed and executed with the caller's `data` struct — **email uses `html/template`** (auto-escaping); **SMS uses `text/template`**. Email also renders `body_plain` when present.
4. On template edit/status change the service calls `email.InvalidateTemplateCache` / `sms.InvalidateTemplateCache` so the next render re-reads the DB.

System templates (`is_system=true`) reject `Update`/`UpdateStatus`/`Delete` in the service layer.

## Implementation

### Branding

| Concern | Location |
|---|---|
| Model / table | `internal/branding/model_branding.go` (`Branding`, `TableName()` → `branding`) |
| DTOs + metadata keys | `internal/branding/types.go` (`BrandingResponseDTO`, `BrandingUpdateRequestDTO`, `BrandingMetadata*` keys) |
| Service | `internal/branding/service_branding.go` (`Get`, `List`, `Create`, `Update`, `UpdateByUUID`, `Activate`, `RestoreSystem`, `Delete`, `GetPublic`, `GetPublicByID`, `GetLogoData`, `SetLogoData`) |
| Repository | `internal/branding/repository_branding.go` (`FindActive`, `FindSystem`, `FindSystemDefault`, `DeactivateAll`, `FindPublicByID`, …) |
| Handler | `internal/branding/handler_branding.go` (`List`, `Create`, `Update`, `Activate`, `RestoreSystem`, `Delete`, `GetPublic`, `ServeLogo`; logo Redis cache helpers) |
| Routes | `internal/branding/routes.go` (`BrandingRoute`, `BrandingPublicRoute`); mounted at `internal/server/router.go:95` (control) and `:310` (public) |
| Client resolver | `internal/branding/client_branding_resolver.go` |
| Validation | `internal/branding/validation_branding.go` (URL fields must be http/https; length caps; layout enum) |
| Seeded system themes | `internal/shared/branding_theme_defaults.go` (palettes for `default`/`light`/`dark`) + seeder `internal/setup/seeder/015_branding.go` |
| Migration | `internal/platform/database/migration/002_create_branding_table.go` |

**`branding` table columns:** `branding_id` (BIGSERIAL PK), `branding_uuid` (UUID unique), `tenant_id` (FK → `tenants`, `ON DELETE CASCADE`), `name`, `company_name`, `logo_url`, `logo_data` (BYTEA), `logo_content_type`, `favicon_url`, `support_url`, `privacy_policy_url`, `terms_of_service_url`, `metadata` (JSONB, default `'{}'`), `is_system`, `is_active`, `created_by`/`updated_by` (FK → `users`, added in migration 026), `created_at`/`updated_at`, `deleted_at` (soft delete).

**`metadata` JSONB holds:** `colors` (primary/secondary/accent, app/top-panel/side-panel/auth-form/auth-visual backgrounds, borders, text), `font.family`, `effects`, `components` (per-component background/hover/border/radius/text/size tokens incl. buttons, tables, inputs, badges, switch), `layout`, `logo_label`, `show_logo_label`, `logo_detail`, `identity_logo_label`, `identity_show_logo_label`, `login_form_logo_detail`, `login_form_logo_placement`. Preferences with no column are merged in via `mergeBrandingPreferences` (`handler_branding.go:366`).

**Branding endpoints (control API, `/api/v1`):**

| Method | Path | Permission | Handler |
|---|---|---|---|
| GET | `/branding` | `branding:read` | `List` |
| POST | `/branding` | `branding:create` | `Create` |
| PUT | `/branding/{branding_uuid}` | `branding:update` | `Update` |
| PATCH | `/branding/{branding_uuid}/restore` | `branding:update` | `RestoreSystem` |
| PATCH | `/branding/{branding_uuid}/activate` | `branding:activate` | `Activate` |
| DELETE | `/branding/{branding_uuid}` | `branding:delete` | `Delete` |
| GET | `/public/branding` | none (public `:8081`) | `GetPublic` (`?tenant_id=`) |
| GET | `/public/branding/{branding_id}/logo` | none (public `:8081`) | `ServeLogo` |

> Caveat: `FindPublicByID` (`repository_branding.go:57`) selects a `settings` column that does not exist on the table (the column is `metadata`); its only service caller `GetPublicByID` is not wired to any route, so the defect is latent.

### Email templates

| Concern | Location |
|---|---|
| Model / table | `internal/branding/model_email_template.go` (`EmailTemplate` → `email_templates`) |
| Service | `internal/branding/service_email_template.go` (`GetAll`, `GetByUUID`, `Create`, `Update`, `UpdateStatus`, `Delete`) |
| Repository | `internal/branding/repository_email_template.go` |
| Handler | `internal/branding/handler_email_template.go` |
| Routes | `internal/branding/routes.go` `EmailTemplateRoute` (mounted `router.go:93`) |
| Validation | `internal/branding/validation_email_template.go` (name ≤100, subject ≤255, body required, status ∈ {active,inactive}) |
| Renderer | `internal/platform/email/template.go` (`RenderTemplate`, `fetchTemplate`, `InvalidateTemplateCache`) |
| Seed bodies | `internal/platform/templates/emailtemplate/*.go` + seeder `internal/setup/seeder/010_email_template.go` |
| Migration | `internal/platform/database/migration/058_create_email_templates_table.go` |

**Table columns:** `email_template_id` (BIGSERIAL PK), `email_template_uuid`, `tenant_id` (FK CASCADE), `name`, `subject`, `body_html`, `body_plain`, `parameters_doc`, `status` (CHECK `active|inactive`, default `active`), `is_default`, `is_system`, `created_by`/`updated_by`, timestamps, `deleted_at`.

**Seeded template names** (per tenant, `is_system`, `010_email_template.go`): `user:invite`, `user:password:reset`, `user:email:verification`, `user:magic_link`, `user:ciba:notification`, `user:device:approved`, `user:email:change`, `user:email:changed`, `user:mfa:enroll`, `user:mfa:stepup`.

**Email endpoints (control API, `/api/v1`):**

| Method | Path | Permission | Handler |
|---|---|---|---|
| GET | `/email_templates` | `email-template:read` | `GetAll` (paginated; filters: name, status, is_default, is_system) |
| GET | `/email_templates/{email_template_uuid}` | `email-template:read` | `Get` |
| PUT | `/email_templates/{email_template_uuid}` | `email-template:update` | `Update` |
| PATCH | `/email_templates/{email_template_uuid}/status` | `email-template:update` | `UpdateStatus` |

> `Create` and `Delete` exist in the handler/service but are **not registered** in `EmailTemplateRoute`, so new/removed email templates are only managed by the seeder. Same for SMS below.

### SMS templates

| Concern | Location |
|---|---|
| Model / table | `internal/branding/model_sms_template.go` (`SMSTemplate` → `sms_templates`) |
| Service / Repo / Handler | `internal/branding/service_sms_template.go`, `repository_sms_template.go`, `handler_sms_template.go` |
| Routes | `internal/branding/routes.go` `SMSTemplateRoute` (mounted `router.go:94`) |
| Validation | `internal/branding/validation_sms_template.go` |
| Renderer | `internal/platform/sms/template.go` (`text/template`) |
| Seed | `internal/setup/seeder/011_sms_template.go` |
| Migration | `internal/platform/database/migration/059_create_sms_templates_table.go` |

**Table columns:** `sms_template_id`, `sms_template_uuid`, `tenant_id`, `name`, `description`, `message`, `parameters_doc`, `status`, `is_default`, `is_system`, `created_by`/`updated_by`, timestamps, `deleted_at`. **Seeded names:** `sms:login:otp`, `sms:mfa:enroll`, `sms:mfa:stepup`.

**SMS endpoints (control API, `/api/v1`):** `GET /sms_templates` (`sms-template:read`), `GET /sms_templates/{sms_template_uuid}` (read), `PUT /sms_templates/{sms_template_uuid}` (`sms-template:update`), `PATCH /sms_templates/{sms_template_uuid}/status` (update).

### Login templates — not implemented

`types.go` defines `LoginTemplate*` DTOs and `internal/shared/constants.go:96` defines style constants (`modern`, `classic`, `minimal`, `corporate`, `creative`, `custom`), but there is **no `login_templates` table, model, repository, service, handler, or route**. Hosted-login appearance is instead driven entirely by the branding `metadata.layout` + theme tokens. Treat login templates as a stubbed/planned surface.

## Configuration

- **No dedicated branding/template env vars.** All theming and message content is stored as **per-tenant DB rows**, seeded on tenant creation (`SeedEmailTemplates`, `SeedSMSTemplates`, `015_branding.go`).
- **Redis** (`email.RedisClient` / `sms.RedisClient` / branding logo cache via `platform/cache`): when configured, enables the 15-minute rendered-template cache and the 1-hour logo cache. When nil, everything falls back to direct DB reads (no failure).
- **Per-tenant settings** are the rows themselves: the active branding theme, custom themes, and any edited/added email and SMS templates.
- **Endpoint exposure**: management endpoints live on the internal control API (`:8080`, VPN-only) behind JWT + tenant-context + permission middleware; only the two public branding reads live on `:8081`.

## Security considerations

- **Tenant isolation**: every service method scopes by `tenant_id`; cross-tenant UUIDs resolve to `not found`. Management routes require JWT auth, tenant-context middleware, and granular permissions (`branding:*`, `email-template:*`, `sms-template:*`).
- **Public surface is intentionally minimal**: `GET /public/branding` returns colors, layout, logo/favicon URLs, and legal URLs only — never logo bytes, `created_by`, or internal IDs beyond the UUID. Logo bytes are served from a separate cached endpoint.
- **Logo upload hardening**: content-type allowlist (PNG/JPEG/WebP) and a 256 KB size cap block oversized or script-bearing uploads; bytes are stored in Postgres, not on a filesystem/URL fetch, avoiding SSRF from remote logo URLs.
- **URL validation**: logo/favicon/support/privacy/terms URLs must parse as `http`/`https` with a host (`validation_branding.go`), and are length-capped at 2048.
- **Template injection surface**: email bodies render through `html/template` (contextual auto-escaping). **SMS bodies render through `text/template` with no escaping** — acceptable for plain-text SMS, but template authors control the body and interpolated `data` comes from server-side flow structs, not raw end-user input.
- **Immutable system content**: seeded system branding themes and system email/SMS templates cannot be deleted (and system messages cannot be edited), preserving working auth flows even if an admin misconfigures a custom theme.
- **Custom CSS is unsupported** by design — there is no CSS-injection vector; theming is confined to structured token values in `metadata`.

## Related

- `./clients.md` — OAuth clients can attach a `branding_id`; the connections endpoint resolves per-client branding via `ClientBrandingResolver`.
- Email/SMS **delivery** (SMTP transport, provider config) is a separate subsystem in `internal/notifier` + `internal/platform/email`/`internal/platform/sms` — this doc covers only the template content those flows render.
- Consuming flows: `internal/mfa`, `internal/authn` (SMS login), `internal/user` (invite, email change, phone verify), `internal/oauth` (device approval, CIBA) call the template renderers.
