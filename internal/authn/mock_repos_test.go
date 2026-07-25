package authn

import (
	"context"
	"encoding/json"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/maintainerd/maintainerd-auth/internal/authevent"
	"github.com/maintainerd/maintainerd-auth/internal/branding"
	"github.com/maintainerd/maintainerd-auth/internal/platform/apperror"
	"github.com/maintainerd/maintainerd-auth/internal/platform/signedurl"
	"gorm.io/gorm"
)

// ---------------------------------------------------------------------------
// Mock: RoleRepository
// ---------------------------------------------------------------------------

type mockRoleRepo struct {
	findPaginatedFn         func(RoleRepositoryGetFilter) (*PaginationResult[Role], error)
	findByNameAndTenantIDFn func(string, int64) (*Role, error)
}

func (m *mockRoleRepo) WithTx(_ *gorm.DB) RoleRepository { return m }
func (m *mockRoleRepo) Create(e *Role) (*Role, error)    { return e, nil }
func (m *mockRoleRepo) CreateOrUpdate(e *Role) (*Role, error) {
	return e, nil
}
func (m *mockRoleRepo) FindAll(p ...string) ([]Role, error) { return nil, nil }
func (m *mockRoleRepo) FindByUUID(id any, p ...string) (*Role, error) {
	return nil, nil
}
func (m *mockRoleRepo) FindByUUIDs(ids []string, p ...string) ([]Role, error) { return nil, nil }
func (m *mockRoleRepo) FindByID(id any, p ...string) (*Role, error)           { return nil, nil }
func (m *mockRoleRepo) UpdateByUUID(id, data any) (*Role, error)              { return nil, nil }
func (m *mockRoleRepo) UpdateByID(id, data any) (*Role, error)                { return nil, nil }
func (m *mockRoleRepo) DeleteByUUID(id any) error                             { return nil }
func (m *mockRoleRepo) DeleteByID(id any) error                               { return nil }
func (m *mockRoleRepo) Paginate(c map[string]any, pg, lim int, p ...string) (*PaginationResult[Role], error) {
	return nil, nil
}
func (m *mockRoleRepo) FindPaginated(f RoleRepositoryGetFilter) (*PaginationResult[Role], error) {
	if m.findPaginatedFn != nil {
		return m.findPaginatedFn(f)
	}
	return &PaginationResult[Role]{}, nil
}
func (m *mockRoleRepo) FindByNameAndTenantID(name string, tenantID int64) (*Role, error) {
	if m.findByNameAndTenantIDFn != nil {
		return m.findByNameAndTenantIDFn(name, tenantID)
	}
	return nil, nil
}

// ---------------------------------------------------------------------------
// Mock: UserRoleRepository
// ---------------------------------------------------------------------------

type mockUserRoleRepo struct {
	createFn                func(*UserRole) (*UserRole, error)
	findByUserIDAndRoleIDFn func(int64, int64) (*UserRole, error)
}

func (m *mockUserRoleRepo) WithTx(_ *gorm.DB) UserRoleRepository { return m }
func (m *mockUserRoleRepo) Create(e *UserRole) (*UserRole, error) {
	if m.createFn != nil {
		return m.createFn(e)
	}
	return e, nil
}
func (m *mockUserRoleRepo) CreateOrUpdate(e *UserRole) (*UserRole, error) { return e, nil }
func (m *mockUserRoleRepo) FindAll(p ...string) ([]UserRole, error)       { return nil, nil }
func (m *mockUserRoleRepo) FindByUUID(id any, p ...string) (*UserRole, error) {
	return nil, nil
}
func (m *mockUserRoleRepo) FindByUUIDs(ids []string, p ...string) ([]UserRole, error) {
	return nil, nil
}
func (m *mockUserRoleRepo) FindByID(id any, p ...string) (*UserRole, error) { return nil, nil }
func (m *mockUserRoleRepo) UpdateByUUID(id, data any) (*UserRole, error)    { return nil, nil }
func (m *mockUserRoleRepo) UpdateByID(id, data any) (*UserRole, error)      { return nil, nil }
func (m *mockUserRoleRepo) DeleteByUUID(id any) error                       { return nil }
func (m *mockUserRoleRepo) DeleteByID(id any) error                         { return nil }
func (m *mockUserRoleRepo) Paginate(c map[string]any, pg, lim int, p ...string) (*PaginationResult[UserRole], error) {
	return nil, nil
}
func (m *mockUserRoleRepo) FindByUserIDAndRoleID(userID, roleID int64) (*UserRole, error) {
	if m.findByUserIDAndRoleIDFn != nil {
		return m.findByUserIDAndRoleIDFn(userID, roleID)
	}
	return nil, nil
}

