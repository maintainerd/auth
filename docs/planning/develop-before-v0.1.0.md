# Develop Before v0.1.0

This document is the implementation tracker for all work that must be completed before tagging `v0.1.0`. Every item below is in scope and has a defined goal, file scope, implementation checklist, and acceptance criterion.

## How to use this tracker

- Leave an item unchecked while any implementation step or acceptance criterion remains incomplete.
- Check each nested task as it is completed.
- Check the parent item only after its acceptance criterion has been verified.
- Run the mandatory lint, commit, and release work in G6 last.

## Related tracker ownership and sequencing

- `registration-flows.md` is the schema and naming refactor that moves branding and `allow_registration` to the client and renames the flow domain. Complete its phases B–E before dependent client-branding and registration-flow UI work referenced by this tracker.
- Branding-on-client and registration-flow correctness are owned by `registration-flows.md`. This tracker's A3, A6, D2, and D3 are retained as superseded pointers; the remaining B, C, E, F, and G items are independent `v0.1.0` gaps.
- The registration entry contract is owned by `registration-flows.md` A3 and D8: self-service registration is an entry mode of the OAuth2 authorization-code flow (`/oauth/authorize?...&screen_hint=signup`, optional `registration_flow=<identifier>`), completing by issuing an authorization code to the client's registered `redirect_uri`. Only invites use a signed, expiring URL whose token is the sole authority. No standalone registration page may accept an arbitrary `callback_url`.
- A1's completed CSS-variable injection mechanism remains valid. After the refactor, its branding input comes from the client resolver defined by `registration-flows.md` E1, with tenant fallback.

## Progress summary

- [ ] A — Branding, theming, and login customization (4/6 remaining; A3/A6 owned by `registration-flows.md`)
- [ ] B — MFA enrollment (1/1 remaining)
- [ ] C — Admin operability (5/5 remaining)
- [ ] D — Details read-back and form/DTO mismatch (3/3 remaining; D2/D3 owned by `registration-flows.md`)
- [ ] E — End-user flows (4/4 remaining)
- [ ] F — IAM (1/1 remaining)
- [ ] G — Cleanups and hardening (6/6 remaining)
- [ ] All pre-v0.1.0 work is complete

## A — Branding, theming, and login customization

### A1 — Apply branding colors in the hosted login

- [x] Complete A1.
- **Model note:** The color/font/background injection implemented here is unchanged. The branding applied by this mechanism is resolved through the client per `registration-flows.md` E1, with tenant-active branding as fallback.
- **Repos/files:**
  - [ID] `maintainerd-auth-identity/src/components/auth/AppBootstrap.tsx` (branding is already fetched through `/tenant/{identifier}` and exposed as `TenantEntity.branding.metadata.colors`)
  - [ID] `maintainerd-auth-identity/src/components/layout/LoginLayout.tsx`
  - [ID] `maintainerd-auth-identity/src/index.css`
- **Implementation:**
  - [x] During bootstrap, read `branding.metadata.colors` and the configured font.
  - [x] Add a small `applyBranding(colors)` helper that injects the values as CSS custom properties on `:root`, using `document.documentElement.style.setProperty(...)` (for example, `--primary`, `--background`, and the other supported tokens).
  - [x] Refactor `index.css` so hardcoded color tokens use `var(--token, <fallback>)`, allowing injected values to override the defaults.
  - [x] Apply the configured branding gradient/background as well as the individual color tokens.
- **Acceptance:**
  - [x] Changing a tenant's colors in the console visibly changes the hosted login, not only the logo.

### A2 — Login layout options (centered / full-page / split)

- [x] Complete A2.
- **Repos/files:**
  - [BE] `internal/branding` model and original migration `003` (edit in place to add the column)
  - [BE] `internal/shared/constants.go` (existing `LoginTemplate*` constants)
  - [BE] Tenant public branding DTO: `internal/tenant/types.go` (`BrandingPublic`)
  - [CON] `maintainerd-auth-console/src/pages/branding/templates/form/BrandingForm.tsx`
  - [CON] `maintainerd-auth-console/src/services/api/branding/types.ts`
  - [ID] `maintainerd-auth-identity/src/components/layout/LoginLayout.tsx`
