package federation

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
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
