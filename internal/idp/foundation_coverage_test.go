package idp

import (
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/maintainerd/auth/internal/shared"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/datatypes"
)

func TestTableNames(t *testing.T) {
	assert.Equal(t, "tenants", Tenant{}.TableName())
	assert.Equal(t, "user_identities", UserIdentity{}.TableName())
	assert.Equal(t, "users", User{}.TableName())
	assert.Equal(t, "clients", Client{}.TableName())
	assert.Equal(t, "roles", Role{}.TableName())
	assert.Equal(t, "user_roles", UserRole{}.TableName())
	assert.Equal(t, "identity_providers", IdentityProvider{}.TableName())
	assert.Equal(t, "signup_flows", SignupFlow{}.TableName())
	assert.Equal(t, "signup_flow_roles", SignupFlowRole{}.TableName())
}

func TestModelBeforeCreate(t *testing.T) {
	t.Run("identity provider assigns uuid only when empty", func(t *testing.T) {
		idp := &IdentityProvider{}
		require.NoError(t, idp.BeforeCreate(nil))
		assert.NotEqual(t, uuid.Nil, idp.IdentityProviderUUID)

		existing := uuid.New()
		idp.IdentityProviderUUID = existing
		require.NoError(t, idp.BeforeCreate(nil))
		assert.Equal(t, existing, idp.IdentityProviderUUID)
	})

	t.Run("signup flow assigns uuid and default status", func(t *testing.T) {
		flow := &SignupFlow{}
		require.NoError(t, flow.BeforeCreate(nil))
		assert.NotEqual(t, uuid.Nil, flow.SignupFlowUUID)
		assert.Equal(t, shared.StatusActive, flow.Status)

		existing := uuid.New()
		flow.SignupFlowUUID = existing
		flow.Status = shared.StatusInactive
		require.NoError(t, flow.BeforeCreate(nil))
		assert.Equal(t, existing, flow.SignupFlowUUID)
		assert.Equal(t, shared.StatusInactive, flow.Status)
	})

	t.Run("signup flow role assigns uuid only when empty", func(t *testing.T) {
		role := &SignupFlowRole{}
		require.NoError(t, role.BeforeCreate(nil))
		assert.NotEqual(t, uuid.Nil, role.SignupFlowRoleUUID)

		existing := uuid.New()
		role.SignupFlowRoleUUID = existing
		require.NoError(t, role.BeforeCreate(nil))
		assert.Equal(t, existing, role.SignupFlowRoleUUID)
	})
}

func TestDepsHelpers(t *testing.T) {
	t.Run("validate tenant access", func(t *testing.T) {
		target := &Tenant{TenantID: tenantID}
		assert.Error(t, ValidateTenantAccess(nil, target))
		assert.Error(t, ValidateTenantAccess(&User{}, nil))
		assert.Error(t, ValidateTenantAccess(&User{}, target))
		assert.NoError(t, ValidateTenantAccess(&User{UserIdentities: []UserIdentity{{TenantID: tenantID}}}, target))
		assert.NoError(t, ValidateTenantAccess(&User{UserIdentities: []UserIdentity{{TenantID: 99, Tenant: &Tenant{IsSystem: true}}}}, target))
		assert.Error(t, ValidateTenantAccess(&User{UserIdentities: []UserIdentity{{TenantID: 99}}}, target))
	})

	t.Run("tenant dto handles nil and full value", func(t *testing.T) {
		assert.Nil(t, toTenantServiceDataResult(nil))
		now := time.Now()
		tnt := &Tenant{
			TenantUUID:  uuid.New(),
			Name:        "tenant",
			DisplayName: "Tenant",
			Description: "desc",
			Identifier:  "tenant",
			Status:      shared.StatusActive,
			IsPublic:    true,
			IsSystem:    true,
			CreatedAt:   now,
			UpdatedAt:   now,
		}
		dto := toTenantServiceDataResult(tnt)
		require.NotNil(t, dto)
		assert.Equal(t, tnt.TenantUUID, dto.TenantUUID)
		assert.Equal(t, tnt.Name, dto.Name)
		assert.True(t, dto.IsPublic)
		assert.True(t, dto.IsSystem)
	})
}

