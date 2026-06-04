# IP Restriction Rules

## Overview

IP restriction rules control which source IP addresses are allowed or denied access to the authentication service. They act as a network-layer gatekeeper, enforcing access policies before any application logic runs. This is a fundamental security control recommended by every major security framework.

## Industry Standards & Background

### What Are IP Restrictions?

IP-based access control is one of the oldest and most reliable perimeter security mechanisms. An IP restriction rule evaluates the source IP address of incoming requests against a configured list of allowable or deniable addresses and takes an action (permit or deny).

### Relevant Standards

| Standard | Reference | Key Guidance |
|----------|-----------|-------------|
| **NIST SP 800-41 Rev 1** | Guidelines on Firewalls and Firewall Policy | Defines packet-filtering rules based on source/destination IP; recommends default-deny policies for sensitive systems. |
| **NIST SP 800-53 Rev 5** | SC-7 (Boundary Protection) | Requires organizations to monitor and control communications at external/internal boundaries. |
| **OWASP ASVS v4** | V1.4.4, V14.5 | Applications should restrict access to administrative interfaces by IP allow-listing. |
| **CIS Controls v8** | Control 3.3, 4.4 | Configure network access control; use firewalls to filter unauthorized traffic. |
| **PCI DSS v4.0** | Req 1.2, 1.3 | Restrict inbound/outbound traffic to that which is necessary; deny all other traffic. |
| **ISO 27001:2022** | A.8.20 – Network Security | Control network access based on business and security requirements. |
| **RFC 791** | Internet Protocol (IPv4) | Defines the IP addressing format used in source-IP evaluation. |
| **RFC 4632** | Classless Inter-domain Routing (CIDR) | Defines CIDR notation (e.g., `192.168.1.0/24`) used for IP range matching. |

### How It Typically Works

1. **Rule types**: Systems maintain either an *allow-list* (only listed IPs are permitted) or a *deny-list* (listed IPs are blocked), or both.
2. **Evaluation order**: Rules are evaluated in a defined order — typically deny rules first, then allow rules, with a default action (deny or allow) applied if no rule matches.
3. **CIDR support**: Enterprise systems support CIDR notation to express entire subnets in a single rule (e.g., `10.0.0.0/8`).
4. **IPv6 support**: Modern systems support both IPv4 and IPv6 addresses.
5. **Temporary vs. permanent**: Some systems support time-bounded rules (e.g., grant access for 24 hours).
6. **Audit trail**: Every rule change is logged with who changed it and when.
7. **Bypass protection**: Critical administrative endpoints should always enforce IP rules regardless of authentication status.

### Common Use Cases

- **Admin panel lock-down**: Only corporate VPN IPs can access the admin API (`port 8080`).
- **Geo-blocking**: Block IP ranges associated with countries where the service should not operate.
- **Incident response**: Immediately deny a specific IP observed in an ongoing attack.
- **Compliance zones**: Restrict access to data-processing endpoints from approved data-center IPs only.
- **Partner API access**: Allow only known partner IPs to hit specific API endpoints.

---

## Our Implementation

### Architecture

IP restriction rules are a **tenant-level** resource stored in PostgreSQL. They are managed through the admin API (port 8080) and enforced at the middleware layer.

### Data Model

| Field | Type | Description |
|-------|------|-------------|
| `id` | UUID | Primary key |
| `tenant_id` | UUID | Foreign key → tenant |
| `description` | string | Human-readable description of the rule |
| `type` | enum | Rule type: `allow`, `deny`, `whitelist`, `blacklist` |
| `ip_address` | string | The IPv4 address to match |
| `status` | enum | `active` or `inactive` |
| `created_by` | UUID | User who created the rule |
| `updated_by` | UUID | User who last updated the rule |
| `created_at` | timestamp | Creation time |
| `updated_at` | timestamp | Last update time |
| `deleted_at` | timestamp | Soft-delete time (nullable) |

**Source files:**
- Model: `internal/model/ip_restriction_rule.go`
- Migration: `internal/database/migration/038_create_ip_restriction_rules.go`

### API Endpoints