- **Implementation:**
  - [x] Add a `layout VARCHAR` column to the branding table.
  - [x] Support exactly these values: `centered`, `full_page`, and `split`.
  - [x] Default the layout to `centered`.
  - [x] Add `layout` to the branding model.
  - [x] Add `layout` to the branding create and update DTOs.
  - [x] Add `layout` to the public branding payload.
  - [x] Add a layout `<Select>` to the console branding form.
  - [x] Make the identity app's `LoginLayout` select its rendering from `branding.layout`.
  - [x] Implement the three layouts: centered card, full-page, and split-screen with a brand panel.
  - [x] Ensure all approximately 18 authentication pages render inside the selected layout.
- **Acceptance:**
  - [x] Selecting a layout in the console changes how the hosted login renders for that tenant.

### A3 — Per-client branding resolution (superseded)

- [ ] Complete through `registration-flows.md` E1 and E2.
- **Ownership:** Client-owned branding resolution, tenant fallback, and the `/oauth/connections` branding payload are implemented and accepted in that tracker. Do not resolve branding through a registration flow.

### A4 — Logo storage (database blob behind a URL)

- [ ] Complete A4.
- **Repos/files:**
  - [BE] `internal/branding` model, original migration, and a logo-serving endpoint
  - [CON] `maintainerd-auth-console/src/pages/branding/templates/form/BrandingForm.tsx`
  - [CON] `maintainerd-auth-console/src/services/api/branding/`
- **Implementation:**
  - [ ] Store logo bytes either in branding columns or a branding-assets row.
  - [ ] Add `logo_data BYTEA`.
  - [ ] Add `logo_content_type VARCHAR`.
  - [ ] Add `GET /public/branding/{branding_id}/logo`.
  - [ ] Stream stored bytes from the endpoint with `Content-Type`, `ETag`, and `Cache-Control` headers.
  - [ ] On upload, store the bytes and set `logo_url` to the serving endpoint.
  - [ ] Enforce a maximum logo size of 256 KB.
  - [ ] Allow PNG, JPEG, and WebP.
  - [ ] Reject SVG.
  - [ ] Add a file-upload control beside the existing URL field in the console.
  - [ ] Continue accepting external logo URLs.
- **Acceptance:**
  - [ ] An admin can upload a PNG, the logo is stored in the database and served through the endpoint, `logo_url` points to it, and the login renders it.
  - [ ] External logo URLs still work.

### A5 — Fix `/public/branding` tenant resolution

- [ ] Complete A5.
- **Repos/files:**
  - [BE] `internal/branding/handler_branding.go` (`GetPublic(ctx, 1)` currently hardcodes tenant 1)
- **Implementation:**
  - [ ] Remove the hardcoded tenant 1 lookup.
  - [ ] Resolve the tenant from the request through the host/subdomain or a `tenant_id`/`client_id` query parameter.
  - [ ] Fall back to the system tenant when the request cannot be resolved to a tenant.
- **Acceptance:**
  - [ ] `/public/branding` returns the correct tenant's branding instead of always returning tenant 1.

### A6 — Console self-theming from branding (superseded)

- [ ] Complete through `registration-flows.md` E4 and F7.
- **Ownership:** Console theming resolves the console system client's branding, with tenant-active fallback, and applies it across the console. Do not theme the console directly from tenant branding when an explicit console-client branding exists.

## B — MFA enrollment (build it; keep requirable MFA enabled)

### B1 — MFA enrollment UI in the hosted login

