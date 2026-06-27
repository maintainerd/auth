package oauth

// Authorization Endpoint

// OAuthAuthorizeRequestDTO captures the query parameters for the
// GET /oauth/authorize endpoint (RFC 6749 §4.1.1).
type OAuthAuthorizeRequestDTO struct {
	ResponseType        string `json:"response_type"`
	ClientID            string `json:"client_id"`
	RedirectURI         string `json:"redirect_uri"`
	Scope               string `json:"scope"`
	State               string `json:"state"`
	Nonce               string `json:"nonce"`
	IdpHint             string `json:"idp_hint"`
	Prompt              string `json:"prompt"`
	CodeChallenge       string `json:"code_challenge"`
	CodeChallengeMethod string `json:"code_challenge_method"`
}

// OAuthAuthorizeResponseDTO is returned on a successful authorization request
// when the caller is already authenticated and has consented.
type OAuthAuthorizeResponseDTO struct {
	RedirectURI string `json:"redirect_uri"`
}

// OAuthConnectionsResponseDTO lists the login options a client offers so the
// hosted identity app can render its login page: whether username/password is
// available and which OAuth2 providers are connected. It never exposes provider
// config or secrets.
type OAuthConnectionsResponseDTO struct {
	PasswordEnabled bool                 `json:"password_enabled"`
	Connections     []OAuthConnectionDTO `json:"connections"`
}

// OAuthConnectionDTO is one connected identity provider (an OAuth2 login button).
// Identifier is what the identity app passes back as idp_hint on /oauth/authorize.
type OAuthConnectionDTO struct {
	Identifier   string `json:"identifier"`
	DisplayName  string `json:"display_name"`
	Provider     string `json:"provider"`
	ProviderType string `json:"provider_type"`
	IsDefault    bool   `json:"is_default"`
	DisplayOrder int    `json:"display_order"`
}

// OAuthConsentRequiredResponseDTO is returned when the user must approve scopes
// before the authorization code can be issued.
type OAuthConsentRequiredResponseDTO struct {
	ConsentChallenge string `json:"consent_challenge"`
	RedirectURI      string `json:"redirect_uri"`
}

// Consent Endpoint

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

// OAuthConsentDecisionResponseDTO is the redirect returned after the user
// approves or denies consent.
type OAuthConsentDecisionResponseDTO struct {
	RedirectURI string `json:"redirect_uri"`
}

// Token Endpoint

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
	// JWT assertion for private_key_jwt / client_secret_jwt (RFC 7523)
	ClientAssertionType string `json:"client_assertion_type"`
	ClientAssertion     string `json:"client_assertion"`
	// DPoP proof JWK thumbprint (RFC 9449) — set by the handler after proof validation.
	DPoPThumbprint string `json:"-"`
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

// Revocation Endpoint (RFC 7009)

// OAuthRevokeRequestDTO captures the form-encoded body of the
// POST /oauth/revoke endpoint.
type OAuthRevokeRequestDTO struct {
	Token         string `json:"token"`
	TokenTypeHint string `json:"token_type_hint"`
	// Client credentials (from body when token_endpoint_auth_method=client_secret_post)
	ClientID     string `json:"client_id"`
	ClientSecret string `json:"client_secret"`
}

// Introspection Endpoint (RFC 7662)

// OAuthIntrospectRequestDTO captures the form-encoded body of the
// POST /oauth/introspect endpoint.
type OAuthIntrospectRequestDTO struct {
	Token         string `json:"token"`
	TokenTypeHint string `json:"token_type_hint"`
	ClientID      string `json:"client_id"`
	ClientSecret  string `json:"client_secret"`
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

// Discovery / Well-Known (RFC 8414)

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
	DPoPSigningAlgValues  []string `json:"dpop_signing_alg_values_supported,omitempty"`
}

// JWKS (RFC 7517)

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

// UserInfo (OpenID Connect Core §5.3)

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

// Consent Grant Management (admin)

