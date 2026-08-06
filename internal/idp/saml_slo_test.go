package idp

import (
	"bytes"
	"compress/flate"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/base64"
	"encoding/pem"
	"math/big"
	"net/url"
	"testing"
	"time"

	"github.com/beevik/etree"
	crewsaml "github.com/crewjam/saml"
	dsig "github.com/russellhaering/goxmldsig"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// samlTestIdP is a throwaway signing identity standing in for a remote SAML IdP.
type samlTestIdP struct {
	key     *rsa.PrivateKey
	cert    *x509.Certificate
	certDER []byte
	certPEM string
}

func newSAMLTestIdP(t *testing.T) *samlTestIdP {
	t.Helper()

	key, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	template := &x509.Certificate{
		SerialNumber: big.NewInt(time.Now().UnixNano()),
		Subject:      pkix.Name{CommonName: "saml-test-idp"},
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().Add(24 * time.Hour),
		KeyUsage:     x509.KeyUsageDigitalSignature,
	}
	der, err := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
	require.NoError(t, err)
	cert, err := x509.ParseCertificate(der)
	require.NoError(t, err)

	return &samlTestIdP{
		key:     key,
		cert:    cert,
		certDER: der,
		certPEM: string(pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})),
	}
}

// signEnveloped returns el with an enveloped XML-DSIG signature, serialized.
func (i *samlTestIdP) signEnveloped(t *testing.T, el *etree.Element) []byte {
	t.Helper()

	keyStore := dsig.TLSCertKeyStore(tls.Certificate{PrivateKey: i.key, Certificate: [][]byte{i.certDER}})
	ctx := dsig.NewDefaultSigningContext(keyStore)
	signed, err := ctx.SignEnveloped(el)
	require.NoError(t, err)

	doc := etree.NewDocument()
	doc.SetRoot(signed)
	out, err := doc.WriteToBytes()
	require.NoError(t, err)
	return out
}

// signRedirectQuery builds the signed query string of an HTTP-Redirect binding
// message exactly as SAML 2.0 Bindings §3.4.4.1 prescribes.
func (i *samlTestIdP) signRedirectQuery(t *testing.T, param, xmlBody, relayState string) string {
	t.Helper()

	var compressed bytes.Buffer
	w, err := flate.NewWriter(&compressed, 9)
	require.NoError(t, err)
	_, err = w.Write([]byte(xmlBody))
	require.NoError(t, err)
	require.NoError(t, w.Close())

	encoded := url.QueryEscape(base64.StdEncoding.EncodeToString(compressed.Bytes()))
	query := param + "=" + encoded
	if relayState != "" {
		query += "&RelayState=" + url.QueryEscape(relayState)
	}
	query += "&SigAlg=" + url.QueryEscape(sigAlgRSASHA256)

	digest := sha256.Sum256([]byte(query))
	sig, err := rsa.SignPKCS1v15(rand.Reader, i.key, crypto.SHA256, digest[:])
	require.NoError(t, err)

	return query + "&Signature=" + url.QueryEscape(base64.StdEncoding.EncodeToString(sig))
}

func samlTestLogoutRequestXML(id, issuer, destination, nameID string, issueInstant time.Time) *etree.Element {
	req := crewsaml.LogoutRequest{
		ID:           id,
		Version:      "2.0",
		IssueInstant: issueInstant,
		Destination:  destination,
		Issuer: &crewsaml.Issuer{
			Format: "urn:oasis:names:tc:SAML:2.0:nameid-format:entity",
			Value:  issuer,
		},
		NameID: &crewsaml.NameID{
			Format: string(crewsaml.PersistentNameIDFormat),
			Value:  nameID,
		},
	}
	return req.Element()
}

func samlTestLogoutResponseXML(id, issuer, destination, inResponseTo, status string, issueInstant time.Time) *etree.Element {
	res := crewsaml.LogoutResponse{
		ID:           id,
		InResponseTo: inResponseTo,
		Version:      "2.0",
		IssueInstant: issueInstant,
		Destination:  destination,
		Issuer: &crewsaml.Issuer{
			Format: "urn:oasis:names:tc:SAML:2.0:nameid-format:entity",
			Value:  issuer,
		},
		Status: crewsaml.Status{StatusCode: crewsaml.StatusCode{Value: status}},
	}
	return res.Element()
}

