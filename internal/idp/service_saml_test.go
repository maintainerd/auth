package idp

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
	"testing"
	"time"

	crewsaml "github.com/crewjam/saml"
	"github.com/maintainerd/maintainerd-auth/internal/shared"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakeSAMLStore is an in-memory WebAuthnSessionStore for SAML tests. Missing keys
// return an error from GetSession, mirroring the real (Redis) store.
type fakeSAMLStore struct {
	mu   sync.Mutex
	data map[string][]byte
}

func newFakeSAMLStore() *fakeSAMLStore {
	return &fakeSAMLStore{data: map[string][]byte{}}
}

func (f *fakeSAMLStore) SetSession(_ context.Context, key string, value any, _ time.Duration) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	b, err := json.Marshal(value)
	if err != nil {
		return err
	}
	f.data[key] = b
	return nil
}

func (f *fakeSAMLStore) GetSession(_ context.Context, key string, dest any) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	b, ok := f.data[key]
	if !ok {
		return errors.New("not found")
	}
	return json.Unmarshal(b, dest)
}

func (f *fakeSAMLStore) DeleteSession(_ context.Context, key string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	delete(f.data, key)
	return nil
}

// ---------------------------------------------------------------------------
// F1: SAML redirect_uri must match a client's registered redirect URIs.
// ---------------------------------------------------------------------------

func TestValidateSAMLRedirectURI(t *testing.T) {
	registered := []ClientURI{
		{ClientID: 5, URI: "https://app.example.com/callback", Type: shared.ClientURITypeRedirect},
		{ClientID: 5, URI: "https://app.example.com/other", Type: "logout_uri"},
	}
	newSvc := func() *federationService {
		return &federationService{
			clientRepo: &mockClientRepo{
				findRedirectURIsFn: func(int64) ([]ClientURI, error) { return registered, nil },
			},
		}
	}

	t.Run("matching registered redirect uri is allowed", func(t *testing.T) {
		err := newSvc().validateSAMLRedirectURI(5, "https://app.example.com/callback")
		require.NoError(t, err)
	})

	t.Run("unregistered redirect uri is rejected", func(t *testing.T) {
		err := newSvc().validateSAMLRedirectURI(5, "https://evil.example.com/steal")
		require.Error(t, err)
	})

	t.Run("non-redirect-type uri does not count as a match", func(t *testing.T) {
		// The logout_uri is registered but not as a redirect_uri, so it must not match.
		err := newSvc().validateSAMLRedirectURI(5, "https://app.example.com/other")
		require.Error(t, err)
	})

	t.Run("dangerous scheme is rejected", func(t *testing.T) {
		err := newSvc().validateSAMLRedirectURI(5, "javascript:alert(1)")
		require.Error(t, err)
	})

	t.Run("no registered uris rejects", func(t *testing.T) {
		svc := &federationService{
			clientRepo: &mockClientRepo{
				findRedirectURIsFn: func(int64) ([]ClientURI, error) { return nil, nil },
			},
		}
		err := svc.validateSAMLRedirectURI(5, "https://app.example.com/callback")
		require.Error(t, err)
	})
}

// ---------------------------------------------------------------------------
// F2: SAML email is only verified when its domain is on the allow-list.
// ---------------------------------------------------------------------------

func TestExtractSAMLClaims_EmailNotVerifiedByPresence(t *testing.T) {
	assertion := &crewsaml.Assertion{
		AttributeStatements: []crewsaml.AttributeStatement{{
			Attributes: []crewsaml.Attribute{{
				Name:   "email",
				Values: []crewsaml.AttributeValue{{Value: "user@corp.example"}},
			}},
		}},
	}
	meta := extractSAMLClaims(assertion, nil)
	assert.Equal(t, "user@corp.example", meta.Email)
	assert.False(t, meta.EmailVerified, "email must never be verified from mere presence")
}

func TestSAMLEmailDomainAllowed(t *testing.T) {
	allowed := []IdentityProviderEmailDomain{
		{Domain: "corp.example"},
		{Domain: "Other.Example"},
	}

	t.Run("allow-listed domain is verified", func(t *testing.T) {
		assert.True(t, samlEmailDomainAllowed("user@corp.example", allowed))
	})
	t.Run("allow-list is case-insensitive", func(t *testing.T) {
		assert.True(t, samlEmailDomainAllowed("user@OTHER.EXAMPLE", allowed))
	})
	t.Run("non-allow-listed domain is not verified", func(t *testing.T) {
		assert.False(t, samlEmailDomainAllowed("user@evil.example", allowed))
	})
	t.Run("empty allow-list is never verified", func(t *testing.T) {
		assert.False(t, samlEmailDomainAllowed("user@corp.example", nil))
	})
	t.Run("empty email is never verified", func(t *testing.T) {
		assert.False(t, samlEmailDomainAllowed("", allowed))
	})
}

// ---------------------------------------------------------------------------
// F5: RelayState replay protection.
// ---------------------------------------------------------------------------

func TestSAMLRelayStateRoundTripCarriesRequestID(t *testing.T) {
	token, err := newSAMLRelayState("prov", "client-abc", "https://app.example.com/cb", 7, "req-id-123")
	require.NoError(t, err)

	rs, err := verifyRelayState(token)
	require.NoError(t, err)
	assert.Equal(t, "prov", rs.ProviderIdentifier)
	assert.Equal(t, "client-abc", rs.ClientID)
	assert.Equal(t, "https://app.example.com/cb", rs.RedirectURI)
	assert.Equal(t, int64(7), rs.TenantID)
	assert.Equal(t, "req-id-123", rs.RequestID)
	assert.NotEmpty(t, rs.Nonce)
}

func TestEnforceRelayStateSingleUse(t *testing.T) {
	t.Run("first use ok, replay rejected", func(t *testing.T) {
		svc := &federationService{samlStore: newFakeSAMLStore()}
		require.NoError(t, svc.enforceRelayStateSingleUse(context.Background(), "nonce-1"))
		err := svc.enforceRelayStateSingleUse(context.Background(), "nonce-1")
		require.Error(t, err, "replay of the same relay state must be rejected")
	})

	t.Run("distinct nonces are independent", func(t *testing.T) {
		svc := &federationService{samlStore: newFakeSAMLStore()}
		require.NoError(t, svc.enforceRelayStateSingleUse(context.Background(), "nonce-a"))
		require.NoError(t, svc.enforceRelayStateSingleUse(context.Background(), "nonce-b"))
	})

	t.Run("empty nonce rejected", func(t *testing.T) {
		svc := &federationService{samlStore: newFakeSAMLStore()}
		require.Error(t, svc.enforceRelayStateSingleUse(context.Background(), ""))
	})

	t.Run("nil store fails closed", func(t *testing.T) {
		svc := &federationService{}
		require.Error(t, svc.enforceRelayStateSingleUse(context.Background(), "nonce-x"))
	})
}
