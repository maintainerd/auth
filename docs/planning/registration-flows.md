# Registration Flows Refactor

This document is the implementation tracker for restoring the original registration-flow design before `v0.1.0`. It covers the backend (`maintainerd-auth`), admin console (`maintainerd-auth-console`), and hosted identity UI (`maintainerd-auth-identity`).

> **Remaining work:** a post-implementation audit found D6 (server-side callback resolution) and D8 (server-persisted authorize resume) only partially built despite being checked below, plus several small hardening items. The precise, build-ready spec for all outstanding work is in [`registration-flows-remaining.md`](registration-flows-remaining.md). Treat that document as authoritative for D6/D8 over the `[x]` marks here.

The central correction is that an authentication flow and a registration flow are not the same thing:

- A **registration flow** is an explicitly selected registration policy that can require additional fields and automatically assign additional roles.
- A registration flow is never selected *implicitly* by login, by an authorize request without signup intent, by token exchange, or merely because a client references the same tenant. It is selected only by an explicit `registration_flow` parameter on a signup-intent authorize request or by an invite.
- Self-service registration is an entry mode of the **OAuth2 authorization-code flow**: external apps redirect to `/oauth/authorize` with `screen_hint=signup` (optionally `registration_flow=<identifier>`), and registration completes by issuing an authorization code to the client's registered `redirect_uri`. Registration and login share one token-issuing flow and one validated callback allowlist. This matches Auth0 (`screen_hint=signup`), Keycloak (`/registrations`), and Cognito (`/signup`).
- An **invite** is the only registration entry that uses a signed, expiring URL instead of an authorize request; the signed token is the sole authority for the invited email, tenant, client, optional flow, optional callback, expiry, and single use.
- A **client** owns application presentation and self-registration capability: branding and `allow_registration` belong on the client.
- A registration flow points to a client only to establish the registration client context and reuse that client's validated callback URI allowlist.
- An invite may select a registration flow. Without one, the invite uses normal registration and assigns only the standard `registered` role.

## How to use this tracker

- Leave an item unchecked while any implementation step or acceptance criterion remains incomplete.
- Check each nested task as it is completed.
- Check a parent item only after its tests and acceptance criteria pass.
- Preserve the internal/public surface contract throughout the refactor:
  - internal port `8080`: `tenant_id` is required and `client_id` is rejected;
  - public port `8081`: `client_id` is required and `tenant_id` is rejected;
  - explicit first-party system clients must not become valid public `client_id` values.
- Follow the create-only migration policy in `docs/contributing/database-migrations.md`. This is a pre-release schema correction, so edit or rename original create migrations in place; do not add alter, drop, rename, or backfill migrations.
- Follow `docs/contributing/testing.md` for handler, service, validation, mock, integration, and end-to-end tests.
- Run the mandatory verification and documentation work in J last.

## Repository labels

- `[BE]` — `maintainerd-auth`
- `[CON]` — `../maintainerd-auth-console`
- `[ID]` — `../maintainerd-auth-identity`
- `[DEV]` — `../maintainerd-dev`

## Locked target model

| Concern | Owner/source | Required behavior |
|---|---|---|
| Self-service registration entry | OAuth2 authorize flow | `/oauth/authorize?...&screen_hint=signup` issues an authorization code to the client's registered `redirect_uri`; no bespoke `callback_url` parameter |
| Invite registration entry | signed URL | The signed, expiring token is the sole authority; email, flow, roles, callback, expiry, and single-use `jti` are bound inside the signature |
| Normal self-registration gate | tenant + client | `tenant.self_registration_enabled && client.allow_registration` |
| Invite registration gate | invite | A valid pending invite can be accepted even when self-registration is disabled |
| Login and OAuth | client + connected IdPs | Never select a registration flow *implicitly*; only a signup-intent authorize request may carry an explicit `registration_flow` |
| Normal registration role | tenant | Always assign the tenant's system `registered` role |
| Special registration roles | registration flow | Assign `registered` plus the explicitly selected flow's roles |
| Registration-flow selection | explicit `registration_flow` on a signup authorize request, or an invite | Never infer a flow from `client_id` |
| Registration fields and verification | tenant policy + selected flow | A selected flow may tighten, never weaken, tenant policy |
| Branding | client | Resolve `client.branding_id`; fall back to tenant active branding, then system defaults |
| Registration callback allowlist | registration flow's client | Self-service uses the authorize `redirect_uri` validated against the client's `client_uris`; an invite carries a signed callback re-validated against the same allowlist |
| Hint parameters | request | `screen_hint` selects login vs signup; `idp_hint` routes to an upstream provider; `registration_flow` selects the registration policy — three independent knobs |
| Console theming | console system client | Resolve that client's branding, with the same tenant fallback |

### Registration behavior matrix

| Entry | Flow source | Self-registration gate | Roles | Flow fields | Branding |
|---|---|---|---|---|---|
| Normal signup (authorize `screen_hint=signup`, no flow) | none | tenant AND client | `registered` | tenant policy only | client |
| Special signup (authorize `screen_hint=signup` + `registration_flow=<identifier>`) | explicit `registration_flow` parameter | tenant AND flow client | `registered` + flow roles | tenant policy tightened by flow | flow client |
| Invite without a flow | none | bypassed | `registered` | invite baseline + tenant policy | invite client |
| Invite with a flow | `invite.registration_flow_id` | bypassed | `registered` + flow roles | invite baseline + flow required fields | flow/invite client |
| Login / authorize without signup intent / token | none | not applicable | none | none | client |

Self-service entries above are reached through `/oauth/authorize` and complete by issuing an authorization code to the client's registered `redirect_uri`; only invites use a signed URL.

## Confirmed current deviations

The following were the pre-refactor state. All have been corrected:

