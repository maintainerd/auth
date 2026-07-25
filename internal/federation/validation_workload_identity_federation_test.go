package federation

import (
	"fmt"
	"strings"
	"testing"

	"github.com/maintainerd/maintainerd-auth/internal/platform/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func baseCreateDTO() WorkloadIdentityFederationCreateRequestDTO {
	return WorkloadIdentityFederationCreateRequestDTO{
		ClientUUID:     testClientUUID.String(),
		Name:           "github-actions",
		IssuerURL:      "https://token.actions.githubusercontent.com",
		Audience:       "https://api.maintainerd.local",
		SubjectPattern: "repo:org/repo:*",
		AllowedScopes:  []string{"deploy:write"},
	}
}

func TestWIFCreateDTO_Validate(t *testing.T) {
	t.Run("valid", func(t *testing.T) {
		dto := baseCreateDTO()
		assert.NoError(t, dto.Validate())
	})

	t.Run("client_uuid required", func(t *testing.T) {
		dto := baseCreateDTO()
		dto.ClientUUID = ""
		assert.Error(t, dto.Validate())
	})

	t.Run("client_uuid must be a UUID", func(t *testing.T) {
		dto := baseCreateDTO()
		dto.ClientUUID = "not-a-uuid"
		assert.Error(t, dto.Validate())
	})

	t.Run("name required", func(t *testing.T) {
		dto := baseCreateDTO()
		dto.Name = ""
		assert.Error(t, dto.Validate())
	})

	t.Run("name too long", func(t *testing.T) {
		dto := baseCreateDTO()
		dto.Name = strings.Repeat("a", 101)
		assert.Error(t, dto.Validate())
	})

	t.Run("issuer_url required", func(t *testing.T) {
		dto := baseCreateDTO()
		dto.IssuerURL = ""
		assert.Error(t, dto.Validate())
	})

	t.Run("issuer_url must be https", func(t *testing.T) {
		dto := baseCreateDTO()
		dto.IssuerURL = "http://insecure.example.com"
		assert.Error(t, dto.Validate())
	})

	t.Run("audience required", func(t *testing.T) {
		dto := baseCreateDTO()
		dto.Audience = ""
		assert.Error(t, dto.Validate())
	})

	t.Run("subject_pattern required", func(t *testing.T) {
		dto := baseCreateDTO()
		dto.SubjectPattern = ""
		assert.Error(t, dto.Validate())
	})

	t.Run("empty scope rejected", func(t *testing.T) {
		dto := baseCreateDTO()
		dto.AllowedScopes = []string{""}
		assert.Error(t, dto.Validate())
	})
}

func TestWIFUpdateDTO_Validate(t *testing.T) {
	valid := WorkloadIdentityFederationUpdateRequestDTO{
		Name:           "github-actions",
		IssuerURL:      "https://token.actions.githubusercontent.com",
		Audience:       "https://api.maintainerd.local",
		SubjectPattern: "repo:org/repo:*",
		AllowedScopes:  []string{"deploy:write"},
	}

	t.Run("valid", func(t *testing.T) {
		assert.NoError(t, valid.Validate())
	})

	t.Run("name required", func(t *testing.T) {
		dto := valid
		dto.Name = ""
		assert.Error(t, dto.Validate())
	})

	t.Run("issuer_url must be https", func(t *testing.T) {
		dto := valid
		dto.IssuerURL = "ftp://example.com"
		assert.Error(t, dto.Validate())
	})

	t.Run("subject_pattern required", func(t *testing.T) {
		dto := valid
		dto.SubjectPattern = ""
		assert.Error(t, dto.Validate())
	})
}

