# Tenant, Membership, Ownership, and User Lifecycle

This document defines the Maintainerd Auth rules connecting tenants, tenant
members, tenant owners, and users. These rules are business invariants: every
transport and internal caller must preserve them, not only the management UI.

## Domain model

### Tenant

A tenant is an isolation boundary for identities, credentials, clients, roles,
policies, security settings, and operational configuration.

There are two tenant classes:

- **System tenant**: the root administrative tenant. Exactly one live system
  tenant may exist, and it cannot be deleted.
- **Regular tenant**: an application/customer tenant managed by its members and,
  for privileged lifecycle operations, by the system tenant.

New regular tenants are created with `is_completed = false`. Baseline roles,
permissions, clients, identity-provider configuration, and branding are seeded
transactionally, but the tenant is not complete until its first owner is
assigned. Assigning the first owner changes `is_completed` to `true`.

### User

A user is an authentication identity. Removing a user from a tenant and deleting
the user are different operations:

- Removing membership removes access to one tenant.
- Deleting a user affects the user account and its credentials according to the
  user deletion lifecycle.

A user may belong to multiple tenants through active `tenant_members` rows.

### Tenant member

`tenant_members` is the authoritative relationship between a user and a tenant.
An active relationship has one of two roles:

- `owner`
- `member`

The database permits at most one active owner per tenant through the partial
unique index `uq_tenant_members_one_owner`. It also prevents duplicate active
membership for the same tenant and user.

## Ownership invariants

1. Every completed tenant has exactly one active owner.
2. A tenant can never have two active owners.
3. An owner cannot be directly demoted to `member`.
4. An owner cannot be directly removed from `tenant_members`.
5. Ownership changes use the atomic ownership-transfer operation.
6. The system tenant owner cannot be transferred, demoted, removed, or deleted.
7. The system tenant owner is established only by the initial setup workflow.
8. A regular tenant's first owner and later ownership transfers require an actor
   who is an active member of the system tenant and whose transport identity has
   the required permission.

### Ownership transfer

Promoting an existing regular-tenant member to `owner` is the ownership-transfer
operation. In one database transaction it:

1. Resolves the current owner.
2. Demotes the current owner to `member`.
3. Revokes the current owner's tenant-scoped `super-admin` IAM role.
4. Promotes the selected member to `owner`.
5. Grants the new owner the tenant-scoped `super-admin` IAM role.
6. Commits all membership and IAM changes together.

If any step fails, the transaction rolls back. The previous owner and IAM role
remain unchanged.

Direct removal of the previous owner is allowed only after ownership has been
transferred, at which point that user is an ordinary member.

## Authorization matrix

Transport permission checks and service-level membership checks are both
required. A permission by itself does not override the system-tenant boundary.

| Operation | Regular tenant member with permission | System tenant member with permission |
| --- | --- | --- |
| Read tenant members | Yes, for own tenant | Yes |
| Add ordinary member | Yes, for own tenant | Yes |
| Remove ordinary member | Yes, for own tenant | Yes |
| Assign first owner | No | Yes |
| Transfer regular tenant ownership | No | Yes |
| Directly demote/remove an owner | No | No; transfer first |
| Change system tenant ownership | No | No |
| Delete regular tenant | No | Yes, with `tenant:delete` and step-up |
| Delete system tenant | No | No |

REST management operations use the authenticated user's ID. Mutating tenant
gRPC requests carry `actor_user_uuid`; the service resolves that UUID and
verifies the actor's active membership. gRPC authorization still requires the
service principal permission configured for the RPC.

## User deletion rules

Before deleting a user, the user service queries every active owner membership.

- If the user owns the system tenant, deletion is always rejected.
- If the user owns any regular tenant, deletion is rejected until ownership is
  transferred and the old owner is removed or retained as a member.
- A user with only ordinary memberships may be deleted subject to normal tenant
  access and permission checks.
- Removing an ordinary membership does not delete the user account.

These checks use active membership rows (`deleted_at IS NULL`). A soft-deleted
membership no longer grants tenant access or blocks user deletion.

## Tenant deletion rules

### Authorization

A regular tenant may be deleted only when all of the following are true:

1. The actor is authenticated.
2. The transport authorizes `tenant:delete`.
3. The operation has fresh step-up authentication where the transport requires
   it.
4. The actor is an active member of the system tenant.
5. The target is a regular tenant.

The tenant service repeats the system-membership and `is_system` checks inside
the deletion transaction. The system tenant cannot be deleted even if a caller
holds `tenant:delete`.

### Soft deletion and purge

Tenant deletion runs transactionally:

1. Tenant-owned runtime and credential records in the configured cascade set are
   deleted or soft-deleted according to their model.
2. The tenant row is soft-deleted by setting `deleted_at`.
3. A `tenant.deleted` integration event is written in the same transaction.
4. The retention job permanently deletes expired soft-deleted tenant rows.
5. PostgreSQL `ON DELETE CASCADE` constraints permanently remove remaining
   tenant-owned rows when the tenant row is purged.

The system tenant is excluded from the purge query as defense in depth.

## Database enforcement

The canonical create migration for `tenant_members` enforces:

- `role IN ('owner', 'member')`
- foreign keys to tenants and users with `ON DELETE CASCADE`
- one active membership per `(tenant_id, user_id)`
- one active owner per `tenant_id`

The database unique-owner index is the concurrency backstop. Service checks
provide domain-specific errors, while the index prevents two concurrent owner
assignments from both succeeding.

## Transaction and failure rules

- Membership mutation and its integration event share one transaction.
- Ownership membership changes and `super-admin` IAM changes share one
  transaction.
- Tenant cascade deletion, tenant soft deletion, and the deletion event share
  one transaction.
- Failed mutations must not leave half-applied membership or IAM state.
- Authorization is checked again by the service; handlers are not the business
  rule boundary.

## Required tests

Changes to this area must cover:

- concurrent duplicate-owner rejection by the database;
- first-owner assignment completing a regular tenant;
- regular member rejection when assigning or transferring ownership;
- successful ownership transfer by a system-tenant actor;
- rollback when owner IAM revocation or grant fails;
- rejection of system tenant ownership changes;
- rejection of direct owner demotion and removal;
- rejection of user deletion while any owner membership exists;
- successful ordinary member removal without user deletion;
- regular tenant deletion by a system-tenant actor;
- rejection of regular-tenant actors deleting tenants;
- unconditional rejection of system tenant deletion;
- equivalent REST and gRPC service-level outcomes.