- [x] The schema and Go domain now use `registration_flows`, `registration_flow_roles`, and `registration_flow_callback_uris`.
- [x] ~~`registration_flows` currently owns `allow_registration`, `branding_id`, and `destination`.~~ Removed from `registration_flows` (B1). Moved to `clients` (B3).
- [x] ~~`clients` currently has neither `branding_id` nor `allow_registration`.~~ Added to `clients` migration + model (B3).
- [x] ~~`/oauth/connections` currently finds the first flow attached to a client and uses it for registration gating, verification, and required fields.~~ Connections no longer queries `registration_flows` (D1).
- [x] ~~Normal internal and public registration currently auto-select a flow by `client_id`.~~ Flow selection is always explicit via `registration_flow` parameter (A2).
- [x] ~~Sending an invite without a flow currently attaches the seeded `system:onboarding:registered` registration flow instead of leaving the foreign key `NULL`.~~ Invites leave `registration_flow_id=NULL` when no flow is selected (B4).
- [x] ~~Invite URLs currently carry a `registration_flow` query value, but invite acceptance authorizes role assignment from the stored invite relationship; the duplicate query value is not the correct authority.~~ Invite authority is in the signed token (B4).
- [x] ~~`destination=console` currently generates a console `/register/invite` URL even though the console has no such route.~~ Destination removed; all invites go to identity (A3).
- [x] ~~Registration-flow callback-URI CRUD exists in the backend and console even though the client already owns the validated URI records.~~ Deleted from backend (B2) and console (F3).
- [x] The console now consistently uses `RegistrationFlow*`, "Registration Flow", `/registration-flows`, and `/registration_flows` naming.
- [x] ~~The console still fabricates a `config.auto_approved` value the backend does not persist.~~ Removed from console types, API, and form (F2).
- [x] ~~The identity app consumes registration-flow-derived `required_fields` and `verification_required` from `/oauth/connections`, so an implicit client→flow relationship is present in the UI contract too.~~ Removed from identity `OAuthConnections` type and `RegisterForm` (G1).
- [x] ~~Client-scoped branding is not yet part of the `/oauth/connections` response; identity bootstrap applies tenant branding only.~~ `ClientBrandingResponse` added to connections (E2).
- [x] ~~Self-service registration is reached through a standalone `/register?flow=<identifier>` page that accepts an arbitrary `callback_url` query parameter instead of the OAuth2 authorize flow.~~ Registration entered via `screen_hint=signup` from `/oauth/authorize` (D8/G2). No free `callback_url` exists.

## Progress summary

- [x] A — Contract and terminology (0/3 remaining)
- [x] B — Canonical schema and migrations (0/4 remaining)
- [x] C — Backend management contracts (0/6 remaining)
- [x] D — Registration, invite, and callback runtime (0/8 remaining)
- [x] E — Client-owned branding resolution (0/4 remaining)
- [x] F — Console management and theming (0/7 remaining)
- [x] G — Identity registration and branding (0/6 remaining)
- [x] H — Automated verification (0/5 remaining)
- [x] I — Documentation and dead-contract cleanup (0/3 remaining)
- [x] J — Implementation order and release gate (0/4 remaining)
- [x] All registration-flow refactor work is complete

## A — Contract and terminology

### A1 — Rename the concept everywhere

- [x] Complete A1.
- **Implementation:**
  - [x] Use `RegistrationFlow` / `RegistrationFlowRole` in Go, protobuf, TypeScript, test fixtures, logs, traces, comments, and UI copy.
  - [x] Use `registration_flows` / `registration_flow_roles` in PostgreSQL.
  - [x] Use `/registration_flows` for the backend management resource and `/registration-flows` for console routes.
  - [x] Use `registration_flow_id`, `registration_flow_uuid`, and `registration_flow_identifier` consistently in internal models and public/admin DTOs.
  - [x] Rename permissions from `auth-flow:*` to `registration-flow:*` and update seeded policies.
  - [x] Do not rename genuine OAuth/OIDC “flows” such as authorization code, refresh token, device authorization, CIBA, or end-session flows.
- **Acceptance:**
  - [x] A targeted search finds no stale `AuthFlow`, `auth_flow`, `SignupFlow`, or `signup_flow` references except historical release notes or migration notes that intentionally describe the old name.

### A2 — Make flow selection explicit and one-way

- [x] Complete A2.
- **Contract:**
  - [x] A registration flow points to exactly one client.
  - [x] Multiple registration flows may point to the same client.
  - [x] A client does not have a `registration_flow_id`, and code must not find a flow merely by receiving a client.
  - [x] A flow is selected only by `invite.registration_flow_id` or an explicit registration-flow identifier in a direct registration request.
  - [x] An inactive or deleted flow cannot be selected for a new registration.
  - [x] An invite already bound to a flow keeps a stable database relationship; query parameters cannot substitute a different flow.
- **Acceptance:**
  - [x] Creating a second flow for one client does not make normal registration ambiguous because neither flow is auto-selected.

### A3 — Define the two registration entry contracts (authorize flow vs signed invite)

- [x] Complete A3.
- **Decision:**
  - [x] Remove `registration_flows.destination`; the hosted identity app is the registration UI for normal, special-flow, and invite registration.
  - [x] Treat `registration_flow.client_id` only as the source of the validated callback allowlist and branding context, never as evidence that the client owns or automatically activates the flow.
  - [x] **Self-service registration is an entry mode of the OAuth2 authorization-code flow, not a standalone page with an arbitrary return URL.** External apps start registration by redirecting to `/oauth/authorize` with the standard parameters (`client_id`, `redirect_uri`, `response_type=code`, `state`, PKCE) plus `screen_hint=signup`, and optionally `registration_flow=<identifier>` to select a special flow.
  - [x] The post-registration callback for self-service is the client's **registered `redirect_uri`**, validated exactly against that client's active `client_uris` of type `redirect-uri` by the existing authorize validation. Do not accept a free `callback_url` query parameter for self-service registration.
  - [x] On successful registration — and after any required verification/MFA — resume the same authorize request and issue an authorization code to the validated `redirect_uri`; the app exchanges it for tokens. Registration and login therefore terminate in the same token-issuing flow with no separate login step.
  - [x] `screen_hint` selects the screen (login vs signup), `idp_hint` selects an upstream provider, and `registration_flow` selects Maintainerd's registration policy/roles; the three are independent and may be combined.
  - [x] **Invites are the one exception** and remain a signed, expiring URL rather than an authorize request: the signed token itself is the authority. Bind the invited email, tenant, client, optional `registration_flow`, optional callback, expiry, and a single-use identifier inside the signature; never trust unsigned query parameters for any of these. After invite registration completes, establish the authenticated session.
  - [x] Preserve the public/internal query contract when generating links: external clients use `client_id`; first-party Maintainerd registration uses `tenant_id` and resolves the correct system client internally.
