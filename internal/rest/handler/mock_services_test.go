package handler

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/maintainerd/auth/internal/dto"
	"github.com/maintainerd/auth/internal/model"
	"github.com/maintainerd/auth/internal/service"
	"gorm.io/datatypes"
)

// ---------------------------------------------------------------------------
// mockLoginService
// ---------------------------------------------------------------------------

type mockLoginService struct {
	loginPublicFn    func(string, string, string, string) (*dto.LoginResponseDTO, error)
	loginFn          func(string, string, *string, *string) (*dto.LoginResponseDTO, error)
	getUserByEmailFn func(string, int64) (*model.User, error)
}

func (m *mockLoginService) LoginPublic(u, p, c, pr string) (*dto.LoginResponseDTO, error) {
	if m.loginPublicFn != nil {
		return m.loginPublicFn(u, p, c, pr)
	}
	return nil, nil
}
func (m *mockLoginService) Login(u, p string, c, pr *string) (*dto.LoginResponseDTO, error) {
	if m.loginFn != nil {
		return m.loginFn(u, p, c, pr)
	}
	return nil, nil
}
func (m *mockLoginService) GetUserByEmail(e string, tid int64) (*model.User, error) {
	if m.getUserByEmailFn != nil {
		return m.getUserByEmailFn(e, tid)
	}
	return nil, nil
}

// ---------------------------------------------------------------------------
// mockRegisterService
// ---------------------------------------------------------------------------

type mockRegisterService struct {
	registerPublicFn       func(string, string, string, *string, *string, string, string) (*dto.RegisterResponseDTO, error)
	registerInvitePublicFn func(string, string, string, string, string) (*dto.RegisterResponseDTO, error)
	registerFn             func(string, string, string, *string, *string, *string, *string) (*dto.RegisterResponseDTO, error)
	registerInviteFn       func(string, string, string, *string, *string) (*dto.RegisterResponseDTO, error)
}

func (m *mockRegisterService) RegisterPublic(u, f, p string, e, ph *string, c, pr string) (*dto.RegisterResponseDTO, error) {
	if m.registerPublicFn != nil {
		return m.registerPublicFn(u, f, p, e, ph, c, pr)
	}
	return nil, nil
}
func (m *mockRegisterService) RegisterInvitePublic(u, p, c, pr, t string) (*dto.RegisterResponseDTO, error) {
	if m.registerInvitePublicFn != nil {
		return m.registerInvitePublicFn(u, p, c, pr, t)
	}
	return nil, nil
}
func (m *mockRegisterService) Register(u, f, p string, e, ph, c, pr *string) (*dto.RegisterResponseDTO, error) {
	if m.registerFn != nil {
		return m.registerFn(u, f, p, e, ph, c, pr)
	}
	return nil, nil
}
func (m *mockRegisterService) RegisterInvite(u, p, t string, c, pr *string) (*dto.RegisterResponseDTO, error) {
	if m.registerInviteFn != nil {
		return m.registerInviteFn(u, p, t, c, pr)
	}
	return nil, nil
}

// ---------------------------------------------------------------------------
// mockSetupService
// ---------------------------------------------------------------------------

type mockSetupService struct {
	getSetupStatusFn func() (*dto.SetupStatusResponseDTO, error)
	createTenantFn   func(dto.CreateTenantRequestDTO) (*dto.CreateTenantResponseDTO, error)
	createAdminFn    func(dto.CreateAdminRequestDTO) (*dto.CreateAdminResponseDTO, error)
	createProfileFn  func(dto.CreateProfileRequestDTO) (*dto.CreateProfileResponseDTO, error)
}

func (m *mockSetupService) GetSetupStatus() (*dto.SetupStatusResponseDTO, error) {
	if m.getSetupStatusFn != nil {
		return m.getSetupStatusFn()
	}
	return nil, nil
}
func (m *mockSetupService) CreateTenant(req dto.CreateTenantRequestDTO) (*dto.CreateTenantResponseDTO, error) {
	if m.createTenantFn != nil {
		return m.createTenantFn(req)
	}
	return nil, nil
}
func (m *mockSetupService) CreateAdmin(req dto.CreateAdminRequestDTO) (*dto.CreateAdminResponseDTO, error) {
	if m.createAdminFn != nil {
		return m.createAdminFn(req)
	}
	return nil, nil
}
func (m *mockSetupService) CreateProfile(req dto.CreateProfileRequestDTO) (*dto.CreateProfileResponseDTO, error) {
	if m.createProfileFn != nil {
		return m.createProfileFn(req)
	}
	return nil, nil
}

// ---------------------------------------------------------------------------
// mockForgotPasswordService
// ---------------------------------------------------------------------------

type mockForgotPasswordService struct {
	sendPasswordResetEmailFn func(string, *string, *string, bool) (*dto.ForgotPasswordResponseDTO, error)
}

func (m *mockForgotPasswordService) SendPasswordResetEmail(_ context.Context, email string, clientID, providerID *string, isInternal bool) (*dto.ForgotPasswordResponseDTO, error) {
	if m.sendPasswordResetEmailFn != nil {
		return m.sendPasswordResetEmailFn(email, clientID, providerID, isInternal)
	}
	return nil, nil
}

// ---------------------------------------------------------------------------
// mockResetPasswordService
// ---------------------------------------------------------------------------

type mockResetPasswordService struct {
	resetPasswordFn func(string, string, *string, *string) (*dto.ResetPasswordResponseDTO, error)
}

func (m *mockResetPasswordService) ResetPassword(token, newPassword string, clientID, providerID *string) (*dto.ResetPasswordResponseDTO, error) {
	if m.resetPasswordFn != nil {
		return m.resetPasswordFn(token, newPassword, clientID, providerID)
	}
	return nil, nil
}

// ---------------------------------------------------------------------------
// mockAPIService
// ---------------------------------------------------------------------------

type mockAPIService struct {
	getFn                func(service.APIServiceGetFilter) (*service.APIServiceGetResult, error)
	getByUUIDFn          func(uuid.UUID, int64) (*service.APIServiceDataResult, error)
	getServiceIDByUUIDFn func(uuid.UUID) (int64, error)
	createFn             func(int64, string, string, string, string, string, bool, string) (*service.APIServiceDataResult, error)
	updateFn             func(uuid.UUID, int64, string, string, string, string, string, string) (*service.APIServiceDataResult, error)
	setStatusByUUIDFn    func(uuid.UUID, int64, string) (*service.APIServiceDataResult, error)
	deleteByUUIDFn       func(uuid.UUID, int64) (*service.APIServiceDataResult, error)
}

func (m *mockAPIService) Get(f service.APIServiceGetFilter) (*service.APIServiceGetResult, error) {
	if m.getFn != nil {
		return m.getFn(f)
	}
	return &service.APIServiceGetResult{}, nil
}
func (m *mockAPIService) GetByUUID(id uuid.UUID, tid int64) (*service.APIServiceDataResult, error) {
	if m.getByUUIDFn != nil {
		return m.getByUUIDFn(id, tid)
	}
	return nil, nil
}
func (m *mockAPIService) GetServiceIDByUUID(id uuid.UUID) (int64, error) {
	if m.getServiceIDByUUIDFn != nil {
		return m.getServiceIDByUUIDFn(id)
	}
	return 0, nil
}
func (m *mockAPIService) Create(tid int64, n, dn, desc, t, s string, isSys bool, svcUUID string) (*service.APIServiceDataResult, error) {
	if m.createFn != nil {
		return m.createFn(tid, n, dn, desc, t, s, isSys, svcUUID)
	}
	return nil, nil
}
func (m *mockAPIService) Update(id uuid.UUID, tid int64, n, dn, desc, t, s, svcUUID string) (*service.APIServiceDataResult, error) {
	if m.updateFn != nil {
		return m.updateFn(id, tid, n, dn, desc, t, s, svcUUID)
	}
	return nil, nil
}
func (m *mockAPIService) SetStatusByUUID(id uuid.UUID, tid int64, s string) (*service.APIServiceDataResult, error) {
	if m.setStatusByUUIDFn != nil {
		return m.setStatusByUUIDFn(id, tid, s)
	}
	return nil, nil
}
func (m *mockAPIService) DeleteByUUID(id uuid.UUID, tid int64) (*service.APIServiceDataResult, error) {
	if m.deleteByUUIDFn != nil {
		return m.deleteByUUIDFn(id, tid)
	}
	return nil, nil
}

// ---------------------------------------------------------------------------
// mockAPIKeyService
// ---------------------------------------------------------------------------

type mockAPIKeyService struct {
	getFn                 func(service.APIKeyServiceGetFilter, uuid.UUID) (*service.APIKeyServiceGetResult, error)
	getByUUIDFn           func(uuid.UUID, int64, uuid.UUID) (*service.APIKeyServiceDataResult, error)
	getConfigByUUIDFn     func(uuid.UUID, int64) (datatypes.JSON, error)
	createFn              func(int64, string, string, datatypes.JSON, *time.Time, *int, string) (*service.APIKeyServiceDataResult, string, error)
	updateFn              func(uuid.UUID, int64, *string, *string, datatypes.JSON, *time.Time, *int, *string, uuid.UUID) (*service.APIKeyServiceDataResult, error)
	setStatusByUUIDFn     func(uuid.UUID, int64, string) (*service.APIKeyServiceDataResult, error)
	deleteFn              func(uuid.UUID, int64, uuid.UUID) (*service.APIKeyServiceDataResult, error)
	validateAPIKeyFn      func(string) (*service.APIKeyServiceDataResult, error)
	getAPIKeyAPIsFn       func(uuid.UUID, int, int, string, string) (*service.APIKeyAPIServicePaginatedResult, error)
	addAPIKeyAPIsFn       func(uuid.UUID, []uuid.UUID) error
	removeAPIKeyAPIFn     func(uuid.UUID, uuid.UUID) error
	getAPIKeyAPIPermsFn   func(uuid.UUID, uuid.UUID) ([]service.PermissionServiceDataResult, error)
	addAPIKeyAPIPermsFn   func(uuid.UUID, uuid.UUID, []uuid.UUID) error
	removeAPIKeyAPIPermFn func(uuid.UUID, uuid.UUID, uuid.UUID) error
}

