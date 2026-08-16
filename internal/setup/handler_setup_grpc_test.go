package setup

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/maintainerd/maintainerd-auth/internal/platform/apperror"
	authv1 "github.com/maintainerd/maintainerd-auth/internal/platform/gen/go/maintainerd/auth"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestSetupGRPCHandler_GetSetupStatus(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		h := NewSetupGRPCHandler(&mockSetupService{
			getSetupStatusFn: func() (*SetupStatusResponseDTO, error) {
				return &SetupStatusResponseDTO{
					IsTenantSetup:   true,
					IsAdminSetup:    true,
					IsProfileSetup:  true,
					IsSetupComplete: false,
				}, nil
			},
		})

		resp, err := h.GetSetupStatus(context.Background(), &authv1.GetSetupStatusRequest{})

		require.NoError(t, err)
		assert.True(t, resp.IsTenantSetup)
		assert.True(t, resp.IsAdminSetup)
		assert.True(t, resp.IsProfileSetup)
		assert.False(t, resp.IsSetupComplete)
	})

	t.Run("error", func(t *testing.T) {
		h := NewSetupGRPCHandler(&mockSetupService{
			getSetupStatusFn: func() (*SetupStatusResponseDTO, error) {
				return nil, apperror.NewValidation("bad status")
			},
		})

		_, err := h.GetSetupStatus(context.Background(), &authv1.GetSetupStatusRequest{})

		assert.Equal(t, codes.InvalidArgument, status.Code(err))
	})
}

func TestSetupGRPCHandler_CreateTenant(t *testing.T) {
	tenantUUID := uuid.New()
	h := NewSetupGRPCHandler(&mockSetupService{
		createTenantFn: func(req CreateTenantRequestDTO) (*CreateTenantResponseDTO, error) {
			require.NotNil(t, req.Description)
			require.NotNil(t, req.Metadata)
			assert.Equal(t, "Maintainerd", req.DisplayName)
			assert.Equal(t, "Auth tenant", *req.Description)
			assert.Equal(t, "en", *req.Metadata.Language)
			assert.Equal(t, "UTC", *req.Metadata.Timezone)
			return &CreateTenantResponseDTO{
				Tenant: TenantResponseDTO{
					TenantUUID:  tenantUUID,
					Name:        req.Name,
					DisplayName: req.DisplayName,
				},
				DefaultClientID:   "client",
				DefaultProviderID: "provider",
			}, nil
		},
	})

	resp, err := h.CreateTenant(context.Background(), &authv1.CreateTenantRequest{
		Name:        "maintainerd",
		DisplayName: "Maintainerd",
		Description: "Auth tenant",
		Metadata: &authv1.TenantMetadata{
			Language: "en",
			Timezone: "UTC",
		},
	})

	require.NoError(t, err)
	assert.Equal(t, tenantUUID.String(), resp.TenantId)
	assert.Equal(t, "maintainerd", resp.Name)
	assert.Equal(t, "client", resp.DefaultClientId)

	_, err = NewSetupGRPCHandler(&mockSetupService{
		createTenantFn: func(CreateTenantRequestDTO) (*CreateTenantResponseDTO, error) {
			return nil, apperror.NewValidation("bad tenant")
		},
	}).CreateTenant(context.Background(), &authv1.CreateTenantRequest{})
	assert.Equal(t, codes.InvalidArgument, status.Code(err))
}

func TestSetupGRPCHandler_CreateAdmin(t *testing.T) {
	userUUID := uuid.New()
	h := NewSetupGRPCHandler(&mockSetupService{
		createAdminFn: func(req CreateAdminRequestDTO) (*CreateAdminResponseDTO, error) {
			assert.Equal(t, "admin", req.Username)
			assert.Equal(t, "Admin User", *req.Fullname)
			assert.Equal(t, "secret123", req.Password)
			assert.Equal(t, "admin@example.com", req.Email)
			return &CreateAdminResponseDTO{
				User: UserResponseDTO{
					UserUUID: userUUID,
					Username: req.Username,
					Fullname: *req.Fullname,
					Email:    req.Email,
					Status:   "active",
				},
			}, nil
		},
	})

	resp, err := h.CreateAdmin(context.Background(), &authv1.CreateAdminRequest{
		Username: "admin",
		Fullname: "Admin User",
		Password: "secret123",
		Email:    "admin@example.com",
	})

	require.NoError(t, err)
	assert.Equal(t, userUUID.String(), resp.UserId)
	assert.Equal(t, "active", resp.Status)

	_, err = NewSetupGRPCHandler(&mockSetupService{
		createAdminFn: func(CreateAdminRequestDTO) (*CreateAdminResponseDTO, error) {
			return nil, apperror.NewValidation("bad admin")
		},
	}).CreateAdmin(context.Background(), &authv1.CreateAdminRequest{})
	assert.Equal(t, codes.InvalidArgument, status.Code(err))
}