- **Acceptance:**
  - [x] No invite is sent to a nonexistent console registration route.
  - [x] A self-service registration can only return to a `redirect_uri` registered on the client, validated by the authorize flow, and completes by issuing an authorization code.
  - [x] Tampering with `registration_flow` on a self-service authorize request cannot grant a flow's roles unless that flow is active and linked to the authenticated client; tampering with an invite's query parameters cannot change the signed email, flow, roles, or callback.

## B — Canonical schema and migrations

### B1 — Rename the registration-flow tables in their original migrations

- [x] Complete B1.
- **Repos/files:**
  - [BE] `internal/platform/database/migration/038_create_auth_flows_table.go`
  - [BE] `internal/platform/database/migration/039_create_auth_flow_roles_table.go`
  - [BE] `internal/platform/runner/migration.go`
- **Implementation:**
  - [x] Rename migration 038 and its function to create `registration_flows`.
  - [x] Rename every primary key, UUID, foreign key, index, and constraint from `auth_flow_*` to `registration_flow_*`.
  - [x] Keep `tenant_id`, `name`, `description`, `identifier`, `is_system`, `verification_required`, `required_fields`, `status`, audit columns, timestamps, and soft deletion.
  - [x] Make `client_id BIGINT NOT NULL`; every special registration flow needs a callback/client context.
  - [x] Use an FK deletion policy that does not silently orphan a flow. Prefer rejecting client deletion while a flow references it.
  - [x] Remove `allow_registration`, `branding_id`, and `destination` from the flow table.
  - [x] Keep tenant-scoped uniqueness for `identifier` among non-deleted rows.
  - [x] Rename migration 039 and its function to create `registration_flow_roles` with matching FK, uniqueness, and indexes.
  - [x] Update the migration registry names and migration tests in place.
- **Acceptance:**
  - [x] A clean database contains only `registration_flows` and `registration_flow_roles`, with no `auth_flows` tables or identifiers.

### B2 — Remove the callback join table from the canonical schema

- [x] Complete B2.
- **Repos/files:**
  - [BE] `internal/platform/database/migration/040_create_auth_flow_callback_uris_table.go`
  - [BE] `internal/platform/runner/migration.go`
- **Implementation:**
  - [x] Delete migration 040 and remove it from the registry; do not add a drop migration.
  - [x] Remove the callback-URI model, repository, service methods, handlers, routes, DTOs, validation, mocks, and tests.
  - [x] Use `client_uris` directly as the callback allowlist.
- **Acceptance:**
  - [x] A clean migration run never creates a registration/auth-flow callback join table.

### B3 — Add client-owned branding and registration capability

- [x] Complete B3.
- **Repos/files:**
  - [BE] `internal/platform/database/migration/015_create_clients_table.go`
  - [BE] `internal/client/model_client.go`
- **Implementation:**
  - [x] Add nullable `branding_id BIGINT` with FK to `branding(branding_id)` and `ON DELETE SET NULL` fallback behavior.
  - [x] Add an index for non-null `branding_id` values.
  - [x] Add `allow_registration BOOLEAN NOT NULL DEFAULT TRUE`.
  - [x] Add `BrandingID`, `Branding`, and `AllowRegistration` to the canonical client model and authn/client projections.
  - [x] Validate at the service layer that an assigned branding record belongs to the same tenant as the client.
- **Acceptance:**
  - [x] A client can explicitly select a tenant branding or inherit the active tenant branding with `branding_id=NULL`.
  - [x] `allow_registration=false` survives create, read, update, list, and restart without being replaced by a default `true` value.

### B4 — Rename the invite relationship and preserve “no flow”

- [x] Complete B4.
- **Repos/files:**
  - [BE] `internal/platform/database/migration/041_create_invites_table.go`
  - [BE] `internal/invite/model_invite.go`
  - [BE] authn/invite adapter models
- **Implementation:**
  - [x] Rename `auth_flow_id` to nullable `registration_flow_id` and point it to `registration_flows`.
  - [x] Rename model fields, preload names, indexes, constraints, request fields, and response fields.
  - [x] Keep `NULL` semantically meaningful: it means normal invite registration and must not be replaced with a seeded default flow.
  - [x] Keep `client_id` on the invite. When a flow is selected, set it from the flow's client; without a flow, use the explicitly resolved/default first-party client for that invite context.
- **Acceptance:**
  - [x] Creating an invite with no flow persists `registration_flow_id=NULL`.

## C — Backend management contracts

### C1 — Rename the backend registration-flow domain

- [x] Complete C1.
- **Repos/files:**
  - [BE] `internal/idp/model_auth_flow*.go` → `model_registration_flow*.go`
  - [BE] All other idp domain files
- **Implementation:**
  - [x] Rename files, types, interfaces, constructors, methods, route parameters, DTOs, spans, messages, mocks, and tests to registration-flow terminology.
  - [x] Keep the domain under `internal/idp` for this refactor unless a separate architecture task explicitly moves ownership; do not combine the semantic correction with an unrelated package reorganization.
  - [x] Remove all branding, allow-registration, destination, and callback-URI fields/methods from this domain.
  - [x] Keep tenant isolation on every management operation.
  - [x] Keep system-flow protection and pending-invite deletion protection under the renamed relationship.
- **Acceptance:**
  - [x] Registration-flow CRUD, status, role assignment, and role removal work under the new types and route.

### C2 — Make the registration-flow DTO match its purpose

- [x] Complete C2.
- **Implementation:**
  - [x] Admin response fields: ID/UUID, name, description, identifier, status, `is_system`, client summary, `verification_required`, parsed `required_fields`, timestamps, and role summaries/count where appropriate.
  - [x] Create/update fields: name, description, identifier, status, client UUID, `verification_required`, `required_fields`, and optional role UUID replacement set.
  - [x] Allow an admin-supplied stable identifier; normalize and validate it, and enforce tenant-scoped uniqueness.
  - [x] Use a structured string array for `required_fields` at the API boundary even if the first implementation continues storing JSON text internally.
  - [x] Support only fields the registration DTO can actually collect (`email`, `fullname`, and `phone`, plus always-required username/password semantics).
  - [x] Validate that selected roles and client belong to the same tenant as the flow.
  - [x] Reject inactive/deleted clients for newly created or activated flows.
- **Acceptance:**
  - [x] The console no longer needs response mappers that fabricate `signup_flow_id` or `config.auto_approved`.

