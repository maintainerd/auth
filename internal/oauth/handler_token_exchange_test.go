package oauth

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/maintainerd/maintainerd-auth/internal/platform/apperror"
	"github.com/stretchr/testify/assert"
)

type mockOAuthTokenExchangeService struct {
	exchangeFn func(context.Context, OAuthTokenExchangeRequestDTO, OAuthClientCredentials) (*OAuthTokenExchangeResponseDTO, *apperror.OAuthError)
}

func (m *mockOAuthTokenExchangeService) Exchange(ctx context.Context, req OAuthTokenExchangeRequestDTO, creds OAuthClientCredentials) (*OAuthTokenExchangeResponseDTO, *apperror.OAuthError) {
	if m.exchangeFn != nil {
		return m.exchangeFn(ctx, req, creds)
	}
	return nil, nil
}

func TestOAuthTokenExchangeHandler_Exchange(t *testing.T) {
	t.Run("body parse error returns 400", func(t *testing.T) {
		r := httptest.NewRequest(http.MethodPost, "/oauth/token", errReader{})
		r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		w := httptest.NewRecorder()
		NewOAuthTokenExchangeHandler(&mockOAuthTokenExchangeService{}).Exchange(w, r)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("validation error returns 400", func(t *testing.T) {
		r := formPost(t, "/oauth/token", url.Values{})
		w := httptest.NewRecorder()
		NewOAuthTokenExchangeHandler(&mockOAuthTokenExchangeService{}).Exchange(w, r)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("service error returns oauth error", func(t *testing.T) {
		svc := &mockOAuthTokenExchangeService{
			exchangeFn: func(context.Context, OAuthTokenExchangeRequestDTO, OAuthClientCredentials) (*OAuthTokenExchangeResponseDTO, *apperror.OAuthError) {
				return nil, apperror.NewOAuthInvalidClient("bad client")
			},
		}
		r := formPost(t, "/oauth/token", url.Values{
			"client_id":          {"app"},
			"subject_token":      {"subject"},
			"subject_token_type": {"urn:ietf:params:oauth:token-type:access_token"},
		})
		w := httptest.NewRecorder()
		NewOAuthTokenExchangeHandler(svc).Exchange(w, r)
		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("success returns 200", func(t *testing.T) {
		svc := &mockOAuthTokenExchangeService{
			exchangeFn: func(_ context.Context, req OAuthTokenExchangeRequestDTO, creds OAuthClientCredentials) (*OAuthTokenExchangeResponseDTO, *apperror.OAuthError) {
				assert.Equal(t, "subject", req.SubjectToken)
				assert.Equal(t, "app", req.ClientID)
				assert.Equal(t, "app", creds.ClientID)
				return &OAuthTokenExchangeResponseDTO{AccessToken: "token", TokenType: "Bearer", ExpiresIn: 60}, nil
			},
		}
		r := formPost(t, "/oauth/token", url.Values{
			"client_id":          {"app"},
			"subject_token":      {"subject"},
			"subject_token_type": {"urn:ietf:params:oauth:token-type:access_token"},
		})
		w := httptest.NewRecorder()
		NewOAuthTokenExchangeHandler(svc).Exchange(w, r)
		assert.Equal(t, http.StatusOK, w.Code)
	})
}

type fakeWorkloadIdentityExchanger struct {
	called bool
	got    WorkloadTokenExchangeInput
	result *WorkloadTokenExchangeResult
}

func (f *fakeWorkloadIdentityExchanger) ExchangeWorkloadToken(_ context.Context, in WorkloadTokenExchangeInput) (*WorkloadTokenExchangeResult, *apperror.OAuthError) {
	f.called = true
	f.got = in
	return f.result, nil
}

// installWorkloadExchanger swaps the package-level exchanger for the duration of
// a test. The token endpoint reads it directly, so there is no seam to inject.
func installWorkloadExchanger(t *testing.T, e WorkloadIdentityExchanger) {
	t.Helper()
	orig := workloadIdentityExchanger
	t.Cleanup(func() { workloadIdentityExchanger = orig })
	workloadIdentityExchanger = e
}

// The workload identity path is keyless — no client credentials are presented —
// so the `aud` of the token it mints is decided entirely by request parameters.
// The exchanger read only `audience`, so `resource` was accepted and then
// discarded, and a request naming two different targets had one of them dropped
// without a word. Both must be refused before anything is minted.
func TestOAuthTokenExchangeHandler_WorkloadIdentityTargetIsValidated(t *testing.T) {
	const wifForm = "urn:ietf:params:oauth:token-type:jwt"

	t.Run("a resource that is not an absolute URI is invalid_target", func(t *testing.T) {
		exchanger := &fakeWorkloadIdentityExchanger{}
		installWorkloadExchanger(t, exchanger)

		r := formPost(t, "/oauth/token", url.Values{
			"grant_type":         {"urn:ietf:params:oauth:grant-type:token-exchange"},
			"subject_token":      {"external-oidc-token"},
			"subject_token_type": {wifForm},
			"resource":           {"not-an-absolute-uri"},
		})
		w := httptest.NewRecorder()
		NewOAuthTokenExchangeHandler(&mockOAuthTokenExchangeService{}).Exchange(w, r)

		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Contains(t, w.Body.String(), "invalid_target")
		assert.False(t, exchanger.called, "no token may be minted for a target the server could not parse")
	})

	t.Run("audience and resource naming different targets is invalid_target", func(t *testing.T) {
		exchanger := &fakeWorkloadIdentityExchanger{}
		installWorkloadExchanger(t, exchanger)

		r := formPost(t, "/oauth/token", url.Values{
			"grant_type":         {"urn:ietf:params:oauth:grant-type:token-exchange"},
			"subject_token":      {"external-oidc-token"},
			"subject_token_type": {wifForm},
			"audience":           {"https://api.a.example.com"},
			"resource":           {"https://api.b.example.com"},
		})
		w := httptest.NewRecorder()
		NewOAuthTokenExchangeHandler(&mockOAuthTokenExchangeService{}).Exchange(w, r)

		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Contains(t, w.Body.String(), "invalid_target")
		assert.False(t, exchanger.called)
	})

	t.Run("a resource-only request reaches the exchanger as the audience", func(t *testing.T) {
		exchanger := &fakeWorkloadIdentityExchanger{
			result: &WorkloadTokenExchangeResult{AccessToken: "at", TokenType: "Bearer", ExpiresIn: 60},
		}
		installWorkloadExchanger(t, exchanger)

		r := formPost(t, "/oauth/token", url.Values{
			"grant_type":         {"urn:ietf:params:oauth:grant-type:token-exchange"},
			"subject_token":      {"external-oidc-token"},
			"subject_token_type": {wifForm},
			"resource":           {"https://api.example.com"},
		})
		w := httptest.NewRecorder()
		NewOAuthTokenExchangeHandler(&mockOAuthTokenExchangeService{}).Exchange(w, r)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.True(t, exchanger.called)
		assert.Equal(t, "https://api.example.com", exchanger.got.Audience,
			"the exchanger only inspects Audience, so an RFC 8707 resource has to arrive there or it is silently ignored")
	})
}