func (m *mockAPIKeyService) Get(f service.APIKeyServiceGetFilter, u uuid.UUID) (*service.APIKeyServiceGetResult, error) {
	if m.getFn != nil {
		return m.getFn(f, u)
	}
	return &service.APIKeyServiceGetResult{}, nil
}
func (m *mockAPIKeyService) GetByUUID(id uuid.UUID, tid int64, u uuid.UUID) (*service.APIKeyServiceDataResult, error) {
	if m.getByUUIDFn != nil {
		return m.getByUUIDFn(id, tid, u)
	}
	return nil, nil
}
func (m *mockAPIKeyService) GetConfigByUUID(id uuid.UUID, tid int64) (datatypes.JSON, error) {
	if m.getConfigByUUIDFn != nil {
		return m.getConfigByUUIDFn(id, tid)
	}
	return nil, nil
}
func (m *mockAPIKeyService) Create(tid int64, n, desc string, cfg datatypes.JSON, exp *time.Time, rl *int, s string) (*service.APIKeyServiceDataResult, string, error) {
	if m.createFn != nil {
		return m.createFn(tid, n, desc, cfg, exp, rl, s)
	}
	return nil, "", nil
}
func (m *mockAPIKeyService) Update(id uuid.UUID, tid int64, n, desc *string, cfg datatypes.JSON, exp *time.Time, rl *int, s *string, u uuid.UUID) (*service.APIKeyServiceDataResult, error) {
	if m.updateFn != nil {
		return m.updateFn(id, tid, n, desc, cfg, exp, rl, s, u)
	}
	return nil, nil
}
func (m *mockAPIKeyService) SetStatusByUUID(id uuid.UUID, tid int64, s string) (*service.APIKeyServiceDataResult, error) {
	if m.setStatusByUUIDFn != nil {
		return m.setStatusByUUIDFn(id, tid, s)
	}
	return nil, nil
}
func (m *mockAPIKeyService) Delete(id uuid.UUID, tid int64, u uuid.UUID) (*service.APIKeyServiceDataResult, error) {
	if m.deleteFn != nil {
		return m.deleteFn(id, tid, u)
	}
	return nil, nil
}
func (m *mockAPIKeyService) ValidateAPIKey(k string) (*service.APIKeyServiceDataResult, error) {
	if m.validateAPIKeyFn != nil {
		return m.validateAPIKeyFn(k)
	}
	return nil, nil
}
func (m *mockAPIKeyService) GetAPIKeyAPIs(id uuid.UUID, pg, lim int, sb, so string) (*service.APIKeyAPIServicePaginatedResult, error) {
	if m.getAPIKeyAPIsFn != nil {
		return m.getAPIKeyAPIsFn(id, pg, lim, sb, so)
	}
	return &service.APIKeyAPIServicePaginatedResult{}, nil
}
func (m *mockAPIKeyService) AddAPIKeyAPIs(id uuid.UUID, apis []uuid.UUID) error {
	if m.addAPIKeyAPIsFn != nil {
		return m.addAPIKeyAPIsFn(id, apis)
	}
	return nil
}
func (m *mockAPIKeyService) RemoveAPIKeyAPI(id, api uuid.UUID) error {
	if m.removeAPIKeyAPIFn != nil {
		return m.removeAPIKeyAPIFn(id, api)
	}
	return nil
}
func (m *mockAPIKeyService) GetAPIKeyAPIPermissions(id, api uuid.UUID) ([]service.PermissionServiceDataResult, error) {
	if m.getAPIKeyAPIPermsFn != nil {
		return m.getAPIKeyAPIPermsFn(id, api)
	}
	return nil, nil
}
func (m *mockAPIKeyService) AddAPIKeyAPIPermissions(id, api uuid.UUID, perms []uuid.UUID) error {
	if m.addAPIKeyAPIPermsFn != nil {
		return m.addAPIKeyAPIPermsFn(id, api, perms)
	}
	return nil
}
func (m *mockAPIKeyService) RemoveAPIKeyAPIPermission(id, api, perm uuid.UUID) error {
	if m.removeAPIKeyAPIPermFn != nil {
		return m.removeAPIKeyAPIPermFn(id, api, perm)
	}
	return nil
}

// ---------------------------------------------------------------------------
// mockClientService
// ---------------------------------------------------------------------------

type mockClientService struct {
	getFn                 func(service.ClientServiceGetFilter) (*service.ClientServiceGetResult, error)
	getByUUIDFn           func(uuid.UUID, int64) (*service.ClientServiceDataResult, error)
	getSecretByUUIDFn     func(uuid.UUID, int64) (*service.ClientSecretServiceDataResult, error)
	getConfigByUUIDFn     func(uuid.UUID, int64) (datatypes.JSON, error)
	createFn              func(int64, string, string, string, string, datatypes.JSON, string, bool, string, uuid.UUID) (*service.ClientServiceDataResult, error)
	updateFn              func(uuid.UUID, int64, string, string, string, string, datatypes.JSON, string, bool, uuid.UUID) (*service.ClientServiceDataResult, error)
	setStatusByUUIDFn     func(uuid.UUID, int64, string, uuid.UUID) (*service.ClientServiceDataResult, error)
	deleteByUUIDFn        func(uuid.UUID, int64, uuid.UUID) (*service.ClientServiceDataResult, error)
	createURIFn           func(uuid.UUID, int64, string, string, uuid.UUID) (*service.ClientServiceDataResult, error)
	updateURIFn           func(uuid.UUID, int64, uuid.UUID, string, string, uuid.UUID) (*service.ClientServiceDataResult, error)
	deleteURIFn           func(uuid.UUID, int64, uuid.UUID, uuid.UUID) (*service.ClientServiceDataResult, error)
	getClientAPIsFn       func(int64, uuid.UUID) ([]service.ClientAPIServiceDataResult, error)
	addClientAPIsFn       func(int64, uuid.UUID, []uuid.UUID) error
	removeClientAPIFn     func(int64, uuid.UUID, uuid.UUID) error
	getClientAPIPermsFn   func(int64, uuid.UUID, uuid.UUID) ([]service.PermissionServiceDataResult, error)
	addClientAPIPermsFn   func(int64, uuid.UUID, uuid.UUID, []uuid.UUID) error
	removeClientAPIPermFn func(int64, uuid.UUID, uuid.UUID, uuid.UUID) error
}

func (m *mockClientService) Get(f service.ClientServiceGetFilter) (*service.ClientServiceGetResult, error) {
	if m.getFn != nil {
		return m.getFn(f)
	}
	return &service.ClientServiceGetResult{}, nil
}
func (m *mockClientService) GetByUUID(id uuid.UUID, tid int64) (*service.ClientServiceDataResult, error) {
	if m.getByUUIDFn != nil {
		return m.getByUUIDFn(id, tid)
	}
	return nil, nil
}
func (m *mockClientService) GetSecretByUUID(id uuid.UUID, tid int64) (*service.ClientSecretServiceDataResult, error) {
	if m.getSecretByUUIDFn != nil {
		return m.getSecretByUUIDFn(id, tid)
	}
	return nil, nil
}
func (m *mockClientService) GetConfigByUUID(id uuid.UUID, tid int64) (datatypes.JSON, error) {
	if m.getConfigByUUIDFn != nil {
		return m.getConfigByUUIDFn(id, tid)
	}
	return nil, nil
}
func (m *mockClientService) Create(tid int64, n, dn, ct, d string, cfg datatypes.JSON, s string, isDef bool, idpUUID string, actor uuid.UUID) (*service.ClientServiceDataResult, error) {
	if m.createFn != nil {
		return m.createFn(tid, n, dn, ct, d, cfg, s, isDef, idpUUID, actor)
	}
	return nil, nil
}
func (m *mockClientService) Update(id uuid.UUID, tid int64, n, dn, ct, d string, cfg datatypes.JSON, s string, isDef bool, actor uuid.UUID) (*service.ClientServiceDataResult, error) {
	if m.updateFn != nil {
		return m.updateFn(id, tid, n, dn, ct, d, cfg, s, isDef, actor)
	}
	return nil, nil
}
func (m *mockClientService) SetStatusByUUID(id uuid.UUID, tid int64, s string, actor uuid.UUID) (*service.ClientServiceDataResult, error) {
	if m.setStatusByUUIDFn != nil {
		return m.setStatusByUUIDFn(id, tid, s, actor)
	}
	return nil, nil
}
func (m *mockClientService) DeleteByUUID(id uuid.UUID, tid int64, actor uuid.UUID) (*service.ClientServiceDataResult, error) {
	if m.deleteByUUIDFn != nil {
		return m.deleteByUUIDFn(id, tid, actor)
	}
	return nil, nil
}
func (m *mockClientService) CreateURI(id uuid.UUID, tid int64, uri, uriType string, actor uuid.UUID) (*service.ClientServiceDataResult, error) {
	if m.createURIFn != nil {
		return m.createURIFn(id, tid, uri, uriType, actor)
	}
	return nil, nil
}
func (m *mockClientService) UpdateURI(id uuid.UUID, tid int64, uriID uuid.UUID, uri, uriType string, actor uuid.UUID) (*service.ClientServiceDataResult, error) {
	if m.updateURIFn != nil {
		return m.updateURIFn(id, tid, uriID, uri, uriType, actor)
	}
	return nil, nil
}
func (m *mockClientService) DeleteURI(id uuid.UUID, tid int64, uriID uuid.UUID, actor uuid.UUID) (*service.ClientServiceDataResult, error) {
	if m.deleteURIFn != nil {
		return m.deleteURIFn(id, tid, uriID, actor)
	}
	return nil, nil
}
func (m *mockClientService) GetClientAPIs(tid int64, id uuid.UUID) ([]service.ClientAPIServiceDataResult, error) {
	if m.getClientAPIsFn != nil {
		return m.getClientAPIsFn(tid, id)
	}
	return nil, nil
}
func (m *mockClientService) AddClientAPIs(tid int64, id uuid.UUID, apis []uuid.UUID) error {
	if m.addClientAPIsFn != nil {
		return m.addClientAPIsFn(tid, id, apis)
	}
	return nil
}
func (m *mockClientService) RemoveClientAPI(tid int64, id, api uuid.UUID) error {
	if m.removeClientAPIFn != nil {
		return m.removeClientAPIFn(tid, id, api)
	}
	return nil
}
func (m *mockClientService) GetClientAPIPermissions(tid int64, id, api uuid.UUID) ([]service.PermissionServiceDataResult, error) {
	if m.getClientAPIPermsFn != nil {
		return m.getClientAPIPermsFn(tid, id, api)
	}
	return nil, nil
}
func (m *mockClientService) AddClientAPIPermissions(tid int64, id, api uuid.UUID, perms []uuid.UUID) error {
	if m.addClientAPIPermsFn != nil {
		return m.addClientAPIPermsFn(tid, id, api, perms)
	}
	return nil
}
func (m *mockClientService) RemoveClientAPIPermission(tid int64, id, api, perm uuid.UUID) error {
	if m.removeClientAPIPermFn != nil {
		return m.removeClientAPIPermFn(tid, id, api, perm)
	}
	return nil
}