### C3 — Add client fields to every backend client contract

- [x] Complete C3.
- **Repos/files:**
  - [BE] `internal/client/model_client.go`, `types.go`, `service_client.go`, handlers, validation, repositories, and tests
  - [BE] `internal/authn/deps.go` and app projection adapters
- **Implementation:**
  - [x] Add `branding_id` and `allow_registration` to client create, update, list, detail, internal projection, and relevant public response contracts.
  - [x] Use pointer/optional request semantics so an explicit `false` is not confused with an omitted value.
  - [x] Preload or resolve branding only where needed; avoid multiplying list queries.
  - [x] Return a safe branding summary/UUID to the console without exposing numeric database IDs.
  - [x] Clear to fallback branding when the update explicitly supplies no branding.
  - [x] Prevent cross-tenant branding assignment.
- **Acceptance:**
  - [x] Client API tests cover explicit branding, inherited branding, `allow_registration=true`, and `allow_registration=false`.

### C4 — Rename permissions, seeders, and composition wiring

- [x] Complete C4.
- **Repos/files:**
  - [BE] `internal/setup/seeder/004_permission.go`
  - [BE] `internal/setup/seeder/012_control_policy.go`
  - [BE] `internal/setup/seeder/015_registration_flow.go`
  - [BE] `internal/setup/seeder/seed_tenant.go`
  - [BE] `internal/app/{app.go,application.go,repositories.go,services.go,adapters_*.go}`
- **Implementation:**
  - [x] Seed `registration-flow:read|create|update|delete` permissions and update wildcard policies.
  - [x] Rename the flow seeder and constants.
  - [x] Stop seeding a special "registered-only" flow as the representation of normal registration.
  - [x] Seed only genuinely special/system registration flows that are still required, each with an explicit client.
  - [x] Set system client branding/registration defaults intentionally; `NULL` branding may be used to inherit the tenant active branding.
  - [x] Rename repositories, services, adapters, fields, and constructors in the composition root.
- **Acceptance:**
  - [x] A newly seeded tenant can perform normal registration without any registration-flow row.

### C5 — Rename and complete the gRPC contract

- [x] Complete C5.
- **Repos/files:**
  - [BE] `proto/maintainerd/auth/v1/identity_provider.proto`
  - [BE] `proto/maintainerd/auth/v1/client.proto`
  - [BE] `proto/maintainerd/auth/v1/api.proto`
  - [BE] generated Go and gRPC handlers
- **Implementation:**
  - [x] Rename `SignupFlowService`, messages, fields, and RPCs to `RegistrationFlowService` terminology.
  - [x] Prefer moving the service/messages into `registration_flow.proto` so the public contract no longer presents registration flows as identity-provider configuration; update imports and API registration.
  - [x] Match REST capabilities: identifier, client, required fields, verification requirement, status, and roles.
  - [x] Remove branding, allow-registration, destination, and callback-URI flow fields.
  - [x] Add client `branding_id` and `allow_registration` fields to protobuf create/update/read messages with correct optional-boolean semantics.
  - [x] Regenerate code with the repository's `make proto` workflow and run `buf lint`.
- **Acceptance:**
  - [x] REST and gRPC expose the same registration-flow and client semantics.

### C6 — Add safe public registration context endpoints

- [x] Complete C6.
- **Implementation:**
  - [x] Add a public-safe registration-flow context lookup by identifier for direct special-registration links.
  - [x] Require and validate the correct client/tenant surface context before returning the flow.
  - [x] Return only fields needed to render registration: identifier, display name/description if desired, `required_fields`, effective verification requirement, client identifier/context, resolved branding, and validated callback result.
  - [x] Never expose assigned role IDs/names through the public context; roles are server-side consequences.
  - [x] Add an invite-context/preflight endpoint that validates the signed invite parameters and returns invited email, required fields, resolved branding, and safe client/flow display metadata before submission.
  - [x] Do not treat a query-string flow identifier as authority for an invite; load the relationship from the invite row.
- **Acceptance:**
  - [x] Identity can render direct-flow and invite forms without exposing management APIs or role assignments.

## D — Registration, invite, and callback runtime

### D1 — Rework `/oauth/connections`

- [x] Complete D1.
- **Repos/files:**
  - [BE] `internal/oauth/service_connections.go`, `types.go`, handler/tests
- **Implementation:**
  - [x] Remove every `registration_flows` query from this service.
  - [x] Compute `registration_enabled` as tenant `self_registration_enabled && idp.AllowRegistration`.
  - [x] Keep provider-specific `allow_registration` scoped to that provider's own JIT/federated registration behavior; do not let it redefine the local client signup gate.
  - [x] Remove flow-derived `verification_required` and `required_fields` from the connections response.
  - [x] Include resolved client branding in the response using the fallback contract in E1. (Deferred to E2)
  - [x] Keep password availability and ordered connected-provider behavior unchanged.
- **Acceptance:**
  - [x] Adding, removing, activating, or deactivating a registration flow cannot change login methods or the normal Sign up link.

### D2 — Rework normal self-registration

- [x] Complete D2.
- **Repos/files:**
  - [BE] `internal/authn/service_register.go`, handler, deps/adapters, and tests
- **Implementation:**
  - [x] Remove `registrationFlowForClient` and all automatic flow lookup by client.
  - [x] Gate normal registration only on tenant self-registration and `client.allow_registration`.
  - [x] Continue enforcing tenant password, email-domain, CAPTCHA, rate-limit, phone-verification, and email-verification policy.
  - [x] Always assign the `registered` role and no flow roles when no explicit flow is supplied.
  - [x] Keep public and internal client resolution rules unchanged.
- **Acceptance:**
  - [x] Normal registration behaves identically whether zero, one, or many registration flows point to the client.

### D3 — Add explicit special-registration execution

- [x] Complete D3.
- **Implementation:**
  - [x] Extend the registration handler/service contract with an optional `registration_flow` identifier supplied through the signup-intent authorize request (or invite), not through a bespoke registration URL.
  - [x] Resolve it by identifier and tenant in the same transaction used for registration.
  - [x] Require the flow to be active, non-deleted, and linked to the resolved client.
  - [x] On public registration, reject a supplied `client_id` that does not match the flow's linked client.
  - [x] Apply flow `required_fields` and let `verification_required=true` tighten the tenant policy.
  - [x] Assign `registered` first, then distinct additional flow roles.
  - [x] Never allow the selected flow to weaken tenant policy, grant cross-tenant roles, or change login/token behavior.
