package dto

import (
	"time"

	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/google/uuid"
	"github.com/maintainerd/auth/internal/security"
)

// ──────────────────────────────────────────────────────────────────────────────
// Authorization Endpoint
// ──────────────────────────────────────────────────────────────────────────────

// OAuthAuthorizeRequestDTO captures the query parameters for the
// GET /oauth/authorize endpoint (RFC 6749 §4.1.1).
type OAuthAuthorizeRequestDTO struct {
	ResponseType        string `json:"response_type"`
	ClientID            string `json:"client_id"`
	RedirectURI         string `json:"redirect_uri"`
	Scope               string `json:"scope"`
	State               string `json:"state"`
	Nonce               string `json:"nonce"`
	CodeChallenge       string `json:"code_challenge"`
	CodeChallengeMethod string `json:"code_challenge_method"`
}

// Validate sanitises inputs and checks required OAuth parameters.
func (r *OAuthAuthorizeRequestDTO) Validate() error {
	r.ResponseType = security.SanitizeInput(r.ResponseType)
	r.ClientID = security.SanitizeInput(r.ClientID)
	r.RedirectURI = security.SanitizeInput(r.RedirectURI)
	r.Scope = security.SanitizeInput(r.Scope)
	r.State = security.SanitizeInput(r.State)
	r.Nonce = security.SanitizeInput(r.Nonce)
	r.CodeChallenge = security.SanitizeInput(r.CodeChallenge)
	r.CodeChallengeMethod = security.SanitizeInput(r.CodeChallengeMethod)

	return validation.ValidateStruct(r,
		validation.Field(&r.ResponseType,
			validation.Required.Error("response_type is required"),
			validation.In("code").Error("response_type must be 'code'"),
		),
		validation.Field(&r.ClientID,
			validation.Required.Error("client_id is required"),
			validation.Length(1, 255).Error("client_id must not exceed 255 characters"),
		),
		validation.Field(&r.RedirectURI,
			validation.Required.Error("redirect_uri is required"),
			validation.Length(1, 2048).Error("redirect_uri must not exceed 2048 characters"),
		),
		validation.Field(&r.CodeChallenge,
			validation.Required.Error("code_challenge is required"),
			validation.Length(43, 128).Error("code_challenge must be between 43 and 128 characters"),
		),
		validation.Field(&r.CodeChallengeMethod,
			validation.Required.Error("code_challenge_method is required"),
			validation.In("S256").Error("code_challenge_method must be 'S256'"),
		),
		validation.Field(&r.State,
			validation.Length(0, 512).Error("state must not exceed 512 characters"),
		),
		validation.Field(&r.Scope,
			validation.Length(0, 1024).Error("scope must not exceed 1024 characters"),
		),
		validation.Field(&r.Nonce,
			validation.Length(0, 512).Error("nonce must not exceed 512 characters"),
		),
	)
}

// OAuthAuthorizeResponseDTO is returned on a successful authorization request
// when the caller is already authenticated and has consented.
type OAuthAuthorizeResponseDTO struct {
	RedirectURI string `json:"redirect_uri"`
}

// OAuthConsentRequiredResponseDTO is returned when the user must approve scopes
// before the authorization code can be issued.
type OAuthConsentRequiredResponseDTO struct {
	ConsentChallenge string `json:"consent_challenge"`
	RedirectURI      string `json:"redirect_uri"`
}

// ──────────────────────────────────────────────────────────────────────────────
// Consent Endpoint
// ──────────────────────────────────────────────────────────────────────────────

// OAuthConsentChallengeResponseDTO describes a pending consent challenge for
// the frontend to display.
type OAuthConsentChallengeResponseDTO struct {
	ChallengeID string   `json:"challenge_id"`
	ClientName  string   `json:"client_name"`
	ClientUUID  string   `json:"client_uuid"`
	Scopes      []string `json:"scopes"`
	RedirectURI string   `json:"redirect_uri"`
	ExpiresAt   int64    `json:"expires_at"`
}

// OAuthConsentDecisionDTO captures the user's decision (approve or deny).
type OAuthConsentDecisionDTO struct {
	ChallengeID string `json:"challenge_id"`
	Approved    bool   `json:"approved"`
}

// Validate sanitises inputs and checks that the challenge ID is a valid UUID.
func (r *OAuthConsentDecisionDTO) Validate() error {
	r.ChallengeID = security.SanitizeInput(r.ChallengeID)

	return validation.ValidateStruct(r,
		validation.Field(&r.ChallengeID,
			validation.Required.Error("challenge_id is required"),
			validation.By(func(value any) error {
				s, _ := value.(string)
				if _, err := uuid.Parse(s); err != nil {
					return validation.NewError("validation_uuid", "challenge_id must be a valid UUID")
				}
				return nil
			}),
		),
	)
}

// OAuthConsentDecisionResponseDTO is the redirect returned after the user
// approves or denies consent.
type OAuthConsentDecisionResponseDTO struct {
	RedirectURI string `json:"redirect_uri"`
}