// ---------------------------------------------------------------------------
// Mock: InviteRepository
// ---------------------------------------------------------------------------

type mockInviteRepo struct {
	findByTokenFn func(string) (*Invite, error)
	markAsUsedFn  func(uuid.UUID) error
}

func (m *mockInviteRepo) WithTx(_ *gorm.DB) InviteRepository { return m }
func (m *mockInviteRepo) Create(e *Invite) (*Invite, error)  { return e, nil }
func (m *mockInviteRepo) CreateOrUpdate(e *Invite) (*Invite, error) {
	return e, nil
}
func (m *mockInviteRepo) FindAll(p ...string) ([]Invite, error) { return nil, nil }
func (m *mockInviteRepo) FindByUUID(id any, p ...string) (*Invite, error) {
	return nil, nil
}
func (m *mockInviteRepo) FindByUUIDs(ids []string, p ...string) ([]Invite, error) { return nil, nil }
func (m *mockInviteRepo) FindByID(id any, p ...string) (*Invite, error)           { return nil, nil }
func (m *mockInviteRepo) UpdateByUUID(id, data any) (*Invite, error)              { return nil, nil }
func (m *mockInviteRepo) UpdateByID(id, data any) (*Invite, error)                { return nil, nil }
func (m *mockInviteRepo) DeleteByUUID(id any) error                               { return nil }
func (m *mockInviteRepo) DeleteByID(id any) error                                 { return nil }
func (m *mockInviteRepo) Paginate(c map[string]any, pg, lim int, p ...string) (*PaginationResult[Invite], error) {
	return nil, nil
}
func (m *mockInviteRepo) FindByToken(token string) (*Invite, error) {
	if m.findByTokenFn != nil {
		return m.findByTokenFn(token)
	}
	return nil, nil
}
func (m *mockInviteRepo) FindByTokenForUpdate(token string) (*Invite, error) {
	if m.findByTokenFn != nil {
		return m.findByTokenFn(token)
	}
	return nil, nil
}
func (m *mockInviteRepo) MarkAsUsed(id uuid.UUID) error {
	if m.markAsUsedFn != nil {
		return m.markAsUsedFn(id)
	}
	return nil
}

// ---------------------------------------------------------------------------
// Mock: branding.EmailTemplateRepository
// ---------------------------------------------------------------------------

type mockEmailTemplateRepo struct {
	findByNameFn func(string) (*branding.EmailTemplate, error)
}

