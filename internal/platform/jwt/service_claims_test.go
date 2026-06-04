package jwt

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGenerateAccessTokenWithOptions_ServiceClaims(t *testing.T) {
	initTestJWTKeys(t)
	token, err := GenerateAccessTokenWithOptions(
		"serviceA",
		"",
		"https://auth.example.com",
		"clientA",
		"clientA",
		"providerA",
		&AccessTokenOptions{Service: "serviceA", SubjectType: "service"},
	)
	require.NoError(t, err)

	claims, err := ValidateToken(token)
	require.NoError(t, err)
	require.Equal(t, "serviceA", claims["sub"])
	require.Equal(t, "serviceA", claims["svc"])
	require.Equal(t, "service", claims["sub_type"])
}