- [ ] Complete B1.
- **Repos/files:**
  - [ID] `maintainerd-auth-identity/src/services/api/mfa.ts`; all required functions already exist: `fetchMFAStatus`, `beginTOTPEnrollment`, `finishTOTPEnrollment`, `beginSMSEnrollment`, `verifySMSEnrollment`, `beginEmailOtpEnrollment`, `beginWebAuthnRegistration`, `finishWebAuthnRegistration`, `regenerateBackupCodes`, and the `disable*` functions
  - [ID] `maintainerd-auth-identity/src/App.tsx` (routes)
  - [ID] New pages under `maintainerd-auth-identity/src/pages/account/mfa/`
- **Implementation:**
  - [ ] Add an authenticated MFA management page.
  - [ ] Show a status list of the user's enrolled factors.
  - [ ] Implement TOTP enrollment with QR code display and verification.
  - [ ] Implement SMS enrollment with phone-number entry and OTP verification.
  - [ ] Implement email OTP enrollment.
  - [ ] Implement WebAuthn/passkey registration.
  - [ ] Implement backup-code regeneration and display.
  - [ ] Allow the user to disable each enrolled factor.
  - [ ] Wire every action to the existing functions in `mfa.ts`.
  - [ ] Add a post-login enrollment prompt when the tenant requires MFA but the user has no enrolled factor.
- **Acceptance:**
  - [ ] A user can enroll and disable every factor type and regenerate backup codes from the hosted app.
  - [ ] Tenants that require MFA can onboard users end to end.
  - [ ] Required MFA remains enabled; it does not need to be disabled because enrollment now exists.

## C — Admin operability (console wiring; one small backend route)

### C1 — IdP “test connection” UI

- [ ] Complete C1.
- **Repos/files:**
  - [CON] `maintainerd-auth-console/src/services/api/identity-providers/` (add a `testConnection` function and endpoint constant in `config.ts`)
  - [CON] `maintainerd-auth-console/src/pages/identity-providers/form/IdentityProviderAddOrUpdateForm.tsx`
  - [BE] Backend is already implemented at `POST /identity_providers/test` in `internal/idp/routes.go:79`
- **Implementation:**
  - [ ] Add a **Test connection** button to the IdP form.
  - [ ] POST the form's current, unsaved configuration.
  - [ ] Display every per-check result, including discovery and JWKS success/failure.
- **Acceptance:**
  - [ ] An admin can click **Test connection** and see pass/fail results before saving the provider.

### C2 — Webhook delivery history (register route and list UI)

- [ ] Complete C2.
- **Repos/files:**
  - [BE] `internal/webhook/routes.go` (register a GET route)
  - [BE] `internal/webhook/repository_delivery_history.go` (`FindByTenantID` already exists)
  - [BE] A webhook delivery-history handler
  - [CON] A service under `maintainerd-auth-console/src/services/api/`
  - [CON] A deliveries list view under the console webhook pages
- **Implementation:**
  - [ ] Add either `GET /webhook-endpoints/{id}/deliveries` or `GET /webhook-deliveries`.
  - [ ] Return paginated, tenant-scoped webhook delivery history.
  - [ ] Add a console **Deliveries** tab/list.
  - [ ] Show status, timestamp, and response code for each delivery.
  - [ ] Include a replay action/link that connects to C3.
- **Acceptance:**
  - [ ] An admin can see past webhook deliveries and whether each succeeded or failed.

### C3 — Webhook replay UI

- [ ] Complete C3.
- **Repos/files:**
  - [CON] `maintainerd-auth-console/src/services/api/` (`config.ts:95` already contains the endpoint constant)
  - [CON] The webhook deliveries view from C2
  - [BE] Backend is already implemented at `POST /webhook-replay` in `internal/webhook/handler_replay.go:59`
- **Implementation:**
  - [ ] Add a **Replay** action to each delivery row.
  - [ ] Call the existing replay endpoint.
  - [ ] Surface the replay result to the admin.
- **Acceptance:**
  - [ ] An admin can replay a failed delivery from the console.

