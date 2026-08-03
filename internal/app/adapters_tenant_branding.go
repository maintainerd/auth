package app

import (
	"context"
	"strings"

	"github.com/maintainerd/maintainerd-auth/internal/branding"
	"github.com/maintainerd/maintainerd-auth/internal/client"
)

type tenantClientBrandingReader struct {
	clientRepo      client.ClientRepository
	brandingService branding.BrandingService
}

func newTenantClientBrandingReader(clientRepo client.ClientRepository, brandingService branding.BrandingService) *tenantClientBrandingReader {
	return &tenantClientBrandingReader{clientRepo: clientRepo, brandingService: brandingService}
}

func (r *tenantClientBrandingReader) GetPublicClientBranding(ctx context.Context, tenantID int64, clientIdentifier string) (*branding.BrandingServiceDataResult, error) {
	if r == nil || r.clientRepo == nil || r.brandingService == nil || tenantID <= 0 {
		return nil, nil
	}

	clientIdentifier = strings.TrimSpace(clientIdentifier)
	if clientIdentifier == "" {
		return nil, nil
	}

	c, err := r.clientRepo.FindByIdentifier(clientIdentifier)
	if err != nil || c == nil {
		return nil, err
	}
	if c.TenantID != tenantID || c.BrandingID == nil {
		return nil, nil
	}

	return r.brandingService.GetPublicByID(ctx, tenantID, *c.BrandingID)
}
