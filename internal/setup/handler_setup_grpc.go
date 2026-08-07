package setup

import (
	"context"

	"github.com/maintainerd/maintainerd-auth/internal/platform/apperror"
	authv1 "github.com/maintainerd/maintainerd-auth/internal/platform/gen/go/maintainerd/auth"
)

// The bootstrap creates are replay-guarded like every other provisioning RPC.
//
// The setup service refuses a second run once the system tenant exists, so a
// retry could never duplicate a tenant — but it answers with a conflict, which
// reads to core as "somebody else already claimed this instance" when the truth
// is "your own first call landed and the response was lost". On the one sequence
// that has no human watching it, that difference decides whether provisioning
// converges or halts.

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
	return h.createTenant(ctx, req)
}

func (h *SetupGRPCHandler) createTenant(ctx context.Context, req *authv1.CreateTenantRequest) (*authv1.CreateTenantResponse, error) {
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
	return h.createAdmin(ctx, req)
}

func (h *SetupGRPCHandler) createAdmin(ctx context.Context, req *authv1.CreateAdminRequest) (*authv1.CreateAdminResponse, error) {
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
	return h.createProfile(ctx, req)
}

func (h *SetupGRPCHandler) createProfile(ctx context.Context, req *authv1.CreateProfileRequest) (*authv1.CreateProfileResponse, error) {
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
		Name:           req.GetName(),
		DisplayName:    req.GetDisplayName(),
		Description:    optionalString(req.GetDescription()),
		Version:        req.GetVersion(),
		AllowedActions: req.GetAllowedActions(),
		PolicyName:     req.GetPolicyName(),
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

// The Ensure* RPCs below are the orchestrator-provisioning surface. Each is a
// thin mapping onto the service; the setup-open check, validation and the
// get-or-create semantics all live there, so the REST and gRPC transports cannot
// disagree about what provisioning means.

func (h *SetupGRPCHandler) EnsureControlClient(ctx context.Context, req *authv1.EnsureControlClientRequest) (*authv1.EnsureControlClientResponse, error) {
	resp, err := h.setupService.EnsureControlClient(ctx, EnsureControlClientRequestDTO{
		Name:        req.GetName(),
		DisplayName: req.GetDisplayName(),
		ServiceName: req.GetServiceName(),
		JWKS:        req.GetJwks(),
		JWKSUri:     req.GetJwksUri(),
		Audience:    req.GetAudience(),
	})
	if err != nil {
		return nil, apperror.ToGRPCError(err)
	}
	return &authv1.EnsureControlClientResponse{
		ClientUuid:              resp.ClientUUID,
		ClientId:                resp.ClientID,
		TokenEndpointAuthMethod: resp.TokenEndpointAuthMethod,
		ServiceUuid:             resp.ServiceUUID,
		AlreadyExisted:          resp.AlreadyExisted,
	}, nil
}

func (h *SetupGRPCHandler) EnsureResourceAPI(ctx context.Context, req *authv1.EnsureResourceAPIRequest) (*authv1.EnsureResourceAPIResponse, error) {
	permissions := make([]EnsureResourceAPIPermissionDTO, 0, len(req.GetPermissions()))
	for _, p := range req.GetPermissions() {
		permissions = append(permissions, EnsureResourceAPIPermissionDTO{
			Name:        p.GetName(),
			Description: p.GetDescription(),
		})
	}
	resp, err := h.setupService.EnsureResourceAPI(ctx, EnsureResourceAPIRequestDTO{
		ServiceName:        req.GetServiceName(),
		ServiceDisplayName: req.GetServiceDisplayName(),
		Name:               req.GetName(),
		DisplayName:        req.GetDisplayName(),
		Identifier:         req.GetIdentifier(),
		Permissions:        permissions,
	})
	if err != nil {
		return nil, apperror.ToGRPCError(err)
	}
	return &authv1.EnsureResourceAPIResponse{
		ServiceUuid:     resp.ServiceUUID,
		ApiUuid:         resp.APIUUID,
		Identifier:      resp.Identifier,
		PermissionNames: resp.PermissionNames,
		AlreadyExisted:  resp.AlreadyExisted,
	}, nil
}

func (h *SetupGRPCHandler) EnsureRole(ctx context.Context, req *authv1.EnsureRoleRequest) (*authv1.EnsureRoleResponse, error) {
	resp, err := h.setupService.EnsureRole(ctx, EnsureRoleRequestDTO{
		Name:             req.GetName(),
		Description:      req.GetDescription(),
		PermissionNames:  req.GetPermissionNames(),
		AssignToUserUUID: req.GetAssignToUserUuid(),
	})
	if err != nil {
		return nil, apperror.ToGRPCError(err)
	}
	return &authv1.EnsureRoleResponse{
		RoleUuid:        resp.RoleUUID,
		Name:            resp.Name,
		PermissionNames: resp.PermissionNames,
		Assigned:        resp.Assigned,
		AlreadyExisted:  resp.AlreadyExisted,
	}, nil
}

func (h *SetupGRPCHandler) EnsureConsoleClient(ctx context.Context, req *authv1.EnsureConsoleClientRequest) (*authv1.EnsureConsoleClientResponse, error) {
	resp, err := h.setupService.EnsureConsoleClient(ctx, EnsureConsoleClientRequestDTO{
		Name:                   req.GetName(),
		DisplayName:            req.GetDisplayName(),
		Domain:                 req.GetDomain(),
		RedirectURIs:           req.GetRedirectUris(),
		PostLogoutRedirectURIs: req.GetPostLogoutRedirectUris(),
	})
	if err != nil {
		return nil, apperror.ToGRPCError(err)
	}
	return &authv1.EnsureConsoleClientResponse{
		ClientUuid:             resp.ClientUUID,
		ClientId:               resp.ClientID,
		RedirectUris:           resp.RedirectURIs,
		PostLogoutRedirectUris: resp.PostLogoutRedirectURIs,
		AlreadyExisted:         resp.AlreadyExisted,
	}, nil
}