All endpoints are under the admin API (port 8080), prefixed with the tenant/user-pool context.

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/ip-restriction-rules` | List rules with pagination and filtering |
| `GET` | `/ip-restriction-rules/{uuid}` | Get a single rule by ID |
| `POST` | `/ip-restriction-rules` | Create a new rule |
| `PUT` | `/ip-restriction-rules/{uuid}` | Update an existing rule |
| `DELETE` | `/ip-restriction-rules/{uuid}` | Soft-delete a rule |
| `PATCH` | `/ip-restriction-rules/{uuid}/status` | Toggle rule status (active/inactive) |

**Source files:**
- Handler: `internal/rest/ip_restriction_rule_handler.go`
- Routes: `internal/rest/ip_restriction_rule_routes.go`

### Filtering & Pagination

The list endpoint supports:
- **Pagination**: `page` (default 1), `limit` (default 20)
- **Filtering**: `type` (allow/deny/whitelist/blacklist), `status` (active/inactive)
- **Search**: text search across description and IP address fields

**Source files:**
- DTOs: `internal/dto/ip_restriction_rule.go`

### Service Layer

The service provides six operations:
- `List` — paginated + filtered listing
- `GetByID` — single rule lookup
- `Create` — validation + insert
- `Update` — validation + update
- `Delete` — soft delete
- `UpdateStatus` — toggle active/inactive

All operations are traced with OpenTelemetry spans.

**Source files:**
- Service: `internal/service/ip_restriction_rule_service.go`
- Repository: `internal/repository/ip_restriction_rule_repository.go`

### Validation Rules

| Field | Rules |
|-------|-------|
| `description` | Optional, max 500 characters |
| `type` | Required, must be one of: `allow`, `deny`, `whitelist`, `blacklist` |
| `ip_address` | Required, must be a valid IPv4 address |
| `status` | Required for status update, must be `active` or `inactive` |

---

## Requirements Checklist

### Core CRUD Operations
- [x] Create IP restriction rule
- [x] Read single IP restriction rule by ID
- [x] List IP restriction rules with pagination
- [x] Update IP restriction rule
- [x] Soft-delete IP restriction rule
- [x] Toggle rule status (active/inactive)

### Data Validation
- [x] Validate IP address format (IPv4)
- [x] Validate rule type (allow/deny/whitelist/blacklist)
- [x] Validate status values (active/inactive)
- [x] Validate description length (max 500)
- [ ] Validate CIDR notation (e.g., `192.168.1.0/24`)
- [ ] Validate IPv6 addresses
- [ ] Validate no duplicate IP + type combinations
- [ ] Validate IP range syntax (e.g., `192.168.1.1-192.168.1.255`)

### Rule Types
- [x] Allow-list rules
- [x] Deny-list rules
- [x] Whitelist (alias for allow)
- [x] Blacklist (alias for deny)
- [ ] Consolidate synonyms (allow = whitelist, deny = blacklist) or deprecate legacy terms

### IP Format Support
- [x] IPv4 single address support
- [ ] IPv4 CIDR range support (e.g., `10.0.0.0/8`)
- [ ] IPv6 single address support
- [ ] IPv6 CIDR range support (e.g., `2001:db8::/32`)
- [ ] Wildcard support (e.g., `192.168.1.*`)
- [ ] IP range support (e.g., `192.168.1.1-192.168.1.100`)

### Rule Evaluation & Enforcement
- [ ] Middleware that evaluates rules on every request
- [ ] Configurable evaluation order (deny-first vs. allow-first)
- [ ] Default action when no rule matches (configurable deny/allow)
- [ ] Apply rules at admin API (port 8080) level
- [ ] Apply rules at public API (port 8081) level
- [ ] Bypass protection for super-admin endpoints
- [ ] Cache evaluated rules in Redis for performance
- [ ] Rule evaluation with `X-Forwarded-For` / `X-Real-IP` header awareness
- [ ] Rule evaluation behind reverse proxies (trusted proxy configuration)

### Filtering & Search
- [x] Filter by rule type
- [x] Filter by status (active/inactive)
- [x] Pagination support
- [ ] Filter by IP address (exact match)
- [ ] Filter by IP address (partial/prefix match)
- [ ] Filter by date range (created_at)
- [ ] Sort by any column
- [ ] Bulk operations (enable/disable multiple rules)

### Audit & Logging
- [x] Track created_by on rule creation
- [x] Track updated_by on rule update
- [x] Timestamps on all operations (created_at, updated_at)
- [x] Soft delete with deleted_at
- [ ] Audit log for rule changes (who changed what, when, from what value)
- [ ] Log blocked requests with source IP and matched rule
- [ ] Alert/notification on rule violation (repeated blocked attempts)
- [ ] Export audit log

### Security
- [x] Admin-only API access (port 8080)
- [x] OpenTelemetry tracing on all operations
- [ ] Rate limiting on rule management endpoints
- [ ] Prevent self-lockout (warn if rule would block current admin IP)
- [ ] Dry-run mode (test a rule before activating)
- [ ] Temporary rules with auto-expiration (TTL)

### Import/Export
- [ ] Bulk import rules from CSV/JSON
- [ ] Export rules to CSV/JSON
- [ ] Import from common firewall formats (iptables, AWS Security Groups)

### Testing
- [x] Unit tests for service layer (100% coverage)
- [x] Unit tests for DTO validation
- [x] Unit tests for handler layer
- [ ] Integration tests with real database
- [ ] Integration tests for middleware enforcement
- [ ] Load tests for rule evaluation performance

---

## References

- [NIST SP 800-41 Rev 1 — Guidelines on Firewalls and Firewall Policy](https://csrc.nist.gov/publications/detail/sp/800-41/rev-1/final)
- [NIST SP 800-53 Rev 5 — Security and Privacy Controls](https://csrc.nist.gov/publications/detail/sp/800-53/rev-5/final)
- [OWASP ASVS v4.0](https://owasp.org/www-project-application-security-verification-standard/)
- [CIS Controls v8](https://www.cisecurity.org/controls)
- [PCI DSS v4.0](https://www.pcisecuritystandards.org/)
- [RFC 791 — Internet Protocol](https://datatracker.ietf.org/doc/html/rfc791)
- [RFC 4632 — CIDR](https://datatracker.ietf.org/doc/html/rfc4632)