### C4 — Audit-events export UI

- [ ] Complete C4.
- **Repos/files:**
  - [CON] `maintainerd-auth-console/src/services/api/auth-events/index.ts` (add `exportAuthEvents`)
  - [CON] The console `AuthEventListing.tsx` page
  - [BE] Backend is already implemented at `GET /auth-events/export` in `internal/authevent/routes.go:27`
- **Implementation:**
  - [ ] Add an **Export** button with CSV and JSON options.
  - [ ] Pass the listing's current filters to the export endpoint.
  - [ ] Download the returned export file.
- **Acceptance:**
  - [ ] An admin can export the filtered audit log.

### C5 — Client-to-IdP connection update UI

- [ ] Complete C5.
- **Repos/files:**
  - [CON] The client-detail **Identity Providers / Connections** UI
  - [CON] `maintainerd-auth-console/src/services/api/clients/`
  - [BE] Backend is already implemented at `PUT /clients/{uuid}/identity_providers/{uuid}` in `internal/client/routes.go:133`
- **Implementation:**
  - [ ] Add an edit control to each connected-provider row.
  - [ ] Allow `is_default`, `enabled`, and `display_order` to be changed.
  - [ ] Call the existing PUT endpoint with those changes.
  - [ ] Optionally wire `GET /clients/{uuid}/identity_providers` to refresh the list with live data.
- **Acceptance:**
  - [ ] An admin can toggle, reorder, and set a client's default connections after connecting them.

## D — Details read-back and form/DTO mismatch (console)

### D1 — IdP details: display the new fields

- [ ] Complete D1.
- **Repos/files:**
  - [CON] `maintainerd-auth-console/src/pages/identity-providers/details/components/IdentityProviderInformationTab.tsx`
  - [CON] `maintainerd-auth-console/src/pages/identity-providers/components/provider-config/providerConfigSchemas.ts` (add field keys and `connectionValue()` cases)
- **Implementation:**
  - [ ] Display `allow_token_federation` as read-only on the **Connection** tab.
  - [ ] Display `allowed_audiences` as read-only on the **Connection** tab.
  - [ ] Display `allow_registration` as read-only on the **Connection** tab. This is the identity provider's JIT/federated-registration flag and is unaffected by removal of the old flow-level flag.
  - [ ] Add a **Token Federation** badge to the IdP list column.
- **Acceptance:**
  - [ ] Opening an IdP shows the current state of all three fields.

### D2 — Registration Flow details (superseded)

- [ ] Complete through `registration-flows.md` F3.
- **Ownership:** The renamed registration-flow details page displays `verification_required` and `required_fields`; `allow_registration` belongs to the client and must not appear as a flow field.

### D3 — Remove the old flow `config`/`auto_approved` mismatch (superseded)

- [ ] Complete through `registration-flows.md` F2 and I2.
- **Ownership:** The registration-flow form cleanup and dead-contract removal are completed as part of the domain rename; do not maintain a second implementation task here.

## E — End-user flows (identity)

### E1 — SMS passwordless login

- [ ] Complete E1.
- **Repos/files:**
  - [ID] New functions under `maintainerd-auth-identity/src/services/api/`
  - [ID] A new identity login screen and route
  - [BE] Backend is already implemented at `POST /sms-login/send` and `POST /sms-login/verify` in `internal/authn/routes.go:237`
- **Implementation:**
  - [ ] Add a **Sign in with SMS** option.
  - [ ] Implement phone entry followed by code sending, code verification, and session establishment.
  - [ ] Add the required endpoint constants to `config.ts`.
  - [ ] Add the page and route.
- **Acceptance:**
  - [ ] A user can log in with a phone number and OTP without a password.

### E2 — Linked-identities management

- [ ] Complete E2.
- **Repos/files:**
  - [ID] A new identity-app service module and authenticated account page
  - [BE] Backend is already implemented at `GET`, `POST`, and `DELETE /account/identities` in `internal/idp/routes.go:32`