func (m *mockEmailTemplateRepo) Create(e *branding.EmailTemplate) (*branding.EmailTemplate, error) {
	return e, nil
}
func (m *mockEmailTemplateRepo) CreateOrUpdate(e *branding.EmailTemplate) (*branding.EmailTemplate, error) {
	return e, nil
}
func (m *mockEmailTemplateRepo) FindAll(p ...string) ([]branding.EmailTemplate, error) {
	return nil, nil
}
func (m *mockEmailTemplateRepo) FindByUUID(id any, p ...string) (*branding.EmailTemplate, error) {
	return nil, nil
}
func (m *mockEmailTemplateRepo) FindByUUIDs(ids []string, p ...string) ([]branding.EmailTemplate, error) {
	return nil, nil
}
func (m *mockEmailTemplateRepo) FindByID(id any, p ...string) (*branding.EmailTemplate, error) {
	return nil, nil
}
func (m *mockEmailTemplateRepo) UpdateByUUID(id, data any) (*branding.EmailTemplate, error) {
	return nil, nil
}
func (m *mockEmailTemplateRepo) UpdateByID(id, data any) (*branding.EmailTemplate, error) {
	return nil, nil
}
func (m *mockEmailTemplateRepo) DeleteByUUID(id any) error { return nil }
func (m *mockEmailTemplateRepo) DeleteByID(id any) error   { return nil }
func (m *mockEmailTemplateRepo) Paginate(c map[string]any, pg, lim int, p ...string) (*branding.PaginationResult[branding.EmailTemplate], error) {
	return nil, nil
}
func (m *mockEmailTemplateRepo) FindByUUIDAndTenantID(id uuid.UUID, tenantID int64, p ...string) (*branding.EmailTemplate, error) {
	return nil, nil
}
func (m *mockEmailTemplateRepo) FindByName(name string) (*branding.EmailTemplate, error) {
	if m.findByNameFn != nil {
		return m.findByNameFn(name)
	}
	return nil, nil
}
func (m *mockEmailTemplateRepo) FindByNameAndTenantID(name string, tenantID int64) (*branding.EmailTemplate, error) {
	if m.findByNameFn != nil {
		return m.findByNameFn(name)
	}
	return nil, nil
}
func (m *mockEmailTemplateRepo) FindPaginated(f branding.EmailTemplateRepositoryGetFilter) (*branding.PaginationResult[branding.EmailTemplate], error) {
	return nil, nil
}

// ---------------------------------------------------------------------------
// TestMain: ensure package-level env vars required by handler tests persist
// across test functions that use os.Unsetenv in defers.
// ---------------------------------------------------------------------------

func TestMain(m *testing.M) {
	if os.Getenv("HMAC_SECRET_KEY") == "" {
		os.Setenv("HMAC_SECRET_KEY", "test-secret-key-for-unit-tests") //nolint:errcheck
	}
	_ = signedurl.Configure([]byte(os.Getenv("HMAC_SECRET_KEY")))
	m.Run()
}

// ---------------------------------------------------------------------------
// Shared test sentinel errors
// ---------------------------------------------------------------------------

var (
	errUnauthorized = apperror.NewUnauthorized("unauthorized")
	errValidation   = apperror.NewValidation("validation error")
)

// ---------------------------------------------------------------------------
// Mock: LoginService
// ---------------------------------------------------------------------------

type mockLoginService struct {
	loginPublicFn      func(usernameOrEmail, password string, clientID, tenantID *string) (*LoginResponseDTO, error)
	loginFn            func(usernameOrEmail, password string, clientID, tenantID *string) (*LoginResponseDTO, error)
	completeMFALoginFn func(challengeToken, method, code string, assertion []byte, clientID, tenantID *string) (*LoginResponseDTO, error)
	sendMFALoginSMSFn  func(challengeToken string) error
	beginMFAWebAuthnFn func(challengeToken string) (json.RawMessage, error)
	refreshTokenFn     func(refreshToken, sessionID string) (*LoginResponseDTO, error)
	getUserByEmailFn   func(ctx context.Context, email string, tenantID int64) (*User, error)
	logoutFn           func(ctx context.Context, accessToken string) error
}

func (m *mockLoginService) CompleteMFALogin(_ context.Context, challengeToken, method, code string, assertion []byte, clientID, tenantID *string) (*LoginResponseDTO, error) {
	if m.completeMFALoginFn != nil {
		return m.completeMFALoginFn(challengeToken, method, code, assertion, clientID, tenantID)
	}
	return nil, nil
}

func (m *mockLoginService) SendMFALoginSMS(_ context.Context, challengeToken string) error {
	if m.sendMFALoginSMSFn != nil {
		return m.sendMFALoginSMSFn(challengeToken)
	}
	return nil
}

func (m *mockLoginService) SendMFALoginEmailOTP(_ context.Context, _ string) error {
	return nil
}

func (m *mockLoginService) BeginMFALoginWebAuthn(_ context.Context, challengeToken string) (json.RawMessage, error) {
	if m.beginMFAWebAuthnFn != nil {
		return m.beginMFAWebAuthnFn(challengeToken)
	}
	return json.RawMessage(`{}`), nil
}