func TestMatchSubjectPattern(t *testing.T) {
	cases := []struct {
		pattern string
		subject string
		want    bool
	}{
		{"repo:org/repo:*", "repo:org/repo:ref:refs/heads/main", true},
		{"repo:org/repo:*", "repo:other/repo:ref:refs/heads/main", false},
		{"system:serviceaccount:prod:deploy-bot", "system:serviceaccount:prod:deploy-bot", true},
		{"system:serviceaccount:prod:*", "system:serviceaccount:prod:deploy-bot", true},
		{"system:serviceaccount:prod:*", "system:serviceaccount:staging:deploy-bot", false},
		{"", "anything", false},
		{"*", "anything", true},
	}
	for _, c := range cases {
		assert.Equalf(t, c.want, matchSubjectPattern(c.pattern, c.subject), "pattern=%q subject=%q", c.pattern, c.subject)
	}
}

func TestIntersectScopes(t *testing.T) {
	t.Run("empty requested returns all allowed", func(t *testing.T) {
		got, ok := intersectScopes("", []string{"a", "b"})
		assert.True(t, ok)
		assert.Equal(t, []string{"a", "b"}, got)
	})

	t.Run("subset granted", func(t *testing.T) {
		got, ok := intersectScopes("a", []string{"a", "b"})
		assert.True(t, ok)
		assert.Equal(t, []string{"a"}, got)
	})

	t.Run("requested scope outside allow-list rejected", func(t *testing.T) {
		_, ok := intersectScopes("c", []string{"a", "b"})
		assert.False(t, ok)
	})
}

func TestUnverifiedIssuer(t *testing.T) {
	t.Run("non-jwt returns false", func(t *testing.T) {
		_, ok := unverifiedIssuer("not-a-jwt")
		assert.False(t, ok)
	})

	t.Run("extracts iss", func(t *testing.T) {
		// header.payload.signature where payload = {"iss":"https://issuer.example"}
		// base64url("{\"iss\":\"https://issuer.example\"}")
		token := "eyJhbGciOiJub25lIn0." + base64URLPayload(`{"iss":"https://issuer.example"}`) + ".sig"
		iss, ok := unverifiedIssuer(token)
		assert.True(t, ok)
		assert.Equal(t, "https://issuer.example", iss)
	})
}

func TestAudienceMatches(t *testing.T) {
	t.Run("string aud", func(t *testing.T) {
		assert.True(t, audienceMatches(map[string]interface{}{"aud": "x"}, "x"))
		assert.False(t, audienceMatches(map[string]interface{}{"aud": "x"}, "y"))
	})
	t.Run("array aud", func(t *testing.T) {
		claims := map[string]interface{}{"aud": []interface{}{"a", "b"}}
		assert.True(t, audienceMatches(claims, "b"))
		assert.False(t, audienceMatches(claims, "c"))
	})
	t.Run("empty audience never matches", func(t *testing.T) {
		assert.False(t, audienceMatches(map[string]interface{}{"aud": ""}, ""))
	})
}

// subject_pattern is the ONLY thing separating one workload from another on a shared
// public issuer: anyone can get a token from token.actions.githubusercontent.com, and
// the audience is chosen by the requesting workflow. An unanchored pattern therefore
// lets any workload on that issuer mint this tenant's token, unauthenticated.
func TestSubjectPatternBreadth(t *testing.T) {
	t.Run("rejects a bare wildcard", func(t *testing.T) {
		dto := baseCreateDTO()
		dto.SubjectPattern = "*"
		err := dto.Validate()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "must not start with a wildcard")
	})

	t.Run("rejects a leading wildcard", func(t *testing.T) {
		for _, pattern := range []string{"*:ref:refs/heads/main", "*", "?repo:org/x", "**"} {
			dto := baseCreateDTO()
			dto.SubjectPattern = pattern
			assert.Error(t, dto.Validate(), pattern)
		}
	})

	t.Run("rejects a pattern with too little literal text", func(t *testing.T) {
		dto := baseCreateDTO()
		dto.SubjectPattern = "rep*"
		err := dto.Validate()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "too broad")
	})

	t.Run("accepts an anchored pattern", func(t *testing.T) {
		for _, pattern := range []string{
			"repo:my-org/my-repo:*",
			"system:serviceaccount:production:*",
			"repo:my-org/*",
		} {
			dto := baseCreateDTO()
			dto.SubjectPattern = pattern
			assert.NoError(t, dto.Validate(), pattern)
		}
	})

	// An exact pattern is always safe regardless of length.
	t.Run("accepts a short literal pattern", func(t *testing.T) {
		dto := baseCreateDTO()
		dto.SubjectPattern = "ci"
		assert.NoError(t, dto.Validate())
	})

	t.Run("applies to update as well", func(t *testing.T) {
		update := WorkloadIdentityFederationUpdateRequestDTO{
			Name:           "github-ci",
			IssuerURL:      "https://token.actions.githubusercontent.com",
			Audience:       "https://auth.example.com",
			SubjectPattern: "*",
		}
		assert.Error(t, update.Validate())
	})
}