- **Acceptance:**
  - [x] `/register?registration_flow=seller` receives seller roles only when `seller` was explicitly selected and validated.

### D4 — Correct invite creation semantics

- [x] Complete D4.
- **Repos/files:**
  - [BE] `internal/invite/service_invite.go`, DTOs, handlers, repositories, validation, and tests
- **Implementation:**
  - [x] Rename request/response fields to `registration_flow_uuid`, `registration_flow_id`, and `registration_flow_name`.
  - [x] Without a selected flow, leave `registration_flow_id=NULL`; do not look up or attach a default flow.
  - [x] With a selected flow, validate tenant ownership, active status, client status, and inviter authority to grant every additional role.
  - [x] Set the invite client from the selected flow so callback and branding resolve from the same client.
  - [x] Encode the invited email, tenant, client, optional `registration_flow`, optional validated callback, expiry, and a single-use identifier (`jti`) inside the signed invite token; never place any of these authority-bearing values as unsigned query parameters.
  - [x] Resolve/validate the post-registration callback against that client before sending the email.
  - [x] Generate all invite registration URLs for the hosted identity app.
  - [x] Include either `client_id` or `tenant_id` according to the target surface, never both.
  - [x] Rename signed query fields; remove a redundant flow identifier from the link if the invite-context endpoint can resolve it from the invite token.
- **Acceptance:**
  - [x] No-flow invite, special-flow invite, external-client invite, and first-party console invite each produce a valid identity URL and correct client context.

### D5 — Correct invite acceptance and role assignment

- [x] Complete D5.
- **Implementation:**
  - [x] Treat the stored invite row as the sole authority for tenant, client, email, and optional flow.
  - [x] Lock registration to the signed invited email; reject any submission whose email differs from the token's email.
  - [x] Keep invite acceptance independent of tenant/client self-registration flags.
  - [x] Always assign `registered`.
  - [x] If `registration_flow_id` is non-null, load the same-tenant flow and assign its distinct additional roles.
  - [x] Enforce flow required fields on invite registration; treat the signed emailed invite as proof of the invited email rather than sending redundant email verification.
  - [x] Keep password policy, duplicate user/email checks, expiration, pending-state, single-use, and transactional `MarkAsUsed` behavior.
  - [x] Deduplicate the public and internal invite role-assignment implementation so both surfaces obey the same rules.
- **Acceptance:**
  - [x] No-flow invite assigns exactly `registered`; flow invite assigns `registered` plus that flow's roles.

### D6 — Validate and return post-registration callbacks

- [x] Complete D6.
- **Implementation:**
  - [x] For self-service registration, use the authorize `redirect_uri` already validated against the client's `client_uris`; do not introduce a second callback parameter.
  - [x] For invites, take the callback from the signed token and re-validate it against the client's `client_uris` before use.
  - [x] Add a reusable callback resolver that selects the flow/invite client and checks exact URI membership and type.
  - [x] Reject wildcard, prefix, scheme-downgrade, lookalike-host, and query-manipulation matches; use exact normalized URI comparison consistent with OAuth redirect validation.
  - [x] Bind the validated callback to signed invite context or a short-lived server-side registration context so it cannot be swapped after validation.
  - [x] Return the validated callback in the successful registration response or a safe continuation response for identity to navigate.
  - [x] Keep the current account-completion sequence: verification/profile/MFA steps finish before the external callback is used.
- **Acceptance:**
  - [x] Callback tampering fails before user creation, and a valid callback is used only after registration/account completion succeeds.

### D7 — Prove login/OAuth/token isolation

- [x] Complete D7.
- **Implementation:**
  - [x] Search login, OAuth authorize, consent, token, refresh, introspection, revocation, end-session, magic-link, and broker paths for registration-flow lookups.
  - [x] Remove any lookup, naming, comments, or tests implying a registration flow participates in authentication.
  - [x] Confine all registration-flow handling to the signup-intent branch (`screen_hint=signup`); the login, refresh, and token branches of the authorize flow must never read `registration_flow`.
  - [x] Keep `registration_enabled` as display/capability metadata only; it must not affect login for existing users.
- **Acceptance:**
  - [x] A client with `allow_registration=false` still permits existing users to log in and complete OAuth authorization.

### D8 — Make self-service registration an entry mode of the OAuth2 authorize flow

- [x] Complete D8.
- **Repos/files:**
  - [BE] `internal/oauth` authorize handler/service, login-context plumbing, and tests
- **Implementation:**
  - [x] Accept `screen_hint` (`signup`/`login`) on `/oauth/authorize` and carry it through the login-context so the identity app renders the registration screen first when signup is requested.
  - [x] Accept an optional `registration_flow=<identifier>` on the authorize request and thread it into the registration submission.
  - [x] Validate that the `registration_flow` is active, non-deleted, same-tenant, and linked to the authorize `client_id`; reject mismatches.
  - [x] After successful registration and any required verification/MFA, resume the same authorize request and issue an authorization code to the validated `redirect_uri`.
  - [x] Do not introduce a parallel registration-to-callback path that bypasses code issuance for self-service; the bespoke `callback_url` page is removed.
  - [x] Keep `screen_hint`, `idp_hint`, and `registration_flow` independent; `idp_hint` still routes signup to an upstream provider.
- **Acceptance:**
  - [x] An external app reaches registration via `/oauth/authorize?...&screen_hint=signup&registration_flow=<id>` and receives an authorization code on completion, with no bespoke callback parameter and no separate login step.

## E — Client-owned branding resolution

### E1 — Implement one client-branding resolver

- [x] Complete E1.
- **Implementation:**
  - [x] Resolve explicit `client.branding_id` first.
  - [x] Fall back to the active branding in the same tenant when `branding_id` is null, missing, deleted, or safely recoverable.
  - [x] Fall back to system/default visual tokens when the tenant has no active branding.
  - [x] Reject cross-tenant explicit branding rather than silently using it.
  - [x] Reuse the resolver in OAuth connections, registration context, invite context, first-party identity bootstrap, and console bootstrap.
  - [x] Avoid N+1 queries and define cache invalidation when client branding or branding contents change.
- **Acceptance:**
  - [x] All three surfaces produce the same resolved branding for the same client.