func (m *mockLoginService) SetMFAFactorAuthenticator(MFAFactorAuthenticator) {}

func (m *mockLoginService) SetUserLockoutRepository(UserLockoutRepository) {}

func (m *mockLoginService) SetTokenRevoker(AccessTokenRevoker) {}

func (m *mockLoginService) MagicLinkMFAChallenge(context.Context, *User, int64) (*LoginResponseDTO, error) {
	return nil, nil
}

func (m *mockLoginService) SMSMFAChallenge(context.Context, *User, int64) (*LoginResponseDTO, error) {
	return nil, nil
}

func (m *mockLoginService) IssueMagicLinkSession(context.Context, string, *User, *Client) (*LoginResponseDTO, error) {
	return nil, nil
}

func (m *mockLoginService) LoginPublic(ctx context.Context, usernameOrEmail, password string, clientID, tenantID *string) (*LoginResponseDTO, error) {
	if m.loginPublicFn != nil {
		return m.loginPublicFn(usernameOrEmail, password, clientID, tenantID)
	}
	return nil, nil
}

func (m *mockLoginService) Login(ctx context.Context, usernameOrEmail, password string, clientID, tenantID *string) (*LoginResponseDTO, error) {
	if m.loginFn != nil {
		return m.loginFn(usernameOrEmail, password, clientID, tenantID)
	}
	return nil, nil
}

func (m *mockLoginService) RefreshToken(ctx context.Context, refreshToken string, sessionID string) (*LoginResponseDTO, error) {
	if m.refreshTokenFn != nil {
		return m.refreshTokenFn(refreshToken, sessionID)
	}
	return nil, nil
}

func (m *mockLoginService) GetUserByEmail(ctx context.Context, email string, tenantID int64) (*User, error) {
	if m.getUserByEmailFn != nil {
		return m.getUserByEmailFn(ctx, email, tenantID)
	}
	return nil, nil
}

func (m *mockLoginService) Logout(ctx context.Context, accessToken string) error {
	if m.logoutFn != nil {
		return m.logoutFn(ctx, accessToken)
	}
	return nil
}

func (m *mockLoginService) ForgetTrustedDevice(ctx context.Context, token string) {}

// ---------------------------------------------------------------------------
// Mock: RegisterService
// ---------------------------------------------------------------------------

type mockRegisterService struct {
	registerPublicFn       func(username, fullname, password string, email, phone *string, clientID, tenantID *string, registrationFlowIdentifier string) (*RegisterResponseDTO, error)
	registerFn             func(username, fullname, password string, email, phone *string, clientID, tenantID *string, registrationFlowIdentifier string) (*RegisterResponseDTO, error)
	registerInvitePublicFn func(username, password, clientID, tenantID, inviteToken string) (*RegisterResponseDTO, error)
	registerInviteFn       func(username, password string, clientID, tenantID *string, inviteToken string) (*RegisterResponseDTO, error)
}

func (m *mockRegisterService) RegisterPublic(ctx context.Context, username, fullname, password string, email, phone *string, clientID, tenantID *string, registrationFlowIdentifier string) (*RegisterResponseDTO, error) {
	if m.registerPublicFn != nil {
		return m.registerPublicFn(username, fullname, password, email, phone, clientID, tenantID, registrationFlowIdentifier)
	}
	return nil, nil
}

func (m *mockRegisterService) Register(ctx context.Context, username, fullname, password string, email, phone *string, clientID, tenantID *string, registrationFlowIdentifier string) (*RegisterResponseDTO, error) {
	if m.registerFn != nil {
		return m.registerFn(username, fullname, password, email, phone, clientID, tenantID, registrationFlowIdentifier)
	}
	return nil, nil
}

func (m *mockRegisterService) RegisterInvitePublic(ctx context.Context, username, password, clientID, tenantID, inviteToken string) (*RegisterResponseDTO, error) {
	if m.registerInvitePublicFn != nil {
		return m.registerInvitePublicFn(username, password, clientID, tenantID, inviteToken)
	}
	return nil, nil
}