// ──────────────────────────────────────────────────────────────────────────────
// Token Endpoint
// ──────────────────────────────────────────────────────────────────────────────

// OAuthTokenRequestDTO captures the form-encoded body of the
// POST /oauth/token endpoint (RFC 6749 §4.1.3, §6).
type OAuthTokenRequestDTO struct {
	GrantType    string `json:"grant_type"`
	Code         string `json:"code"`
	RedirectURI  string `json:"redirect_uri"`
	CodeVerifier string `json:"code_verifier"`
	RefreshToken string `json:"refresh_token"`
	Scope        string `json:"scope"`
	// Client credentials (from body when token_endpoint_auth_method=client_secret_post)
	ClientID     string `json:"client_id"`
	ClientSecret string `json:"client_secret"`
}

// Validate sanitises inputs and checks grant-type-specific required fields.
func (r *OAuthTokenRequestDTO) Validate() error {
	r.GrantType = security.SanitizeInput(r.GrantType)
	r.Code = security.SanitizeInput(r.Code)
	r.RedirectURI = security.SanitizeInput(r.RedirectURI)
	r.CodeVerifier = security.SanitizeInput(r.CodeVerifier)
	r.RefreshToken = security.SanitizeInput(r.RefreshToken)
	r.Scope = security.SanitizeInput(r.Scope)
	r.ClientID = security.SanitizeInput(r.ClientID)
	r.ClientSecret = security.SanitizeInput(r.ClientSecret)

	return validation.ValidateStruct(r,
		validation.Field(&r.GrantType,
			validation.Required.Error("grant_type is required"),
			validation.In("authorization_code", "refresh_token", "client_credentials").
				Error("grant_type must be one of: authorization_code, refresh_token, client_credentials"),
		),
	)
}