- **Implementation:**
  - [ ] Add an authenticated page that lists linked identities.
  - [ ] Add a link action that accepts/provides an external token.
  - [ ] Add an unlink action.
- **Acceptance:**
  - [ ] A user can view, link, and unlink external identities.

### E3 — Backup-code account recovery

- [ ] Complete E3.
- **Repos/files:**
  - [ID] A new standalone page and route in the identity app
  - [BE] Backend `RecoveryRoute` is already implemented as an unauthenticated backup-code flow
- **Implementation:**
  - [ ] Add a standalone **Use a backup code** recovery screen.
  - [ ] Keep this recovery screen distinct from the mid-login backup-code step.
- **Acceptance:**
  - [ ] A locked-out user can recover an account with a backup code.

### E4 — Account-locked and rate-limit screens

- [ ] Complete E4.
- **Repos/files:**
  - [ID] Error handling in `maintainerd-auth-identity/src/pages/.../LoginForm.tsx`
  - [ID] New dedicated identity-app error pages
- **Implementation:**
  - [ ] Detect backend account-lockout responses.
  - [ ] Detect HTTP `429` responses.
  - [ ] Show a clear **account temporarily locked** screen instead of a generic inline error.
  - [ ] Show a clear **too many attempts, try again later** screen instead of a generic inline error.
- **Acceptance:**
  - [ ] Lockout and rate-limit states render as dedicated, clear screens.

## F — IAM

### F1 — Standalone Permissions page (console)

- [ ] Complete F1.
- **Repos/files:**
  - [CON] `maintainerd-auth-console/src/App.tsx` (add a route)
  - [CON] New pages under `maintainerd-auth-console/src/pages/permissions/`
  - [CON] Existing permissions client under `maintainerd-auth-console/src/services/api/`
  - [BE] Backend already has full permission CRUD
- **Implementation:**
  - [ ] Add a top-level **Permissions** route/page in addition to API-nested permission management.
  - [ ] Implement global permission listing.
  - [ ] Implement global permission creation.
  - [ ] Implement global permission editing.
  - [ ] Implement global permission deletion.
- **Acceptance:**
  - [ ] An admin can manage permissions globally, not only under an API.

## G — Cleanups and hardening

### G1 — Remove duplicate `UpdateIdentityProviderRequest` interface

- [ ] Complete G1.
- **Repos/files:**
  - [CON] `maintainerd-auth-console/src/services/api/identity-providers/types.ts`
- **Implementation:**
  - [ ] Delete the second, incomplete `UpdateIdentityProviderRequest` declaration.
  - [ ] Keep the declaration that contains the Mode B fields.
- **Acceptance:**
  - [ ] Exactly one complete `UpdateIdentityProviderRequest` interface remains.

### G2 — Remove dead `src/services/api/apis/` directory

- [ ] Complete G2.
- **Repos/files:**
  - [CON] `maintainerd-auth-console/src/services/api/apis/`
  - [CON] The live directory is `maintainerd-auth-console/src/services/api/`
- **Implementation:**
  - [ ] Delete the unused `src/services/api/apis/` directory.
  - [ ] Find and fix any stray imports that still reference it.
- **Acceptance:**
  - [ ] The dead directory and all references to it are gone.

### G3 — Add `RequireStepUp` to API-permission routes

- [ ] Complete G3.
- **Repos/files:**
  - [BE] `internal/client/routes.go:152`
  - [BE] `internal/client/routes.go:155`
- **Implementation:**
  - [ ] Add `middleware.RequireStepUp` to the API-permission add route.
  - [ ] Add `middleware.RequireStepUp` to the API-permission remove route.
  - [ ] Match the protection used by sibling mutating routes.
- **Acceptance:**
  - [ ] Both API-permission mutation routes require step-up authentication.

### G4 — Add new fields to the backend IdP list DTO

