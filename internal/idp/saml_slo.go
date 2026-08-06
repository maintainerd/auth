package idp

import (
	"bytes"
	"compress/flate"
	"crypto"
	"crypto/ecdsa"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/sha512"
	"crypto/x509"
	"encoding/base64"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/beevik/etree"
	dsig "github.com/russellhaering/goxmldsig"
	"github.com/russellhaering/goxmldsig/etreeutils"
)

// samlMaxMessageSize bounds an inflated SAML message. The HTTP-Redirect binding
// carries DEFLATE-compressed XML, so without a ceiling a few kilobytes of query
// string can expand into gigabytes of heap — a decompression bomb against an
// endpoint that is, by design, unauthenticated.
const samlMaxMessageSize = 5 << 20 // 5 MiB

// samlLogoutClockSkew is how far an inbound logout message's IssueInstant may
// sit outside our clock. crewjam's 90-second MaxIssueDelay is tuned for an
// assertion handed straight back by the browser; an SLO message can cross more
// hops, so a slightly wider window avoids rejecting honest logouts. It is still
// a bounded freshness check — never disabled.
const samlLogoutClockSkew = 5 * time.Minute

// samlLogoutRequestNoncePrefix namespaces the single-use marker recorded for an
// inbound IdP-initiated LogoutRequest ID. A captured LogoutRequest is otherwise
// replayable for as long as its IssueInstant stays fresh, which lets an attacker
// keep terminating a user's sessions on demand.
const samlLogoutRequestNoncePrefix = "saml:slo-request:"

// Signature algorithms accepted on a redirect-binding (detached) signature.
// SHA-1 is deliberately absent: it is collision-broken and no longer acceptable
// for a signature that authorises terminating a user's sessions. An IdP still
// pinned to rsa-sha1 must be reconfigured rather than silently trusted.
const (
	sigAlgRSASHA256   = "http://www.w3.org/2001/04/xmldsig-more#rsa-sha256"
	sigAlgRSASHA384   = "http://www.w3.org/2001/04/xmldsig-more#rsa-sha384"
	sigAlgRSASHA512   = "http://www.w3.org/2001/04/xmldsig-more#rsa-sha512"
	sigAlgECDSASHA256 = "http://www.w3.org/2001/04/xmldsig-more#ecdsa-sha256"
	sigAlgECDSASHA384 = "http://www.w3.org/2001/04/xmldsig-more#ecdsa-sha384"
	sigAlgECDSASHA512 = "http://www.w3.org/2001/04/xmldsig-more#ecdsa-sha512"
)

// samlLogoutRequestEnvelope is the subset of a SAML 2.0 LogoutRequest this
// server acts on. It is a local struct rather than crewsaml.LogoutRequest
// because that type models its Signature as an *etree.Element, which
// encoding/xml cannot populate.
type samlLogoutRequestEnvelope struct {
	XMLName      xml.Name   `xml:"urn:oasis:names:tc:SAML:2.0:protocol LogoutRequest"`
	ID           string     `xml:"ID,attr"`
	Version      string     `xml:"Version,attr"`
	IssueInstant time.Time  `xml:"IssueInstant,attr"`
	NotOnOrAfter *time.Time `xml:"NotOnOrAfter,attr"`
	Destination  string     `xml:"Destination,attr"`
	Issuer       string     `xml:"urn:oasis:names:tc:SAML:2.0:assertion Issuer"`
	NameID       string     `xml:"urn:oasis:names:tc:SAML:2.0:assertion NameID"`
	SessionIndex []string   `xml:"urn:oasis:names:tc:SAML:2.0:protocol SessionIndex"`
}

// samlLogoutResponseEnvelope is the subset of a SAML 2.0 LogoutResponse this
// server acts on.
type samlLogoutResponseEnvelope struct {
	XMLName      xml.Name  `xml:"urn:oasis:names:tc:SAML:2.0:protocol LogoutResponse"`
	ID           string    `xml:"ID,attr"`
	InResponseTo string    `xml:"InResponseTo,attr"`
	IssueInstant time.Time `xml:"IssueInstant,attr"`
	Destination  string    `xml:"Destination,attr"`
	Issuer       string    `xml:"urn:oasis:names:tc:SAML:2.0:assertion Issuer"`
	Status       struct {
		StatusCode struct {
			Value string `xml:"Value,attr"`
		} `xml:"urn:oasis:names:tc:SAML:2.0:protocol StatusCode"`
	} `xml:"urn:oasis:names:tc:SAML:2.0:protocol Status"`
}