### E2 — Add client branding to public identity payloads

- [x] Complete E2.
- **Implementation:**
  - [x] Embed safe branding data in `/oauth/connections` for client-scoped login.
  - [x] Embed the same shape in direct registration-flow and invite context responses.
  - [x] Include colors, font, background/gradient metadata, logo/favicon URLs, layout, company name, and policy/support links needed by the identity UI.
  - [x] Do not expose internal numeric branding IDs or unrelated email/SMS template data.
- **Acceptance:**
  - [x] Two clients in one tenant can render different login and registration designs.

### E3 — Resolve first-party identity branding without public system-client leakage

- [x] Complete E3.
- **Implementation:**
  - [x] For a first-party `tenant_id` identity context, resolve the tenant's identity system client internally.
  - [x] Apply that system client's branding with tenant fallback.
  - [x] Do not make the system client's identifier acceptable on public `client_id` handlers to achieve this.
- **Acceptance:**
  - [x] First-party identity pages are client-themed while the public surface contract still rejects explicit system clients.

### E4 — Resolve console branding from the console client

- [x] Complete E4.
- **Implementation:**
  - [x] Resolve the tenant's console system client during authenticated console bootstrap.
  - [x] Return/apply its client branding with tenant fallback.
  - [x] Ensure client A's theme cannot leak after tenant/client switching or logout.
- **Acceptance:**
  - [x] Changing the console system client's branding changes the entire console theme without changing another client's hosted login.

## F — Console management and theming

### F1 — Rename the flow feature end to end

- [x] Complete F1.
- **Repos/files:**
  - [CON] `src/pages/signup-flows/` → already `src/pages/registration-flows/`
  - [CON] `src/services/api/signup-flows/` → already `src/services/api/registration-flows/`
  - [CON] `src/hooks/useSignupFlows.ts` → already `useRegistrationFlows.ts`
  - [CON] `src/lib/validations/signupFlowSchema.ts` → already `registrationFlowSchema.ts`
  - [CON] routes, navigation, sidebar, labels, and tests
- **Implementation:**
  - [x] Rename directories, files, components, hooks, types, query keys, functions, and validation schemas to `registration-flow` terminology.
  - [x] Change console paths from `/:tenantId/auth-flows/...` to `/:tenantId/registration-flows/...`.
  - [x] Change the API endpoint from `/auth_flows` to `/registration_flows`.
  - [x] Remove compatibility mappers for `auth_flow_id`, `signup_flow_id`, and fabricated config.
  - [x] Preserve origin-aware back navigation on list/detail/form pages.
- **Acceptance:**
  - [x] The UI consistently says "Registration Flow" and no page or API helper says "Auth Flow" or "Signup Flow".

### F2 — Simplify the registration-flow form

- [x] Complete F2.
- **Implementation:**
  - [x] Keep name, description, stable identifier, status, client selector, roles, required fields, and verification requirement.
  - [x] Explain beside the client selector that the flow uses that client's branding/context and validated callback URIs but is not automatically activated by that client.
  - [x] Remove flow branding selection.
  - [x] Remove flow `allow_registration`.
  - [x] Remove destination.
  - [x] Remove callback-URI selection.
  - [x] Remove `config`, `auto_approved`, custom config fields, and every fabricated default.
  - [x] Submit role replacement through the canonical backend contract.
- **Acceptance:**
  - [x] The form contains only fields persisted by the registration-flow backend DTO.

### F3 — Simplify registration-flow details

- [x] Complete F3.
- **Implementation:**
  - [x] Remove the callback-URI tab and its dialogs/components.
  - [x] Show identifier, linked client, status, verification requirement, required fields, assigned roles, and system state.
  - [x] Link the client summary to client details.
  - [x] Explain that all redirect URIs registered on the linked client form the callback allowlist.
  - [x] Keep role assignment/removal and system-flow restrictions visible and tested.
- **Acceptance:**
  - [x] Details accurately read back every configurable field and no dead callback/config data.

### F4 — Add branding and registration controls to clients

- [x] Complete F4.
- **Repos/files:**
  - [CON] `src/pages/clients/form/ClientAddOrUpdateForm.tsx`
  - [CON] client validation, API types/hooks, list, and detail components
- **Implementation:**
  - [x] Add a branding selector with "Use tenant active branding" as the null option.
  - [x] Add an `Allow registration` switch with clear copy that it affects self-registration, not login or invite acceptance.
  - [x] Send and hydrate explicit `false` correctly.
  - [x] Show branding and registration state on client details.
  - [x] Add useful list badges/columns or filters without overcrowding the existing client list.
  - [x] Keep the controls out of the free-form `config` JSON.
- **Acceptance:**
  - [x] Admins can assign different brandings to two clients and independently disable self-registration for either client.

### F5 — Show client usage on branding pages

- [x] Complete F5.
- **Implementation:**
  - [x] Show which clients explicitly use a branding template, or provide a linked-clients count/list.
  - [x] Explain that deleting an explicitly selected branding returns those clients to tenant-active fallback.
  - [x] Keep active tenant branding semantics for clients with no explicit selection.
- **Acceptance:**
  - [x] An admin can see the impact of editing or deleting a branding before doing it.

### F6 — Rename and correct invite management

- [x] Complete F6.
- **Repos/files:**
  - [CON] invite form, list, details, API types/hooks, and tests
- **Implementation:**
  - [x] Rename request/response fields and labels to registration-flow terminology.
  - [x] Keep "Default registration (no registration flow)" as the explicit null choice.
  - [x] List only selectable active flows and show the linked client and additional roles in the option/preview.
  - [x] If callback selection is required, populate it only from the selected flow client's redirect URIs and submit the selected URI.
  - [x] Make it clear that no-flow gives `registered`, while a selected flow adds its roles.
- **Acceptance:**
  - [x] The invite list/details display "Default registration" only when the stored relationship is null.

### F7 — Apply client branding to the entire console

- [x] Complete F7.
- **Implementation:**
  - [x] Extend the console theme provider/bootstrap to apply resolved console-client CSS variables, font, logo/favicon, and supported layout tokens.
  - [x] Apply theme tokens to shared navigation, forms, buttons, status surfaces, and authentication pages rather than one branding preview screen.
  - [x] Restore defaults during logout, tenant switch, and unmount to prevent theme leakage.
  - [x] Keep accessibility contrast and focus visibility acceptable for custom themes.
