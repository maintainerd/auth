package cookie

import (
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

// A Domain-scoped cookie can ONLY be deleted by a Set-Cookie that carries the
// same Domain. If logout emitted a host-only delete, the shared cookie would
// survive and the user would stay signed in on every other subdomain.
func TestClearAuthCookies_DeleteCarriesSharedDomain(t *testing.T) {
	withCookieDomain(t, "auth.example.com")
	w := httptest.NewRecorder()

	ClearAuthCookies(w)

	var secureAccessDelete string
	for _, h := range w.Result().Header["Set-Cookie"] {
		if strings.HasPrefix(h, "__Secure-access_token=") {
			secureAccessDelete = h
		}
	}
	assert.NotEmpty(t, secureAccessDelete, "the shared-domain cookie must be cleared")
	assert.Contains(t, secureAccessDelete, "Domain=auth.example.com")
	assert.Contains(t, secureAccessDelete, "Max-Age=0")
}
