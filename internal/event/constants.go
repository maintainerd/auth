package event

// Category constants for integration event types.
const (
	CategoryUser    = "USER"
	CategoryTenant  = "TENANT"
	CategoryRole    = "IAM"
	CategoryClient  = "CLIENT"
	CategorySession = "SESSION"
	CategoryService = "SERVICE"
	CategoryAPI     = "API"
	CategoryPolicy  = "IAM"
	CategoryIAM     = "IAM"
)

// Integration event type constants — canonical catalog.
const (
	// Group 1 — User identity
	EventTypeUserCreated      = "user.created"
	EventTypeUserUpdated      = "user.updated"
	EventTypeUserStatusChanged = "user.status_changed"
	EventTypeUserDeleted      = "user.deleted"
	EventTypeUserRoleAssigned = "user.role_assigned"
	EventTypeUserRoleRemoved  = "user.role_removed"

	// Group 2 — Authorization model
	EventTypeRoleCreated           = "role.created"
	EventTypeRoleUpdated           = "role.updated"
	EventTypeRoleDeleted           = "role.deleted"
	EventTypeRolePermissionsChanged = "role.permissions_changed"
	EventTypePermissionCreated     = "permission.created"
	EventTypePermissionUpdated     = "permission.updated"
	EventTypePermissionDeleted     = "permission.deleted"
	EventTypeIAMPolicyUpdated      = "iam.policy.updated"
	EventTypePolicyCreated         = "policy.created"
	EventTypePolicyDeleted         = "policy.deleted"
	EventTypeIAMServicePolicyAssigned  = "iam.service.policy.assigned"
	EventTypeIAMServicePolicyRemoved   = "iam.service.policy.removed"

	// Group 3 — Tenant / organization
	EventTypeTenantCreated      = "tenant.created"
	EventTypeTenantUpdated      = "tenant.updated"
	EventTypeTenantStatusChanged = "tenant.status_changed"
	EventTypeTenantDeleted      = "tenant.deleted"
	EventTypeTenantMemberAdded  = "tenant_member.added"
	EventTypeTenantMemberRemoved = "tenant_member.removed"

	// Group 4 — OAuth clients & credentials
	EventTypeClientCreated       = "client.created"
	EventTypeClientUpdated       = "client.updated"
	EventTypeClientDeleted       = "client.deleted"
	EventTypeClientStatusChanged = "client.status_changed"
	EventTypeClientSecretRotated = "client.secret_rotated"
	EventTypeAPIKeyCreated       = "api_key.created"
	EventTypeAPIKeyStatusChanged = "api_key.status_changed"
	EventTypeAPIKeyRevoked       = "api_key.revoked"

	// Group 5 — Sessions, identities & service principals
	EventTypeSessionRevoked        = "session.revoked"
	EventTypeTokenRevoked          = "token.revoked"
	EventTypeIdentityLinked        = "identity.linked"
	EventTypeIdentityUnlinked      = "identity.unlinked"
	EventTypeAPICreated            = "api.created"
	EventTypeAPIUpdated            = "api.updated"
	EventTypeAPIStatusChanged      = "api.status_changed"
	EventTypeAPIDeleted            = "api.deleted"
	EventTypeServiceCreated        = "service.created"
	EventTypeServiceUpdated        = "service.updated"
	EventTypeServiceStatusChanged  = "service.status_changed"
	EventTypeServiceDeleted        = "service.deleted"
)

