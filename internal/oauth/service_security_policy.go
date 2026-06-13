package oauth

import (
	"log/slog"
	"time"

	"github.com/maintainerd/auth/internal/platform/jsonutil"
	"github.com/maintainerd/auth/internal/platform/jwt"
	"github.com/maintainerd/auth/internal/secpolicy"
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
	jwt.SetTokenLeeway(policy.ClockSkewLeewaySeconds)
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
	}
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
			if client != nil && client.TenantID > 0 {
				if opts.ExtraClaims == nil {
					opts.ExtraClaims = map[string]any{}
				}
				opts.ExtraClaims["tenant_id"] = client.TenantID
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