func TestVerifySAMLRedirectSignature(t *testing.T) {
	idp := newSAMLTestIdP(t)
	body := `<samlp:LogoutRequest xmlns:samlp="urn:oasis:names:tc:SAML:2.0:protocol" ID="id-1"/>`

	t.Run("valid detached signature is accepted", func(t *testing.T) {
		query := idp.signRedirectQuery(t, "SAMLRequest", body, "relay-1")
		require.NoError(t, verifySAMLRedirectSignature(query, "SAMLRequest", idp.cert))
	})

	t.Run("tampered message is rejected", func(t *testing.T) {
		query := idp.signRedirectQuery(t, "SAMLRequest", body, "relay-1")
		other := idp.signRedirectQuery(t, "SAMLRequest", `<samlp:LogoutRequest xmlns:samlp="urn:oasis:names:tc:SAML:2.0:protocol" ID="id-2"/>`, "relay-1")
		// Swap in a different SAMLRequest while keeping the original signature.
		tampered := "SAMLRequest=" + mustRawValue(t, other, "SAMLRequest") +
			"&RelayState=" + mustRawValue(t, query, "RelayState") +
			"&SigAlg=" + mustRawValue(t, query, "SigAlg") +
			"&Signature=" + mustRawValue(t, query, "Signature")
		require.Error(t, verifySAMLRedirectSignature(tampered, "SAMLRequest", idp.cert))
	})

	t.Run("tampered relay state is rejected", func(t *testing.T) {
		query := idp.signRedirectQuery(t, "SAMLRequest", body, "relay-1")
		tampered := "SAMLRequest=" + mustRawValue(t, query, "SAMLRequest") +
			"&RelayState=relay-2" +
			"&SigAlg=" + mustRawValue(t, query, "SigAlg") +
			"&Signature=" + mustRawValue(t, query, "Signature")
		require.Error(t, verifySAMLRedirectSignature(tampered, "SAMLRequest", idp.cert))
	})

	t.Run("unsigned message is rejected", func(t *testing.T) {
		unsigned := "SAMLRequest=" + url.QueryEscape("anything")
		require.Error(t, verifySAMLRedirectSignature(unsigned, "SAMLRequest", idp.cert))
	})

	t.Run("sha-1 is refused", func(t *testing.T) {
		err := verifyDetachedSignature([]byte("x"), []byte("y"), "http://www.w3.org/2000/09/xmldsig#rsa-sha1", idp.cert)
		require.Error(t, err, "a collision-broken algorithm must not authorise ending sessions")
	})

	t.Run("signature from another key is rejected", func(t *testing.T) {
		other := newSAMLTestIdP(t)
		query := other.signRedirectQuery(t, "SAMLRequest", body, "")
		require.Error(t, verifySAMLRedirectSignature(query, "SAMLRequest", idp.cert))
	})
}

func mustRawValue(t *testing.T, rawQuery, key string) string {
	t.Helper()
	value, ok := rawQueryValue(rawQuery, key)
	require.True(t, ok, "query is missing %s", key)
	return value
}

func TestVerifySAMLEnvelopedSignature(t *testing.T) {
	idp := newSAMLTestIdP(t)
	el := samlTestLogoutRequestXML("id-1", "https://idp.test/entity", "https://auth.test/federation/saml/slo/prov", "nameid-1", time.Now())

	t.Run("valid signature is accepted", func(t *testing.T) {
		signed := idp.signEnveloped(t, el.Copy())
		require.NoError(t, verifySAMLEnvelopedSignature(signed, idp.cert))
	})

	t.Run("unsigned message is rejected", func(t *testing.T) {
		doc := etree.NewDocument()
		doc.SetRoot(el.Copy())
		raw, err := doc.WriteToBytes()
		require.NoError(t, err)
		require.Error(t, verifySAMLEnvelopedSignature(raw, idp.cert))
	})

	t.Run("signature from another key is rejected", func(t *testing.T) {
		other := newSAMLTestIdP(t)
		signed := other.signEnveloped(t, el.Copy())
		require.Error(t, verifySAMLEnvelopedSignature(signed, idp.cert),
			"only the provider's configured certificate may authorise a logout")
	})
}

