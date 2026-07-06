package authn

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"gorm.io/gorm"
)

// mockAccountLinkRepo is a function-field mock of AccountLinkRequestRepository.
type mockAccountLinkRepo struct {
	createFn        func(*AccountLinkRequest) (*AccountLinkRequest, error)
	findByTokenFn   func(string) (*AccountLinkRequest, error)
	markConfirmedFn func(int64, time.Time) error
	markExpiredFn   func(int64, time.Time) error
	expireStaleFn   func(time.Time) (int64, error)
}

func (m *mockAccountLinkRepo) Create(e *AccountLinkRequest) (*AccountLinkRequest, error) {
	if m.createFn != nil {
		return m.createFn(e)
	}
	return e, nil
}
func (m *mockAccountLinkRepo) CreateOrUpdate(e *AccountLinkRequest) (*AccountLinkRequest, error) {
	return e, nil
}
func (m *mockAccountLinkRepo) WithTx(*gorm.DB) AccountLinkRequestRepository { return m }
func (m *mockAccountLinkRepo) FindByToken(token string) (*AccountLinkRequest, error) {
	if m.findByTokenFn != nil {
		return m.findByTokenFn(token)
	}
	return nil, nil
}
func (m *mockAccountLinkRepo) MarkConfirmed(id int64, at time.Time) error {
	if m.markConfirmedFn != nil {
		return m.markConfirmedFn(id, at)
	}
	return nil
}
func (m *mockAccountLinkRepo) MarkExpired(id int64, at time.Time) error {
	if m.markExpiredFn != nil {
		return m.markExpiredFn(id, at)
	}
	return nil
}
func (m *mockAccountLinkRepo) ExpireStale(now time.Time) (int64, error) {
	if m.expireStaleFn != nil {
		return m.expireStaleFn(now)
	}
	return 0, nil
}

// mockIdentityLinker is a function-field mock of AccountIdentityLinker.
type mockIdentityLinker struct {
	findFn func(int64, string, string) (int64, bool, error)
	linkFn func(int64, int64, string, string, []byte) error
}

func (m *mockIdentityLinker) FindLinkedUserID(tenantID int64, provider, sub string) (int64, bool, error) {
	if m.findFn != nil {
		return m.findFn(tenantID, provider, sub)
	}
	return 0, false, nil
}
func (m *mockIdentityLinker) LinkIdentity(tenantID, userID int64, provider, sub string, claims []byte) error {
	if m.linkFn != nil {
		return m.linkFn(tenantID, userID, provider, sub, claims)
	}
	return nil
}

func pendingLinkRequest() *AccountLinkRequest {
	return &AccountLinkRequest{
		AccountLinkRequestID:   1,
		AccountLinkRequestUUID: uuid.New(),
		TenantID:               1,
		ExistingUserID:         7,
		ProviderName:           "google",
		ProviderSubject:        "google|abc",
		Status:                 "pending",
		ConfirmationToken:      "tok",
		ExpiresAt:              time.Now().Add(10 * time.Minute),
		ProviderClaims:         []byte(`{"email":"a@b.com"}`),
	}
}

func existingUserRepo() *mockUserRepo {
	return &mockUserRepo{findByIDFn: func(any, ...string) (*User, error) { return &User{UserID: 7}, nil }}
}

func TestAccountLinkService_Initiate(t *testing.T) {
	var created *AccountLinkRequest
	repo := &mockAccountLinkRepo{createFn: func(e *AccountLinkRequest) (*AccountLinkRequest, error) {
		created = e
		return e, nil
	}}
	svc := NewAccountLinkRequestService(repo, &mockUserRepo{}, &mockIdentityLinker{})
	res, err := svc.Initiate(context.Background(), InitiateAccountLinkInput{
		TenantID: 1, ExistingUserID: 7, ProviderName: "google", ProviderSubject: "google|abc", ProviderEmail: "a@b.com",
	})
	assert.NoError(t, err)
	assert.Equal(t, "pending", res.Status)
	assert.NotEmpty(t, created.ConfirmationToken)
	// TTL ~15 minutes.
	ttl := time.Until(created.ExpiresAt)
	assert.Greater(t, ttl, 14*time.Minute)
	assert.LessOrEqual(t, ttl, 15*time.Minute)
}

