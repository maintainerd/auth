package app

import (
	"context"

	"github.com/maintainerd/maintainerd-auth/internal/oauth"
	"gorm.io/gorm"
)

// oauthDPoPRequirementResolver resolves a client's DPoP requirement by its
// presented client_id (identifier) for the token endpoint's RFC 9449 §8 nonce
// gate. It queries the clients table directly so the oauth handler stays
// decoupled from the client domain.
type oauthDPoPRequirementResolver struct {
	db *gorm.DB
}

func newOAuthDPoPRequirementResolver(db *gorm.DB) oauth.DPoPRequirementResolver {
	return &oauthDPoPRequirementResolver{db: db}
}

func (a *oauthDPoPRequirementResolver) ResolveDPoPRequirement(_ context.Context, clientID string) (oauth.DPoPRequirement, bool) {
	if clientID == "" {
		return oauth.DPoPRequirement{}, false
	}
	var row struct {
		DPoPRequired bool
		TenantID     int64
		ClientID     int64
	}
	err := a.db.Table("clients").
		Select("dpop_required, tenant_id, client_id").
		Where("identifier = ? AND status = ? AND deleted_at IS NULL", clientID, "active").
		Take(&row).Error
	if err != nil {
		return oauth.DPoPRequirement{}, false
	}
	return oauth.DPoPRequirement{
		Required:         row.DPoPRequired,
		TenantID:         row.TenantID,
		InternalClientID: row.ClientID,
	}, true
}
