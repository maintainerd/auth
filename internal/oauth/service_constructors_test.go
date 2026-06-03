package oauth

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestOAuthServiceConstructors(t *testing.T) {
	db, _ := newMockDB(t)

	assert.IsType(t, &oauthCIBAService{}, NewOAuthCIBAService(db, &mockClientRepo{}, &mockOAuthCIBARepo{}, &mockUserRepo{}, &mockAuthEventService{}))
	assert.IsType(t, &oauthDeviceService{}, NewOAuthDeviceService(db, &mockClientRepo{}, &mockOAuthDeviceCodeRepo{}, &mockUserRepo{}, &mockUserIdentityRepo{}, &mockAuthEventService{}))
	assert.IsType(t, &oauthPARService{}, NewOAuthPARService(db, &mockClientRepo{}, &mockClientURIRepo{}, &mockOAuthPARRepo{}, &mockAuthEventService{}))
	assert.IsType(t, &oauthRegisterService{}, NewOAuthRegisterService(db, &mockClientRepo{}, &mockClientURIRepo{}, &mockTenantRepo{}, &mockAuthEventService{}))
	assert.IsType(t, &oauthSessionService{}, NewOAuthSessionService(db, &mockClientRepo{}, &mockUserRepo{}, &mockOAuthRefreshTokenRepo{}, &mockAuthEventService{}))
	assert.IsType(t, &oauthTokenExchangeService{}, NewOAuthTokenExchangeService(db, &mockClientRepo{}, &mockUserRepo{}, &mockAuthEventService{}))
}