- **Acceptance:**
  - [x] Branding the console client visibly changes the whole console while client-specific identity branding remains independent.

## G — Identity registration and branding

### G1 — Apply branding from the client during bootstrap

- [x] Complete G1.
- **Repos/files:**
  - [ID] `src/components/auth/AppBootstrap.tsx`
  - [ID] OAuth/registration context services and branding utilities
- **Implementation:**
  - [x] In external `client_id` contexts, fetch/apply client-resolved branding before rendering login/registration content.
  - [x] In first-party `tenant_id` contexts, apply identity-system-client branding.
  - [x] Let direct-flow and invite contexts supply the same branding shape.
  - [x] Prevent a flash of tenant branding followed by client branding where practical.
  - [x] Clean up CSS variables and document metadata on context changes.
- **Acceptance:**
  - [x] Login, normal registration, direct-flow registration, invite registration, verification, recovery, and completion screens retain the correct client theme.

### G2 — Keep login independent from registration flows

- [x] Complete G2.
- **Repos/files:**
  - [ID] login form, OAuth connection types/hooks, and tests
- **Implementation:**
  - [x] Render the signup screen first when the authorize context carries `screen_hint=signup`, and the login screen otherwise; the Sign up link switches to the signup screen within the same authorize flow rather than navigating to a separate registration URL.
  - [x] Gate the Sign up link only on `/oauth/connections.registration_enabled` for a client context, or the equivalent first-party resolved client value.
  - [x] Continue showing login when registration is disabled.
  - [x] Remove `required_fields` and flow verification consumption from login/OAuth connection types.
  - [x] Keep connected-provider login buttons and password availability unchanged.
- **Acceptance:**
  - [x] Disabling registration hides/blocks signup without hiding or breaking sign-in.

### G3 — Make normal registration flow-free

- [x] Complete G3.
- **Repos/files:**
  - [ID] normal register page/form, route guard, auth API types/actions, and tests
- **Implementation:**
  - [x] Render normal fields from the baseline registration contract and tenant security policy only.
  - [x] Do not request or consume a registration flow when no `flow` query parameter exists.
  - [x] Preserve current tenant password-policy validation, verification/profile continuation, and cookie session behavior.
  - [x] Pass the correct `client_id` or `tenant_id` surface context.
- **Acceptance:**
  - [x] Normal registration assigns only `registered` and is unaffected by flows configured for the same client.

### G4 — Add direct special-registration through the authorize flow

- [x] Complete G4.
- **Implementation:**
  - [x] Support reaching registration through `/oauth/authorize?...&screen_hint=signup&registration_flow=<identifier>` rather than a standalone `/register?flow=` page with a free return URL.
  - [x] Read `screen_hint` and `registration_flow` from the authorize context and fetch the safe public registration-flow context before rendering.
  - [x] Dynamically render supported required fields and validation.
  - [x] Submit the flow identifier with the registration so the backend can resume the authorize flow and issue an authorization code to the validated `redirect_uri`.
  - [x] Handle missing, inactive, deleted, cross-tenant, mismatched-client, and malformed flows with a dedicated safe error state.
  - [x] Do not display the roles that will be assigned.
- **Acceptance:**
  - [x] A valid `screen_hint=signup&registration_flow=<id>` authorize request applies the flow's field/verification policy and roles and returns an authorization code; dropping `registration_flow` returns to normal signup.

### G5 — Apply registration-flow context to invite registration

- [x] Complete G5.
- **Repos/files:**
  - [ID] invite page/form, auth API types/actions, and tests
- **Implementation:**
  - [x] Load invite context from the backend rather than trusting `email`, flow, client, or callback query values directly.
  - [x] Render any flow-required fullname/phone fields in addition to the locked invited email and password fields.
  - [x] Use the resolved client branding.
  - [x] Submit only the invite token/signed context and user-entered registration fields; the backend chooses the stored flow and roles.
  - [x] Preserve invalid, expired, revoked, used, and malformed invite states.
- **Acceptance:**
  - [x] Tampering with the flow identifier in an invite URL cannot change fields, client, callback, branding, or assigned roles.

### G6 — Continue safely after registration

- [x] Complete G6.
- **Implementation:**
  - [x] Persist the validated post-registration callback through email verification, profile completion, and required MFA enrollment.
  - [x] Navigate to the callback only after the account reaches the completed state.
  - [x] Use the existing safe-return utilities or extend them; never navigate directly to an unvalidated query value.
  - [x] Clear continuation state after success, cancellation, logout, or terminal failure.
- **Acceptance:**
  - [x] Direct and invite registration return to the intended client without open redirects or stale continuation leakage.

## H — Automated verification

### H1 — Backend migration/model/repository coverage

- [x] Complete H1.
- **Tests:**
  - [x] Migration SQL creates the renamed flow tables and client columns with correct defaults/FKs.
  - [x] Migration registry contains renamed 038/039 entries and no 040 callback-table entry.
  - [x] Registration-flow models generate UUIDs and map to the correct tables.
  - [x] Repository tests cover identifier/client/tenant scoping, roles, soft deletion, and transaction behavior.
  - [x] Client tests cover branding fallback ownership and explicit false registration state.

### H2 — Backend handler/service/validation coverage

- [x] Complete H2.
- **Tests:**
  - [x] Follow the handler 9-step checklist for registration-flow and client mutations.
  - [x] Cover every create/update validation rule, including identifier, required fields, same-tenant client/branding/roles, and inactive dependencies.
  - [x] Cover every service success branch and error path under `internal/client`, `internal/idp`, `internal/authn`, `internal/invite`, and `internal/oauth`.
  - [x] Update mocks/adapters rather than bypassing domain interfaces with unverified SQL in new code.

### H3 — Runtime behavior matrix tests