// OAuthTokenResponseDTO is the JSON body returned by the token endpoint on
// success (RFC 6749 §5.1).
type OAuthTokenResponseDTO struct {
	AccessToken  string `json:"access_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int64  `json:"expires_in"`
	RefreshToken string `json:"refresh_token,omitempty"`
	IDToken      string `json:"id_token,omitempty"`
	Scope        string `json:"scope,omitempty"`
}

// ──────────────────────────────────────────────────────────────────────────────
// Revocation Endpoint (RFC 7009)
// ──────────────────────────────────────────────────────────────────────────────

// OAuthRevokeRequestDTO captures the form-encoded body of the
// POST /oauth/revoke endpoint.
type OAuthRevokeRequestDTO struct {
	Token         string `json:"token"`
	TokenTypeHint string `json:"token_type_hint"`
	// Client credentials (from body when token_endpoint_auth_method=client_secret_post)
	ClientID     string `json:"client_id"`
	ClientSecret string `json:"client_secret"`
}

// Validate sanitises inputs and checks the required token field.
func (r *OAuthRevokeRequestDTO) Validate() error {
	r.Token = security.SanitizeInput(r.Token)
	r.TokenTypeHint = security.SanitizeInput(r.TokenTypeHint)
	r.ClientID = security.SanitizeInput(r.ClientID)
	r.ClientSecret = security.SanitizeInput(r.ClientSecret)

	return validation.ValidateStruct(r,
		validation.Field(&r.Token,
			validation.Required.Error("token is required"),
		),
		validation.Field(&r.TokenTypeHint,
			validation.In("access_token", "refresh_token", "").
				Error("token_type_hint must be 'access_token' or 'refresh_token'"),
		),
	)
}

// ──────────────────────────────────────────────────────────────────────────────
// Introspection Endpoint (RFC 7662)
// ──────────────────────────────────────────────────────────────────────────────

// OAuthIntrospectRequestDTO captures the form-encoded body of the
// POST /oauth/introspect endpoint.
type OAuthIntrospectRequestDTO struct {
	Token         string `json:"token"`
	TokenTypeHint string `json:"token_type_hint"`
}

// Validate sanitises inputs and checks the required token field.
func (r *OAuthIntrospectRequestDTO) Validate() error {
	r.Token = security.SanitizeInput(r.Token)
	r.TokenTypeHint = security.SanitizeInput(r.TokenTypeHint)

	return validation.ValidateStruct(r,
		validation.Field(&r.Token,
			validation.Required.Error("token is required"),
		),
		validation.Field(&r.TokenTypeHint,
			validation.In("access_token", "refresh_token", "").
				Error("token_type_hint must be 'access_token' or 'refresh_token'"),
		),
	)
}

// OAuthIntrospectResponseDTO is the JSON body returned by the introspection
// endpoint (RFC 7662 §2.2).
type OAuthIntrospectResponseDTO struct {
	Active    bool   `json:"active"`
	Scope     string `json:"scope,omitempty"`
	ClientID  string `json:"client_id,omitempty"`
	Username  string `json:"username,omitempty"`
	TokenType string `json:"token_type,omitempty"`
	Exp       int64  `json:"exp,omitempty"`
	Iat       int64  `json:"iat,omitempty"`
	Nbf       int64  `json:"nbf,omitempty"`
	Sub       string `json:"sub,omitempty"`
	Aud       string `json:"aud,omitempty"`
	Iss       string `json:"iss,omitempty"`
	Jti       string `json:"jti,omitempty"`
}

// ──────────────────────────────────────────────────────────────────────────────
// Discovery / Well-Known (RFC 8414)
// ──────────────────────────────────────────────────────────────────────────────

// OAuthDiscoveryResponseDTO is the JSON body for the
// GET /.well-known/openid-configuration endpoint.
type OAuthDiscoveryResponseDTO struct {
	Issuer                string   `json:"issuer"`
	AuthorizationEndpoint string   `json:"authorization_endpoint"`
	TokenEndpoint         string   `json:"token_endpoint"`
	UserinfoEndpoint      string   `json:"userinfo_endpoint"`
	JwksURI               string   `json:"jwks_uri"`
	RevocationEndpoint    string   `json:"revocation_endpoint"`
	IntrospectionEndpoint string   `json:"introspection_endpoint"`
	ScopesSupported       []string `json:"scopes_supported"`
	ResponseTypesSupp     []string `json:"response_types_supported"`
	GrantTypesSupported   []string `json:"grant_types_supported"`
	SubjectTypesSupported []string `json:"subject_types_supported"`
	IDTokenSignAlgValues  []string `json:"id_token_signing_alg_values_supported"`
	TokenEndpointAuth     []string `json:"token_endpoint_auth_methods_supported"`
	CodeChallengeMethods  []string `json:"code_challenge_methods_supported"`
}

// ──────────────────────────────────────────────────────────────────────────────
// JWKS (RFC 7517)
// ──────────────────────────────────────────────────────────────────────────────

// JWKSResponseDTO is the JSON Web Key Set.
type JWKSResponseDTO struct {
	Keys []JWKKeyDTO `json:"keys"`
}

// JWKKeyDTO is a single JSON Web Key (RSA public key).
type JWKKeyDTO struct {
	Kty string `json:"kty"`
	Use string `json:"use"`
	Kid string `json:"kid"`
	Alg string `json:"alg"`
	N   string `json:"n"`
	E   string `json:"e"`
}

// ──────────────────────────────────────────────────────────────────────────────
// UserInfo (OpenID Connect Core §5.3)
// ──────────────────────────────────────────────────────────────────────────────

// OAuthUserInfoResponseDTO is the JSON body for GET /oauth/userinfo.
type OAuthUserInfoResponseDTO struct {
	Sub           string `json:"sub"`
	Email         string `json:"email,omitempty"`
	EmailVerified bool   `json:"email_verified,omitempty"`
	Phone         string `json:"phone_number,omitempty"`
	PhoneVerified bool   `json:"phone_number_verified,omitempty"`
	Name          string `json:"name,omitempty"`
	Picture       string `json:"picture,omitempty"`
	UpdatedAt     int64  `json:"updated_at,omitempty"`
}

// ──────────────────────────────────────────────────────────────────────────────
// Consent Grant Management (admin)
// ──────────────────────────────────────────────────────────────────────────────

// OAuthConsentGrantResponseDTO represents a persisted consent grant.
type OAuthConsentGrantResponseDTO struct {
	ConsentGrantUUID string   `json:"consent_grant_id"`
	ClientName       string   `json:"client_name"`
	ClientUUID       string   `json:"client_uuid"`
	Scopes           []string `json:"scopes"`
	GrantedAt        string   `json:"granted_at"`
	UpdatedAt        string   `json:"updated_at"`
}

// ──────────────────────────────────────────────────────────────────────────────
// Internal types used by services
// ──────────────────────────────────────────────────────────────────────────────

// OAuthClientCredentials holds the resolved client_id and client_secret from
// either the Authorization header (Basic auth) or the POST body.
type OAuthClientCredentials struct {
	ClientID     string
	ClientSecret string
}

// OAuthAuthorizeResult is the internal result returned by the authorize service
// method. One of RedirectURI or ConsentChallenge will be set.
type OAuthAuthorizeResult struct {
	// RedirectURI is the full redirect (including ?code=...&state=...) when
	// the authorization code was issued immediately.
	RedirectURI string
	// ConsentChallenge is set when user consent is required. The frontend
	// must redirect the user to the consent page.
	ConsentChallenge string
}

// OAuthConsentDecisionResult is the internal result from processing consent.
type OAuthConsentDecisionResult struct {
	RedirectURI string
}

// OAuthTokenResult is the internal result from the token service.
type OAuthTokenResult struct {
	AccessToken  string
	TokenType    string
	ExpiresIn    int64
	RefreshToken string
	IDToken      string
	Scope        string
}

// OAuthTokenIssuedAt is used internally to track when a token was issued.
type OAuthTokenIssuedAt struct {
	Time time.Time
}
