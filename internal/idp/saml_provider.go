package idp

import (
	"crypto/hmac"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"net/url"
	"strings"
	"time"

	crewsaml "github.com/crewjam/saml"
	"github.com/google/uuid"
	"github.com/maintainerd/maintainerd-auth/internal/platform/config"
)

// samlRelayState is HMAC-signed and passed as the SAML RelayState parameter.
// It carries all context needed to resume the flow after the IdP redirects
// the user back to our ACS endpoint.
type samlRelayState struct {
	ProviderIdentifier string `json:"pi"`
	ClientID           string `json:"cid"`
	RedirectURI        string `json:"ruri"`
	// The tenant is intentionally NOT carried here. RelayState is HMAC-signed but
	// NOT encrypted and traverses the user's browser and the external SAML IdP, so
	// any field is readable by third parties. The tenant is re-derived server-side
	// from the provider identifier at ACS/SLO, so an internal PK never needs to
	// travel — leaving it out closes an internal-identifier disclosure.
	// RequestID is the ID of the SAML AuthnRequest this flow generated. It is
	// carried back through RelayState and fed to ParseResponse as the only
	// accepted InResponseTo value, binding the IdP's Response to the exact
	// request we issued (anti-replay / anti-injection).
	RequestID string `json:"rid"`
	Nonce     string `json:"n"`
	IssuedAt  int64  `json:"iat"`
	// Purpose pins the RelayState to one protocol exchange (SSO or SLO). Both
	// exchanges are signed with the same HMAC key and carry the same fields, so
	// without it a live SSO RelayState could be replayed at the SLO endpoint (and
	// vice versa) to drive a flow it was never issued for.
	Purpose string `json:"p"`
}

// RelayState purposes. verifyRelayStateForPurpose rejects any mismatch, so a
// token minted for one exchange is inert in the other.
const (
	samlRelayPurposeSSO = "sso"
	samlRelayPurposeSLO = "slo"
)

// signRelayState encodes rs as JSON, computes HMAC-SHA256 over it with the
// global HMACSecretKey, and returns "{base64url(json)}.{base64url(sig)}".
func signRelayState(rs *samlRelayState) (string, error) {
	data, err := json.Marshal(rs)
	if err != nil {
		return "", err
	}
	b64 := base64.RawURLEncoding.EncodeToString(data)
	mac := hmac.New(sha256.New, config.HMACSecretKey)
	mac.Write([]byte(b64))
	sig := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
	return b64 + "." + sig, nil
}

// verifyRelayState verifies the HMAC and returns the decoded samlRelayState.
// It also rejects tokens older than 15 minutes.
func verifyRelayState(token string) (*samlRelayState, error) {
	parts := strings.SplitN(token, ".", 2)
	if len(parts) != 2 {
		return nil, fmt.Errorf("malformed relay state")
	}
	b64, sig64 := parts[0], parts[1]

	mac := hmac.New(sha256.New, config.HMACSecretKey)
	mac.Write([]byte(b64))
	expected := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
	if !hmac.Equal([]byte(sig64), []byte(expected)) {
		return nil, fmt.Errorf("relay state signature mismatch")
	}

	data, err := base64.RawURLEncoding.DecodeString(b64)
	if err != nil {
		return nil, fmt.Errorf("relay state decode failed: %w", err)
	}
	var rs samlRelayState
	if err := json.Unmarshal(data, &rs); err != nil {
		return nil, fmt.Errorf("relay state unmarshal failed: %w", err)
	}
	if time.Since(time.Unix(rs.IssuedAt, 0)) > 15*time.Minute {
		return nil, fmt.Errorf("relay state expired")
	}
	return &rs, nil
}

// verifyRelayStateForPurpose verifies the RelayState and additionally requires
// it to have been issued for the given exchange. A blank purpose is rejected
// rather than treated as a wildcard: every RelayState this server mints stamps
// one, so a missing purpose means the token was not minted here or was crafted.
func verifyRelayStateForPurpose(token, purpose string) (*samlRelayState, error) {
	rs, err := verifyRelayState(token)
	if err != nil {
		return nil, err
	}
	if rs.Purpose != purpose {
		return nil, fmt.Errorf("relay state was not issued for %s", purpose)
	}
	return rs, nil
}

// parseSAMLConfig unmarshals the JSONB config column into SAMLProviderConfig.
func parseSAMLConfig(idp *IdentityProvider) (*SAMLProviderConfig, error) {
	if len(idp.Config) == 0 {
		return nil, fmt.Errorf("identity provider has no configuration")
	}
	var cfg SAMLProviderConfig
	if err := json.Unmarshal(idp.Config, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse SAML config: %w", err)
	}
	if cfg.EntityID == "" || cfg.SSOURL == "" || cfg.Certificate == "" {
		return nil, fmt.Errorf("SAML config missing required fields: entity_id, sso_url, certificate")
	}
	return &cfg, nil
}

// parsePEMCertificate decodes a PEM-encoded X.509 certificate string.
func parsePEMCertificate(pemStr string) (*x509.Certificate, error) {
	block, _ := pem.Decode([]byte(pemStr))
	if block == nil {
		return nil, fmt.Errorf("failed to decode PEM block from certificate")
	}
	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse X.509 certificate: %w", err)
	}
	return cert, nil
}

