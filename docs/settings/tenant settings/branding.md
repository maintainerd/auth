# Branding

## Overview

Branding configuration controls the visual identity of the authentication service's user-facing pages — login screens, registration forms, email templates, and error pages. It enables white-labeling so that the authentication experience matches the organization's brand identity rather than showing a generic interface.

## Industry Standards & Background

### What Is White-Label Branding?

White-label branding allows a platform to be reskinned with a customer's visual identity. In auth services (Auth0, Cognito, Keycloak), branding covers logos, colors, fonts, and legal/support links displayed on externally-facing pages. The goal is a seamless experience where end users see the organization's brand, not the underlying auth provider.

### Relevant Standards & Guidelines

| Standard | Reference | Key Guidance |
|----------|-----------|-------------|
| **WCAG 2.1 / 2.2** | Web Content Accessibility Guidelines | Minimum 4.5:1 contrast ratio for normal text, 3:1 for large text (AA). Color must not be the only way to convey information. |
| **ISO 9241-210** | Ergonomics of Human-System Interaction | User-centered design process for interactive systems — relevant for login UX. |
| **Section 508** | US Accessibility Standards | Federal requirement for accessible user interfaces; aligns with WCAG 2.0 Level AA. |
| **EU Directive 2019/882** | European Accessibility Act | Requires digital products/services to be accessible to persons with disabilities. |
| **GDPR Art. 13, 14** | Transparency Obligations | Requires clear privacy policy links and data processing disclosures visible to users. |
| **Material Design** | Google Design System | Guidelines for color systems, typography scales, and component theming. |
| **CSS Custom Properties** | W3C CSS Variables | The underlying web technology (`--brand-primary`, `--brand-secondary`) for runtime theming. |

### How Auth Branding Typically Works

1. **Logo & favicon**: Organization uploads logos (login page, email header) and a favicon.
2. **Color scheme**: Primary, secondary, and accent colors are set; the system derives button colors, backgrounds, hover states.
3. **Typography**: A font-family is specified, applied to all rendered text.
4. **Custom CSS**: Advanced users inject custom CSS for fine-grained control beyond color/font settings.
5. **Legal links**: Privacy policy, terms of service, and support URLs are displayed in the footer of all public pages.
6. **Preview**: A live preview shows how changes will look before saving.
7. **Email templates**: Branding colors and logos are applied to transactional email headers/footers.

### What Leading Auth Platforms Offer

| Feature | Auth0 | Cognito | Keycloak | Clerk |
|---------|-------|---------|----------|-------|
| Logo/favicon | ✅ | ✅ | ✅ | ✅ |
| Colors (primary/secondary) | ✅ | Limited | ✅ | ✅ |
| Custom CSS | ✅ | ❌ | ✅ (themes) | ❌ |
| Custom fonts | ✅ | ❌ | ✅ | ✅ |
| Email template branding | ✅ | ✅ | ✅ | ✅ |
| Dark mode | ✅ | ❌ | Via theme | ✅ |
| Custom HTML | ✅ (Universal Login) | ❌ | ✅ (FreeMarker) | ❌ |
| Live preview | ✅ | ❌ | ❌ | ✅ |
| Multiple brands per tenant | ✅ (per-app) | ❌ | ✅ (per-realm) | ❌ |

---

## Our Implementation

### Architecture

Branding is a **tenant-level** singleton resource (one branding config per tenant). It is stored in PostgreSQL and managed through the admin API (port 8080). Public-facing pages read branding values at render time.

### Data Model

| Field | Type | Description |
|-------|------|-------------|
| `id` | UUID | Primary key |
| `tenant_id` | UUID | Foreign key → tenant |
| `company_name` | string | Organization name displayed on login pages |
| `logo_url` | string | URL to the organization's logo image |
| `favicon_url` | string | URL to the favicon |
| `primary_color` | string | Primary brand color (hex, e.g., `#1a73e8`) |
| `secondary_color` | string | Secondary brand color |
| `accent_color` | string | Accent/highlight color |
| `font_family` | string | CSS font-family string |
| `custom_css` | text | Custom CSS injected into public pages (max 50,000 chars) |
| `support_url` | string | Link to organization's support page (max 2048 chars, URL format) |
| `privacy_policy_url` | string | Link to privacy policy (max 2048 chars, URL format) |
| `terms_of_service_url` | string | Link to terms of service (max 2048 chars, URL format) |
| `metadata` | JSONB | Additional branding data |
| `created_at` | timestamp | Creation time |
| `updated_at` | timestamp | Last update time |
| `deleted_at` | timestamp | Soft-delete time (nullable) |

**Source files:**
- Model: `internal/model/branding.go`
- Migration: `internal/database/migration/003_create_brandings.go`