// ---------------------------------------------------------------------------
// mockTenantService
// ---------------------------------------------------------------------------

type mockTenantService struct {
	getFn                    func(service.TenantServiceGetFilter) (*service.TenantServiceGetResult, error)
	getByUUIDFn              func(uuid.UUID) (*service.TenantServiceDataResult, error)
	getDefaultFn             func() (*service.TenantServiceDataResult, error)
	getByIdentifierFn        func(string) (*service.TenantServiceDataResult, error)
	createFn                 func(string, string, string, string, bool, bool) (*service.TenantServiceDataResult, error)
	updateFn                 func(uuid.UUID, string, string, string, string, bool) (*service.TenantServiceDataResult, error)
	setStatusByUUIDFn        func(uuid.UUID, string) (*service.TenantServiceDataResult, error)
	setActivePublicByUUIDFn  func(uuid.UUID) (*service.TenantServiceDataResult, error)
	setDefaultStatusByUUIDFn func(uuid.UUID) (*service.TenantServiceDataResult, error)
	deleteByUUIDFn           func(uuid.UUID) (*service.TenantServiceDataResult, error)
}

func (m *mockTenantService) Get(f service.TenantServiceGetFilter) (*service.TenantServiceGetResult, error) {
	if m.getFn != nil {
		return m.getFn(f)
	}
	return &service.TenantServiceGetResult{}, nil
}
func (m *mockTenantService) GetByUUID(id uuid.UUID) (*service.TenantServiceDataResult, error) {
	if m.getByUUIDFn != nil {
		return m.getByUUIDFn(id)
	}
	return nil, nil
}
func (m *mockTenantService) GetDefault() (*service.TenantServiceDataResult, error) {
	if m.getDefaultFn != nil {
		return m.getDefaultFn()
	}
	return nil, nil
}
func (m *mockTenantService) GetByIdentifier(id string) (*service.TenantServiceDataResult, error) {
	if m.getByIdentifierFn != nil {
		return m.getByIdentifierFn(id)
	}
	return nil, nil
}
func (m *mockTenantService) Create(n, dn, desc, s string, isPublic, isDef bool) (*service.TenantServiceDataResult, error) {
	if m.createFn != nil {
		return m.createFn(n, dn, desc, s, isPublic, isDef)
	}
	return nil, nil
}
func (m *mockTenantService) Update(id uuid.UUID, n, dn, desc, s string, isPublic bool) (*service.TenantServiceDataResult, error) {
	if m.updateFn != nil {
		return m.updateFn(id, n, dn, desc, s, isPublic)
	}
	return nil, nil
}
func (m *mockTenantService) SetStatusByUUID(id uuid.UUID, s string) (*service.TenantServiceDataResult, error) {
	if m.setStatusByUUIDFn != nil {
		return m.setStatusByUUIDFn(id, s)
	}
	return nil, nil
}
func (m *mockTenantService) SetActivePublicByUUID(id uuid.UUID) (*service.TenantServiceDataResult, error) {
	if m.setActivePublicByUUIDFn != nil {
		return m.setActivePublicByUUIDFn(id)
	}
	return nil, nil
}
func (m *mockTenantService) SetDefaultStatusByUUID(id uuid.UUID) (*service.TenantServiceDataResult, error) {
	if m.setDefaultStatusByUUIDFn != nil {
		return m.setDefaultStatusByUUIDFn(id)
	}
	return nil, nil
}
func (m *mockTenantService) DeleteByUUID(id uuid.UUID) (*service.TenantServiceDataResult, error) {
	if m.deleteByUUIDFn != nil {
		return m.deleteByUUIDFn(id)
	}
	return nil, nil
}

// ---------------------------------------------------------------------------
// mockServiceService
// ---------------------------------------------------------------------------

type mockServiceService struct {
	getFn             func(service.ServiceServiceGetFilter) (*service.ServiceServiceGetResult, error)
	getByUUIDFn       func(uuid.UUID, int64) (*service.ServiceServiceDataResult, error)
	createFn          func(string, string, string, string, bool, string, int64) (*service.ServiceServiceDataResult, error)
	updateFn          func(uuid.UUID, int64, string, string, string, string, bool, string) (*service.ServiceServiceDataResult, error)
	setStatusByUUIDFn func(uuid.UUID, int64, string) (*service.ServiceServiceDataResult, error)
	deleteByUUIDFn    func(uuid.UUID, int64) (*service.ServiceServiceDataResult, error)
	assignPolicyFn    func(uuid.UUID, uuid.UUID, int64) error
	removePolicyFn    func(uuid.UUID, uuid.UUID, int64) error
}

func (m *mockServiceService) Get(f service.ServiceServiceGetFilter) (*service.ServiceServiceGetResult, error) {
	if m.getFn != nil {
		return m.getFn(f)
	}
	return &service.ServiceServiceGetResult{}, nil
}
func (m *mockServiceService) GetByUUID(id uuid.UUID, tid int64) (*service.ServiceServiceDataResult, error) {
	if m.getByUUIDFn != nil {
		return m.getByUUIDFn(id, tid)
	}
	return nil, nil
}
func (m *mockServiceService) Create(n, dn, desc, v string, isSys bool, s string, tid int64) (*service.ServiceServiceDataResult, error) {
	if m.createFn != nil {
		return m.createFn(n, dn, desc, v, isSys, s, tid)
	}
	return nil, nil
}
func (m *mockServiceService) Update(id uuid.UUID, tid int64, n, dn, desc, v string, isSys bool, s string) (*service.ServiceServiceDataResult, error) {
	if m.updateFn != nil {
		return m.updateFn(id, tid, n, dn, desc, v, isSys, s)
	}
	return nil, nil
}
func (m *mockServiceService) SetStatusByUUID(id uuid.UUID, tid int64, s string) (*service.ServiceServiceDataResult, error) {
	if m.setStatusByUUIDFn != nil {
		return m.setStatusByUUIDFn(id, tid, s)
	}
	return nil, nil
}
func (m *mockServiceService) DeleteByUUID(id uuid.UUID, tid int64) (*service.ServiceServiceDataResult, error) {
	if m.deleteByUUIDFn != nil {
		return m.deleteByUUIDFn(id, tid)
	}
	return nil, nil
}
func (m *mockServiceService) AssignPolicy(svcID, polID uuid.UUID, tid int64) error {
	if m.assignPolicyFn != nil {
		return m.assignPolicyFn(svcID, polID, tid)
	}
	return nil
}
func (m *mockServiceService) RemovePolicy(svcID, polID uuid.UUID, tid int64) error {
	if m.removePolicyFn != nil {
		return m.removePolicyFn(svcID, polID, tid)
	}
	return nil
}

// ---------------------------------------------------------------------------
// mockRoleService
// ---------------------------------------------------------------------------