// samlStatusSuccess is the only Status this server treats as a completed logout.
const samlStatusSuccess = "urn:oasis:names:tc:SAML:2.0:status:Success"

// samlInboundMessage is a decoded, signature-verified SAML message plus the
// RelayState it travelled with.
type samlInboundMessage struct {
	XML        []byte
	RelayState string
	IsRequest  bool
}

// readSAMLLogoutMessage pulls the SAML message out of an SLO request, decodes
// it, and verifies its signature against the provider's configured certificate.
//
// A signature is REQUIRED on both directions. The SLO endpoint is
// unauthenticated by protocol design, so the XML signature is the ONLY proof
// that the IdP — and not a passer-by with the URL — asked for these sessions to
// end. An unsigned message is rejected outright rather than trusted because it
// happened to arrive at the right endpoint.
func readSAMLLogoutMessage(r *http.Request, cert *x509.Certificate) (*samlInboundMessage, error) {
	if cert == nil {
		return nil, fmt.Errorf("provider has no signing certificate configured")
	}

	// HTTP-Redirect binding: the message rides in the query string and, when
	// signed, carries a DETACHED signature over the raw query parameters
	// (SAML 2.0 Bindings §3.4.4.1) rather than an enveloped XML signature.
	if raw := r.URL.Query().Get("SAMLRequest"); raw != "" {
		return readSAMLRedirectMessage(r, cert, "SAMLRequest", raw, true)
	}
	if raw := r.URL.Query().Get("SAMLResponse"); raw != "" {
		return readSAMLRedirectMessage(r, cert, "SAMLResponse", raw, false)
	}

	if err := r.ParseForm(); err != nil {
		return nil, fmt.Errorf("failed to parse form")
	}
	if raw := r.PostForm.Get("SAMLRequest"); raw != "" {
		return readSAMLPostMessage(cert, raw, r.PostForm.Get("RelayState"), true)
	}
	if raw := r.PostForm.Get("SAMLResponse"); raw != "" {
		return readSAMLPostMessage(cert, raw, r.PostForm.Get("RelayState"), false)
	}
	return nil, fmt.Errorf("no SAMLRequest or SAMLResponse present")
}

func readSAMLRedirectMessage(r *http.Request, cert *x509.Certificate, param, raw string, isRequest bool) (*samlInboundMessage, error) {
	decoded, err := base64.StdEncoding.DecodeString(raw)
	if err != nil {
		return nil, fmt.Errorf("message is not valid base64")
	}
	inflated, err := inflateSAMLMessage(decoded)
	if err != nil {
		return nil, err
	}
	// The redirect binding normally carries a detached query signature, but some
	// IdPs sign the XML itself and deflate that. Either proof is accepted; the
	// absence of both is not — verifySAMLEnvelopedSignature rejects an unsigned
	// document, so this fallback can only ever admit a verified message.
	if _, signed := rawQueryValue(r.URL.RawQuery, "Signature"); signed {
		if err := verifySAMLRedirectSignature(r.URL.RawQuery, param, cert); err != nil {
			return nil, err
		}
	} else if err := verifySAMLEnvelopedSignature(inflated, cert); err != nil {
		return nil, err
	}
	return &samlInboundMessage{
		XML:        inflated,
		RelayState: r.URL.Query().Get("RelayState"),
		IsRequest:  isRequest,
	}, nil
}

func readSAMLPostMessage(cert *x509.Certificate, raw, relayState string, isRequest bool) (*samlInboundMessage, error) {
	decoded, err := base64.StdEncoding.DecodeString(raw)
	if err != nil {
		return nil, fmt.Errorf("message is not valid base64")
	}
	if len(decoded) > samlMaxMessageSize {
		return nil, fmt.Errorf("message exceeds the maximum accepted size")
	}
	if err := verifySAMLEnvelopedSignature(decoded, cert); err != nil {
		return nil, err
	}
	return &samlInboundMessage{XML: decoded, RelayState: relayState, IsRequest: isRequest}, nil
}

// inflateSAMLMessage DEFLATE-decompresses a redirect-binding message under a
// hard output ceiling (see samlMaxMessageSize).
func inflateSAMLMessage(compressed []byte) ([]byte, error) {
	reader := flate.NewReader(bytes.NewReader(compressed))
	defer func() { _ = reader.Close() }()

	inflated, err := io.ReadAll(io.LimitReader(reader, samlMaxMessageSize+1))
	if err != nil {
		return nil, fmt.Errorf("message could not be inflated")
	}
	if len(inflated) > samlMaxMessageSize {
		return nil, fmt.Errorf("message exceeds the maximum accepted size")
	}
	return inflated, nil
}