// OAuthConsentGrantResponseDTO represents a persisted consent grant.
type OAuthConsentGrantResponseDTO struct {
	ConsentGrantUUID string   `json:"consent_grant_id"`
	ClientName       string   `json:"client_name"`
	ClientUUID       string   `json:"client_uuid"`
	Scopes           []string `json:"scopes"`
	GrantedAt        string   `json:"granted_at"`
	UpdatedAt        string   `json:"updated_at"`
}

// OAuth Authorization Server Metadata (RFC 8414)

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
	DPoPSigningAlgValues          []string `json:"dpop_signing_alg_values_supported,omitempty"`
	DPoPBindingRequired           bool     `json:"dpop_bound_access_tokens_required,omitempty"`
}

// Pushed Authorization Requests (RFC 9126)

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

// OAuthPARResponseDTO is returned by POST /oauth/par on success (RFC 9126 §2.2).
type OAuthPARResponseDTO struct {
	RequestURI string `json:"request_uri"`
	ExpiresIn  int    `json:"expires_in"`
}

// Device Authorization Grant (RFC 8628)

// OAuthDeviceAuthorizationRequestDTO is the form-encoded body of
// POST /oauth/device_authorization (RFC 8628 §3.1).
type OAuthDeviceAuthorizationRequestDTO struct {
	ClientID     string `json:"client_id"`
	ClientSecret string `json:"client_secret"`
	Scope        string `json:"scope"`
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

// OAuthDeviceTokenRequestDTO captures the fields for polling POST /oauth/token
// with grant_type=urn:ietf:params:oauth:grant-type:device_code (RFC 8628 §3.4).
type OAuthDeviceTokenRequestDTO struct {
	DeviceCode   string `json:"device_code"`
	ClientID     string `json:"client_id"`
	ClientSecret string `json:"client_secret"`
}

// Token Exchange (RFC 8693)

// OAuthTokenExchangeRequestDTO captures the fields for
// POST /oauth/token with grant_type=urn:ietf:params:oauth:grant-type:token-exchange
// (RFC 8693 §2.1).
type OAuthTokenExchangeRequestDTO struct {
	SubjectToken       string `json:"subject_token"`
	SubjectTokenType   string `json:"subject_token_type"`
	ActorToken         string `json:"actor_token"`
	ActorTokenType     string `json:"actor_token_type"`
	RequestedTokenType string `json:"requested_token_type"`
	Resource           string `json:"resource"`
	Audience           string `json:"audience"`
	Scope              string `json:"scope"`
	ClientID           string `json:"client_id"`
	ClientSecret       string `json:"client_secret"`
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

// Dynamic Client Registration (RFC 7591)

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

// Backchannel Logout (OIDC Back-Channel Logout 1.0)

// OAuthBackchannelLogoutRequestDTO is the form-encoded body for
// POST /oauth/logout/backchannel.
type OAuthBackchannelLogoutRequestDTO struct {
	LogoutToken string `json:"logout_token"`
}

// RP-Initiated Logout (OIDC Session Management 1.0)

// OAuthEndSessionRequestDTO captures the query parameters for
// GET /oauth/end_session (OIDC Session Management 1.0).
type OAuthEndSessionRequestDTO struct {
	IDTokenHint           string `json:"id_token_hint"`
	PostLogoutRedirectURI string `json:"post_logout_redirect_uri"`
	State                 string `json:"state"`
	ClientID              string `json:"client_id"`
}

// CIBA — Client-Initiated Backchannel Authentication (RFC 9126)

// OAuthCIBARequestDTO is the form-encoded body for POST /oauth/bc-authorize.
type OAuthCIBARequestDTO struct {
	Scope                   string `json:"scope"`
	ClientNotificationToken string `json:"client_notification_token"`
	ACRValues               string `json:"acr_values"`
	LoginHint               string `json:"login_hint"`
	LoginHintToken          string `json:"login_hint_token"`
	IDTokenHint             string `json:"id_token_hint"`
	BindingMessage          string `json:"binding_message"`
	UserCode                string `json:"user_code"`
	RequestedExpiry         int    `json:"requested_expiry"`
	ClientID                string `json:"client_id"`
	ClientSecret            string `json:"client_secret"`
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