// ParsePEMCertExpiry returns the NotAfter time from a PEM certificate.
// Exported for use in service_provider.go.
func ParsePEMCertExpiry(pemStr string) (*time.Time, error) {
	cert, err := parsePEMCertificate(pemStr)
	if err != nil {
		return nil, err
	}
	t := cert.NotAfter
	return &t, nil
}

// buildIDPEntityDescriptor constructs the crewjam/saml EntityDescriptor that
// describes the remote IdP from our stored SAMLProviderConfig.
func buildIDPEntityDescriptor(cfg *SAMLProviderConfig) (*crewsaml.EntityDescriptor, error) {
	cert, err := parsePEMCertificate(cfg.Certificate)
	if err != nil {
		return nil, fmt.Errorf("invalid IdP certificate: %w", err)
	}
	certDER := base64.StdEncoding.EncodeToString(cert.Raw)

	ed := &crewsaml.EntityDescriptor{
		EntityID: cfg.EntityID,
		IDPSSODescriptors: []crewsaml.IDPSSODescriptor{{
			SSODescriptor: crewsaml.SSODescriptor{
				RoleDescriptor: crewsaml.RoleDescriptor{
					KeyDescriptors: []crewsaml.KeyDescriptor{{
						Use: "signing",
						KeyInfo: crewsaml.KeyInfo{
							X509Data: crewsaml.X509Data{
								X509Certificates: []crewsaml.X509Certificate{{
									Data: certDER,
								}},
							},
						},
					}},
				},
			},
			SingleSignOnServices: []crewsaml.Endpoint{
				{Binding: crewsaml.HTTPRedirectBinding, Location: cfg.SSOURL},
				{Binding: crewsaml.HTTPPostBinding, Location: cfg.SSOURL},
			},
		}},
	}
	// The IdP's Single Logout endpoint has to be part of the descriptor or
	// GetSLOBindingLocation resolves to "" and every LogoutRequest we build is
	// addressed nowhere. Only advertised when the admin configured slo_url —
	// an empty location would make us claim an SLO capability the IdP lacks.
	if cfg.SLOURL != "" {
		ed.IDPSSODescriptors[0].SingleLogoutServices = []crewsaml.Endpoint{
			{Binding: crewsaml.HTTPRedirectBinding, Location: cfg.SLOURL},
			{Binding: crewsaml.HTTPPostBinding, Location: cfg.SLOURL},
		}
	}
	return ed, nil
}

// buildSAMLServiceProvider constructs a crewjam/saml ServiceProvider from the
// IdP config and our SP endpoint URLs.
func buildSAMLServiceProvider(cfg *SAMLProviderConfig, spEntityID, acsURL, metadataURL, sloURL string) (*crewsaml.ServiceProvider, error) {
	idpMeta, err := buildIDPEntityDescriptor(cfg)
	if err != nil {
		return nil, err
	}

	parsedACS, err := url.Parse(acsURL)
	if err != nil {
		return nil, fmt.Errorf("invalid ACS URL: %w", err)
	}
	parsedMeta, err := url.Parse(metadataURL)
	if err != nil {
		return nil, fmt.Errorf("invalid metadata URL: %w", err)
	}
	// Our SLO endpoint is only published (and only accepted as a LogoutResponse
	// Destination) when the IdP actually has an SLO endpoint to talk to. An SP
	// whose metadata advertises a SingleLogoutService it cannot complete invites
	// IdPs to start logouts that can never finish.
	var parsedSLO url.URL
	var logoutBindings []string
	if cfg.SLOURL != "" && sloURL != "" {
		p, sloErr := url.Parse(sloURL)
		if sloErr != nil {
			return nil, fmt.Errorf("invalid SLO URL: %w", sloErr)
		}
		parsedSLO = *p
		// LogoutBindings is what makes Metadata() emit the SingleLogoutService
		// entries; without it the SP metadata an IdP imports says nothing about
		// where to send a logout, and SLO never starts.
		logoutBindings = []string{crewsaml.HTTPRedirectBinding, crewsaml.HTTPPostBinding}
	}

	nameIDFmt := crewsaml.PersistentNameIDFormat
	if cfg.NameIDFormat != "" {
		nameIDFmt = crewsaml.NameIDFormat(cfg.NameIDFormat)
	}

	sp := &crewsaml.ServiceProvider{
		EntityID:              spEntityID,
		MetadataURL:           *parsedMeta,
		AcsURL:                *parsedACS,
		SloURL:                parsedSLO,
		LogoutBindings:        logoutBindings,
		IDPMetadata:           idpMeta,
		AuthnNameIDFormat:     nameIDFmt,
		AllowIDPInitiated:     false,
		MetadataValidDuration: 24 * time.Hour,
	}
	return sp, nil
}