// verifySAMLRedirectSignature validates the detached signature carried in the
// query string of a redirect-binding message (SAML 2.0 Bindings §3.4.4.1).
//
// The signed string is the RAW, still-percent-encoded parameters in the fixed
// order SAMLRequest|SAMLResponse, RelayState (when present), SigAlg. Re-encoding
// the parsed values would produce a different byte string than the IdP signed
// (escaping is not canonical), so the segments are lifted straight out of
// RawQuery.
func verifySAMLRedirectSignature(rawQuery, messageParam string, cert *x509.Certificate) error {
	sigRaw, ok := rawQueryValue(rawQuery, "Signature")
	if !ok || sigRaw == "" {
		return fmt.Errorf("logout message is not signed")
	}
	sigEncoded, err := url.QueryUnescape(sigRaw)
	if err != nil {
		return fmt.Errorf("signature is not valid")
	}
	signature, err := base64.StdEncoding.DecodeString(sigEncoded)
	if err != nil {
		return fmt.Errorf("signature is not valid base64")
	}

	sigAlgRaw, ok := rawQueryValue(rawQuery, "SigAlg")
	if !ok {
		return fmt.Errorf("logout message is missing SigAlg")
	}
	sigAlg, err := url.QueryUnescape(sigAlgRaw)
	if err != nil {
		return fmt.Errorf("SigAlg is not valid")
	}

	messageRaw, ok := rawQueryValue(rawQuery, messageParam)
	if !ok {
		return fmt.Errorf("logout message is missing %s", messageParam)
	}

	signed := messageParam + "=" + messageRaw
	if relayState, present := rawQueryValue(rawQuery, "RelayState"); present {
		signed += "&RelayState=" + relayState
	}
	signed += "&SigAlg=" + sigAlgRaw

	return verifyDetachedSignature([]byte(signed), signature, sigAlg, cert)
}

// rawQueryValue returns a query parameter's value exactly as it appears in the
// raw query string, without unescaping it.
func rawQueryValue(rawQuery, key string) (string, bool) {
	for _, segment := range strings.Split(rawQuery, "&") {
		name, value, found := strings.Cut(segment, "=")
		if !found {
			name = segment
		}
		if name == key {
			return value, true
		}
	}
	return "", false
}

func verifyDetachedSignature(signed, signature []byte, sigAlg string, cert *x509.Certificate) error {
	var hashed []byte
	var hash crypto.Hash
	switch sigAlg {
	case sigAlgRSASHA256, sigAlgECDSASHA256:
		sum := sha256.Sum256(signed)
		hashed, hash = sum[:], crypto.SHA256
	case sigAlgRSASHA384, sigAlgECDSASHA384:
		sum := sha512.Sum384(signed)
		hashed, hash = sum[:], crypto.SHA384
	case sigAlgRSASHA512, sigAlgECDSASHA512:
		sum := sha512.Sum512(signed)
		hashed, hash = sum[:], crypto.SHA512
	default:
		// Unknown or deliberately-unsupported (SHA-1) algorithm. Fail closed:
		// an algorithm we cannot check is not an algorithm we can accept.
		return fmt.Errorf("unsupported signature algorithm")
	}

	switch pub := cert.PublicKey.(type) {
	case *rsa.PublicKey:
		if err := rsa.VerifyPKCS1v15(pub, hash, hashed, signature); err != nil {
			return fmt.Errorf("signature verification failed")
		}
		return nil
	case *ecdsa.PublicKey:
		if !ecdsa.VerifyASN1(pub, hashed, signature) {
			return fmt.Errorf("signature verification failed")
		}
		return nil
	default:
		return fmt.Errorf("unsupported signing key type")
	}
}