- [ ] Complete G4.
- **Repos/files:**
  - [BE] `internal/idp/service_provider.go` list-response mapper
- **Implementation:**
  - [ ] Include `allow_registration` in the list DTO. This is the identity provider's JIT/federated-registration flag, not the removed flow-level flag.
  - [ ] Include `allow_token_federation` in the list DTO.
  - [ ] Include `allowed_audiences` in the list DTO.
  - [ ] Ensure list views and badges receive the new data.
- **Acceptance:**
  - [ ] The IdP list response exposes all three fields.

### G5 — Route `/federation/token` through `resolveFederatedPrincipal`

- [ ] Complete G5.
- **Repos/files:**
  - [BE] `internal/idp/service_federation.go` (`ExchangeExternalToken`)
  - [BE] `internal/idp/service_federated_principal.go` (`resolveFederatedPrincipal`)
- **Implementation:**
  - [ ] Refactor `ExchangeExternalToken` to call the shared `resolveFederatedPrincipal` path.
  - [ ] Mint the Maintainerd token on top of the resolved principal.
  - [ ] Preserve existing behavior.
  - [ ] Keep one shared validation and just-in-time provisioning path.
- **Acceptance:**
  - [ ] `/federation/token` uses `resolveFederatedPrincipal` without changing externally observable behavior.

### G6 — Commit and lint (mandatory, last)

- [ ] Complete G6.
- **Repos/files:**
  - [ALL] `maintainerd-auth`
  - [ALL] `maintainerd-auth-console`
  - [ALL] `maintainerd-auth-identity`
- **Implementation:**
  - [ ] Complete A1–G5 before starting G6.
  - [ ] Run `gofmt` on the backend and fix all formatting issues.
  - [ ] Run `golangci-lint` on the backend and fix all failures.
  - [ ] Run the frontend linters for both frontend repositories and fix all failures.
  - [ ] Commit the entire working tree, including tenant isolation, Mode B, registration gating, and all A–G work.
  - [ ] Tag `v0.1.0` only after every item and acceptance criterion in this document is complete.
- **Acceptance:**
  - [ ] All backend and frontend formatting/lint checks pass.
  - [ ] The complete working tree is committed.
  - [ ] `v0.1.0` is tagged after completion.

## Required execution order

- [ ] Phase 1 — A1, A4, and A5: establish the branding base.
- [ ] Phase 2 — Complete `registration-flows.md` B–E before its client-branding and console-theming work; A3 and A6 here are superseded pointers.
- [ ] Phase 3 — B1: build MFA enrollment.
- [ ] Phase 4 — C1–C5: wire admin operations; C2 includes the small backend route.
- [ ] Phase 5 — Complete D1 here and complete D2/D3 through `registration-flows.md` F2/F3/I2.
- [ ] Phase 6 — E1–E4: complete end-user identity flows.
- [ ] Phase 7 — F1: add global permission management.
- [ ] Phase 8 — G1–G5: complete cleanups and hardening.
- [ ] Phase 9 — G6: lint, commit, and tag last.

## Cross-cutting implementation rules

### Database migrations

- [ ] Edit the original create migration in place for changes to existing tables, including branding `layout`, `logo_data`, and `logo_content_type`.
- [ ] Do not create add/alter/drop/backfill migrations for existing tables while the project remains pre-release.
- [ ] If a genuinely new table is required, create a new migration and append it to the migration registry, following the create-only migration rule.

### Surface boundaries

- [ ] Keep branding and connections public endpoints on the public surface.
- [ ] Keep admin CRUD on the internal surface.

### Release scope

- [ ] Treat every item in A–G as in scope for `v0.1.0`; there are no open scope decisions.
- [ ] Build and verify every item before the final commit, lint, and tag steps.
- [ ] Commit and lint all tenant-isolation, Mode B, registration-gating, and A–G work together as required by G6.
- [ ] Tag `v0.1.0` only after the entire tracker is complete.