// samlSPURLs returns the SP entity ID, ACS URL, metadata URL and Single Logout
// URL for the given provider identifier using the configured public hostname.
func samlSPURLs(providerIdentifier string) (entityID, acsURL, metadataURL, sloURL string) {
	base := strings.TrimRight(config.AppPublicHostname, "/")
	entityID = base + "/federation/saml/metadata/" + providerIdentifier
	acsURL = base + "/federation/saml/acs/" + providerIdentifier
	metadataURL = base + "/federation/saml/metadata/" + providerIdentifier
	sloURL = base + "/federation/saml/slo/" + providerIdentifier
	return
}

// newSAMLRelayState creates a signed relay state token for the given flow params.
// requestID is the AuthnRequest ID so the ACS handler can bind the IdP Response
// to the request we issued.
func newSAMLRelayState(providerIdentifier, clientID, redirectURI, requestID string) (string, error) {
	return newSAMLRelayStateForPurpose(samlRelayPurposeSSO, providerIdentifier, clientID, redirectURI, requestID)
}

// newSAMLLogoutRelayState creates the RelayState for an SP-initiated Single
// Logout. redirectURI is the already-validated post-logout landing page and
// requestID is the LogoutRequest ID, so the SLO endpoint can tie the IdP's
// LogoutResponse back to the logout this server started.
func newSAMLLogoutRelayState(providerIdentifier, clientID, redirectURI, requestID string) (string, error) {
	return newSAMLRelayStateForPurpose(samlRelayPurposeSLO, providerIdentifier, clientID, redirectURI, requestID)
}

func newSAMLRelayStateForPurpose(purpose, providerIdentifier, clientID, redirectURI, requestID string) (string, error) {
	rs := &samlRelayState{
		ProviderIdentifier: providerIdentifier,
		ClientID:           clientID,
		RedirectURI:        redirectURI,
		RequestID:          requestID,
		Nonce:              uuid.New().String(),
		IssuedAt:           time.Now().Unix(),
		Purpose:            purpose,
	}
	return signRelayState(rs)
}

// extractSAMLClaims maps SAML assertion attributes to IdentityMetadata using
// the configured attribute mapping (or well-known fallback names).
func extractSAMLClaims(assertion *crewsaml.Assertion, attrMapping map[string]string) IdentityMetadata {
	attrs := make(map[string]string)
	for _, stmt := range assertion.AttributeStatements {
		for _, attr := range stmt.Attributes {
			if len(attr.Values) == 0 {
				continue
			}
			v := attr.Values[0].Value
			attrs[attr.Name] = v
			if attr.FriendlyName != "" {
				attrs[attr.FriendlyName] = v
			}
		}
	}

	// Apply admin-configured mapping first.
	resolve := func(targets ...string) string {
		// Check configured mapping first.
		for src, dst := range attrMapping {
			for _, t := range targets {
				if dst == t {
					if v, ok := attrs[src]; ok {
						return v
					}
				}
			}
		}
		// Fall back to well-known attribute names.
		for _, t := range targets {
			if v, ok := attrs[t]; ok {
				return v
			}
		}
		return ""
	}

	email := resolve("email",
		"urn:oid:0.9.2342.19200300.100.1.3",
		"http://schemas.xmlsoap.org/ws/2005/05/identity/claims/emailaddress",
		"http://schemas.xmlsoap.org/claims/EmailAddress",
	)
	givenName := resolve("given_name", "givenName", "gn",
		"urn:oid:2.5.4.42",
		"http://schemas.xmlsoap.org/ws/2005/05/identity/claims/givenname",
	)
	familyName := resolve("family_name", "sn", "surname",
		"urn:oid:2.5.4.4",
		"http://schemas.xmlsoap.org/ws/2005/05/identity/claims/surname",
	)
	displayName := resolve("name", "displayName", "cn",
		"urn:oid:2.16.840.1.113730.3.1.241",
		"http://schemas.xmlsoap.org/ws/2005/05/identity/claims/name",
	)
	if displayName == "" && (givenName != "" || familyName != "") {
		displayName = strings.TrimSpace(givenName + " " + familyName)
	}

	return IdentityMetadata{
		Email: email,
		// EmailVerified is intentionally NOT derived from the mere presence of an
		// email attribute. A SAML IdP asserting an email address is not proof that
		// the address is owned by the subject, and treating it as verified would
		// let the merge-with-existing-user gate in provisionUser fire on an
		// unproven email. The caller (HandleSAMLResponse) sets EmailVerified only
		// when the email's domain is on the provider's configured allow-list.
		EmailVerified: false,
		Name:          displayName,
		GivenName:     givenName,
		FamilyName:    familyName,
	}
}

// samlEmailDomainAllowed reports whether the asserted email's domain is on the
// provider's configured email_domains allow-list. Only an allow-listed domain is
// treated as proof the SAML IdP is authoritative for that email, which is the
// precondition for merging into (rather than creating separately from) an
// existing local account.
func samlEmailDomainAllowed(email string, allowed []IdentityProviderEmailDomain) bool {
	domain := emailDomain(email)
	if domain == "" || len(allowed) == 0 {
		return false
	}
	for _, d := range allowed {
		if strings.EqualFold(strings.TrimSpace(d.Domain), domain) {
			return true
		}
	}
	return false
}