type mockRoleService struct {
	getFn                func(service.RoleServiceGetFilter) (*service.RoleServiceGetResult, error)
	getByUUIDFn          func(uuid.UUID, int64) (*service.RoleServiceDataResult, error)
	getRolePermissionsFn func(service.RoleServiceGetPermissionsFilter) (*service.RoleServiceGetPermissionsResult, error)
	createFn             func(string, string, bool, bool, string, string, uuid.UUID) (*service.RoleServiceDataResult, error)
	updateFn             func(uuid.UUID, int64, string, string, bool, bool, string, uuid.UUID) (*service.RoleServiceDataResult, error)
	setStatusByUUIDFn    func(uuid.UUID, int64, string, uuid.UUID) (*service.RoleServiceDataResult, error)
	deleteByUUIDFn       func(uuid.UUID, int64, uuid.UUID) (*service.RoleServiceDataResult, error)
	addRolePermsFn       func(uuid.UUID, int64, []uuid.UUID, uuid.UUID) (*service.RoleServiceDataResult, error)
	removeRolePermsFn    func(uuid.UUID, int64, uuid.UUID, uuid.UUID) (*service.RoleServiceDataResult, error)
}

func (m *mockRoleService) Get(f service.RoleServiceGetFilter) (*service.RoleServiceGetResult, error) {
	if m.getFn != nil {
		return m.getFn(f)
	}
	return &service.RoleServiceGetResult{}, nil
}
func (m *mockRoleService) GetByUUID(id uuid.UUID, tid int64) (*service.RoleServiceDataResult, error) {
	if m.getByUUIDFn != nil {
		return m.getByUUIDFn(id, tid)
	}
	return nil, nil
}
func (m *mockRoleService) GetRolePermissions(f service.RoleServiceGetPermissionsFilter) (*service.RoleServiceGetPermissionsResult, error) {
	if m.getRolePermissionsFn != nil {
		return m.getRolePermissionsFn(f)
	}
	return &service.RoleServiceGetPermissionsResult{}, nil
}
func (m *mockRoleService) Create(_ context.Context, n, desc string, isDef, isSys bool, s, tenantUUID string, actor uuid.UUID) (*service.RoleServiceDataResult, error) {
	if m.createFn != nil {
		return m.createFn(n, desc, isDef, isSys, s, tenantUUID, actor)
	}
	return nil, nil
}
func (m *mockRoleService) Update(_ context.Context, id uuid.UUID, tid int64, n, desc string, isDef, isSys bool, s string, actor uuid.UUID) (*service.RoleServiceDataResult, error) {
	if m.updateFn != nil {
		return m.updateFn(id, tid, n, desc, isDef, isSys, s, actor)
	}
	return nil, nil
}
func (m *mockRoleService) SetStatusByUUID(_ context.Context, id uuid.UUID, tid int64, s string, actor uuid.UUID) (*service.RoleServiceDataResult, error) {
	if m.setStatusByUUIDFn != nil {
		return m.setStatusByUUIDFn(id, tid, s, actor)
	}
	return nil, nil
}
func (m *mockRoleService) DeleteByUUID(_ context.Context, id uuid.UUID, tid int64, actor uuid.UUID) (*service.RoleServiceDataResult, error) {
	if m.deleteByUUIDFn != nil {
		return m.deleteByUUIDFn(id, tid, actor)
	}
	return nil, nil
}
func (m *mockRoleService) AddRolePermissions(_ context.Context, id uuid.UUID, tid int64, perms []uuid.UUID, actor uuid.UUID) (*service.RoleServiceDataResult, error) {
	if m.addRolePermsFn != nil {
		return m.addRolePermsFn(id, tid, perms, actor)
	}
	return nil, nil
}
func (m *mockRoleService) RemoveRolePermissions(_ context.Context, id uuid.UUID, tid int64, perm uuid.UUID, actor uuid.UUID) (*service.RoleServiceDataResult, error) {
	if m.removeRolePermsFn != nil {
		return m.removeRolePermsFn(id, tid, perm, actor)
	}
	return nil, nil
}

// ---------------------------------------------------------------------------
// mockUserService
// ---------------------------------------------------------------------------

type mockUserService struct {
	getFn             func(service.UserServiceGetFilter) (*service.UserServiceGetResult, error)
	getByUUIDFn       func(uuid.UUID, int64) (*service.UserServiceDataResult, error)
	createFn          func(string, string, *string, *string, string, string, datatypes.JSON, string, uuid.UUID) (*service.UserServiceDataResult, error)
	updateFn          func(uuid.UUID, int64, string, string, *string, *string, string, datatypes.JSON, uuid.UUID) (*service.UserServiceDataResult, error)
	setStatusFn       func(uuid.UUID, int64, string, uuid.UUID) (*service.UserServiceDataResult, error)
	verifyEmailFn     func(uuid.UUID, int64) (*service.UserServiceDataResult, error)
	verifyPhoneFn     func(uuid.UUID, int64) (*service.UserServiceDataResult, error)
	completeAccountFn func(uuid.UUID, int64) (*service.UserServiceDataResult, error)
	deleteByUUIDFn    func(uuid.UUID, int64, uuid.UUID) (*service.UserServiceDataResult, error)
	assignUserRolesFn func(uuid.UUID, []uuid.UUID, int64) (*service.UserServiceDataResult, error)
	removeUserRoleFn  func(uuid.UUID, uuid.UUID, int64) (*service.UserServiceDataResult, error)
	getUserRolesFn    func(uuid.UUID) ([]service.RoleServiceDataResult, error)
	getUserIdentsFn   func(uuid.UUID) ([]service.UserIdentityServiceDataResult, error)
}

func (m *mockUserService) Get(f service.UserServiceGetFilter) (*service.UserServiceGetResult, error) {
	if m.getFn != nil {
		return m.getFn(f)
	}
	return &service.UserServiceGetResult{}, nil
}
func (m *mockUserService) GetByUUID(id uuid.UUID, tid int64) (*service.UserServiceDataResult, error) {
	if m.getByUUIDFn != nil {
		return m.getByUUIDFn(id, tid)
	}
	return nil, nil
}
func (m *mockUserService) Create(u, fn string, e, ph *string, pw, s string, meta datatypes.JSON, tUUID string, creator uuid.UUID) (*service.UserServiceDataResult, error) {
	if m.createFn != nil {
		return m.createFn(u, fn, e, ph, pw, s, meta, tUUID, creator)
	}
	return nil, nil
}
func (m *mockUserService) Update(_ context.Context, id uuid.UUID, tid int64, u, fn string, e, ph *string, s string, meta datatypes.JSON, updater uuid.UUID) (*service.UserServiceDataResult, error) {
	if m.updateFn != nil {
		return m.updateFn(id, tid, u, fn, e, ph, s, meta, updater)
	}
	return nil, nil
}
func (m *mockUserService) SetStatus(_ context.Context, id uuid.UUID, tid int64, s string, updater uuid.UUID) (*service.UserServiceDataResult, error) {
	if m.setStatusFn != nil {
		return m.setStatusFn(id, tid, s, updater)
	}
	return nil, nil
}
func (m *mockUserService) VerifyEmail(_ context.Context, id uuid.UUID, tid int64) (*service.UserServiceDataResult, error) {
	if m.verifyEmailFn != nil {
		return m.verifyEmailFn(id, tid)
	}
	return nil, nil
}
func (m *mockUserService) VerifyPhone(_ context.Context, id uuid.UUID, tid int64) (*service.UserServiceDataResult, error) {
	if m.verifyPhoneFn != nil {
		return m.verifyPhoneFn(id, tid)
	}
	return nil, nil
}
func (m *mockUserService) CompleteAccount(_ context.Context, id uuid.UUID, tid int64) (*service.UserServiceDataResult, error) {
	if m.completeAccountFn != nil {
		return m.completeAccountFn(id, tid)
	}
	return nil, nil
}
func (m *mockUserService) DeleteByUUID(_ context.Context, id uuid.UUID, tid int64, deleter uuid.UUID) (*service.UserServiceDataResult, error) {
	if m.deleteByUUIDFn != nil {
		return m.deleteByUUIDFn(id, tid, deleter)
	}
	return nil, nil
}
func (m *mockUserService) AssignUserRoles(_ context.Context, id uuid.UUID, roles []uuid.UUID, tid int64) (*service.UserServiceDataResult, error) {
	if m.assignUserRolesFn != nil {
		return m.assignUserRolesFn(id, roles, tid)
	}
	return nil, nil
}
func (m *mockUserService) RemoveUserRole(_ context.Context, id, role uuid.UUID, tid int64) (*service.UserServiceDataResult, error) {
	if m.removeUserRoleFn != nil {
		return m.removeUserRoleFn(id, role, tid)
	}
	return nil, nil
}
func (m *mockUserService) GetUserRoles(id uuid.UUID) ([]service.RoleServiceDataResult, error) {
	if m.getUserRolesFn != nil {
		return m.getUserRolesFn(id)
	}
	return nil, nil
}
func (m *mockUserService) GetUserIdentities(id uuid.UUID) ([]service.UserIdentityServiceDataResult, error) {
	if m.getUserIdentsFn != nil {
		return m.getUserIdentsFn(id)
	}
	return nil, nil
}

func (m *mockUserService) FindBySubAndClientID(sub, clientID string) (*model.User, error) {
	return nil, nil
}

// ---------------------------------------------------------------------------
// mockProfileService
// ---------------------------------------------------------------------------

