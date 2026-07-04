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
	TenantID           int64  `json:"tid"`
	Nonce              string `json:"n"`
	IssuedAt           int64  `json:"iat"`
}

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
	return ed, nil
}

// buildSAMLServiceProvider constructs a crewjam/saml ServiceProvider from the
// IdP config and our SP endpoint URLs.
func buildSAMLServiceProvider(cfg *SAMLProviderConfig, spEntityID, acsURL, metadataURL string) (*crewsaml.ServiceProvider, error) {
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

	nameIDFmt := crewsaml.PersistentNameIDFormat
	if cfg.NameIDFormat != "" {
		nameIDFmt = crewsaml.NameIDFormat(cfg.NameIDFormat)
	}

	sp := &crewsaml.ServiceProvider{
		EntityID:              spEntityID,
		MetadataURL:           *parsedMeta,
		AcsURL:                *parsedACS,
		IDPMetadata:           idpMeta,
		AuthnNameIDFormat:     nameIDFmt,
		AllowIDPInitiated:     false,
		MetadataValidDuration: 24 * time.Hour,
	}
	return sp, nil
}

// samlSPURLs returns the SP entity ID, ACS URL, and metadata URL for the
// given provider identifier using the configured public hostname.
func samlSPURLs(providerIdentifier string) (entityID, acsURL, metadataURL string) {
	base := strings.TrimRight(config.AppPublicHostname, "/")
	entityID = base + "/federation/saml/metadata/" + providerIdentifier
	acsURL = base + "/federation/saml/acs/" + providerIdentifier
	metadataURL = base + "/federation/saml/metadata/" + providerIdentifier
	return
}

// newSAMLRelayState creates a signed relay state token for the given flow params.
func newSAMLRelayState(providerIdentifier, clientID, redirectURI string, tenantID int64) (string, error) {
	rs := &samlRelayState{
		ProviderIdentifier: providerIdentifier,
		ClientID:           clientID,
		RedirectURI:        redirectURI,
		TenantID:           tenantID,
		Nonce:              uuid.New().String(),
		IssuedAt:           time.Now().Unix(),
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
		Email:         email,
		EmailVerified: email != "",
		Name:          displayName,
		GivenName:     givenName,
		FamilyName:    familyName,
	}
}