func TestSetupGRPCHandler_CreateProfile(t *testing.T) {
	profileUUID := uuid.New()
	display := "Admin User"
	h := NewSetupGRPCHandler(&mockSetupService{
		createProfileFn: func(req CreateProfileRequestDTO) (*CreateProfileResponseDTO, error) {
			assert.Equal(t, "Admin", req.FirstName)
			return &CreateProfileResponseDTO{
				Profile: ProfileResponseDTO{
					ProfileUUID: profileUUID.String(),
					FirstName:   req.FirstName,
					DisplayName: &display,
				},
			}, nil
		},
	})

	resp, err := h.CreateProfile(context.Background(), &authv1.CreateProfileRequest{
		FirstName:   "Admin",
		DisplayName: "Admin User",
	})
	require.NoError(t, err)
	assert.Equal(t, profileUUID.String(), resp.ProfileId)
	assert.Equal(t, "Admin", resp.FirstName)
	assert.Equal(t, "Admin User", resp.DisplayName)

	// FirstName is required by CreateProfileRequestDTO.Validate → InvalidArgument.
	_, err = h.CreateProfile(context.Background(), &authv1.CreateProfileRequest{})
	assert.Equal(t, codes.InvalidArgument, status.Code(err))
}

func TestSetupGRPCHandler_RegisterControlService(t *testing.T) {
	h := NewSetupGRPCHandler(&mockSetupService{
		registerControlServiceFn: func(req RegisterControlServiceRequestDTO) (*RegisterControlServiceResponseDTO, error) {
			require.NotNil(t, req.Description)
			assert.Equal(t, "core", req.Name)
			assert.Equal(t, "Core", req.DisplayName)
			assert.Equal(t, "v1", req.Version)
			return &RegisterControlServiceResponseDTO{
				ServiceUUID:       "service-1",
				Name:              req.Name,
				DisplayName:       req.DisplayName,
				PolicyUUID:        "policy-1",
				PolicyName:        "auth-control",
				AlreadyExisted:    true,
				PolicyWasAttached: false,
			}, nil
		},
	})

	resp, err := h.RegisterControlService(context.Background(), &authv1.RegisterControlServiceRequest{
		Name:        "core",
		DisplayName: "Core",
		Description: "control plane",
		Version:     "v1",
	})

	require.NoError(t, err)
	assert.Equal(t, "service-1", resp.ServiceId)
	assert.True(t, resp.AlreadyExisted)
	assert.False(t, resp.PolicyWasAttached)

	_, err = NewSetupGRPCHandler(&mockSetupService{
		registerControlServiceFn: func(RegisterControlServiceRequestDTO) (*RegisterControlServiceResponseDTO, error) {
			return nil, apperror.NewValidation("bad control")
		},
	}).RegisterControlService(context.Background(), &authv1.RegisterControlServiceRequest{})
	assert.Equal(t, codes.InvalidArgument, status.Code(err))
}

func TestSetupGRPCHandler_CompleteSetup(t *testing.T) {
	h := NewSetupGRPCHandler(&mockSetupService{
		completeSetupFn: func() (*CompleteSetupResponseDTO, error) {
			return &CompleteSetupResponseDTO{IsSetupComplete: true}, nil
		},
	})

	resp, err := h.CompleteSetup(context.Background(), &authv1.CompleteSetupRequest{})

	require.NoError(t, err)
	assert.True(t, resp.IsSetupComplete)

	_, err = NewSetupGRPCHandler(&mockSetupService{
		completeSetupFn: func() (*CompleteSetupResponseDTO, error) {
			return nil, apperror.NewValidation("bad complete")
		},
	}).CompleteSetup(context.Background(), &authv1.CompleteSetupRequest{})
	assert.Equal(t, codes.InvalidArgument, status.Code(err))
}

func TestSetupGRPCHandlerHelpers(t *testing.T) {
	assert.Nil(t, optionalString(""))
	assert.Nil(t, tenantMetadataDTO(nil))
	assert.Nil(t, structMap(nil))
}
