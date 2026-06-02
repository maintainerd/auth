package idp

import (
	"context"
	"errors"
	"testing"

	"github.com/maintainerd/auth/internal/platform/apperror"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

type mockFederationUserIdentityRepo struct {
	mockBaseRepo[UserIdentity]
	findByUserIDAndProviderFn func(int64, string) (*UserIdentity, error)
	createFn                  func(*UserIdentity) (*UserIdentity, error)
}

func (m *mockFederationUserIdentityRepo) WithTx(_ *gorm.DB) UserIdentityRepository {
	return m
}

func (m *mockFederationUserIdentityRepo) FindByUserID(_ int64) ([]UserIdentity, error) {
	return nil, nil
}

func (m *mockFederationUserIdentityRepo) FindByProviderAndSub(_, _ string) (*UserIdentity, error) {
	return nil, nil
}

func (m *mockFederationUserIdentityRepo) FindByUserIDAndProvider(userID int64, provider string) (*UserIdentity, error) {
	if m.findByUserIDAndProviderFn != nil {
		return m.findByUserIDAndProviderFn(userID, provider)
	}
	return nil, nil
}

func (m *mockFederationUserIdentityRepo) DeleteByUserID(_ int64) error {
	return nil
}

func (m *mockFederationUserIdentityRepo) Create(identity *UserIdentity) (*UserIdentity, error) {
	if m.createFn != nil {
		return m.createFn(identity)
	}
	return identity, nil
}

func TestFederationServiceProvisionUser_UnverifiedEmailDoesNotMergeExistingAccount(t *testing.T) {
	gormDB, _ := newMockGormDB(t)

	var emailLookupCalled bool
	createdUser := &User{UserID: 200, Email: "owner@example.com"}
	userRepo := &mockUserRepo{
		findByEmailFn: func(string) (*User, error) {
			t.Fatal("global email lookup must not be used for federation merge")
			return nil, nil
		},
		findByEmailAndTenantIDFn: func(string, int64) (*User, error) {
			emailLookupCalled = true
			return &User{UserID: 100, Email: "owner@example.com"}, nil
		},
		createFn: func(user *User) (*User, error) {
			assert.Equal(t, "owner@example.com", user.Email)
			assert.False(t, user.IsEmailVerified)
			return createdUser, nil
		},
	}

	var externalIdentity *UserIdentity
	identityRepo := &mockFederationUserIdentityRepo{
		createFn: func(identity *UserIdentity) (*UserIdentity, error) {
			if identity.Provider == "google" {
				externalIdentity = identity
			}
			return identity, nil
		},
	}

	svc := &federationService{
		userRepo:         userRepo,
		userIdentityRepo: identityRepo,
		idpRepo:          &mockIdentityProviderRepo{},
		roleRepo:         &mockRoleRepo{},
	}

	user, isNew, err := svc.provisionUser(context.Background(), gormDB, &IdentityProvider{
		IdentityProviderID: 10,
		TenantID:           20,
		Provider:           "google",
	}, "external-sub", "owner@example.com", IdentityMetadata{
		Email:         "owner@example.com",
		EmailVerified: false,
	})

	require.NoError(t, err)
	require.NotNil(t, user)
	assert.True(t, isNew)
	assert.Equal(t, int64(200), user.UserID)
	assert.False(t, emailLookupCalled)
	require.NotNil(t, externalIdentity)
	assert.Equal(t, int64(200), externalIdentity.UserID)
}

func TestFederationServiceProvisionUser_VerifiedEmailMergesTenantAccount(t *testing.T) {
	gormDB, _ := newMockGormDB(t)

	existingUser := &User{UserID: 100, Email: "owner@example.com", IsEmailVerified: true}
	var createUserCalled bool
	userRepo := &mockUserRepo{
		findByEmailFn: func(string) (*User, error) {
			t.Fatal("global email lookup must not be used for federation merge")
			return nil, nil
		},
		findByEmailAndTenantIDFn: func(email string, tenantID int64) (*User, error) {
			assert.Equal(t, "owner@example.com", email)
			assert.Equal(t, int64(20), tenantID)
			return existingUser, nil
		},
		createFn: func(user *User) (*User, error) {
			createUserCalled = true
			return user, nil
		},
	}

	var externalIdentity *UserIdentity
	identityRepo := &mockFederationUserIdentityRepo{
		createFn: func(identity *UserIdentity) (*UserIdentity, error) {
			if identity.Provider == "google" {
				externalIdentity = identity
			}
			return identity, nil
		},
	}

	svc := &federationService{
		userRepo:         userRepo,
		userIdentityRepo: identityRepo,
		idpRepo:          &mockIdentityProviderRepo{},
		roleRepo:         &mockRoleRepo{},
	}

	user, isNew, err := svc.provisionUser(context.Background(), gormDB, &IdentityProvider{
		IdentityProviderID: 10,
		TenantID:           20,
		Provider:           "google",
	}, "external-sub", "owner@example.com", IdentityMetadata{
		Email:         "owner@example.com",
		EmailVerified: true,
	})

	require.NoError(t, err)
	require.Same(t, existingUser, user)
	assert.False(t, isNew)
	assert.False(t, createUserCalled)
	require.NotNil(t, externalIdentity)
	assert.Equal(t, int64(100), externalIdentity.UserID)
}

func TestFederationServiceProvisionUser_VerifiedEmailLookupErrorFailsClosed(t *testing.T) {
	gormDB, _ := newMockGormDB(t)
	lookupErr := errors.New("database unavailable")

	userRepo := &mockUserRepo{
		findByEmailAndTenantIDFn: func(string, int64) (*User, error) {
			return nil, lookupErr
		},
		createFn: func(user *User) (*User, error) {
			t.Fatal("user must not be created when verified email lookup fails")
			return user, nil
		},
	}

	svc := &federationService{
		userRepo:         userRepo,
		userIdentityRepo: &mockFederationUserIdentityRepo{},
		idpRepo:          &mockIdentityProviderRepo{},
		roleRepo:         &mockRoleRepo{},
	}

	user, isNew, err := svc.provisionUser(context.Background(), gormDB, &IdentityProvider{
		TenantID: 20,
		Provider: "google",
	}, "external-sub", "owner@example.com", IdentityMetadata{
		Email:         "owner@example.com",
		EmailVerified: true,
	})

	require.Error(t, err)
	assert.Nil(t, user)
	assert.False(t, isNew)
	var internalErr *apperror.InternalError
	assert.ErrorAs(t, err, &internalErr)
}