// verifySAMLEnvelopedSignature validates an enveloped XML-DSIG signature on a
// POST-binding message against the provider's configured certificate.
//
// KeyInfo is stripped before validation so the ONLY trust anchor is the
// certificate the tenant admin configured. Left in place, a message could
// nominate its own certificate and — while goxmldsig would still require that
// certificate to match a root — the intent is to make the configured cert the
// single source of truth rather than depend on that secondary check.
func verifySAMLEnvelopedSignature(raw []byte, cert *x509.Certificate) error {
	doc := etree.NewDocument()
	if err := doc.ReadFromBytes(raw); err != nil {
		return fmt.Errorf("message is not valid XML")
	}
	el := doc.Root()
	if el == nil {
		return fmt.Errorf("message is empty")
	}
	sigEl := el.FindElement("./Signature")
	if sigEl == nil {
		return fmt.Errorf("logout message is not signed")
	}
	if keyInfo := sigEl.FindElement("KeyInfo"); keyInfo != nil {
		sigEl.RemoveChild(keyInfo)
	}

	store := dsig.MemoryX509CertificateStore{Roots: []*x509.Certificate{cert}}
	validationContext := dsig.NewDefaultValidationContext(&store)
	validationContext.IdAttribute = "ID"

	nsCtx, err := etreeutils.NSBuildParentContext(el)
	if err != nil {
		return fmt.Errorf("signature verification failed")
	}
	nsCtx, err = nsCtx.SubContext(el)
	if err != nil {
		return fmt.Errorf("signature verification failed")
	}
	detached, err := etreeutils.NSDetatch(nsCtx, el)
	if err != nil {
		return fmt.Errorf("signature verification failed")
	}
	if _, err := validationContext.Validate(detached); err != nil {
		return fmt.Errorf("signature verification failed")
	}
	return nil
}

// parseSAMLLogoutRequest decodes an inbound LogoutRequest and enforces the
// envelope checks that bind it to THIS service provider and to now: the issuer
// must be the configured IdP, the destination (when the IdP set one) must be our
// own SLO endpoint, and the message must be fresh. Without the destination check
// a LogoutRequest addressed to a different SP is replayable here.
func parseSAMLLogoutRequest(raw []byte, idpEntityID, sloURL string) (*samlLogoutRequestEnvelope, error) {
	var req samlLogoutRequestEnvelope
	if err := xml.Unmarshal(raw, &req); err != nil {
		return nil, fmt.Errorf("logout request is not valid SAML")
	}
	if req.ID == "" {
		return nil, fmt.Errorf("logout request has no ID")
	}
	if strings.TrimSpace(req.Issuer) != idpEntityID {
		return nil, fmt.Errorf("logout request issuer does not match the identity provider")
	}
	if req.Destination != "" && req.Destination != sloURL {
		return nil, fmt.Errorf("logout request is addressed to another service provider")
	}
	if err := checkSAMLLogoutFreshness(req.IssueInstant, req.NotOnOrAfter); err != nil {
		return nil, err
	}
	if strings.TrimSpace(req.NameID) == "" {
		return nil, fmt.Errorf("logout request has no NameID")
	}
	return &req, nil
}

// parseSAMLLogoutResponse decodes the IdP's answer to a LogoutRequest we sent
// and enforces the same envelope binding plus a Success status.
func parseSAMLLogoutResponse(raw []byte, idpEntityID, sloURL, requestID string) (*samlLogoutResponseEnvelope, error) {
	var res samlLogoutResponseEnvelope
	if err := xml.Unmarshal(raw, &res); err != nil {
		return nil, fmt.Errorf("logout response is not valid SAML")
	}
	if strings.TrimSpace(res.Issuer) != idpEntityID {
		return nil, fmt.Errorf("logout response issuer does not match the identity provider")
	}
	if res.Destination != "" && res.Destination != sloURL {
		return nil, fmt.Errorf("logout response is addressed to another service provider")
	}
	// InResponseTo ties the answer to the LogoutRequest carried in our signed
	// RelayState. An IdP that omits it leaves nothing to bind against, so a
	// mismatch and an absence are both refused.
	if res.InResponseTo != requestID {
		return nil, fmt.Errorf("logout response does not answer this logout request")
	}
	if err := checkSAMLLogoutFreshness(res.IssueInstant, nil); err != nil {
		return nil, err
	}
	if res.Status.StatusCode.Value != samlStatusSuccess {
		return nil, fmt.Errorf("identity provider reported logout status %q", res.Status.StatusCode.Value)
	}
	return &res, nil
}

func checkSAMLLogoutFreshness(issueInstant time.Time, notOnOrAfter *time.Time) error {
	if issueInstant.IsZero() {
		return fmt.Errorf("logout message has no IssueInstant")
	}
	now := time.Now()
	if issueInstant.After(now.Add(samlLogoutClockSkew)) || now.Sub(issueInstant) > samlLogoutClockSkew {
		return fmt.Errorf("logout message is not fresh")
	}
	if notOnOrAfter != nil && !now.Before(*notOnOrAfter) {
		return fmt.Errorf("logout message has expired")
	}
	return nil
}
