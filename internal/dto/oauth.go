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

// ──────────────────────────────────────────────────────────────────────────────
// OAuth Authorization Server Metadata (RFC 8414)
// ──────────────────────────────────────────────────────────────────────────────

// OAuthAuthorizationServerMetadataDTO is the JSON body for
// GET /.well-known/oauth-authorization-server (RFC 8414). Unlike the OIDC
// discovery document it omits OIDC-specific fields (userinfo, id_token_alg).
type OAuthAuthorizationServerMetadataDTO struct {
	Issuer                        string   `json:"issuer"`
	AuthorizationEndpoint         string   `json:"authorization_endpoint"`
	TokenEndpoint                 string   `json:"token_endpoint"`
	JwksURI                       string   `json:"jwks_uri"`
	RevocationEndpoint            string   `json:"revocation_endpoint"`
	IntrospectionEndpoint         string   `json:"introspection_endpoint"`
	PAREndpoint                   string   `json:"pushed_authorization_request_endpoint,omitempty"`
	DeviceAuthorizationEndpoint   string   `json:"device_authorization_endpoint,omitempty"`
	RegistrationEndpoint          string   `json:"registration_endpoint,omitempty"`
	BCAuthorizeEndpoint           string   `json:"backchannel_authentication_endpoint,omitempty"`
	EndSessionEndpoint            string   `json:"end_session_endpoint,omitempty"`
	ScopesSupported               []string `json:"scopes_supported"`
	ResponseTypesSupported        []string `json:"response_types_supported"`
	GrantTypesSupported           []string `json:"grant_types_supported"`
	TokenEndpointAuthMethods      []string `json:"token_endpoint_auth_methods_supported"`
	CodeChallengeMethods          []string `json:"code_challenge_methods_supported"`
	BackchannelTokenDeliveryModes []string `json:"backchannel_token_delivery_modes_supported,omitempty"`
}

// ──────────────────────────────────────────────────────────────────────────────
// Pushed Authorization Requests (RFC 9126)
// ──────────────────────────────────────────────────────────────────────────────

// OAuthPARRequestDTO is the form-encoded body of POST /oauth/par.
type OAuthPARRequestDTO struct {
	ResponseType        string `json:"response_type"`
	ClientID            string `json:"client_id"`
	RedirectURI         string `json:"redirect_uri"`
	Scope               string `json:"scope"`
	State               string `json:"state"`
	Nonce               string `json:"nonce"`
	CodeChallenge       string `json:"code_challenge"`
	CodeChallengeMethod string `json:"code_challenge_method"`
	// Client credentials (from body when token_endpoint_auth_method=client_secret_post)
	ClientSecret string `json:"client_secret"`
}