// AllEventTypes returns the full v1.0.0 catalog of integration event types.
func AllEventTypes() []EventTypeSpec {
	return []EventTypeSpec{
		// Group 1 — User identity
		{Key: EventTypeUserCreated, Category: CategoryUser, Description: "User created", Version: 1},
		{Key: EventTypeUserUpdated, Category: CategoryUser, Description: "Identity/profile fields changed", Version: 1},
		{Key: EventTypeUserStatusChanged, Category: CategoryUser, Description: "User activated/suspended/locked", Version: 1},
		{Key: EventTypeUserDeleted, Category: CategoryUser, Description: "User deleted", Version: 1},
		{Key: EventTypeUserRoleAssigned, Category: CategoryUser, Description: "Role assigned to user", Version: 1},
		{Key: EventTypeUserRoleRemoved, Category: CategoryUser, Description: "Role removed from user", Version: 1},

		// Group 2 — Authorization model
		{Key: EventTypeRoleCreated, Category: CategoryIAM, Description: "Role created", Version: 1},
		{Key: EventTypeRoleUpdated, Category: CategoryIAM, Description: "Role updated", Version: 1},
		{Key: EventTypeRoleDeleted, Category: CategoryIAM, Description: "Role deleted", Version: 1},
		{Key: EventTypeRolePermissionsChanged, Category: CategoryIAM, Description: "Role permissions changed", Version: 1},
		{Key: EventTypePermissionCreated, Category: CategoryIAM, Description: "Permission created", Version: 1},
		{Key: EventTypePermissionUpdated, Category: CategoryIAM, Description: "Permission updated", Version: 1},
		{Key: EventTypePermissionDeleted, Category: CategoryIAM, Description: "Permission deleted", Version: 1},
		{Key: EventTypeIAMPolicyUpdated, Category: CategoryIAM, Description: "Policy updated", Version: 1},
		{Key: EventTypePolicyCreated, Category: CategoryIAM, Description: "Policy created", Version: 1},
		{Key: EventTypePolicyDeleted, Category: CategoryIAM, Description: "Policy deleted", Version: 1},
		{Key: EventTypeIAMServicePolicyAssigned, Category: CategoryIAM, Description: "Service-policy link assigned", Version: 1},
		{Key: EventTypeIAMServicePolicyRemoved, Category: CategoryIAM, Description: "Service-policy link removed", Version: 1},

		// Group 3 — Tenant / organization
		{Key: EventTypeTenantCreated, Category: CategoryTenant, Description: "Tenant created", Version: 1},
		{Key: EventTypeTenantUpdated, Category: CategoryTenant, Description: "Tenant attributes changed", Version: 1},
		{Key: EventTypeTenantStatusChanged, Category: CategoryTenant, Description: "Tenant activated/suspended", Version: 1},
		{Key: EventTypeTenantDeleted, Category: CategoryTenant, Description: "Tenant deleted", Version: 1},
		{Key: EventTypeTenantMemberAdded, Category: CategoryTenant, Description: "Member added to tenant", Version: 1},
		{Key: EventTypeTenantMemberRemoved, Category: CategoryTenant, Description: "Member removed from tenant", Version: 1},

		// Group 4 — OAuth clients & credentials
		{Key: EventTypeClientCreated, Category: CategoryClient, Description: "Client created", Version: 1},
		{Key: EventTypeClientUpdated, Category: CategoryClient, Description: "Client updated", Version: 1},
		{Key: EventTypeClientDeleted, Category: CategoryClient, Description: "Client deleted", Version: 1},
		{Key: EventTypeClientStatusChanged, Category: CategoryClient, Description: "Client enabled/disabled", Version: 1},
		{Key: EventTypeClientSecretRotated, Category: CategoryClient, Description: "Client secret rotated", Version: 1},
		{Key: EventTypeAPIKeyCreated, Category: CategoryClient, Description: "API key created", Version: 1},
		{Key: EventTypeAPIKeyStatusChanged, Category: CategoryClient, Description: "API key enabled/disabled", Version: 1},
		{Key: EventTypeAPIKeyRevoked, Category: CategoryClient, Description: "API key revoked", Version: 1},

		// Group 5 — Sessions, identities & service principals
		{Key: EventTypeSessionRevoked, Category: CategorySession, Description: "Session revoked", Version: 1},
		{Key: EventTypeTokenRevoked, Category: CategorySession, Description: "Token revoked", Version: 1},
		{Key: EventTypeIdentityLinked, Category: CategoryUser, Description: "External identity linked", Version: 1},
		{Key: EventTypeIdentityUnlinked, Category: CategoryUser, Description: "External identity unlinked", Version: 1},
		{Key: EventTypeAPICreated, Category: CategoryAPI, Description: "API created", Version: 1},
		{Key: EventTypeAPIUpdated, Category: CategoryAPI, Description: "API updated", Version: 1},
		{Key: EventTypeAPIStatusChanged, Category: CategoryAPI, Description: "API status changed", Version: 1},
		{Key: EventTypeAPIDeleted, Category: CategoryAPI, Description: "API deleted", Version: 1},
		{Key: EventTypeServiceCreated, Category: CategoryService, Description: "Service created", Version: 1},
		{Key: EventTypeServiceUpdated, Category: CategoryService, Description: "Service updated", Version: 1},
		{Key: EventTypeServiceStatusChanged, Category: CategoryService, Description: "Service status changed", Version: 1},
		{Key: EventTypeServiceDeleted, Category: CategoryService, Description: "Service deleted", Version: 1},
	}
}

// EventTypeSpec describes a single integration event type for seeding and discovery.
type EventTypeSpec struct {
	Key         string
	Category    string
	Description string
	Version     int
}