func TestAccountLinkService_Confirm(t *testing.T) {
	t.Run("success creates identity and marks confirmed", func(t *testing.T) {
		linked := false
		confirmed := false
		repo := &mockAccountLinkRepo{
			findByTokenFn:   func(string) (*AccountLinkRequest, error) { return pendingLinkRequest(), nil },
			markConfirmedFn: func(int64, time.Time) error { confirmed = true; return nil },
		}
		linker := &mockIdentityLinker{
			findFn: func(int64, string, string) (int64, bool, error) { return 0, false, nil },
			linkFn: func(int64, int64, string, string, []byte) error { linked = true; return nil },
		}
		svc := NewAccountLinkRequestService(repo, existingUserRepo(), linker)
		res, err := svc.Confirm(context.Background(), "tok", 7, 1)
		assert.NoError(t, err)
		assert.Equal(t, int64(7), res.ExistingUserID)
		assert.True(t, linked)
		assert.True(t, confirmed)
	})

	t.Run("not found", func(t *testing.T) {
		repo := &mockAccountLinkRepo{findByTokenFn: func(string) (*AccountLinkRequest, error) { return nil, nil }}
		svc := NewAccountLinkRequestService(repo, existingUserRepo(), &mockIdentityLinker{})
		_, err := svc.Confirm(context.Background(), "tok", 7, 1)
		assert.Error(t, err)
	})

	t.Run("not pending is rejected", func(t *testing.T) {
		req := pendingLinkRequest()
		req.Status = "confirmed"
		repo := &mockAccountLinkRepo{findByTokenFn: func(string) (*AccountLinkRequest, error) { return req, nil }}
		svc := NewAccountLinkRequestService(repo, existingUserRepo(), &mockIdentityLinker{})
		_, err := svc.Confirm(context.Background(), "tok", 7, 1)
		assert.Error(t, err)
	})

	t.Run("expired is rejected and marked expired", func(t *testing.T) {
		req := pendingLinkRequest()
		req.ExpiresAt = time.Now().Add(-time.Minute)
		markedExpired := false
		repo := &mockAccountLinkRepo{
			findByTokenFn: func(string) (*AccountLinkRequest, error) { return req, nil },
			markExpiredFn: func(int64, time.Time) error { markedExpired = true; return nil },
		}
		svc := NewAccountLinkRequestService(repo, existingUserRepo(), &mockIdentityLinker{})
		_, err := svc.Confirm(context.Background(), "tok", 7, 1)
		assert.Error(t, err)
		assert.True(t, markedExpired)
	})

	t.Run("wrong authenticated user is forbidden", func(t *testing.T) {
		repo := &mockAccountLinkRepo{findByTokenFn: func(string) (*AccountLinkRequest, error) { return pendingLinkRequest(), nil }}
		svc := NewAccountLinkRequestService(repo, existingUserRepo(), &mockIdentityLinker{})
		_, err := svc.Confirm(context.Background(), "tok", 999, 1) // authUserID != existing_user_id
		assert.Error(t, err)
	})

	t.Run("existing user deleted is rejected", func(t *testing.T) {
		repo := &mockAccountLinkRepo{findByTokenFn: func(string) (*AccountLinkRequest, error) { return pendingLinkRequest(), nil }}
		deletedUserRepo := &mockUserRepo{findByIDFn: func(any, ...string) (*User, error) { return nil, nil }}
		svc := NewAccountLinkRequestService(repo, deletedUserRepo, &mockIdentityLinker{})
		_, err := svc.Confirm(context.Background(), "tok", 7, 1)
		assert.Error(t, err)
	})

	t.Run("provider already linked to a different user is rejected", func(t *testing.T) {
		repo := &mockAccountLinkRepo{findByTokenFn: func(string) (*AccountLinkRequest, error) { return pendingLinkRequest(), nil }}
		linker := &mockIdentityLinker{findFn: func(int64, string, string) (int64, bool, error) { return 999, true, nil }}
		svc := NewAccountLinkRequestService(repo, existingUserRepo(), linker)
		_, err := svc.Confirm(context.Background(), "tok", 7, 1)
		assert.Error(t, err)
	})

	t.Run("already linked to same user is idempotent success", func(t *testing.T) {
		linkCalled := false
		confirmed := false
		repo := &mockAccountLinkRepo{
			findByTokenFn:   func(string) (*AccountLinkRequest, error) { return pendingLinkRequest(), nil },
			markConfirmedFn: func(int64, time.Time) error { confirmed = true; return nil },
		}
		linker := &mockIdentityLinker{
			findFn: func(int64, string, string) (int64, bool, error) { return 7, true, nil },
			linkFn: func(int64, int64, string, string, []byte) error { linkCalled = true; return nil },
		}
		svc := NewAccountLinkRequestService(repo, existingUserRepo(), linker)
		_, err := svc.Confirm(context.Background(), "tok", 7, 1)
		assert.NoError(t, err)
		assert.False(t, linkCalled, "must not re-create an existing identity link")
		assert.True(t, confirmed)
	})
}