func (r *OAuthPARRequestDTO) Validate() error {
	r.ResponseType = security.SanitizeInput(r.ResponseType)
	r.ClientID = security.SanitizeInput(r.ClientID)
	r.RedirectURI = security.SanitizeInput(r.RedirectURI)
	r.Scope = security.SanitizeInput(r.Scope)
	r.State = security.SanitizeInput(r.State)
	r.Nonce = security.SanitizeInput(r.Nonce)
	r.CodeChallenge = security.SanitizeInput(r.CodeChallenge)
	r.CodeChallengeMethod = security.SanitizeInput(r.CodeChallengeMethod)
	r.ClientSecret = security.SanitizeInput(r.ClientSecret)

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

// OAuthPARResponseDTO is returned by POST /oauth/par on success (RFC 9126 §2.2).
type OAuthPARResponseDTO struct {
	RequestURI string `json:"request_uri"`
	ExpiresIn  int    `json:"expires_in"`
}

// ──────────────────────────────────────────────────────────────────────────────
// Device Authorization Grant (RFC 8628)
// ──────────────────────────────────────────────────────────────────────────────

// OAuthDeviceAuthorizationRequestDTO is the form-encoded body of
// POST /oauth/device_authorization (RFC 8628 §3.1).
type OAuthDeviceAuthorizationRequestDTO struct {
	ClientID     string `json:"client_id"`
	ClientSecret string `json:"client_secret"`
	Scope        string `json:"scope"`
}

func (r *OAuthDeviceAuthorizationRequestDTO) Validate() error {
	r.ClientID = security.SanitizeInput(r.ClientID)
	r.ClientSecret = security.SanitizeInput(r.ClientSecret)
	r.Scope = security.SanitizeInput(r.Scope)

	return validation.ValidateStruct(r,
		validation.Field(&r.ClientID,
			validation.Required.Error("client_id is required"),
			validation.Length(1, 255).Error("client_id must not exceed 255 characters"),
		),
		validation.Field(&r.Scope,
			validation.Length(0, 1024).Error("scope must not exceed 1024 characters"),
		),
	)
}

// OAuthDeviceAuthorizationResponseDTO is returned on a successful device
// authorization request (RFC 8628 §3.2).
type OAuthDeviceAuthorizationResponseDTO struct {
	DeviceCode              string `json:"device_code"`
	UserCode                string `json:"user_code"`
	VerificationURI         string `json:"verification_uri"`
	VerificationURIComplete string `json:"verification_uri_complete,omitempty"`
	ExpiresIn               int    `json:"expires_in"`
	Interval                int    `json:"interval"`
}

// OAuthDeviceVerifyRequestDTO is posted when the user submits the user_code
// at the verification URI.
type OAuthDeviceVerifyRequestDTO struct {
	UserCode string `json:"user_code"`
}

func (r *OAuthDeviceVerifyRequestDTO) Validate() error {
	r.UserCode = security.SanitizeInput(r.UserCode)

	return validation.ValidateStruct(r,
		validation.Field(&r.UserCode,
			validation.Required.Error("user_code is required"),
			validation.Length(8, 9).Error("user_code must be 8 characters (XXXX-XXXX format)"),
		),
	)
}

// OAuthDeviceTokenRequestDTO captures the fields for polling POST /oauth/token
// with grant_type=urn:ietf:params:oauth:grant-type:device_code (RFC 8628 §3.4).
type OAuthDeviceTokenRequestDTO struct {
	DeviceCode   string `json:"device_code"`
	ClientID     string `json:"client_id"`
	ClientSecret string `json:"client_secret"`
}

func (r *OAuthDeviceTokenRequestDTO) Validate() error {
	r.DeviceCode = security.SanitizeInput(r.DeviceCode)
	r.ClientID = security.SanitizeInput(r.ClientID)
	r.ClientSecret = security.SanitizeInput(r.ClientSecret)

	return validation.ValidateStruct(r,
		validation.Field(&r.DeviceCode,
			validation.Required.Error("device_code is required"),
		),
		validation.Field(&r.ClientID,
			validation.Required.Error("client_id is required"),
		),
	)
}

// ──────────────────────────────────────────────────────────────────────────────
// Token Exchange (RFC 8693)
// ──────────────────────────────────────────────────────────────────────────────

// OAuthTokenExchangeRequestDTO captures the fields for
// POST /oauth/token with grant_type=urn:ietf:params:oauth:grant-type:token-exchange
// (RFC 8693 §2.1).
type OAuthTokenExchangeRequestDTO struct {
	SubjectToken         string `json:"subject_token"`
	SubjectTokenType     string `json:"subject_token_type"`
	ActorToken           string `json:"actor_token"`
	ActorTokenType       string `json:"actor_token_type"`
	RequestedTokenType   string `json:"requested_token_type"`
	Resource             string `json:"resource"`
	Audience             string `json:"audience"`
	Scope                string `json:"scope"`
	ClientID             string `json:"client_id"`
	ClientSecret         string `json:"client_secret"`
}

func (r *OAuthTokenExchangeRequestDTO) Validate() error {
	r.SubjectToken = security.SanitizeInput(r.SubjectToken)
	r.SubjectTokenType = security.SanitizeInput(r.SubjectTokenType)
	r.ActorToken = security.SanitizeInput(r.ActorToken)
	r.ActorTokenType = security.SanitizeInput(r.ActorTokenType)
	r.RequestedTokenType = security.SanitizeInput(r.RequestedTokenType)
	r.Resource = security.SanitizeInput(r.Resource)
	r.Audience = security.SanitizeInput(r.Audience)
	r.Scope = security.SanitizeInput(r.Scope)
	r.ClientID = security.SanitizeInput(r.ClientID)
	r.ClientSecret = security.SanitizeInput(r.ClientSecret)

	const (
		tokenTypeAccessToken  = "urn:ietf:params:oauth:token-type:access_token"
		tokenTypeRefreshToken = "urn:ietf:params:oauth:token-type:refresh_token"
		tokenTypeIDToken      = "urn:ietf:params:oauth:token-type:id_token"
		tokenTypeJWT          = "urn:ietf:params:oauth:token-type:jwt"
	)

	return validation.ValidateStruct(r,
		validation.Field(&r.SubjectToken,
			validation.Required.Error("subject_token is required"),
		),
		validation.Field(&r.SubjectTokenType,
			validation.Required.Error("subject_token_type is required"),
			validation.In(tokenTypeAccessToken, tokenTypeRefreshToken, tokenTypeIDToken, tokenTypeJWT).
				Error("subject_token_type must be a valid token type URI"),
		),
		validation.Field(&r.RequestedTokenType,
			validation.In(tokenTypeAccessToken, tokenTypeRefreshToken, tokenTypeIDToken, tokenTypeJWT, "").
				Error("requested_token_type must be a valid token type URI"),
		),
		validation.Field(&r.ClientID,
			validation.Required.Error("client_id is required"),
		),
		validation.Field(&r.Scope,
			validation.Length(0, 1024).Error("scope must not exceed 1024 characters"),
		),
	)
}

// OAuthTokenExchangeResponseDTO is the response from a successful token exchange
// (RFC 8693 §2.2).
type OAuthTokenExchangeResponseDTO struct {
	AccessToken     string `json:"access_token"`
	IssuedTokenType string `json:"issued_token_type"`
	TokenType       string `json:"token_type"`
	ExpiresIn       int64  `json:"expires_in"`
	Scope           string `json:"scope,omitempty"`
	RefreshToken    string `json:"refresh_token,omitempty"`
}

// ──────────────────────────────────────────────────────────────────────────────
// Dynamic Client Registration (RFC 7591)
// ──────────────────────────────────────────────────────────────────────────────

// OAuthClientRegistrationRequestDTO is the JSON body for
// POST /oauth/register (RFC 7591 §2).
type OAuthClientRegistrationRequestDTO struct {
	ClientName              string   `json:"client_name"`
	RedirectURIs            []string `json:"redirect_uris"`
	GrantTypes              []string `json:"grant_types"`
	ResponseTypes           []string `json:"response_types"`
	Scope                   string   `json:"scope"`
	TokenEndpointAuthMethod string   `json:"token_endpoint_auth_method"`
	LogoURI                 string   `json:"logo_uri"`
	PolicyURI               string   `json:"policy_uri"`
	TOSURI                  string   `json:"tos_uri"`
	Contacts                []string `json:"contacts"`
	// Additional fields used internally to associate the client with a tenant/IDP.
	IdentityProviderID int64 `json:"identity_provider_id"`
}

func (r *OAuthClientRegistrationRequestDTO) Validate() error {
	r.ClientName = security.SanitizeInput(r.ClientName)
	r.Scope = security.SanitizeInput(r.Scope)
	r.TokenEndpointAuthMethod = security.SanitizeInput(r.TokenEndpointAuthMethod)
	r.LogoURI = security.SanitizeInput(r.LogoURI)
	r.PolicyURI = security.SanitizeInput(r.PolicyURI)
	r.TOSURI = security.SanitizeInput(r.TOSURI)

	for i, u := range r.RedirectURIs {
		r.RedirectURIs[i] = security.SanitizeInput(u)
	}
	for i, g := range r.GrantTypes {
		r.GrantTypes[i] = security.SanitizeInput(g)
	}
	for i, rt := range r.ResponseTypes {
		r.ResponseTypes[i] = security.SanitizeInput(rt)
	}

	return validation.ValidateStruct(r,
		validation.Field(&r.ClientName,
			validation.Required.Error("client_name is required"),
			validation.Length(1, 255).Error("client_name must not exceed 255 characters"),
		),
		validation.Field(&r.RedirectURIs,
			validation.Required.Error("redirect_uris is required"),
			validation.Length(1, 10).Error("between 1 and 10 redirect_uris are allowed"),
		),
		validation.Field(&r.TokenEndpointAuthMethod,
			validation.In("client_secret_basic", "client_secret_post", "none", "").
				Error("token_endpoint_auth_method must be client_secret_basic, client_secret_post, or none"),
		),
		validation.Field(&r.Scope,
			validation.Length(0, 1024).Error("scope must not exceed 1024 characters"),
		),
		validation.Field(&r.IdentityProviderID,
			validation.Required.Error("identity_provider_id is required"),
			validation.Min(int64(1)).Error("identity_provider_id must be a positive integer"),
		),
	)
}

// OAuthClientRegistrationResponseDTO is the JSON body returned by
// POST /oauth/register on success (RFC 7591 §3.2).
type OAuthClientRegistrationResponseDTO struct {
	ClientID                string   `json:"client_id"`
	ClientSecret            string   `json:"client_secret,omitempty"`
	ClientIDIssuedAt        int64    `json:"client_id_issued_at"`
	ClientSecretExpiresAt   int64    `json:"client_secret_expires_at"`
	ClientName              string   `json:"client_name"`
	RedirectURIs            []string `json:"redirect_uris"`
	GrantTypes              []string `json:"grant_types"`
	ResponseTypes           []string `json:"response_types"`
	TokenEndpointAuthMethod string   `json:"token_endpoint_auth_method"`
	Scope                   string   `json:"scope,omitempty"`
}

// ──────────────────────────────────────────────────────────────────────────────
// Backchannel Logout (OIDC Back-Channel Logout 1.0)
// ──────────────────────────────────────────────────────────────────────────────

// OAuthBackchannelLogoutRequestDTO is the form-encoded body for
// POST /oauth/logout/backchannel.
type OAuthBackchannelLogoutRequestDTO struct {
	LogoutToken string `json:"logout_token"`
}

func (r *OAuthBackchannelLogoutRequestDTO) Validate() error {
	r.LogoutToken = security.SanitizeInput(r.LogoutToken)

	return validation.ValidateStruct(r,
		validation.Field(&r.LogoutToken,
			validation.Required.Error("logout_token is required"),
		),
	)
}

// ──────────────────────────────────────────────────────────────────────────────
// RP-Initiated Logout (OIDC Session Management 1.0)
// ──────────────────────────────────────────────────────────────────────────────

// OAuthEndSessionRequestDTO captures the query parameters for
// GET /oauth/end_session (OIDC Session Management 1.0).
type OAuthEndSessionRequestDTO struct {
	IDTokenHint            string `json:"id_token_hint"`
	PostLogoutRedirectURI  string `json:"post_logout_redirect_uri"`
	State                  string `json:"state"`
	ClientID               string `json:"client_id"`
}

func (r *OAuthEndSessionRequestDTO) Validate() error {
	r.IDTokenHint = security.SanitizeInput(r.IDTokenHint)
	r.PostLogoutRedirectURI = security.SanitizeInput(r.PostLogoutRedirectURI)
	r.State = security.SanitizeInput(r.State)
	r.ClientID = security.SanitizeInput(r.ClientID)

	return validation.ValidateStruct(r,
		validation.Field(&r.PostLogoutRedirectURI,
			validation.Length(0, 2048).Error("post_logout_redirect_uri must not exceed 2048 characters"),
		),
		validation.Field(&r.State,
			validation.Length(0, 512).Error("state must not exceed 512 characters"),
		),
	)
}

// ──────────────────────────────────────────────────────────────────────────────
// CIBA — Client-Initiated Backchannel Authentication (RFC 9126)
// ──────────────────────────────────────────────────────────────────────────────

// OAuthCIBARequestDTO is the form-encoded body for POST /oauth/bc-authorize.
type OAuthCIBARequestDTO struct {
	Scope                     string `json:"scope"`
	ClientNotificationToken   string `json:"client_notification_token"`
	ACRValues                 string `json:"acr_values"`
	LoginHint                 string `json:"login_hint"`
	LoginHintToken            string `json:"login_hint_token"`
	IDTokenHint               string `json:"id_token_hint"`
	BindingMessage            string `json:"binding_message"`
	UserCode                  string `json:"user_code"`
	RequestedExpiry           int    `json:"requested_expiry"`
	ClientID                  string `json:"client_id"`
	ClientSecret              string `json:"client_secret"`
}

func (r *OAuthCIBARequestDTO) Validate() error {
	r.Scope = security.SanitizeInput(r.Scope)
	r.ClientNotificationToken = security.SanitizeInput(r.ClientNotificationToken)
	r.ACRValues = security.SanitizeInput(r.ACRValues)
	r.LoginHint = security.SanitizeInput(r.LoginHint)
	r.LoginHintToken = security.SanitizeInput(r.LoginHintToken)
	r.IDTokenHint = security.SanitizeInput(r.IDTokenHint)
	r.BindingMessage = security.SanitizeInput(r.BindingMessage)
	r.UserCode = security.SanitizeInput(r.UserCode)
	r.ClientID = security.SanitizeInput(r.ClientID)
	r.ClientSecret = security.SanitizeInput(r.ClientSecret)

	return validation.ValidateStruct(r,
		validation.Field(&r.ClientID,
			validation.Required.Error("client_id is required"),
		),
		validation.Field(&r.Scope,
			validation.Required.Error("scope is required"),
			validation.Length(1, 1024).Error("scope must not exceed 1024 characters"),
		),
		validation.Field(&r.LoginHint,
			validation.When(
				r.LoginHintToken == "" && r.IDTokenHint == "",
				validation.Required.Error("one of login_hint, login_hint_token, or id_token_hint is required"),
			),
		),
		validation.Field(&r.BindingMessage,
			validation.Length(0, 128).Error("binding_message must not exceed 128 characters"),
		),
	)
}

// OAuthCIBAResponseDTO is returned on a successful bc-authorize request.
type OAuthCIBAResponseDTO struct {
	AuthReqID string `json:"auth_req_id"`
	ExpiresIn int    `json:"expires_in"`
	Interval  int    `json:"interval"`
}

// OAuthCIBATokenRequestDTO captures the polling request at POST /oauth/token
// with grant_type=urn:ietf:params:oauth:grant-type:ciba.
type OAuthCIBATokenRequestDTO struct {
	AuthReqID    string `json:"auth_req_id"`
	ClientID     string `json:"client_id"`
	ClientSecret string `json:"client_secret"`
}

func (r *OAuthCIBATokenRequestDTO) Validate() error {
	r.AuthReqID = security.SanitizeInput(r.AuthReqID)
	r.ClientID = security.SanitizeInput(r.ClientID)
	r.ClientSecret = security.SanitizeInput(r.ClientSecret)

	return validation.ValidateStruct(r,
		validation.Field(&r.AuthReqID,
			validation.Required.Error("auth_req_id is required"),
		),
		validation.Field(&r.ClientID,
			validation.Required.Error("client_id is required"),
		),
	)
}