type mockProfileService struct {
	createOrUpdateFn         func(uuid.UUID, string, *string, *string, *string, *string, *string, *time.Time, *string, *string, *string, *string, *string, *string, *string, *string, *string, map[string]any) (*service.ProfileServiceDataResult, error)
	createOrUpdateSpecificFn func(uuid.UUID, uuid.UUID, string, *string, *string, *string, *string, *string, *time.Time, *string, *string, *string, *string, *string, *string, *string, *string, *string, map[string]any) (*service.ProfileServiceDataResult, error)
	getByUUIDFn              func(uuid.UUID, uuid.UUID) (*service.ProfileServiceDataResult, error)
	getByUserUUIDFn          func(uuid.UUID) (*service.ProfileServiceDataResult, error)
	getAllFn                 func(uuid.UUID, *string, *string, *string, *string, *string, *string, *bool, int, int, string, string) (*service.ProfileServiceListResult, error)
	setDefaultFn             func(uuid.UUID, uuid.UUID) (*service.ProfileServiceDataResult, error)
	deleteByUUIDFn           func(uuid.UUID, uuid.UUID) (*service.ProfileServiceDataResult, error)
}

func (m *mockProfileService) CreateOrUpdateProfile(userUUID uuid.UUID, firstName string, middleName, lastName, suffix, displayName, bio *string, birthdate *time.Time, gender, phone, email, address, city, country, timezone, language, profileURL *string, metadata map[string]any) (*service.ProfileServiceDataResult, error) {
	if m.createOrUpdateFn != nil {
		return m.createOrUpdateFn(userUUID, firstName, middleName, lastName, suffix, displayName, bio, birthdate, gender, phone, email, address, city, country, timezone, language, profileURL, metadata)
	}
	return nil, nil
}
func (m *mockProfileService) CreateOrUpdateSpecificProfile(profileUUID uuid.UUID, userUUID uuid.UUID, firstName string, middleName, lastName, suffix, displayName, bio *string, birthdate *time.Time, gender, phone, email, address, city, country, timezone, language, profileURL *string, metadata map[string]any) (*service.ProfileServiceDataResult, error) {
	if m.createOrUpdateSpecificFn != nil {
		return m.createOrUpdateSpecificFn(profileUUID, userUUID, firstName, middleName, lastName, suffix, displayName, bio, birthdate, gender, phone, email, address, city, country, timezone, language, profileURL, metadata)
	}
	return nil, nil
}
func (m *mockProfileService) GetByUUID(profileUUID uuid.UUID, userUUID uuid.UUID) (*service.ProfileServiceDataResult, error) {
	if m.getByUUIDFn != nil {
		return m.getByUUIDFn(profileUUID, userUUID)
	}
	return nil, nil
}
func (m *mockProfileService) GetByUserUUID(userUUID uuid.UUID) (*service.ProfileServiceDataResult, error) {
	if m.getByUserUUIDFn != nil {
		return m.getByUserUUIDFn(userUUID)
	}
	return nil, nil
}
func (m *mockProfileService) GetAll(userUUID uuid.UUID, firstName, lastName, email, phone, city, country *string, isDefault *bool, page, limit int, sortBy, sortOrder string) (*service.ProfileServiceListResult, error) {
	if m.getAllFn != nil {
		return m.getAllFn(userUUID, firstName, lastName, email, phone, city, country, isDefault, page, limit, sortBy, sortOrder)
	}
	return &service.ProfileServiceListResult{}, nil
}
func (m *mockProfileService) SetDefaultProfile(profileUUID uuid.UUID, userUUID uuid.UUID) (*service.ProfileServiceDataResult, error) {
	if m.setDefaultFn != nil {
		return m.setDefaultFn(profileUUID, userUUID)
	}
	return nil, nil
}
func (m *mockProfileService) DeleteByUUID(profileUUID uuid.UUID, userUUID uuid.UUID) (*service.ProfileServiceDataResult, error) {
	if m.deleteByUUIDFn != nil {
		return m.deleteByUUIDFn(profileUUID, userUUID)
	}
	return nil, nil
}

// ---------------------------------------------------------------------------
// mockInviteService
// ---------------------------------------------------------------------------

type mockInviteService struct {
	sendInviteFn func(int64, string, int64, []string) (*model.Invite, error)
}

func (m *mockInviteService) SendInvite(_ context.Context, tenantID int64, email string, userID int64, roleUUIDs []string) (*model.Invite, error) {
	if m.sendInviteFn != nil {
		return m.sendInviteFn(tenantID, email, userID, roleUUIDs)
	}
	return nil, nil
}

// ---------------------------------------------------------------------------
// mockSecuritySettingService
// ---------------------------------------------------------------------------

type mockSecuritySettingService struct {
	getByTenantIDFn        func(int64) (*service.SecuritySettingServiceDataResult, error)
	getGeneralConfigFn     func(int64) (map[string]any, error)
	getPasswordConfigFn    func(int64) (map[string]any, error)
	getSessionConfigFn     func(int64) (map[string]any, error)
	getThreatConfigFn      func(int64) (map[string]any, error)
	getIPConfigFn          func(int64) (map[string]any, error)
	updateGeneralConfigFn  func(int64, map[string]any, int64, string, string) (*service.SecuritySettingServiceDataResult, error)
	updatePasswordConfigFn func(int64, map[string]any, int64, string, string) (*service.SecuritySettingServiceDataResult, error)
	updateSessionConfigFn  func(int64, map[string]any, int64, string, string) (*service.SecuritySettingServiceDataResult, error)
	updateThreatConfigFn   func(int64, map[string]any, int64, string, string) (*service.SecuritySettingServiceDataResult, error)
	updateIPConfigFn       func(int64, map[string]any, int64, string, string) (*service.SecuritySettingServiceDataResult, error)
}

func (m *mockSecuritySettingService) GetByTenantID(tid int64) (*service.SecuritySettingServiceDataResult, error) {
	if m.getByTenantIDFn != nil {
		return m.getByTenantIDFn(tid)
	}
	return nil, nil
}
func (m *mockSecuritySettingService) GetGeneralConfig(tid int64) (map[string]any, error) {
	if m.getGeneralConfigFn != nil {
		return m.getGeneralConfigFn(tid)
	}
	return nil, nil
}
func (m *mockSecuritySettingService) GetPasswordConfig(tid int64) (map[string]any, error) {
	if m.getPasswordConfigFn != nil {
		return m.getPasswordConfigFn(tid)
	}
	return nil, nil
}
func (m *mockSecuritySettingService) GetSessionConfig(tid int64) (map[string]any, error) {
	if m.getSessionConfigFn != nil {
		return m.getSessionConfigFn(tid)
	}
	return nil, nil
}
func (m *mockSecuritySettingService) GetThreatConfig(tid int64) (map[string]any, error) {
	if m.getThreatConfigFn != nil {
		return m.getThreatConfigFn(tid)
	}
	return nil, nil
}
func (m *mockSecuritySettingService) GetIPConfig(tid int64) (map[string]any, error) {
	if m.getIPConfigFn != nil {
		return m.getIPConfigFn(tid)
	}
	return nil, nil
}
func (m *mockSecuritySettingService) UpdateGeneralConfig(tid int64, cfg map[string]any, by int64, ip, ua string) (*service.SecuritySettingServiceDataResult, error) {
	if m.updateGeneralConfigFn != nil {
		return m.updateGeneralConfigFn(tid, cfg, by, ip, ua)
	}
	return nil, nil
}
func (m *mockSecuritySettingService) UpdatePasswordConfig(tid int64, cfg map[string]any, by int64, ip, ua string) (*service.SecuritySettingServiceDataResult, error) {
	if m.updatePasswordConfigFn != nil {
		return m.updatePasswordConfigFn(tid, cfg, by, ip, ua)
	}
	return nil, nil
}
func (m *mockSecuritySettingService) UpdateSessionConfig(tid int64, cfg map[string]any, by int64, ip, ua string) (*service.SecuritySettingServiceDataResult, error) {
	if m.updateSessionConfigFn != nil {
		return m.updateSessionConfigFn(tid, cfg, by, ip, ua)
	}
	return nil, nil
}
func (m *mockSecuritySettingService) UpdateThreatConfig(tid int64, cfg map[string]any, by int64, ip, ua string) (*service.SecuritySettingServiceDataResult, error) {
	if m.updateThreatConfigFn != nil {
		return m.updateThreatConfigFn(tid, cfg, by, ip, ua)
	}
	return nil, nil
}
func (m *mockSecuritySettingService) UpdateIPConfig(tid int64, cfg map[string]any, by int64, ip, ua string) (*service.SecuritySettingServiceDataResult, error) {
	if m.updateIPConfigFn != nil {
		return m.updateIPConfigFn(tid, cfg, by, ip, ua)
	}
	return nil, nil
}

// ---------------------------------------------------------------------------
// mockPermissionService
// ---------------------------------------------------------------------------

type mockPermissionService struct {
	getFn                   func(service.PermissionServiceGetFilter) (*service.PermissionServiceGetResult, error)
	getByUUIDFn             func(uuid.UUID, int64) (*service.PermissionServiceDataResult, error)
	createFn                func(int64, string, string, string, bool, string) (*service.PermissionServiceDataResult, error)
	updateFn                func(uuid.UUID, int64, string, string, string) (*service.PermissionServiceDataResult, error)
	setActiveStatusByUUIDFn func(uuid.UUID, int64) (*service.PermissionServiceDataResult, error)
	setStatusFn             func(uuid.UUID, int64, string) (*service.PermissionServiceDataResult, error)
	deleteByUUIDFn          func(uuid.UUID, int64) (*service.PermissionServiceDataResult, error)
}

