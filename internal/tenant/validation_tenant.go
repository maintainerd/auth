package tenant

import (
	"regexp"

	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/maintainerd/maintainerd-auth/internal/platform/apperror"
	"github.com/maintainerd/maintainerd-auth/internal/shared"
)

// tenantNamePattern enforces a DNS-safe subdomain slug: lowercase alphanumerics
// and hyphens, must start and end with an alphanumeric. The name is used as the
// tenant subdomain ({tenant}.auth.maintainerd.local).
var tenantNamePattern = regexp.MustCompile(`^[a-z0-9]([a-z0-9-]*[a-z0-9])?$`)

// reservedTenantSlugs are names that would shadow a platform host and must never
// be assigned to a tenant. Rejecting them is security-critical: a tenant named
// "console" would otherwise resolve to the system console host.
var reservedTenantSlugs = map[string]struct{}{
	"system":      {},
	"console":     {},
	"api":         {},
	"control-api": {},
	"control":     {},
	"auth":        {},
	"www":         {},
	"admin":       {},
	"root":        {},
	"rabbitmq":    {},
	"prometheus":  {},
	"grafana":     {},
	"signoz":      {},
}

// validateTenantSlug validates a tenant name as a DNS-safe subdomain slug and
// rejects reserved platform slugs. It is enforced at the service layer (create
// and update) so both REST and gRPC creation paths are covered.
func validateTenantSlug(name string) error {
	if name == "" {
		return apperror.NewValidation("Name is required")
	}
	if len(name) > 63 {
		return apperror.NewValidation("Name must not exceed 63 characters")
	}
	if !tenantNamePattern.MatchString(name) {
		return apperror.NewValidation("Name must be a DNS-safe slug: lowercase letters, numbers, and hyphens, starting and ending with an alphanumeric")
	}
	if _, reserved := reservedTenantSlugs[name]; reserved {
		return apperror.NewValidation("Name is reserved and cannot be used")
	}
	return nil
}

// reservedSlugRule is an ozzo-validation rule that rejects reserved tenant slugs
// at the DTO layer (before the request reaches the service).
func reservedSlugRule(value any) error {
	name, _ := value.(string)
	if _, reserved := reservedTenantSlugs[name]; reserved {
		return validation.NewError("validation_reserved_slug", "Name is reserved and cannot be used")
	}
	return nil
}

func (r TenantCreateRequestDTO) Validate() error {
	return validation.ValidateStruct(&r,
		validation.Field(&r.Name,
			validation.Required.Error("Name is required"),
			validation.Length(3, 63).Error("Name must be between 3 and 63 characters"),
			validation.Match(tenantNamePattern).Error("Name must be a DNS-safe slug: lowercase letters, numbers, and hyphens, starting and ending with an alphanumeric"),
			validation.By(reservedSlugRule),
		),
		validation.Field(&r.Description,
			validation.Required.Error("Description is required"),
			validation.Length(8, 200).Error("Description must be between 8 and 200 characters"),
		),
		validation.Field(&r.Status,
			validation.Required.Error("Status is required"),
			validation.In(shared.StatusActive, shared.StatusInactive, shared.StatusPending, shared.StatusSuspended).Error("Status must be active, inactive, pending, or suspended"),
		),
	)
}

func (r TenantUpdateRequestDTO) Validate() error {
	return validation.ValidateStruct(&r,
		validation.Field(&r.Name,
			validation.Required.Error("Name is required"),
			validation.Length(3, 63).Error("Name must be between 3 and 63 characters"),
			validation.Match(tenantNamePattern).Error("Name must be a DNS-safe slug: lowercase letters, numbers, and hyphens, starting and ending with an alphanumeric"),
			validation.By(reservedSlugRule),
		),
		validation.Field(&r.Description,
			validation.Required.Error("Description is required"),
			validation.Length(8, 200).Error("Description must be between 8 and 200 characters"),
		),
		validation.Field(&r.Status,
			validation.Required.Error("Status is required"),
			validation.In(shared.StatusActive, shared.StatusInactive, shared.StatusPending, shared.StatusSuspended).Error("Status must be active, inactive, pending, or suspended"),
		),
	)
}

func (r TenantSetStatusRequestDTO) Validate() error {
	return validation.ValidateStruct(&r,
		validation.Field(&r.Status,
			validation.Required.Error("Status is required"),
			validation.In(shared.StatusActive, shared.StatusInactive, shared.StatusPending, shared.StatusSuspended).Error("Status must be active, inactive, pending, or suspended"),
		),
	)
}

func (r TenantFilterDTO) Validate() error {
	return validation.ValidateStruct(&r,
		validation.Field(&r.PaginationRequestDTO),
	)
}
