package branding

import (
	"context"

	"github.com/google/uuid"
	"github.com/maintainerd/auth/internal/platform/apperror"
	authv1 "github.com/maintainerd/auth/internal/platform/gen/go/maintainerd/auth"
	"github.com/maintainerd/auth/internal/platform/pagination"
	"google.golang.org/protobuf/types/known/timestamppb"
	"gorm.io/datatypes"
)

type TenantResolver interface {
	GetByUUID(ctx context.Context, tenantUUID uuid.UUID) (*TenantServiceDataResult, error)
}

type BrandingGRPCHandler struct {
	authv1.UnimplementedBrandingServiceServer
	tenantResolver TenantResolver
	svc            BrandingService
}

func NewBrandingGRPCHandler(tenantResolver TenantResolver, svc BrandingService) *BrandingGRPCHandler {
	return &BrandingGRPCHandler{tenantResolver: tenantResolver, svc: svc}
}

func (h *BrandingGRPCHandler) GetBranding(ctx context.Context, req *authv1.GetBrandingRequest) (*authv1.GetBrandingResponse, error) {
	tenant, err := h.resolveTenant(ctx, req.GetTenantUuid())
	if err != nil {
		return nil, err
	}
	result, err := h.svc.Get(ctx, tenant.TenantID)
	if err != nil {
		return nil, apperror.ToGRPCError(err)
	}
	return &authv1.GetBrandingResponse{Branding: brandingProto(result)}, nil
}

func (h *BrandingGRPCHandler) UpdateBranding(ctx context.Context, req *authv1.UpdateBrandingRequest) (*authv1.UpdateBrandingResponse, error) {
	tenant, err := h.resolveTenant(ctx, req.GetTenantUuid())
	if err != nil {
		return nil, err
	}
	// Colors are managed via the HTTP API only (JSON palette); the gRPC surface
	// covers the scalar fields.
	var metadata datatypes.JSON
	if m := req.GetMetadata(); m != "" {
		metadata = datatypes.JSON([]byte(m))
	}
	result, err := h.svc.Update(ctx, tenant.TenantID, req.GetName(), req.GetCompanyName(), req.GetLogoUrl(), req.GetFaviconUrl(), metadata, req.GetSupportUrl(), req.GetPrivacyPolicyUrl(), req.GetTermsOfServiceUrl())
	if err != nil {
		return nil, apperror.ToGRPCError(err)
	}
	return &authv1.UpdateBrandingResponse{Branding: brandingProto(result)}, nil
}

func (h *BrandingGRPCHandler) resolveTenant(ctx context.Context, tenantUUID string) (*TenantServiceDataResult, error) {
	parsed, err := parseUUID(tenantUUID, "Tenant UUID")
	if err != nil {
		return nil, err
	}
	result, err := h.tenantResolver.GetByUUID(ctx, parsed)
	if err != nil {
		return nil, apperror.ToGRPCError(err)
	}
	return result, nil
}

func parseUUID(value string, label string) (uuid.UUID, error) {
	if value == "" {
		return uuid.Nil, apperror.ToGRPCError(apperror.NewValidation(label + " is required"))
	}
	parsed, err := uuid.Parse(value)
	if err != nil {
		return uuid.Nil, apperror.ToGRPCError(apperror.NewValidation("Invalid " + label))
	}
	return parsed, nil
}

func optionalStr(value string) *string {
	if value == "" {
		return nil
	}
	return &value
}

func grpcPagination(req *authv1.Pagination) PaginationRequestDTO {
	if req == nil {
		return PaginationRequestDTO{Page: 1, Limit: pagination.DefaultPageSize}
	}
	page, limit := int(req.GetPage()), int(req.GetLimit())
	if page == 0 {
		page = 1
	}
	if limit == 0 {
		limit = pagination.DefaultPageSize
	}
	return PaginationRequestDTO{Page: page, Limit: limit, SortBy: req.GetSortBy(), SortOrder: req.GetSortOrder()}
}

func pageProto(total int64, page, limit, totalPages int) *authv1.PageMetadata {
	return &authv1.PageMetadata{Total: total, Page: int32(page), Limit: int32(limit), TotalPages: int32(totalPages)}
}

func brandingProto(r *BrandingServiceDataResult) *authv1.Branding {
	if r == nil {
		return nil
	}
	return &authv1.Branding{
		BrandingUuid: r.BrandingUUID.String(), Name: r.Name, IsSystem: r.IsSystem, IsActive: r.IsActive,
		CompanyName: r.CompanyName, LogoUrl: r.LogoURL, FaviconUrl: r.FaviconURL,
		SupportUrl: r.SupportURL, PrivacyPolicyUrl: r.PrivacyPolicyURL, TermsOfServiceUrl: r.TermsOfServiceURL,
		Metadata:  string(r.Metadata),
		CreatedAt: timestamppb.New(r.CreatedAt), UpdatedAt: timestamppb.New(r.UpdatedAt),
	}
}
