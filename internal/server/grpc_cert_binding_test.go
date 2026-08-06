package server

import (
	"context"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"encoding/base64"
	"testing"

	"github.com/maintainerd/maintainerd-auth/internal/client"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/status"
)

// boundClientService reports a fixed thumbprint for one client id.
type boundClientService struct {
	client.ClientService
	clientID   string
	thumbprint string
}

func (s *boundClientService) BoundCertThumbprint(_ context.Context, clientIdentifier string) string {
	if clientIdentifier == s.clientID {
		return s.thumbprint
	}
	return ""
}

// ctxWithPeerCert builds a context carrying a VERIFIED chain, which is what the
// binding check reads. PeerCertificates is deliberately left empty: it holds
// whatever the client sent, so a check that used it would compare against a
// certificate nobody verified came from this deployment's CA.
func ctxWithPeerCert(der []byte) context.Context {
	cert := &x509.Certificate{Raw: der}
	return peer.NewContext(context.Background(), &peer.Peer{
		AuthInfo: credentials.TLSInfo{
			State: tls.ConnectionState{VerifiedChains: [][]*x509.Certificate{{cert}}},
		},
	})
}

func thumbprintOf(der []byte) string {
	sum := sha256.Sum256(der)
	return base64.RawURLEncoding.EncodeToString(sum[:])
}

// RFC 8705 §3: a certificate-bound token is usable only by the holder of that
// certificate's private key. Without this, a control-plane token is a bearer
// token — anyone who steals it and holds ANY cert signed by this deployment's CA
// inherits the orchestrator's permissions.
func TestGRPCCertBinding(t *testing.T) {
	coreDER := []byte("core-certificate-der")
	otherDER := []byte("some-other-workload-certificate-der")

	appWithBinding := func() *Application {
		return &Application{ClientService: &boundClientService{
			clientID:   "core-client",
			thumbprint: thumbprintOf(coreDER),
		}}
	}

	t.Run("the bound certificate is accepted", func(t *testing.T) {
		err := enforceGRPCCertBinding(ctxWithPeerCert(coreDER), appWithBinding(), "core-client")
		assert.NoError(t, err)
	})

	// The escalation this exists to stop: a legitimate certificate, issued by the
	// same CA to a different workload, must not be able to wield a stolen token.
	t.Run("a different valid certificate is rejected", func(t *testing.T) {
		err := enforceGRPCCertBinding(ctxWithPeerCert(otherDER), appWithBinding(), "core-client")
		require.Error(t, err)
		assert.Equal(t, codes.Unauthenticated, status.Code(err))
	})

	// Failing open here would make the binding advisory, which is the same as not
	// having it at all.
	t.Run("no certificate at all is rejected", func(t *testing.T) {
		err := enforceGRPCCertBinding(context.Background(), appWithBinding(), "core-client")
		require.Error(t, err)
		assert.Equal(t, codes.Unauthenticated, status.Code(err))
	})

	// Opt-in per client: an existing caller that never registered a thumbprint
	// keeps working, so enabling this cannot break a deployment.
	t.Run("a client with no binding is unaffected", func(t *testing.T) {
		assert.NoError(t, enforceGRPCCertBinding(ctxWithPeerCert(otherDER), appWithBinding(), "unbound-client"))
		assert.NoError(t, enforceGRPCCertBinding(context.Background(), appWithBinding(), "unbound-client"))
	})

	t.Run("a token naming no client is unaffected", func(t *testing.T) {
		assert.NoError(t, enforceGRPCCertBinding(ctxWithPeerCert(coreDER), appWithBinding(), ""))
	})

	// An unwired application must not silently disable the check for a client
	// that IS bound; with no client service there is nothing to look up, so the
	// only clients affected are ones we cannot prove are bound.
	t.Run("no client service leaves the check inert", func(t *testing.T) {
		assert.NoError(t, enforceGRPCCertBinding(ctxWithPeerCert(coreDER), &Application{}, "core-client"))
	})
}

// The thumbprint must be computed the way RFC 8705 §3.1 defines x5t#S256:
// base64url, no padding, over the DER. A padded or hex encoding would never
// match a value an operator generated with the documented openssl recipe.
func TestGRPCPeerCertThumbprintEncoding(t *testing.T) {
	der := []byte("certificate-der-bytes")

	got, ok := grpcPeerCertThumbprint(ctxWithPeerCert(der))
	require.True(t, ok)

	sum := sha256.Sum256(der)
	assert.Equal(t, base64.RawURLEncoding.EncodeToString(sum[:]), got)
	assert.NotContains(t, got, "=", "x5t#S256 is unpadded base64url")
	assert.Len(t, got, 43, "a SHA-256 in unpadded base64url is 43 characters")
}

// Reading an unverified certificate would compare against whatever the client
// sent, defeating the point of the binding.
func TestGRPCPeerCertThumbprintIgnoresUnverifiedCertificates(t *testing.T) {
	ctx := peer.NewContext(context.Background(), &peer.Peer{
		AuthInfo: credentials.TLSInfo{
			State: tls.ConnectionState{
				PeerCertificates: []*x509.Certificate{{Raw: []byte("attacker-supplied")}},
				// No VerifiedChains: nothing established this came from our CA.
			},
		},
	})
	_, ok := grpcPeerCertThumbprint(ctx)
	assert.False(t, ok, "an unverified certificate must not satisfy the binding")
}