func (m *mockPermissionService) Get(f service.PermissionServiceGetFilter) (*service.PermissionServiceGetResult, error) {
	if m.getFn != nil {
		return m.getFn(f)
	}
	return &service.PermissionServiceGetResult{}, nil
}
func (m *mockPermissionService) GetByUUID(id uuid.UUID, tid int64) (*service.PermissionServiceDataResult, error) {
	if m.getByUUIDFn != nil {
		return m.getByUUIDFn(id, tid)
	}
	return nil, nil
}
func (m *mockPermissionService) Create(tid int64, name, desc, status string, isSys bool, apiUUID string) (*service.PermissionServiceDataResult, error) {
	if m.createFn != nil {
		return m.createFn(tid, name, desc, status, isSys, apiUUID)
	}
	return nil, nil
}
func (m *mockPermissionService) Update(id uuid.UUID, tid int64, name, desc, status string) (*service.PermissionServiceDataResult, error) {
	if m.updateFn != nil {
		return m.updateFn(id, tid, name, desc, status)
	}
	return nil, nil
}
func (m *mockPermissionService) SetActiveStatusByUUID(id uuid.UUID, tid int64) (*service.PermissionServiceDataResult, error) {
	if m.setActiveStatusByUUIDFn != nil {
		return m.setActiveStatusByUUIDFn(id, tid)
	}
	return nil, nil
}
func (m *mockPermissionService) SetStatus(id uuid.UUID, tid int64, status string) (*service.PermissionServiceDataResult, error) {
	if m.setStatusFn != nil {
		return m.setStatusFn(id, tid, status)
	}
	return nil, nil
}
func (m *mockPermissionService) DeleteByUUID(id uuid.UUID, tid int64) (*service.PermissionServiceDataResult, error) {
	if m.deleteByUUIDFn != nil {
		return m.deleteByUUIDFn(id, tid)
	}
	return nil, nil
}

// ---------------------------------------------------------------------------
// mockPolicyService
// ---------------------------------------------------------------------------

type mockPolicyService struct {
	getFn                     func(service.PolicyServiceGetFilter) (*service.PolicyServiceGetResult, error)
	getByUUIDFn               func(uuid.UUID, int64) (*service.PolicyServiceDataResult, error)
	getServicesByPolicyUUIDFn func(uuid.UUID, int64, service.PolicyServiceServicesFilter) (*service.PolicyServiceServicesResult, error)
	createFn                  func(int64, string, *string, datatypes.JSON, string, string, bool) (*service.PolicyServiceDataResult, error)
	updateFn                  func(uuid.UUID, int64, string, *string, datatypes.JSON, string, string) (*service.PolicyServiceDataResult, error)
	setStatusByUUIDFn         func(uuid.UUID, int64, string) (*service.PolicyServiceDataResult, error)
	deleteByUUIDFn            func(uuid.UUID, int64) (*service.PolicyServiceDataResult, error)
}

func (m *mockPolicyService) Get(f service.PolicyServiceGetFilter) (*service.PolicyServiceGetResult, error) {
	if m.getFn != nil {
		return m.getFn(f)
	}
	return &service.PolicyServiceGetResult{}, nil
}
func (m *mockPolicyService) GetByUUID(id uuid.UUID, tid int64) (*service.PolicyServiceDataResult, error) {
	if m.getByUUIDFn != nil {
		return m.getByUUIDFn(id, tid)
	}
	return nil, nil
}
func (m *mockPolicyService) GetServicesByPolicyUUID(id uuid.UUID, tid int64, f service.PolicyServiceServicesFilter) (*service.PolicyServiceServicesResult, error) {
	if m.getServicesByPolicyUUIDFn != nil {
		return m.getServicesByPolicyUUIDFn(id, tid, f)
	}
	return &service.PolicyServiceServicesResult{}, nil
}
func (m *mockPolicyService) Create(tid int64, name string, desc *string, doc datatypes.JSON, ver, status string, isSys bool) (*service.PolicyServiceDataResult, error) {
	if m.createFn != nil {
		return m.createFn(tid, name, desc, doc, ver, status, isSys)
	}
	return nil, nil
}
func (m *mockPolicyService) Update(id uuid.UUID, tid int64, name string, desc *string, doc datatypes.JSON, ver, status string) (*service.PolicyServiceDataResult, error) {
	if m.updateFn != nil {
		return m.updateFn(id, tid, name, desc, doc, ver, status)
	}
	return nil, nil
}
func (m *mockPolicyService) SetStatusByUUID(id uuid.UUID, tid int64, status string) (*service.PolicyServiceDataResult, error) {
	if m.setStatusByUUIDFn != nil {
		return m.setStatusByUUIDFn(id, tid, status)
	}
	return nil, nil
}
func (m *mockPolicyService) DeleteByUUID(id uuid.UUID, tid int64) (*service.PolicyServiceDataResult, error) {
	if m.deleteByUUIDFn != nil {
		return m.deleteByUUIDFn(id, tid)
	}
	return nil, nil
}

// ---------------------------------------------------------------------------
// mockEmailTemplateService
// ---------------------------------------------------------------------------

type mockEmailTemplateService struct {
	getAllFn       func(int64, *string, []string, *bool, *bool, int, int, string, string) (*service.EmailTemplateServiceListResult, error)
	getByUUIDFn    func(uuid.UUID, int64) (*service.EmailTemplateServiceDataResult, error)
	createFn       func(int64, string, string, string, *string, string, bool) (*service.EmailTemplateServiceDataResult, error)
	updateFn       func(uuid.UUID, int64, string, string, string, *string, string) (*service.EmailTemplateServiceDataResult, error)
	updateStatusFn func(uuid.UUID, int64, string) (*service.EmailTemplateServiceDataResult, error)
	deleteFn       func(uuid.UUID, int64) (*service.EmailTemplateServiceDataResult, error)
}

func (m *mockEmailTemplateService) GetAll(tid int64, name *string, status []string, isDefault, isSystem *bool, page, limit int, sortBy, sortOrder string) (*service.EmailTemplateServiceListResult, error) {
	if m.getAllFn != nil {
		return m.getAllFn(tid, name, status, isDefault, isSystem, page, limit, sortBy, sortOrder)
	}
	return &service.EmailTemplateServiceListResult{}, nil
}
func (m *mockEmailTemplateService) GetByUUID(id uuid.UUID, tid int64) (*service.EmailTemplateServiceDataResult, error) {
	if m.getByUUIDFn != nil {
		return m.getByUUIDFn(id, tid)
	}
	return nil, nil
}
func (m *mockEmailTemplateService) Create(tid int64, name, subject, bodyHTML string, bodyPlain *string, status string, isDefault bool) (*service.EmailTemplateServiceDataResult, error) {
	if m.createFn != nil {
		return m.createFn(tid, name, subject, bodyHTML, bodyPlain, status, isDefault)
	}
	return nil, nil
}
func (m *mockEmailTemplateService) Update(id uuid.UUID, tid int64, name, subject, bodyHTML string, bodyPlain *string, status string) (*service.EmailTemplateServiceDataResult, error) {
	if m.updateFn != nil {
		return m.updateFn(id, tid, name, subject, bodyHTML, bodyPlain, status)
	}
	return nil, nil
}
func (m *mockEmailTemplateService) UpdateStatus(id uuid.UUID, tid int64, status string) (*service.EmailTemplateServiceDataResult, error) {
	if m.updateStatusFn != nil {
		return m.updateStatusFn(id, tid, status)
	}
	return nil, nil
}
func (m *mockEmailTemplateService) Delete(id uuid.UUID, tid int64) (*service.EmailTemplateServiceDataResult, error) {
	if m.deleteFn != nil {
		return m.deleteFn(id, tid)
	}
	return nil, nil
}

// ---------------------------------------------------------------------------
// mockSMSTemplateService
// ---------------------------------------------------------------------------

type mockSMSTemplateService struct {
	getAllFn       func(int64, *string, []string, *bool, *bool, int, int, string, string) (*service.SMSTemplateServiceListResult, error)
	getByUUIDFn    func(uuid.UUID, int64) (*service.SMSTemplateServiceDataResult, error)
	createFn       func(int64, string, *string, string, *string, string) (*service.SMSTemplateServiceDataResult, error)
	updateFn       func(uuid.UUID, int64, string, *string, string, *string, string) (*service.SMSTemplateServiceDataResult, error)
	updateStatusFn func(uuid.UUID, int64, string) (*service.SMSTemplateServiceDataResult, error)
	deleteFn       func(uuid.UUID, int64) (*service.SMSTemplateServiceDataResult, error)
}

func (m *mockSMSTemplateService) GetAll(tid int64, name *string, status []string, isDefault, isSystem *bool, page, limit int, sortBy, sortOrder string) (*service.SMSTemplateServiceListResult, error) {
	if m.getAllFn != nil {
		return m.getAllFn(tid, name, status, isDefault, isSystem, page, limit, sortBy, sortOrder)
	}
	return &service.SMSTemplateServiceListResult{}, nil
}
func (m *mockSMSTemplateService) GetByUUID(id uuid.UUID, tid int64) (*service.SMSTemplateServiceDataResult, error) {
	if m.getByUUIDFn != nil {
		return m.getByUUIDFn(id, tid)
	}
	return nil, nil
}
func (m *mockSMSTemplateService) Create(tid int64, name string, desc *string, message string, senderID *string, status string) (*service.SMSTemplateServiceDataResult, error) {
	if m.createFn != nil {
		return m.createFn(tid, name, desc, message, senderID, status)
	}
	return nil, nil
}
func (m *mockSMSTemplateService) Update(id uuid.UUID, tid int64, name string, desc *string, message string, senderID *string, status string) (*service.SMSTemplateServiceDataResult, error) {
	if m.updateFn != nil {
		return m.updateFn(id, tid, name, desc, message, senderID, status)
	}
	return nil, nil
}
func (m *mockSMSTemplateService) UpdateStatus(id uuid.UUID, tid int64, status string) (*service.SMSTemplateServiceDataResult, error) {
	if m.updateStatusFn != nil {
		return m.updateStatusFn(id, tid, status)
	}
	return nil, nil
}
func (m *mockSMSTemplateService) Delete(id uuid.UUID, tid int64) (*service.SMSTemplateServiceDataResult, error) {
	if m.deleteFn != nil {
		return m.deleteFn(id, tid)
	}
	return nil, nil
}

