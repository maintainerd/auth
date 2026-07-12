package user

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

type mockUserConsentService struct {
	recordFn       func(ctx context.Context, tx *gorm.DB, userID, tenantID int64, consentType, policyVersion, ipAddress, userAgent string) error
	findByUserIDFn func(ctx context.Context, userID int64) ([]UserConsent, error)
	withdrawFn     func(ctx context.Context, userID, tenantID int64, consentType, ipAddress, userAgent string) error
}

func (m *mockUserConsentService) Withdraw(ctx context.Context, userID, tenantID int64, consentType, ipAddress, userAgent string) error {
	if m.withdrawFn != nil {
		return m.withdrawFn(ctx, userID, tenantID, consentType, ipAddress, userAgent)
	}
	return nil
}

func (m *mockUserConsentService) Record(ctx context.Context, tx *gorm.DB, userID, tenantID int64, consentType, policyVersion, ipAddress, userAgent string) error {
	if m.recordFn != nil {
		return m.recordFn(ctx, tx, userID, tenantID, consentType, policyVersion, ipAddress, userAgent)
	}
	return nil
}

func (m *mockUserConsentService) FindByUserID(ctx context.Context, userID int64) ([]UserConsent, error) {
	if m.findByUserIDFn != nil {
		return m.findByUserIDFn(ctx, userID)
	}
	return nil, nil
}

func TestUserConsentHandler_RecordConsent_NoAuth(t *testing.T) {
	h := NewUserConsentHandler(&mockUserConsentService{}, nil, nil)
	r := httptest.NewRequest(http.MethodPost, "/me/consent", nil)
	w := httptest.NewRecorder()
	h.RecordConsent(w, r)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestUserConsentHandler_RecordConsent_BadJSON(t *testing.T) {
	h := NewUserConsentHandler(&mockUserConsentService{}, nil, nil)
	r := withTenantAndUser(badJSONReq(t, http.MethodPost, "/me/consent"))
	w := httptest.NewRecorder()
	h.RecordConsent(w, r)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestUserConsentHandler_RecordConsent_EmptyBody(t *testing.T) {
	h := NewUserConsentHandler(&mockUserConsentService{}, nil, nil)
	r := withTenantAndUser(httptest.NewRequest(http.MethodPost, "/me/consent", nil))
	r.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.RecordConsent(w, r)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestUserConsentHandler_RecordConsent_MissingFields(t *testing.T) {
	tests := []struct {
		name string
		body map[string]string
	}{
		{"missing consent_type", map[string]string{"policy_version": "v1"}},
		{"missing policy_version", map[string]string{"consent_type": "terms_of_service"}},
		{"invalid consent_type", map[string]string{"consent_type": "invalid", "policy_version": "v1"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := NewUserConsentHandler(&mockUserConsentService{}, nil, nil)
			r := withTenantAndUser(jsonReq(t, http.MethodPost, "/me/consent", tt.body))
			w := httptest.NewRecorder()
			h.RecordConsent(w, r)
			assert.Equal(t, http.StatusBadRequest, w.Code)
		})
	}
}

func TestUserConsentHandler_RecordConsent_ServiceError(t *testing.T) {
	svc := &mockUserConsentService{
		recordFn: func(ctx context.Context, tx *gorm.DB, userID, tenantID int64, consentType, policyVersion, ipAddress, userAgent string) error {
			return assert.AnError
		},
	}
	h := NewUserConsentHandler(svc, nil, nil)
	r := withTenantAndUser(jsonReq(t, http.MethodPost, "/me/consent", map[string]string{
		"consent_type": "terms_of_service", "policy_version": "v1",
	}))
	w := httptest.NewRecorder()
	h.RecordConsent(w, r)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestUserConsentHandler_RecordConsent_Success(t *testing.T) {
	h := NewUserConsentHandler(&mockUserConsentService{}, nil, nil)
	r := withTenantAndUser(jsonReq(t, http.MethodPost, "/me/consent", map[string]string{
		"consent_type": "privacy_policy", "policy_version": "v2",
	}))
	w := httptest.NewRecorder()
	h.RecordConsent(w, r)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestUserConsentHandler_GetUserConsents_MissingUUID(t *testing.T) {
	h := NewUserConsentHandler(&mockUserConsentService{}, nil, nil)
	r := withTenant(httptest.NewRequest(http.MethodGet, "/users/123/consents", nil))
	r = withChiParam(r, "user_uuid", "")
	w := httptest.NewRecorder()
	h.GetUserConsents(w, r)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestUserConsentHandler_GetUserConsents_InvalidUUID(t *testing.T) {
	h := NewUserConsentHandler(&mockUserConsentService{}, nil, nil)
	r := withTenant(httptest.NewRequest(http.MethodGet, "/users/123/consents", nil))
	r = withChiParam(r, "user_uuid", "not-a-uuid")
	w := httptest.NewRecorder()
	h.GetUserConsents(w, r)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestUserConsentHandler_GetUserConsents_UserNotFound(t *testing.T) {
	h := NewUserConsentHandler(&mockUserConsentService{}, nil, &mockUserRepo{
		findByUUIDFn: func(id any, p ...string) (*User, error) {
			return nil, nil
		},
	})
	uu := uuid.New().String()
	r := withTenant(httptest.NewRequest(http.MethodGet, "/users/"+uu+"/consents", nil))
	r = withChiParam(r, "user_uuid", uu)
	w := httptest.NewRecorder()
	h.GetUserConsents(w, r)
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestUserConsentHandler_GetUserConsents_ServiceError(t *testing.T) {
	testUser := &User{UserID: 42, UserIdentities: []UserIdentity{{TenantID: tenantID}}}
	svc := &mockUserConsentService{
		findByUserIDFn: func(ctx context.Context, userID int64) ([]UserConsent, error) {
			return nil, assert.AnError
		},
	}
	h := NewUserConsentHandler(svc, nil, &mockUserRepo{
		findByUUIDFn: func(id any, p ...string) (*User, error) {
			return testUser, nil
		},
	})
	uu := uuid.New().String()
	r := withTenant(httptest.NewRequest(http.MethodGet, "/users/"+uu+"/consents", nil))
	r = withChiParam(r, "user_uuid", uu)
	w := httptest.NewRecorder()
	h.GetUserConsents(w, r)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestUserConsentHandler_GetUserConsents_Success(t *testing.T) {
	testUser := &User{UserID: 42, UserIdentities: []UserIdentity{{TenantID: tenantID}}}
	consentUUID := uuid.New()
	svc := &mockUserConsentService{
		findByUserIDFn: func(ctx context.Context, userID int64) ([]UserConsent, error) {
			return []UserConsent{
				{UserConsentUUID: consentUUID, ConsentType: "terms_of_service", PolicyVersion: "v1", Accepted: true},
			}, nil
		},
	}
	h := NewUserConsentHandler(svc, nil, &mockUserRepo{
		findByUUIDFn: func(id any, p ...string) (*User, error) {
			return testUser, nil
		},
	})
	uu := uuid.New().String()
	r := withTenantAndUser(httptest.NewRequest(http.MethodGet, "/users/"+uu+"/consents", nil))
	r = withChiParam(r, "user_uuid", uu)
	w := httptest.NewRecorder()
	h.GetUserConsents(w, r)
	require.Equal(t, http.StatusOK, w.Code)
}