### API Endpoints

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/branding` | Get current branding configuration |
| `PUT` | `/branding` | Update branding configuration |

**Source files:**
- Handler: `internal/rest/branding_handler.go`
- Routes: `internal/rest/branding_routes.go`

### Service Layer

The service provides two operations:
- `Get` — retrieve the current branding for a tenant
- `Update` — validate and persist new branding

All operations are traced with OpenTelemetry spans.

**Source files:**
- Service: `internal/service/branding_service.go`
- Repository: `internal/repository/branding_repository.go`

### Validation Rules

| Field | Rules |
|-------|-------|
| `company_name` | Optional, max 255 characters |
| `logo_url` | Optional, max 2048 characters, must be valid URL format |
| `favicon_url` | Optional, max 2048 characters, must be valid URL format |
| `primary_color` | Optional, max 20 characters |
| `secondary_color` | Optional, max 20 characters |
| `accent_color` | Optional, max 20 characters |
| `font_family` | Optional, max 255 characters |
| `custom_css` | Optional, max 50,000 characters |
| `support_url` | Optional, max 2048 characters, must be valid URL format |
| `privacy_policy_url` | Optional, max 2048 characters, must be valid URL format |
| `terms_of_service_url` | Optional, max 2048 characters, must be valid URL format |

**Source files:**
- DTO: `internal/dto/branding.go`

---

## Requirements Checklist

### Core Operations
- [x] Get branding configuration
- [x] Update branding configuration
- [ ] Reset to default branding
- [ ] Preview branding before saving
- [ ] Branding version history (undo changes)

### Visual Identity
- [x] Company name
- [x] Logo URL
- [x] Favicon URL
- [x] Primary color
- [x] Secondary color
- [x] Accent color
- [x] Font family
- [x] Custom CSS injection
- [ ] Hex color format validation (e.g., `#1a73e8` or `#fff`)
- [ ] Logo image upload (vs. external URL)
- [ ] Favicon image upload
- [ ] Logo size constraints (recommended dimensions, max file size)
- [ ] Image format validation (PNG, SVG, JPEG, WebP)
- [ ] Dark mode variant (separate color scheme)
- [ ] Color contrast validation (WCAG AA 4.5:1 ratio check)
- [ ] Font loading from Google Fonts / custom font upload
- [ ] Custom CSS sanitization (prevent XSS via CSS injection)

### Legal & Compliance Links
- [x] Support URL
- [x] Privacy policy URL
- [x] Terms of service URL
- [x] URL format validation
- [x] URL max length validation (2048 chars)
- [ ] Link availability check (verify URL returns 2xx)
- [ ] Cookie policy URL
- [ ] GDPR data request URL
- [ ] Accessibility statement URL
- [ ] Custom footer text

### Template Integration
- [ ] Apply branding to login page
- [ ] Apply branding to registration page
- [ ] Apply branding to password reset page
- [ ] Apply branding to MFA challenge page
- [ ] Apply branding to email templates (headers, footers, colors)
- [ ] Apply branding to error pages
- [ ] Apply branding to consent/authorization pages

### Multi-Brand Support
- [ ] Per-user-pool branding override (different branding per client app)
- [ ] Branding inheritance (use tenant branding as default, override per pool)
- [ ] Brand preview by client ID

### Accessibility
- [ ] WCAG 2.1 AA compliance for default theme
- [ ] Color contrast checker integration
- [ ] Screen reader support on branded pages
- [ ] Keyboard navigation on branded pages
- [ ] High contrast mode

### Security
- [x] Custom CSS max length limit (50,000 chars)
- [x] URL length limit (2048 chars)
- [ ] Custom CSS sanitization (strip `url()`, `@import`, `expression()`, JavaScript URLs)
- [ ] Content Security Policy (CSP) headers for custom CSS/fonts
- [ ] Logo/favicon URL validation (prevent SSRF via internal URLs)
- [ ] Subresource Integrity (SRI) for external font CDN

### Monitoring & Observability
- [x] OpenTelemetry tracing on service operations
- [ ] Branding change audit log
- [ ] Brand rendering error tracking
- [ ] Custom CSS parse error detection

### Testing
- [x] Unit tests for service layer (100% coverage)
- [x] Unit tests for DTO validation
- [x] Unit tests for handler layer
- [ ] Visual regression tests for branded pages
- [ ] Accessibility automated tests (axe, Lighthouse)
- [ ] Integration tests with real database

---

## References

- [WCAG 2.1 — Web Content Accessibility Guidelines](https://www.w3.org/TR/WCAG21/)
- [Section 508 Standards](https://www.section508.gov/)
- [EU European Accessibility Act](https://ec.europa.eu/social/main.jsp?catId=1202)
- [GDPR — Transparency Obligations (Art. 13, 14)](https://gdpr-info.eu/art-13-gdpr/)
- [Material Design Color System](https://m3.material.io/styles/color/overview)
- [Auth0 Branding Documentation](https://auth0.com/docs/customize/branding)
- [CSS Custom Properties (W3C)](https://www.w3.org/TR/css-variables-1/)
