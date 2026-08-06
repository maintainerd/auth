package oauth

import (
	"context"
	"log/slog"
	"time"

	clientpkg "github.com/maintainerd/maintainerd-auth/internal/client"
	"github.com/maintainerd/maintainerd-auth/internal/platform/jsonutil"
	"github.com/maintainerd/maintainerd-auth/internal/platform/jwt"
	"github.com/maintainerd/maintainerd-auth/internal/secpolicy"
	"github.com/maintainerd/maintainerd-auth/internal/shared"
)

func oauthEffectiveSessionPolicy(repo secpolicy.SecuritySettingRepository, client *Client) secpolicy.EffectiveSessionPolicy {
	var sessionConfig map[string]any
	var mfaConfig map[string]any

	if repo != nil && client != nil && client.TenantID > 0 {
		if setting, err := repo.FindByTenantID(client.TenantID); err == nil && setting != nil {
			sessionConfig = jsonutil.JSONToMap(setting.SessionConfig)
			mfaConfig = jsonutil.JSONToMap(setting.MFAConfig)
		}
	}

	policy, err := secpolicy.ResolveEffectiveSessionPolicy(sessionConfig, mfaConfig, oauthClientOverrides(client))
	if err != nil {
		slog.Warn("oauth: session policy resolution failed, falling back to defaults", "err", err)
		policy, _ = secpolicy.ResolveEffectiveSessionPolicy(nil, nil, oauthClientOverrides(client))
	}
	return policy
}

func oauthEffectiveTokenPolicy(repo secpolicy.SecuritySettingRepository, client *Client) secpolicy.EffectiveTokenPolicy {
	var tokenConfig map[string]any

	if repo != nil && client != nil && client.TenantID > 0 {
		if setting, err := repo.FindByTenantID(client.TenantID); err == nil && setting != nil {
			tokenConfig = jsonutil.JSONToMap(setting.TokenConfig)
		}
	}

	policy, err := secpolicy.ResolveEffectiveTokenPolicy(tokenConfig, oauthClientOverrides(client))
	if err != nil {
		slog.Warn("oauth: token policy resolution failed, falling back to defaults", "err", err)
		policy, _ = secpolicy.ResolveEffectiveTokenPolicy(nil, oauthClientOverrides(client))
	}
	// clock_skew_leeway_seconds is advisory for external resource servers only;
	// this server validates its own tokens with a fixed small leeway and never a
	// per-tenant global. See platformjwt.tokenValidationLeeway.
	return policy
}

func oauthClientOverrides(client *Client) secpolicy.SecuritySettingClientOverrides {
	if client == nil {
		return secpolicy.SecuritySettingClientOverrides{}
	}
	return secpolicy.SecuritySettingClientOverrides{
		AccessTokenTTL:         client.AccessTokenTTL,
		RefreshTokenTTL:        client.RefreshTokenTTL,
		SessionIdleTimeout:     client.SessionIdleTimeout,
		SessionAbsoluteTimeout: client.SessionAbsoluteTimeout,
		RequiredACR:            client.RequiredACR,
		RequirePKCE:            client.RequirePKCE,
		PublicClient:           isPublicOAuthClient(client),
	}
}

// isPublicOAuthClient reports whether a client presents no credential at the
// token endpoint, either because its type cannot hold one (SPA, native/mobile)
// or because it is registered with token_endpoint_auth_method "none".
//
// Both conditions matter independently: the type is what the registry validates
// against, but authenticateOAuthClient short-circuits on the auth method, and a
// client configured "none" performs no credential check at redemption whatever
// its declared type. Either way the authorization code is the only secret in the
// flow, so PKCE has to be mandatory (RFC 9700 §2.1.1).
func isPublicOAuthClient(client *Client) bool {
	if client == nil {
		return false
	}
	return clientpkg.IsPublicClientType(client.ClientType) ||
		client.TokenEndpointAuthMethod == TokenAuthMethodNone
}

func oauthAccessTokenOptions(repo secpolicy.SecuritySettingRepository, client *Client) *jwt.AccessTokenOptions {
	opts := &jwt.AccessTokenOptions{}
	sessionPolicy := oauthEffectiveSessionPolicy(repo, client)
	tokenPolicy := oauthEffectiveTokenPolicy(repo, client)
	if sessionPolicy.AccessTokenTTLSeconds > 0 {
		opts.AccessTokenTTL = time.Duration(sessionPolicy.AccessTokenTTLSeconds) * time.Second
	}
	if tokenPolicy.SigningAlgorithm != "" {
		opts.SigningAlgorithm = tokenPolicy.SigningAlgorithm
	}
	for _, claim := range tokenPolicy.AdditionalAccessTokenClaims {
		switch claim {
		case "tenant_id":
			// Stamp the tenant's opaque UUID, never the internal PK (least-disclosure
			// per RFC 9068). The resolver is a cached, ctx-agnostic lookup, so a
			// background context is fine here. It goes on the dedicated TenantUUID
			// field rather than ExtraClaims so the issuer's binding is applied last
			// and cannot be displaced by a claim of the same name.
			if client != nil && client.TenantID > 0 {
				opts.TenantUUID = shared.TenantUUIDStringByID(context.Background(), client.TenantID)
			}
		}
	}
	return opts
}

func oauthAccessTokenExpiresIn(repo secpolicy.SecuritySettingRepository, client *Client) int64 {
	policy := oauthEffectiveSessionPolicy(repo, client)
	if policy.AccessTokenTTLSeconds > 0 {
		return int64(policy.AccessTokenTTLSeconds)
	}
	return int64(jwt.AccessTokenTTL.Seconds())
}