// The map's VALUES are destination claim names in the issued token, so a reserved
// value forges that claim. The exchange path drops them at issuance; this refuses
// them at configuration time so the operator is told rather than left with a mapping
// that silently does nothing.
func TestAttributeMappingValidation(t *testing.T) {
	t.Run("rejects a reserved destination claim", func(t *testing.T) {
		for _, reserved := range []string{"sub", "client_id", "svc", "tenant_id", "permissions", "exp"} {
			dto := baseCreateDTO()
			dto.AttributeMapping = map[string]string{"repository": reserved}
			err := dto.Validate()
			require.Error(t, err, reserved)
			assert.Contains(t, err.Error(), "reserved claim")
		}
	})

	t.Run("rejects a malformed destination claim name", func(t *testing.T) {
		for _, bad := range []string{"Has-Caps", "has space", "1leading", "has.dot", ""} {
			dto := baseCreateDTO()
			dto.AttributeMapping = map[string]string{"repository": bad}
			assert.Error(t, dto.Validate(), bad)
		}
	})

	t.Run("rejects an empty external claim key", func(t *testing.T) {
		dto := baseCreateDTO()
		dto.AttributeMapping = map[string]string{"": "repository"}
		assert.Error(t, dto.Validate())
	})

	t.Run("bounds the number of entries", func(t *testing.T) {
		dto := baseCreateDTO()
		mapping := map[string]string{}
		for i := 0; i < maxAttributeMappingEntries+1; i++ {
			mapping[fmt.Sprintf("external_%d", i)] = fmt.Sprintf("internal_%d", i)
		}
		dto.AttributeMapping = mapping
		err := dto.Validate()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "more than")
	})

	t.Run("accepts a sane mapping", func(t *testing.T) {
		dto := baseCreateDTO()
		dto.AttributeMapping = map[string]string{
			"repository": "repository",
			"ref":        "git_ref",
			"workflow":   "workflow_name",
		}
		assert.NoError(t, dto.Validate())
	})

	t.Run("accepts no mapping at all", func(t *testing.T) {
		dto := baseCreateDTO()
		dto.AttributeMapping = nil
		assert.NoError(t, dto.Validate())
	})
}

// Trusting our own issuer turns the exchange endpoint into an unbounded token-refresh
// loop: any platform token can be re-exchanged for a fresh TTL, forever.
func TestSelfIssuerRejected(t *testing.T) {
	orig := config.AppPublicHostname
	config.AppPublicHostname = "https://auth.example.com"
	t.Cleanup(func() { config.AppPublicHostname = orig })

	dto := baseCreateDTO()
	dto.IssuerURL = "https://auth.example.com"
	err := dto.Validate()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "must not be this authorization server")

	// Case and path variants of the same host are the same server.
	dto.IssuerURL = "https://AUTH.EXAMPLE.COM/oauth"
	assert.Error(t, dto.Validate())

	// A genuinely different issuer is unaffected.
	dto.IssuerURL = "https://token.actions.githubusercontent.com"
	assert.NoError(t, dto.Validate())
}

func TestIssuerURLMustBeHTTPS(t *testing.T) {
	dto := baseCreateDTO()
	for _, bad := range []string{
		"http://token.actions.githubusercontent.com",
		"ftp://example.com",
		"not-a-url",
		"https://",
	} {
		dto.IssuerURL = bad
		assert.Error(t, dto.Validate(), bad)
	}
}
