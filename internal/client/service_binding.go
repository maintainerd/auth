package client

import (
	"github.com/google/uuid"
	"github.com/maintainerd/maintainerd-auth/internal/platform/apperror"
	"github.com/maintainerd/maintainerd-auth/internal/shared"
	"gorm.io/gorm"
)

// boundService is a read projection of the services row this package needs. The
// client package does not import iam, so it reads the columns it needs directly.
type boundService struct {
	ServiceID   int64     `gorm:"column:service_id"`
	ServiceUUID uuid.UUID `gorm:"column:service_uuid"`
	TenantID    int64     `gorm:"column:tenant_id"`
	Name        string    `gorm:"column:name"`
	Status      string    `gorm:"column:status"`
}

func (boundService) TableName() string { return "services" }

// resolveServiceBinding turns the requested service UUID into the internal id to
// store on the client, enforcing the rules that make the binding safe.
//
// Binding a client to a service is what makes its tokens carry the `svc` claim, and
// `svc` is the principal the policy bundle and the gRPC authorizer resolve. It is
// therefore a privilege grant, not a label:
//
//   - the service must belong to the SAME tenant, or a client could authenticate as
//     another tenant's principal;
//   - the service must be ACTIVE, mirroring the exchange path, which refuses an
//     inactive principal;
//   - only an m2m client may be bound. A public client (spa, mobile) holds no
//     secret — binding one would let anyone holding the public client_id mint
//     tokens for the service. A traditional client is a user-facing app, not a
//     machine principal.
//
// Returns (nil, nil) when no binding was requested, and (nil, nil) for an explicit
// empty string, which means "unbind".
func resolveServiceBinding(tx *gorm.DB, tenantID int64, clientType string, serviceUUID *string) (*int64, error) {
	if serviceUUID == nil || *serviceUUID == "" {
		return nil, nil
	}

	if clientType != shared.ClientTypeM2M {
		return nil, apperror.NewValidation(
			"only an m2m client can act as a service: a public client cannot keep a credential, " +
				"and a traditional client is a user-facing application")
	}

	parsed, err := uuid.Parse(*serviceUUID)
	if err != nil {
		return nil, apperror.NewValidation("service_id must be a valid UUID")
	}

	var service boundService
	err = tx.Model(&boundService{}).
		Where("service_uuid = ? AND tenant_id = ? AND deleted_at IS NULL", parsed, tenantID).
		First(&service).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, apperror.NewNotFoundWithReason("service not found or access denied")
		}
		return nil, err
	}
	if service.Status != shared.StatusActive {
		return nil, apperror.NewValidation("the service is not active")
	}

	return &service.ServiceID, nil
}

// serviceUUIDForClient renders the bound service as its public UUID for responses.
// The join is preloaded where available; when it is not, the binding is reported as
// absent rather than leaking the internal id.
func serviceUUIDForClient(c *Client) *string {
	if c == nil || c.ServiceID == nil || c.Service == nil {
		return nil
	}
	uuidStr := c.Service.ServiceUUID.String()
	return &uuidStr
}
