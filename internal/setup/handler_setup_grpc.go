package setup

import (
	"context"

	"github.com/maintainerd/maintainerd-auth/internal/platform/apperror"
	authv1 "github.com/maintainerd/maintainerd-auth/internal/platform/gen/go/maintainerd/auth"
)

type SetupGRPCHandler struct {
	authv1.UnimplementedSetupServiceServer
	setupService SetupService
}

func NewSetupGRPCHandler(setupService SetupService) *SetupGRPCHandler {
	return &SetupGRPCHandler{setupService: setupService}
}

func (h *SetupGRPCHandler) GetSetupStatus(ctx context.Context, _ *authv1.GetSetupStatusRequest) (*authv1.GetSetupStatusResponse, error) {
	status, err := h.setupService.GetSetupStatus(ctx)
	if err != nil {
		return nil, apperror.ToGRPCError(err)
	}
	return &authv1.GetSetupStatusResponse{
		IsTenantSetup:   status.IsTenantSetup,
		IsAdminSetup:    status.IsAdminSetup,
		IsProfileSetup:  status.IsProfileSetup,
		IsSetupComplete: status.IsSetupComplete,
	}, nil
}

func (h *SetupGRPCHandler) CreateTenant(ctx context.Context, req *authv1.CreateTenantRequest) (*authv1.CreateTenantResponse, error) {
	resp, err := h.setupService.CreateTenant(ctx, CreateTenantRequestDTO{
		Name:        req.GetName(),
		DisplayName: req.GetDisplayName(),
		Description: optionalString(req.GetDescription()),
		Metadata:    tenantMetadataDTO(req.GetMetadata()),
	})
	if err != nil {
		return nil, apperror.ToGRPCError(err)
	}
	return &authv1.CreateTenantResponse{
		TenantUuid:        resp.Tenant.TenantUUID.String(),
		Name:              resp.Tenant.Name,
		DisplayName:       resp.Tenant.DisplayName,
		DefaultClientId:   resp.DefaultClientID,
		DefaultProviderId: resp.DefaultProviderID,
	}, nil
}

func (h *SetupGRPCHandler) CreateAdmin(ctx context.Context, req *authv1.CreateAdminRequest) (*authv1.CreateAdminResponse, error) {
	resp, err := h.setupService.CreateAdmin(ctx, CreateAdminRequestDTO{
		Username: req.GetUsername(),
		Fullname: func() *string { s := req.GetFullname(); return &s }(),
		Password: req.GetPassword(),
		Email:    req.GetEmail(),
	})
	if err != nil {
		return nil, apperror.ToGRPCError(err)
	}
	return &authv1.CreateAdminResponse{
		UserUuid: resp.User.UserUUID.String(),
		Username: resp.User.Username,
		Fullname: resp.User.Fullname,
		Email:    resp.User.Email,
		Status:   resp.User.Status,
	}, nil
}

func (h *SetupGRPCHandler) CreateProfile(ctx context.Context, req *authv1.CreateProfileRequest) (*authv1.CreateProfileResponse, error) {
	dto := CreateProfileRequestDTO{
		FirstName:   req.GetFirstName(),
		MiddleName:  optionalString(req.GetMiddleName()),
		LastName:    optionalString(req.GetLastName()),
		Suffix:      optionalString(req.GetSuffix()),
		DisplayName: optionalString(req.GetDisplayName()),
		Birthdate:   optionalString(req.GetBirthdate()),
		Gender:      optionalString(req.GetGender()),
		Bio:         optionalString(req.GetBio()),
		Phone:       optionalString(req.GetPhone()),
		Email:       optionalString(req.GetEmail()),
		Address:     optionalString(req.GetAddress()),
		City:        optionalString(req.GetCity()),
		Country:     optionalString(req.GetCountry()),
		Timezone:    optionalString(req.GetTimezone()),
		Language:    optionalString(req.GetLanguage()),
		ProfileURL:  optionalString(req.GetProfileUrl()),
		Metadata:    structMap(req.GetMetadata()),
	}
	if err := dto.Validate(); err != nil {
		return nil, apperror.ToGRPCError(apperror.NewValidation(err.Error()))
	}
	resp, err := h.setupService.CreateProfile(ctx, dto)
	if err != nil {
		return nil, apperror.ToGRPCError(err)
	}
	displayName := ""
	if resp.Profile.DisplayName != nil {
		displayName = *resp.Profile.DisplayName
	}
	return &authv1.CreateProfileResponse{
		ProfileUuid: resp.Profile.ProfileUUID,
		FirstName:   resp.Profile.FirstName,
		DisplayName: displayName,
	}, nil
}

func (h *SetupGRPCHandler) RegisterControlService(ctx context.Context, req *authv1.RegisterControlServiceRequest) (*authv1.RegisterControlServiceResponse, error) {
	resp, err := h.setupService.RegisterControlService(ctx, RegisterControlServiceRequestDTO{
		Name:        req.GetName(),
		DisplayName: req.GetDisplayName(),
		Description: optionalString(req.GetDescription()),
		Version:     req.GetVersion(),
	})
	if err != nil {
		return nil, apperror.ToGRPCError(err)
	}
	return &authv1.RegisterControlServiceResponse{
		ServiceUuid:       resp.ServiceUUID,
		Name:              resp.Name,
		DisplayName:       resp.DisplayName,
		PolicyUuid:        resp.PolicyUUID,
		PolicyName:        resp.PolicyName,
		AlreadyExisted:    resp.AlreadyExisted,
		PolicyWasAttached: resp.PolicyWasAttached,
	}, nil
}

func (h *SetupGRPCHandler) CompleteSetup(ctx context.Context, _ *authv1.CompleteSetupRequest) (*authv1.CompleteSetupResponse, error) {
	resp, err := h.setupService.CompleteSetup(ctx)
	if err != nil {
		return nil, apperror.ToGRPCError(err)
	}
	return &authv1.CompleteSetupResponse{IsSetupComplete: resp.IsSetupComplete}, nil
}

func optionalString(value string) *string {
	if value == "" {
		return nil
	}
	return &value
}

func tenantMetadataDTO(metadata *authv1.TenantMetadata) *TenantMetadataDTO {
	if metadata == nil {
		return nil
	}
	return &TenantMetadataDTO{
		ApplicationLogoURL: optionalString(metadata.GetApplicationLogoUrl()),
		FaviconURL:         optionalString(metadata.GetFaviconUrl()),
		Language:           optionalString(metadata.GetLanguage()),
		Timezone:           optionalString(metadata.GetTimezone()),
		DateFormat:         optionalString(metadata.GetDateFormat()),
		TimeFormat:         optionalString(metadata.GetTimeFormat()),
		PrivacyPolicyURL:   optionalString(metadata.GetPrivacyPolicyUrl()),
		TermOfServiceURL:   optionalString(metadata.GetTermOfServiceUrl()),
	}
}

func structMap(metadata interface{ AsMap() map[string]any }) map[string]any {
	if metadata == nil {
		return nil
	}
	return metadata.AsMap()
}
