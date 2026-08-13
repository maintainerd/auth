// Package event implements the integration event plane for maintainerd-auth.
//
// # Naming convention
//
// Integration events use dotted notation (e.g. "user.created", "role.permissions_changed").
// This is distinct from the audit plane which uses snake_case ("authn_login_success").
// The dotted convention groups events by resource type: <resource>.<action>.
//
// # Versioning
//
// Each event type carries a version number (EventVersion field). Version 1 is the
// initial schema. If an event's fields change in a breaking way, the version is
// incremented and the old version may be retired via the global kill switch
// (event_types.is_active = false).
//
// # Redaction rules (thin events — hard requirement)
//
// Integration events carry identifiers and changed field NAMES only — never their
// values. This is enforced:
//   - changed_fields contains field names like ["email", "status"], not "new@example.com".
//   - The payload carries public UUIDs (event_id, tenant_id, actor_user_id,
//     subject_uuid) and event metadata — never internal database primary keys.
//   - A test must assert the outbox payload contains no resource value fields.
//
// Consumers MUST re-fetch current state from the authenticated API when they
// receive an event.
//
// # Delivery contract
//
//   - At-least-once, unordered. Delivery may duplicate and arrive out of order.
//   - Consumers dedup on event_id and treat ordering as best-effort.
//   - A 404 on refetch means the resource was deleted — a valid terminal outcome.
//   - Best-effort per-tenant ordering (relay processes outbox rows in insertion order).
//
// # Event catalog
//
// 42 event types across 5 groups:
//
//	Group 1 — User identity (user.created, user.updated, user.status_changed,
//	  user.deleted, user.role_assigned, user.role_removed)
//	Group 2 — Authorization model (role.*, permission.*, policy.*, iam.*)
//	Group 3 — Tenant / organization (tenant.*, tenant_member.*)
//	Group 4 — OAuth clients & credentials (client.*)
//	Group 5 — Sessions, identities & service principals (session.*, token.*,
//	  identity.*, api.*, service.*)
//
// # Example payload
//
//	{
//	  "event_id": "550e8400-e29b-41d4-a716-446655440000",
//	  "event_type": "user.updated",
//	  "event_version": 1,
//	  "tenant_id": "770e8400-e29b-41d4-a716-446655440001",
//	  "actor_user_id": "880e8400-e29b-41d4-a716-446655440002",
//	  "subject_uuid": "660e8400-e29b-41d4-a716-446655440001",
//	  "subject_type": "user",
//	  "changed_fields": ["email", "status"],
//	  "occurred_at": "2026-06-06T12:00:00Z",
//	  "trace_id": "abc123def456",
//	  "request_id": "req-789"
//	}
package event