func TestParseSAMLLogoutRequest(t *testing.T) {
	const issuer = "https://idp.test/entity"
	const sloURL = "https://auth.test/federation/saml/slo/prov"

	marshal := func(el *etree.Element) []byte {
		doc := etree.NewDocument()
		doc.SetRoot(el)
		out, err := doc.WriteToBytes()
		require.NoError(t, err)
		return out
	}

	t.Run("well-formed request parses", func(t *testing.T) {
		raw := marshal(samlTestLogoutRequestXML("id-1", issuer, sloURL, "nameid-1", time.Now()))
		req, err := parseSAMLLogoutRequest(raw, issuer, sloURL)
		require.NoError(t, err)
		assert.Equal(t, "nameid-1", req.NameID)
		assert.Equal(t, "id-1", req.ID)
	})

	t.Run("issuer mismatch is rejected", func(t *testing.T) {
		raw := marshal(samlTestLogoutRequestXML("id-1", "https://evil.test/entity", sloURL, "nameid-1", time.Now()))
		_, err := parseSAMLLogoutRequest(raw, issuer, sloURL)
		require.Error(t, err)
	})

	t.Run("request addressed to another SP is rejected", func(t *testing.T) {
		raw := marshal(samlTestLogoutRequestXML("id-1", issuer, "https://other.test/slo", "nameid-1", time.Now()))
		_, err := parseSAMLLogoutRequest(raw, issuer, sloURL)
		require.Error(t, err, "a LogoutRequest minted for another SP must not be replayable here")
	})

	t.Run("stale request is rejected", func(t *testing.T) {
		raw := marshal(samlTestLogoutRequestXML("id-1", issuer, sloURL, "nameid-1", time.Now().Add(-time.Hour)))
		_, err := parseSAMLLogoutRequest(raw, issuer, sloURL)
		require.Error(t, err)
	})

	t.Run("missing NameID is rejected", func(t *testing.T) {
		raw := marshal(samlTestLogoutRequestXML("id-1", issuer, sloURL, "", time.Now()))
		_, err := parseSAMLLogoutRequest(raw, issuer, sloURL)
		require.Error(t, err)
	})
}

func TestParseSAMLLogoutResponse(t *testing.T) {
	const issuer = "https://idp.test/entity"
	const sloURL = "https://auth.test/federation/saml/slo/prov"

	marshal := func(el *etree.Element) []byte {
		doc := etree.NewDocument()
		doc.SetRoot(el)
		out, err := doc.WriteToBytes()
		require.NoError(t, err)
		return out
	}

	t.Run("successful response parses", func(t *testing.T) {
		raw := marshal(samlTestLogoutResponseXML("res-1", issuer, sloURL, "req-1", samlStatusSuccess, time.Now()))
		res, err := parseSAMLLogoutResponse(raw, issuer, sloURL, "req-1")
		require.NoError(t, err)
		assert.Equal(t, "req-1", res.InResponseTo)
	})

	t.Run("response for another request is rejected", func(t *testing.T) {
		raw := marshal(samlTestLogoutResponseXML("res-1", issuer, sloURL, "req-other", samlStatusSuccess, time.Now()))
		_, err := parseSAMLLogoutResponse(raw, issuer, sloURL, "req-1")
		require.Error(t, err)
	})

	t.Run("non-success status is rejected", func(t *testing.T) {
		raw := marshal(samlTestLogoutResponseXML("res-1", issuer, sloURL, "req-1", "urn:oasis:names:tc:SAML:2.0:status:Requester", time.Now()))
		_, err := parseSAMLLogoutResponse(raw, issuer, sloURL, "req-1")
		require.Error(t, err)
	})
}

func TestInflateSAMLMessageRejectsOversizedPayload(t *testing.T) {
	var compressed bytes.Buffer
	w, err := flate.NewWriter(&compressed, 9)
	require.NoError(t, err)
	// Highly compressible payload: a few KiB on the wire, far more once inflated.
	_, err = w.Write(bytes.Repeat([]byte("A"), samlMaxMessageSize+1024))
	require.NoError(t, err)
	require.NoError(t, w.Close())

	_, err = inflateSAMLMessage(compressed.Bytes())
	require.Error(t, err, "a decompression bomb must be refused, not inflated")
}

// A RelayState is only valid for the exchange it was minted for. Both exchanges
// are signed with the same key and carry the same fields, so without the purpose
// stamp a live SSO RelayState could be presented at the SLO endpoint.
func TestRelayStatePurposeIsEnforced(t *testing.T) {
	ssoToken, err := newSAMLRelayState("prov", "client", "https://app.test/cb", 1, "req-1")
	require.NoError(t, err)
	sloToken, err := newSAMLLogoutRelayState("prov", "client", "https://app.test/bye", 1, "req-2")
	require.NoError(t, err)

	t.Run("sso relay state is refused at the SLO endpoint", func(t *testing.T) {
		_, err := verifyRelayStateForPurpose(ssoToken, samlRelayPurposeSLO)
		require.Error(t, err)
	})

	t.Run("slo relay state is refused at the ACS endpoint", func(t *testing.T) {
		_, err := verifyRelayStateForPurpose(sloToken, samlRelayPurposeSSO)
		require.Error(t, err)
	})

	t.Run("each is accepted for its own purpose", func(t *testing.T) {
		sso, err := verifyRelayStateForPurpose(ssoToken, samlRelayPurposeSSO)
		require.NoError(t, err)
		assert.Equal(t, "req-1", sso.RequestID)

		slo, err := verifyRelayStateForPurpose(sloToken, samlRelayPurposeSLO)
		require.NoError(t, err)
		assert.Equal(t, "https://app.test/bye", slo.RedirectURI)
	})
}