// ---------------------------------------------------------------------------
// mockLoginTemplateService
// ---------------------------------------------------------------------------

type mockLoginTemplateService struct {
	getAllFn       func(int64, *string, []string, *string, *bool, *bool, int, int, string, string) (*service.LoginTemplateServiceListResult, error)
	getByUUIDFn    func(uuid.UUID, int64) (*service.LoginTemplateServiceDataResult, error)
	createFn       func(int64, string, *string, string, map[string]any, string) (*service.LoginTemplateServiceDataResult, error)
	updateFn       func(uuid.UUID, int64, string, *string, string, map[string]any, string) (*service.LoginTemplateServiceDataResult, error)
	updateStatusFn func(uuid.UUID, int64, string) (*service.LoginTemplateServiceDataResult, error)
	deleteFn       func(uuid.UUID, int64) (*service.LoginTemplateServiceDataResult, error)
}

func (m *mockLoginTemplateService) GetAll(tid int64, name *string, status []string, tmpl *string, isDefault, isSystem *bool, page, limit int, sortBy, sortOrder string) (*service.LoginTemplateServiceListResult, error) {
	if m.getAllFn != nil {
		return m.getAllFn(tid, name, status, tmpl, isDefault, isSystem, page, limit, sortBy, sortOrder)
	}
	return &service.LoginTemplateServiceListResult{}, nil
}
func (m *mockLoginTemplateService) GetByUUID(id uuid.UUID, tid int64) (*service.LoginTemplateServiceDataResult, error) {
	if m.getByUUIDFn != nil {
		return m.getByUUIDFn(id, tid)
	}
	return nil, nil
}
func (m *mockLoginTemplateService) Create(tid int64, name string, desc *string, tmpl string, metadata map[string]any, status string) (*service.LoginTemplateServiceDataResult, error) {
	if m.createFn != nil {
		return m.createFn(tid, name, desc, tmpl, metadata, status)
	}
	return nil, nil
}
func (m *mockLoginTemplateService) Update(id uuid.UUID, tid int64, name string, desc *string, tmpl string, metadata map[string]any, status string) (*service.LoginTemplateServiceDataResult, error) {
	if m.updateFn != nil {
		return m.updateFn(id, tid, name, desc, tmpl, metadata, status)
	}
	return nil, nil
}
func (m *mockLoginTemplateService) UpdateStatus(id uuid.UUID, tid int64, status string) (*service.LoginTemplateServiceDataResult, error) {
	if m.updateStatusFn != nil {
		return m.updateStatusFn(id, tid, status)
	}
	return nil, nil
}
func (m *mockLoginTemplateService) Delete(id uuid.UUID, tid int64) (*service.LoginTemplateServiceDataResult, error) {
	if m.deleteFn != nil {
		return m.deleteFn(id, tid)
	}
	return nil, nil
}

// ---------------------------------------------------------------------------
// mockIdentityProviderService
// ---------------------------------------------------------------------------

type mockIdentityProviderService struct {
	getFn             func(service.IdentityProviderServiceGetFilter) (*service.IdentityProviderServiceGetResult, error)
	getByUUIDFn       func(uuid.UUID, int64) (*service.IdentityProviderServiceDataResult, error)
	createFn          func(string, string, string, string, datatypes.JSON, string, string, int64, uuid.UUID) (*service.IdentityProviderServiceDataResult, error)
	updateFn          func(uuid.UUID, string, string, string, string, datatypes.JSON, string, int64, uuid.UUID) (*service.IdentityProviderServiceDataResult, error)
	setStatusByUUIDFn func(uuid.UUID, string, int64, uuid.UUID) (*service.IdentityProviderServiceDataResult, error)
	deleteByUUIDFn    func(uuid.UUID, int64, uuid.UUID) (*service.IdentityProviderServiceDataResult, error)
}

func (m *mockIdentityProviderService) Get(f service.IdentityProviderServiceGetFilter) (*service.IdentityProviderServiceGetResult, error) {
	if m.getFn != nil {
		return m.getFn(f)
	}
	return &service.IdentityProviderServiceGetResult{}, nil
}
func (m *mockIdentityProviderService) GetByUUID(id uuid.UUID, tid int64) (*service.IdentityProviderServiceDataResult, error) {
	if m.getByUUIDFn != nil {
		return m.getByUUIDFn(id, tid)
	}
	return nil, nil
}
func (m *mockIdentityProviderService) Create(name, displayName, provider, providerType string, config datatypes.JSON, status, tenantUUID string, tenantID int64, actor uuid.UUID) (*service.IdentityProviderServiceDataResult, error) {
	if m.createFn != nil {
		return m.createFn(name, displayName, provider, providerType, config, status, tenantUUID, tenantID, actor)
	}
	return nil, nil
}
func (m *mockIdentityProviderService) Update(id uuid.UUID, name, displayName, provider, providerType string, config datatypes.JSON, status string, tenantID int64, actor uuid.UUID) (*service.IdentityProviderServiceDataResult, error) {
	if m.updateFn != nil {
		return m.updateFn(id, name, displayName, provider, providerType, config, status, tenantID, actor)
	}
	return nil, nil
}
func (m *mockIdentityProviderService) SetStatusByUUID(id uuid.UUID, status string, tenantID int64, actor uuid.UUID) (*service.IdentityProviderServiceDataResult, error) {
	if m.setStatusByUUIDFn != nil {
		return m.setStatusByUUIDFn(id, status, tenantID, actor)
	}
	return nil, nil
}
func (m *mockIdentityProviderService) DeleteByUUID(id uuid.UUID, tenantID int64, actor uuid.UUID) (*service.IdentityProviderServiceDataResult, error) {
	if m.deleteByUUIDFn != nil {
		return m.deleteByUUIDFn(id, tenantID, actor)
	}
	return nil, nil
}

// ---------------------------------------------------------------------------
// mockIPRestrictionRuleService
// ---------------------------------------------------------------------------

type mockIPRestrictionRuleService struct {
	getAllFn       func(int64, *string, []string, *string, *string, int, int, string, string) (*service.IPRestrictionRuleServiceListResult, error)
	getByUUIDFn    func(int64, uuid.UUID) (*service.IPRestrictionRuleServiceDataResult, error)
	createFn       func(int64, string, string, string, string, int64) (*service.IPRestrictionRuleServiceDataResult, error)
	updateFn       func(int64, uuid.UUID, string, string, string, string, int64) (*service.IPRestrictionRuleServiceDataResult, error)
	updateStatusFn func(int64, uuid.UUID, string, int64) (*service.IPRestrictionRuleServiceDataResult, error)
	deleteFn       func(int64, uuid.UUID) (*service.IPRestrictionRuleServiceDataResult, error)
}

func (m *mockIPRestrictionRuleService) GetAll(tid int64, ruleType *string, status []string, ipAddress, desc *string, page, limit int, sortBy, sortOrder string) (*service.IPRestrictionRuleServiceListResult, error) {
	if m.getAllFn != nil {
		return m.getAllFn(tid, ruleType, status, ipAddress, desc, page, limit, sortBy, sortOrder)
	}
	return &service.IPRestrictionRuleServiceListResult{}, nil
}
func (m *mockIPRestrictionRuleService) GetByUUID(tid int64, id uuid.UUID) (*service.IPRestrictionRuleServiceDataResult, error) {
	if m.getByUUIDFn != nil {
		return m.getByUUIDFn(tid, id)
	}
	return nil, nil
}
func (m *mockIPRestrictionRuleService) Create(tid int64, desc, ruleType, ipAddress, status string, createdBy int64) (*service.IPRestrictionRuleServiceDataResult, error) {
	if m.createFn != nil {
		return m.createFn(tid, desc, ruleType, ipAddress, status, createdBy)
	}
	return nil, nil
}
func (m *mockIPRestrictionRuleService) Update(tid int64, id uuid.UUID, desc, ruleType, ipAddress, status string, updatedBy int64) (*service.IPRestrictionRuleServiceDataResult, error) {
	if m.updateFn != nil {
		return m.updateFn(tid, id, desc, ruleType, ipAddress, status, updatedBy)
	}
	return nil, nil
}
func (m *mockIPRestrictionRuleService) UpdateStatus(tid int64, id uuid.UUID, status string, updatedBy int64) (*service.IPRestrictionRuleServiceDataResult, error) {
	if m.updateStatusFn != nil {
		return m.updateStatusFn(tid, id, status, updatedBy)
	}
	return nil, nil
}
func (m *mockIPRestrictionRuleService) Delete(tid int64, id uuid.UUID) (*service.IPRestrictionRuleServiceDataResult, error) {
	if m.deleteFn != nil {
		return m.deleteFn(tid, id)
	}
	return nil, nil
}

// ---------------------------------------------------------------------------
// mockSignupFlowService
// ---------------------------------------------------------------------------