func (m *mockRegisterService) RegisterInvite(ctx context.Context, username, password string, clientID, tenantID *string, inviteToken string) (*RegisterResponseDTO, error) {
	if m.registerInviteFn != nil {
		return m.registerInviteFn(username, password, clientID, tenantID, inviteToken)
	}
	return nil, nil
}

// ---------------------------------------------------------------------------
// Mock: ResetPasswordService
// ---------------------------------------------------------------------------

type mockResetPasswordService struct {
	resetPasswordFn func(token, newPassword string, clientID, providerID *string) (*ResetPasswordResponseDTO, error)
}

func (m *mockResetPasswordService) ResetPassword(ctx context.Context, token, newPassword string, clientID, providerID *string) (*ResetPasswordResponseDTO, error) {
	if m.resetPasswordFn != nil {
		return m.resetPasswordFn(token, newPassword, clientID, providerID)
	}
	return &ResetPasswordResponseDTO{Success: true}, nil
}

// ---------------------------------------------------------------------------
// Mock: EmailVerificationService
// ---------------------------------------------------------------------------

type mockEmailVerificationService struct {
	sendVerificationEmailFn func(ctx context.Context, email string, clientID, providerID *string) (*SendEmailVerificationResponseDTO, error)
	verifyEmailFn           func(ctx context.Context, email, otp string) (*VerifyEmailResponseDTO, error)
}

func (m *mockEmailVerificationService) SendVerificationEmail(ctx context.Context, email string, clientID, providerID *string) (*SendEmailVerificationResponseDTO, error) {
	if m.sendVerificationEmailFn != nil {
		return m.sendVerificationEmailFn(ctx, email, clientID, providerID)
	}
	return &SendEmailVerificationResponseDTO{Success: true}, nil
}

func (m *mockEmailVerificationService) VerifyEmail(ctx context.Context, email, otp string, _ ...*string) (*VerifyEmailResponseDTO, error) {
	if m.verifyEmailFn != nil {
		return m.verifyEmailFn(ctx, email, otp)
	}
	return &VerifyEmailResponseDTO{Success: true}, nil
}

// ---------------------------------------------------------------------------
// Mock: ForgotPasswordService
// ---------------------------------------------------------------------------

type mockForgotPasswordService struct {
	sendPasswordResetEmailFn func(email string, clientID, providerID *string, isInternal bool) (*ForgotPasswordResponseDTO, error)
}

func (m *mockForgotPasswordService) SendPasswordResetEmail(ctx context.Context, email string, clientID, providerID *string, isInternal bool) (*ForgotPasswordResponseDTO, error) {
	if m.sendPasswordResetEmailFn != nil {
		return m.sendPasswordResetEmailFn(email, clientID, providerID, isInternal)
	}
	return &ForgotPasswordResponseDTO{Success: true}, nil
}

// ---------------------------------------------------------------------------
// Mock: authevent.AuthEventService (shared across login, sms_login tests)
// ---------------------------------------------------------------------------

type mockAuthEventService struct{}

func (m *mockAuthEventService) Log(_ context.Context, _ authevent.AuthEventInput) {}
func (m *mockAuthEventService) FindPaginated(_ context.Context, _ authevent.AuthEventRepositoryGetFilter) (*authevent.PaginationResult[authevent.AuthEventServiceDataResult], error) {
	return &authevent.PaginationResult[authevent.AuthEventServiceDataResult]{}, nil
}
func (m *mockAuthEventService) FindByUUID(_ context.Context, _ int64, _ uuid.UUID) (*authevent.AuthEventServiceDataResult, error) {
	return nil, nil
}
func (m *mockAuthEventService) CountByEventType(_ context.Context, _ string, _ int64) (int64, error) {
	return 0, nil
}
func (m *mockAuthEventService) DeleteOlderThan(_ context.Context, _ time.Time) (int64, error) {
	return 0, nil
}
func (m *mockAuthEventService) Shutdown() {}
