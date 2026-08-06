package iam

import (
	"regexp"
	"strings"

	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/go-ozzo/ozzo-validation/v4/is"
	"github.com/maintainerd/maintainerd-auth/internal/shared"
)

// permissionNameFormat is the shape of a permission name: two to four
// colon-separated lowercase segments, e.g. "reports:read" or "users:read:own".
//
// PermissionMiddleware matches route guards on the permission NAME as an exact
// string, so the name is not a label — it is the authorization token itself.
// Pinning the format keeps a tenant-created name inside one predictable namespace
// instead of letting it be any 3-50 character string.
var permissionNameFormat = regexp.MustCompile(`^[a-z0-9]+([_-][a-z0-9]+)*(:[a-z0-9]+([_-][a-z0-9]+)*){1,3}$`)

// reservedPermissionNamespaces are the first segments the seeder allocates
// (internal/setup/seeder/004_permission.go). They must not be mintable through the
// permission API: an admin holding `permission:create` could otherwise create a
// permission literally named "tenant:delete" under their own tenant and satisfy
// every route guard that requires it — a self-service privilege escalation, since
// the guard compares names and never asks who created the row.
//
// The first segment is the unit of reservation because that is the unit the seeder
// allocates. Names outside these namespaces ("reports:read", "users:read:own") stay
// available; note "user" is reserved but the plural "users" is not.
var reservedPermissionNamespaces = map[string]struct{}{
	"account": {}, "api": {}, "audit": {}, "auth_event": {}, "branding": {},
	"client": {}, "email": {}, "email-config": {}, "email-template": {},
	"idp": {}, "ip-restriction-rule": {}, "notification": {}, "permission": {},
	"policy": {}, "public": {}, "registration-flow": {}, "role": {}, "root": {},
	"security": {}, "security-setting": {}, "service": {}, "settings": {},
	"sms-config": {}, "sms-template": {}, "system": {}, "tenant": {},
	"tenant-setting": {}, "user": {}, "webhook-endpoint": {},
	"workload-identity-federation": {},
}

// permissionNameRules are the name rules shared by create and update. Update
// carries them too: without it a caller creates "reports:read" and then renames it
// to "tenant:delete", which is the same escalation in two calls.
func permissionNameRules() []validation.Rule {
	return []validation.Rule{
		validation.Required.Error("Name is required"),
		validation.Length(3, 50).Error("Name must be between 3 and 50 characters"),
		validation.Match(permissionNameFormat).
			Error("Name must be 2 to 4 lowercase colon-separated segments, e.g. 'reports:read' or 'users:read:own'"),
		validation.By(rejectReservedPermissionNamespace),
	}
}

func rejectReservedPermissionNamespace(value any) error {
	name, _ := value.(string)
	namespace, _, found := strings.Cut(strings.ToLower(strings.TrimSpace(name)), ":")
	if !found {
		// Shape is the format rule's problem; report it once, there.
		return nil
	}
	if _, reserved := reservedPermissionNamespaces[namespace]; reserved {
		return validation.NewError("validation_reserved_namespace",
			"Name may not use the reserved '"+namespace+":' namespace")
	}
	return nil
}

func (r PermissionCreateRequestDTO) Validate() error {
	return validation.ValidateStruct(&r,
		validation.Field(&r.Name, permissionNameRules()...),
		validation.Field(&r.Description,
			validation.Length(0, 200).Error("Description must be at most 200 characters"),
		),
		validation.Field(&r.Status,
			validation.Required.Error("Status is required"),
			validation.In(shared.StatusActive, shared.StatusInactive).Error("Status must be one of: active, inactive"),
		),
		validation.Field(&r.APIUUID,
			validation.Required.Error("API ID is required"),
			is.UUID.Error("API ID must be a valid UUID"),
		),
	)
}

func (r PermissionUpdateRequestDTO) Validate() error {
	return validation.ValidateStruct(&r,
		validation.Field(&r.Name, permissionNameRules()...),
		validation.Field(&r.Description,
			validation.Length(0, 200).Error("Description must be at most 200 characters"),
		),
		validation.Field(&r.Status,
			validation.Required.Error("Status is required"),
			validation.In(shared.StatusActive, shared.StatusInactive).Error("Status must be one of: active, inactive"),
		),
	)
}

func (f PermissionFilterDTO) Validate() error {
	return validation.ValidateStruct(&f,
		validation.Field(&f.Status,
			validation.When(f.Status != nil,
				validation.In(shared.StatusActive, shared.StatusInactive).Error("Status must be 'active' or 'inactive'"),
			),
		),
		validation.Field(&f.PaginationRequestDTO),
	)
}

func (r PermissionStatusUpdateDTO) Validate() error {
	return validation.ValidateStruct(&r,
		validation.Field(&r.Status,
			validation.Required.Error("Status is required"),
			validation.In(shared.StatusActive, shared.StatusInactive).Error("Status must be one of: active, inactive"),
		),
	)
}