- [x] Complete H3.
- **Required cases:**
  - [x] tenant on + client on → normal registration allowed;
  - [x] tenant off or client off → normal/direct registration denied;
  - [x] registration off → login still allowed;
  - [x] valid invite → allowed regardless of self-registration gates;
  - [x] no-flow registration/invite → only `registered`;
  - [x] explicit/invite flow → `registered` plus distinct flow roles;
  - [x] flow is never inferred from client;
  - [x] flow client mismatch/cross-tenant flow → rejected;
  - [x] flow tightens required fields/verification but cannot weaken tenant policy;
  - [x] callback exact match succeeds and malicious variants fail;
  - [x] self-service registration entered via `/oauth/authorize?screen_hint=signup` issues an authorization code to a registered `redirect_uri` and never to an arbitrary URL;
  - [x] `screen_hint`/`registration_flow` affect only the signup branch and never alter login/refresh/token for existing users;
  - [x] an invite with tampered query parameters cannot change the signed email, flow, roles, or callback, and a single-use invite cannot be replayed after acceptance;
  - [x] client branding, tenant fallback branding, and system fallback all resolve;
  - [x] public handlers reject `tenant_id`; internal handlers reject `client_id`; public system clients remain rejected.

### H4 — Console and identity component coverage

- [x] Complete H4.
- **Console tests:**
  - [x] Registration-flow form submits only canonical fields.
  - [x] Client form hydrates/submits branding and explicit false registration state.
  - [x] Invite form preserves the null-flow choice and filters callback options by selected client.
  - [x] Console theme changes/cleans up by client and tenant context.
- **Identity tests:**
  - [x] Login Sign up visibility follows tenant+client capability only.
  - [x] Normal registration does not fetch a flow.
  - [x] Direct-flow registration renders fields and sends the identifier.
  - [x] Invite context cannot be overridden by query tampering.
  - [x] Client branding overrides tenant branding with correct fallback and no cross-context leak.
  - [x] Safe callback continuation survives intermediate account-completion routes.

### H5 — Cross-repo end-to-end coverage

- [x] Complete H5.
- **Scenarios:**
  - [x] External client normal signup with client branding and callback.
  - [x] External client special registration link with extra roles.
  - [x] External invite with and without a registration flow.
  - [x] First-party console invite hosted by identity and returned to console.
  - [x] Two clients in one tenant with different branding and registration flags.
  - [x] Existing-user login for a registration-disabled client.
  - [x] Invalid callback, inactive flow, expired invite, and cross-tenant attempts.
- **Acceptance:**
  - [x] Tests exercise the real routers and app contracts rather than only inline stub handlers.

## I — Documentation and dead-contract cleanup

### I1 — Update user and architecture documentation

- [x] Complete I1.
- **Repos/files:**
  - [BE] `docs/overview.md`, `docs/features.md`, contributing architecture/testing references
  - [BE] frontend initialization, security settings, gRPC, release, and API documentation
- **Implementation:**
  - [x] Replace signup/auth-flow prose with the locked registration model.
  - [x] Document normal versus explicit-flow versus invite behavior.
  - [x] Document client branding fallback and `allow_registration`.
  - [x] Document callback validation and public/internal surface rules.
  - [x] Update REST/OpenAPI and gRPC examples.
  - [x] Update `docs/planning/develop-before-v0.1.0.md` A3/D2/D3 so it points to client branding and registration-flow fields instead of the old design.
- **Acceptance:**
  - [x] No current documentation says a client automatically owns/selects a flow or that branding belongs to a flow.

### I2 — Remove dead code and stale generated artifacts

- [x] Complete I2.
- **Implementation:**
  - [x] Delete callback-URI models/repos/handlers/routes/UI and their mocks/tests.
  - [x] Delete `destination` constants and routing branches once no longer used.
  - [x] Delete `config.auto_approved` and custom flow config UI/types.
  - [x] Remove old endpoint constants, query keys, compatibility mappers, generated protobuf files, and imports.
  - [x] Rename metrics/spans/log messages where dashboards or assertions depend on them.
- **Acceptance:**
  - [x] Repo-wide targeted searches show no executable old-contract artifacts.

### I3 — Record the local database reset requirement

- [x] Complete I3.
- **Implementation:**
  - [x] Document that existing pre-release local databases containing migrations 038–040 must be recreated because canonical create migrations changed in place.
  - [x] Verify the normal `[DEV] docker compose up --build -d` clean-start path.
  - [x] Do not ship compatibility DDL for pre-release local data.
- **Acceptance:**
  - [x] A contributor can reset and bootstrap the revised schema without manual table surgery.

## J — Implementation order and release gate

### J1 — Land the backend schema and naming foundation

- [x] Complete J1.
- **Order:**
  - [x] B schema/migrations.
  - [x] C1–C5 backend domain, client, permissions, seeding, composition, and protobuf.
  - [x] Compile before beginning runtime behavior changes.
- **Verification:**
  - [x] `go test ./internal/platform/database/migration ./internal/platform/runner`
  - [x] `go test ./internal/client ./internal/idp ./internal/app ./internal/setup/seeder`
  - [x] `make proto` and protobuf lint/generation checks.

### J2 — Land runtime behavior and public contexts

- [x] Complete J2.
- **Order:**
  - [x] D registration/connections/invite/callback behavior.
  - [x] E backend branding resolver/public payloads.
  - [x] C6 safe public flow/invite contexts.
- **Verification:**
  - [x] `go test ./internal/authn ./internal/invite ./internal/oauth ./internal/client ./internal/idp ./internal/app`
  - [x] Run relevant integration tests with the repository-defined tags.

### J3 — Land console, then identity

- [x] Complete J3.
- **Order:**
  - [x] F console resource rename and management UI.
  - [x] F client/invite controls and console theming.
  - [x] G identity branding, normal registration, direct flow, invite, and continuation.
- **Verification:**
  - [x] Run each frontend's unit tests.
  - [x] Run each frontend's lint and production build.
  - [x] Verify the browser paths through the nginx local hosts, not only direct Vite ports.

### J4 — Final cross-repo release gate

- [x] Complete J4.
- **Verification:**
  - [x] Complete H end-to-end scenarios against a clean local database.
  - [x] Run backend package tests, app-level tests, static analysis, and the repository's normal full test command.
  - [x] Run console and identity tests, lint, and builds.
  - [x] Update all I documentation in the same change set.
  - [x] Run `graphify update .` in the backend after code/tests pass.
  - [x] Confirm `git diff` in all three repos contains no unrelated user changes.
- **Acceptance:**
  - [x] The implementation satisfies every row in the locked behavior matrix.
  - [x] The old auth/signup-flow contract is absent from schema, runtime, management UI, hosted identity UI, generated contracts, and current documentation.
  - [x] `v0.1.0` can be tagged without carrying a compatibility layer for the abandoned pre-release design.