type mockSignupFlowService struct {
	getAllFn       func(int64, *string, *string, []string, *uuid.UUID, int, int, string, string) (*service.SignupFlowServiceListResult, error)
	getByUUIDFn    func(uuid.UUID, int64) (*service.SignupFlowServiceDataResult, error)
	createFn       func(int64, string, string, map[string]any, string, uuid.UUID) (*service.SignupFlowServiceDataResult, error)
	updateFn       func(uuid.UUID, int64, string, string, map[string]any, string) (*service.SignupFlowServiceDataResult, error)
	updateStatusFn func(uuid.UUID, int64, string) (*service.SignupFlowServiceDataResult, error)
	deleteFn       func(uuid.UUID, int64) (*service.SignupFlowServiceDataResult, error)
	assignRolesFn  func(uuid.UUID, int64, []uuid.UUID) ([]service.SignupFlowRoleServiceDataResult, error)
	getRolesFn     func(uuid.UUID, int64, int, int) (*service.SignupFlowRoleServiceListResult, error)
	removeRoleFn   func(uuid.UUID, int64, uuid.UUID) error
}

func (m *mockSignupFlowService) GetAll(tid int64, name, identifier *string, status []string, clientUUID *uuid.UUID, page, limit int, sortBy, sortOrder string) (*service.SignupFlowServiceListResult, error) {
	if m.getAllFn != nil {
		return m.getAllFn(tid, name, identifier, status, clientUUID, page, limit, sortBy, sortOrder)
	}
	return &service.SignupFlowServiceListResult{}, nil
}
func (m *mockSignupFlowService) GetByUUID(id uuid.UUID, tid int64) (*service.SignupFlowServiceDataResult, error) {
	if m.getByUUIDFn != nil {
		return m.getByUUIDFn(id, tid)
	}
	return nil, nil
}
func (m *mockSignupFlowService) Create(tid int64, name, desc string, config map[string]any, status string, clientUUID uuid.UUID) (*service.SignupFlowServiceDataResult, error) {
	if m.createFn != nil {
		return m.createFn(tid, name, desc, config, status, clientUUID)
	}
	return nil, nil
}
func (m *mockSignupFlowService) Update(id uuid.UUID, tid int64, name, desc string, config map[string]any, status string) (*service.SignupFlowServiceDataResult, error) {
	if m.updateFn != nil {
		return m.updateFn(id, tid, name, desc, config, status)
	}
	return nil, nil
}
func (m *mockSignupFlowService) UpdateStatus(id uuid.UUID, tid int64, status string) (*service.SignupFlowServiceDataResult, error) {
	if m.updateStatusFn != nil {
		return m.updateStatusFn(id, tid, status)
	}
	return nil, nil
}
func (m *mockSignupFlowService) Delete(id uuid.UUID, tid int64) (*service.SignupFlowServiceDataResult, error) {
	if m.deleteFn != nil {
		return m.deleteFn(id, tid)
	}
	return nil, nil
}
func (m *mockSignupFlowService) AssignRoles(id uuid.UUID, tid int64, roles []uuid.UUID) ([]service.SignupFlowRoleServiceDataResult, error) {
	if m.assignRolesFn != nil {
		return m.assignRolesFn(id, tid, roles)
	}
	return nil, nil
}
func (m *mockSignupFlowService) GetRoles(id uuid.UUID, tid int64, page, limit int) (*service.SignupFlowRoleServiceListResult, error) {
	if m.getRolesFn != nil {
		return m.getRolesFn(id, tid, page, limit)
	}
	return &service.SignupFlowRoleServiceListResult{}, nil
}
func (m *mockSignupFlowService) RemoveRole(id uuid.UUID, tid int64, roleID uuid.UUID) error {
	if m.removeRoleFn != nil {
		return m.removeRoleFn(id, tid, roleID)
	}
	return nil
}

// ---------------------------------------------------------------------------
// mockTenantMemberService
// ---------------------------------------------------------------------------

type mockTenantMemberService struct {
	createFn             func(int64, int64, string) (*service.TenantMemberServiceDataResult, error)
	createByUserUUIDFn   func(int64, uuid.UUID, string) (*service.TenantMemberServiceDataResult, error)
	getByUUIDFn          func(uuid.UUID) (*service.TenantMemberServiceDataResult, error)
	getByTenantAndUserFn func(int64, int64) (*service.TenantMemberServiceDataResult, error)
	listByTenantFn       func(int64) ([]service.TenantMemberServiceDataResult, error)
	listByUserFn         func(int64) ([]service.TenantMemberServiceDataResult, error)
	updateRoleFn         func(uuid.UUID, string) (*service.TenantMemberServiceDataResult, error)
	deleteByUUIDFn       func(uuid.UUID) error
	isUserInTenantFn     func(int64, uuid.UUID) (bool, error)
}

func (m *mockTenantMemberService) Create(tenantID, userID int64, role string) (*service.TenantMemberServiceDataResult, error) {
	if m.createFn != nil {
		return m.createFn(tenantID, userID, role)
	}
	return nil, nil
}
func (m *mockTenantMemberService) CreateByUserUUID(tenantID int64, userUUID uuid.UUID, role string) (*service.TenantMemberServiceDataResult, error) {
	if m.createByUserUUIDFn != nil {
		return m.createByUserUUIDFn(tenantID, userUUID, role)
	}
	return nil, nil
}
func (m *mockTenantMemberService) GetByUUID(id uuid.UUID) (*service.TenantMemberServiceDataResult, error) {
	if m.getByUUIDFn != nil {
		return m.getByUUIDFn(id)
	}
	return nil, nil
}
func (m *mockTenantMemberService) GetByTenantAndUser(tenantID, userID int64) (*service.TenantMemberServiceDataResult, error) {
	if m.getByTenantAndUserFn != nil {
		return m.getByTenantAndUserFn(tenantID, userID)
	}
	return nil, nil
}
func (m *mockTenantMemberService) ListByTenant(tenantID int64) ([]service.TenantMemberServiceDataResult, error) {
	if m.listByTenantFn != nil {
		return m.listByTenantFn(tenantID)
	}
	return nil, nil
}
func (m *mockTenantMemberService) ListByUser(userID int64) ([]service.TenantMemberServiceDataResult, error) {
	if m.listByUserFn != nil {
		return m.listByUserFn(userID)
	}
	return nil, nil
}
func (m *mockTenantMemberService) UpdateRole(id uuid.UUID, role string) (*service.TenantMemberServiceDataResult, error) {
	if m.updateRoleFn != nil {
		return m.updateRoleFn(id, role)
	}
	return nil, nil
}
func (m *mockTenantMemberService) DeleteByUUID(id uuid.UUID) error {
	if m.deleteByUUIDFn != nil {
		return m.deleteByUUIDFn(id)
	}
	return nil
}
func (m *mockTenantMemberService) IsUserInTenant(userID int64, tenantUUID uuid.UUID) (bool, error) {
	if m.isUserInTenantFn != nil {
		return m.isUserInTenantFn(userID, tenantUUID)
	}
	return false, nil
}

// ---------------------------------------------------------------------------
// mockUserSettingService
// ---------------------------------------------------------------------------

type mockUserSettingService struct {
	createOrUpdateFn func(uuid.UUID, *string, *string, *string, map[string]any, *string, *bool, *bool, *bool, *string, *bool, *time.Time, *time.Time, *string, *string, *string, *string) (*service.UserSettingServiceDataResult, error)
	getByUUIDFn      func(uuid.UUID) (*service.UserSettingServiceDataResult, error)
	getByUserUUIDFn  func(uuid.UUID) (*service.UserSettingServiceDataResult, error)
	deleteByUUIDFn   func(uuid.UUID) (*service.UserSettingServiceDataResult, error)
}

func (m *mockUserSettingService) CreateOrUpdateUserSetting(userUUID uuid.UUID, timezone, preferredLanguage, locale *string, socialLinks map[string]any, preferredContactMethod *string, marketingEmailConsent, smsNotificationsConsent, pushNotificationsConsent *bool, profileVisibility *string, dataProcessingConsent *bool, termsAcceptedAt, privacyPolicyAcceptedAt *time.Time, emergencyContactName, emergencyContactPhone, emergencyContactEmail, emergencyContactRelation *string) (*service.UserSettingServiceDataResult, error) {
	if m.createOrUpdateFn != nil {
		return m.createOrUpdateFn(userUUID, timezone, preferredLanguage, locale, socialLinks, preferredContactMethod, marketingEmailConsent, smsNotificationsConsent, pushNotificationsConsent, profileVisibility, dataProcessingConsent, termsAcceptedAt, privacyPolicyAcceptedAt, emergencyContactName, emergencyContactPhone, emergencyContactEmail, emergencyContactRelation)
	}
	return nil, nil
}
func (m *mockUserSettingService) GetByUUID(id uuid.UUID) (*service.UserSettingServiceDataResult, error) {
	if m.getByUUIDFn != nil {
		return m.getByUUIDFn(id)
	}
	return nil, nil
}
func (m *mockUserSettingService) GetByUserUUID(userUUID uuid.UUID) (*service.UserSettingServiceDataResult, error) {
	if m.getByUserUUIDFn != nil {
		return m.getByUserUUIDFn(userUUID)
	}
	return nil, nil
}
func (m *mockUserSettingService) DeleteByUUID(id uuid.UUID) (*service.UserSettingServiceDataResult, error) {
	if m.deleteByUUIDFn != nil {
		return m.deleteByUUIDFn(id)
	}
	return nil, nil
}
