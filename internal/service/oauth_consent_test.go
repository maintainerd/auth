package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/maintainerd/auth/internal/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newOAuthConsentSvc(repo *mockOAuthConsentGrantRepo) OAuthConsentService {
	return NewOAuthConsentService(repo)
}

func TestOAuthConsentService_ListGrants(t *testing.T) {
	ctx := context.Background()

	t.Run("returns mapped grants", func(t *testing.T) {
		grantUUID := uuid.New()
		clientUUID := uuid.New()
		now := time.Now()

		svc := newOAuthConsentSvc(&mockOAuthConsentGrantRepo{
			findByUserIDFn: func(_ int64) ([]model.OAuthConsentGrant, error) {
				return []model.OAuthConsentGrant{
					{
						OAuthConsentGrantUUID: grantUUID,
						UserID:                1,
						ClientID:              10,
						TenantID:              100,
						Scopes:                "openid profile email",
						CreatedAt:             now,
						UpdatedAt:             now,
						Client: &model.Client{
							ClientUUID:  clientUUID,
							DisplayName: "Test App",
						},
					},
				}, nil
			},
		})

		grants, err := svc.ListGrants(ctx, 1)
		require.NoError(t, err)
		require.Len(t, grants, 1)
		assert.Equal(t, grantUUID.String(), grants[0].ConsentGrantUUID)
		assert.Equal(t, "Test App", grants[0].ClientName)
		assert.Equal(t, clientUUID.String(), grants[0].ClientUUID)
		assert.Equal(t, []string{"openid", "profile", "email"}, grants[0].Scopes)
	})

	t.Run("returns empty list when no grants", func(t *testing.T) {
		svc := newOAuthConsentSvc(&mockOAuthConsentGrantRepo{
			findByUserIDFn: func(_ int64) ([]model.OAuthConsentGrant, error) {
				return nil, nil
			},
		})

		grants, err := svc.ListGrants(ctx, 1)
		require.NoError(t, err)
		require.Len(t, grants, 0)
	})

	t.Run("grant with nil client", func(t *testing.T) {
		grantUUID := uuid.New()
		now := time.Now()

		svc := newOAuthConsentSvc(&mockOAuthConsentGrantRepo{
			findByUserIDFn: func(_ int64) ([]model.OAuthConsentGrant, error) {
				return []model.OAuthConsentGrant{
					{
						OAuthConsentGrantUUID: grantUUID,
						Scopes:                "openid",
						CreatedAt:             now,
						UpdatedAt:             now,
					},
				}, nil
			},
		})

		grants, err := svc.ListGrants(ctx, 1)
		require.NoError(t, err)
		require.Len(t, grants, 1)
		assert.Empty(t, grants[0].ClientName)
		assert.Empty(t, grants[0].ClientUUID)
	})

	t.Run("repo error", func(t *testing.T) {
		svc := newOAuthConsentSvc(&mockOAuthConsentGrantRepo{
			findByUserIDFn: func(_ int64) ([]model.OAuthConsentGrant, error) {
				return nil, errors.New("db error")
			},
		})

		_, err := svc.ListGrants(ctx, 1)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to retrieve consent grants")
	})
}

func TestOAuthConsentService_RevokeGrant(t *testing.T) {
	ctx := context.Background()

	t.Run("revokes existing grant", func(t *testing.T) {
		grantUUID := uuid.New()
		var deletedUserID, deletedClientID int64

		svc := newOAuthConsentSvc(&mockOAuthConsentGrantRepo{
			findByUserIDFn: func(uid int64) ([]model.OAuthConsentGrant, error) {
				return []model.OAuthConsentGrant{
					{
						OAuthConsentGrantUUID: grantUUID,
						UserID:                uid,
						ClientID:              10,
					},
				}, nil
			},
			deleteByUserAndClientFn: func(uid, cid int64) error {
				deletedUserID = uid
				deletedClientID = cid
				return nil
			},
		})

		err := svc.RevokeGrant(ctx, grantUUID, 1)
		require.NoError(t, err)
		assert.Equal(t, int64(1), deletedUserID)
		assert.Equal(t, int64(10), deletedClientID)
	})

	t.Run("grant not found", func(t *testing.T) {
		svc := newOAuthConsentSvc(&mockOAuthConsentGrantRepo{
			findByUserIDFn: func(_ int64) ([]model.OAuthConsentGrant, error) {
				return []model.OAuthConsentGrant{}, nil
			},
		})

		err := svc.RevokeGrant(ctx, uuid.New(), 1)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "consent grant not found")
	})

	t.Run("grant owned by different user", func(t *testing.T) {
		grantUUID := uuid.New()
		otherUUID := uuid.New()

		svc := newOAuthConsentSvc(&mockOAuthConsentGrantRepo{
			findByUserIDFn: func(_ int64) ([]model.OAuthConsentGrant, error) {
				return []model.OAuthConsentGrant{
					{OAuthConsentGrantUUID: otherUUID, ClientID: 10},
				}, nil
			},
		})

		err := svc.RevokeGrant(ctx, grantUUID, 1)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "consent grant not found")
	})

	t.Run("FindByUserID error", func(t *testing.T) {
		svc := newOAuthConsentSvc(&mockOAuthConsentGrantRepo{
			findByUserIDFn: func(_ int64) ([]model.OAuthConsentGrant, error) {
				return nil, errors.New("db error")
			},
		})

		err := svc.RevokeGrant(ctx, uuid.New(), 1)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to revoke consent grant")
	})

	t.Run("DeleteByUserAndClient error", func(t *testing.T) {
		grantUUID := uuid.New()

		svc := newOAuthConsentSvc(&mockOAuthConsentGrantRepo{
			findByUserIDFn: func(uid int64) ([]model.OAuthConsentGrant, error) {
				return []model.OAuthConsentGrant{
					{OAuthConsentGrantUUID: grantUUID, UserID: uid, ClientID: 10},
				}, nil
			},
			deleteByUserAndClientFn: func(_, _ int64) error {
				return errors.New("delete error")
			},
		})

		err := svc.RevokeGrant(ctx, grantUUID, 1)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to revoke consent grant")
	})
}
