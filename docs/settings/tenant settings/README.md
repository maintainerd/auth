# Settings Documentation

This directory contains detailed documentation for every configurable settings entity in the maintainerd-auth service. Each document covers:

1. **What it is** — industry context, relevant standards (IETF, NIST, OWASP, etc.), and how things are typically done
2. **How we implemented it** — data model, API endpoints, service layer, and validation rules
3. **Requirements checklist** — complete checklist with ✅ for implemented items and ☐ for remaining work

Use these docs to understand what's built, what's missing, and where to contribute.

---

## Settings at a Glance

| Setting | Scope | Type | Doc |
|---------|-------|------|-----|
| [IP Restriction Rules](ip-restriction-rules.md) | Tenant | Many per tenant | Full CRUD with pagination |
| [Email Config](email-config.md) | Tenant | Singleton | Get / Update |
| [SMS Config](sms-config.md) | Tenant | Singleton | Get / Update |
| [Webhook Endpoints](webhook-endpoints.md) | Tenant | Many per tenant | Full CRUD |
| [Branding](branding.md) | Tenant | Singleton | Get / Update |
| [Tenant Settings](tenant-settings.md) | Tenant | Singleton (4 JSONB sub-configs) | Get / Update per sub-config |
| [Security Settings](security-settings/README.md) | User Pool | Singleton (7 JSONB sub-configs) | Get / Update per sub-config |

### Cross-Cutting Concerns

| Document | Scope | Description |
|----------|-------|-------------|
| [Logging & Audit Architecture](logging-and-audit.md) | System-wide | Three-layer logging strategy, `auth_events` redesign, standards references (OWASP, PCI DSS, NIST) |

---

## Implementation Status Summary

### Fully Implemented (config management only — enforcement logic is separate)

| Entity | Migration | Model | Repo | Service | DTO | Handler | Routes | Unit Tests |
|--------|:---------:|:-----:|:----:|:-------:|:---:|:-------:|:------:|:----------:|
| **IP Restriction Rules** | ✅ | ✅ | ✅ | ✅ (6 methods) | ✅ (5 DTOs) | ✅ (6 handlers) | ✅ | ✅ |
| **Email Config** | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |
| **Branding** | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |
| **Tenant Settings** | ✅ | ✅ | ✅ | ✅ (9 methods) | ✅ | ✅ (8 handlers) | ✅ | ✅ |
| **Security Settings** | ✅ | ✅ | ✅ | ✅ (14 methods) | ✅ | ✅ (14 handlers) | ✅ | ✅ |

### Missing Components

| Entity | What's Missing | Impact |
|--------|---------------|--------|
| **SMS Config** | No DTO validation file | Validation not enforced at the DTO layer |
| **Webhook Endpoints** | No DTO validation file | Validation not enforced at the DTO layer |
| **Webhook Endpoints** | No dispatcher engine | Endpoints are configurable but events are not actually delivered |

### Not Yet Implemented (enforcement layer)

None of the settings entities have their enforcement/runtime behavior implemented yet. The current implementation covers **configuration management** (CRUD) — the actual enforcement (e.g., IP blocking middleware, sending emails, rate limiting, session management, lockout, etc.) is future work. See each entity's checklist for specifics.

---

## Architecture Overview

```
┌─────────────────────────────────────────────────────────────────┐
│                        Admin API (port 8080)                     │
│                                                                  │
│  /ip-restriction-rules/*    → IP Restriction CRUD               │
│  /email-config              → Email Config Get/Update           │
│  /sms-config                → SMS Config Get/Update             │
│  /webhook-endpoints/*       → Webhook Endpoint CRUD             │
│  /branding                  → Branding Get/Update               │
│  /tenant-settings/*         → Tenant Settings (4 sub-configs)   │
│  /security-settings/*       → Security Settings (7 sub-configs) │
└─────────────────────────────────────────────────────────────────┘
         │
         ▼
┌─────────────────────┐    ┌─────────────────────┐
│   Service Layer      │    │   Repository Layer   │
│   (validation,       │───▶│   (GORM + PostgreSQL)│
│    tracing, logic)   │    │                      │
└─────────────────────┘    └─────────────────────┘
         │
         ▼
┌─────────────────────────────────────┐
│         PostgreSQL                   │
│                                      │
│  ip_restriction_rules               │
│  email_configs                      │
│  sms_configs                        │
│  webhook_endpoints                  │
│  brandings                          │
│  tenant_settings (4 JSONB cols)     │
│  security_settings (7 JSONB cols)   │
│  security_settings_audit            │
└─────────────────────────────────────┘
```

---

## Contributing

1. Pick an unchecked item from any entity's checklist
2. Read the entity doc to understand the standards and implementation context
3. Follow the [architecture guide](architecture.md) for layer conventions
4. Add tests (see [testing guide](testing.md))
5. Update the checklist in the entity doc when done