func TestFederationPureHelpers(t *testing.T) {
	claims := map[string]interface{}{
		"email":          "User@Example.COM",
		"email_verified": true,
		"name":           "Test User",
		"given_name":     "Test",
		"family_name":    "User",
		"picture":        "https://example.com/avatar.png",
		"locale":         "en",
		"non_string":     123,
		"non_bool":       "yes",
	}

	assert.Equal(t, "Test User", stringClaim(claims, "name"))
	assert.Empty(t, stringClaim(claims, "missing"))
	assert.Empty(t, stringClaim(claims, "non_string"))
	assert.True(t, boolClaim(claims, "email_verified"))
	assert.False(t, boolClaim(claims, "missing"))
	assert.False(t, boolClaim(claims, "non_bool"))

	meta := extractMetadata(claims, map[string]string{"email": "email", "name": "name"})
	assert.Equal(t, "User@Example.COM", meta.Email)
	assert.True(t, meta.EmailVerified)
	assert.Equal(t, "Test User", meta.Name)

	assert.Equal(t, "test_user", deriveUsername(meta, "fallback@example.com"))
	assert.Equal(t, "fallback", deriveUsername(IdentityMetadata{}, "fallback@example.com"))
	assert.True(t, len(deriveUsername(IdentityMetadata{}, "")) > len("user_"))
	assert.Equal(t, "example.com", emailDomain("User@Example.COM"))
	assert.Empty(t, emailDomain("bad-email"))

	metadata, err := json.Marshal(IdentityMetadata{Email: "user@example.com", Name: "User", Picture: "pic"})
	require.NoError(t, err)
	dto := identityToDTO(&UserIdentity{
		UserIdentityUUID: uuid.New(),
		Provider:         shared.ProviderDefault,
		Sub:              "sub",
		Metadata:         datatypes.JSON(metadata),
		CreatedAt:        time.Unix(1, 0).UTC(),
	})
	require.NotNil(t, dto)
	assert.True(t, dto.IsDefault)
	assert.Equal(t, "user@example.com", *dto.Email)

	idp := &IdentityProvider{Identifier: "google", Provider: "google", DisplayName: "Google"}
	hrd := hrdResponseFrom(idp)
	assert.Equal(t, "google", hrd.ProviderIdentifier)
	assert.Equal(t, "Google", hrd.DisplayName)
}

func TestRoutesRegister(t *testing.T) {
	r := chi.NewRouter()
	FederationPublicRoute(r, NewFederationHandler(&mockFederationService{}))
	FederationIdentityRoute(r, NewFederationHandler(&mockFederationService{}), nil, nil)
	IdentityProviderRoute(r, NewIdentityProviderHandler(&mockIdentityProviderService{}), nil, nil)
	SignupFlowRoute(r, NewSignupFlowHandler(&mockSignupFlowService{}), nil, nil)

	for _, tc := range []struct {
		method string
		path   string
	}{
		{http.MethodPost, "/federation/token"},
		{http.MethodPost, "/federation/oauth2/callback"},
		{http.MethodGet, "/federation/hrd"},
		{http.MethodGet, "/account/identities/"},
		{http.MethodPost, "/account/identities/link"},
		{http.MethodDelete, "/account/identities/abc"},
		{http.MethodGet, "/identity_providers/"},
		{http.MethodGet, "/identity_providers/abc"},
		{http.MethodPost, "/identity_providers/"},
		{http.MethodPut, "/identity_providers/abc"},
		{http.MethodPut, "/identity_providers/abc/status"},
		{http.MethodDelete, "/identity_providers/abc"},
		{http.MethodGet, "/signup_flows/"},
		{http.MethodGet, "/signup_flows/abc"},
		{http.MethodPost, "/signup_flows/"},
		{http.MethodPut, "/signup_flows/abc"},
		{http.MethodPatch, "/signup_flows/abc/status"},
		{http.MethodDelete, "/signup_flows/abc"},
		{http.MethodPost, "/signup_flows/abc/roles/"},
		{http.MethodGet, "/signup_flows/abc/roles/"},
		{http.MethodDelete, "/signup_flows/abc/roles/role"},
	} {
		t.Run(tc.method+" "+tc.path, func(t *testing.T) {
			match := chi.NewRouteContext()
			assert.True(t, r.Match(match, tc.method, tc.path))
		})
	}
}
