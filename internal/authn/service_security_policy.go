package authn

import (
	"context"

	"github.com/google/uuid"
	"github.com/maintainerd/maintainerd-auth/internal/platform/jsonutil"
	platformjwt "github.com/maintainerd/maintainerd-auth/internal/platform/jwt"
	"github.com/maintainerd/maintainerd-auth/internal/secpolicy"
)

func resolveEffectiveSessionPolicy(repo secpolicy.SecuritySettingRepository, client *Client) secpolicy.EffectiveSessionPolicy {
	var sessionConfig map[string]any
	var mfaConfig map[string]any

	tenantID := clientTenantID(client)
	if repo != nil && tenantID > 0 {
		if setting, err := repo.FindByTenantID(tenantID); err == nil && setting != nil {
			sessionConfig = jsonutil.JSONToMap(setting.SessionConfig)
			mfaConfig = jsonutil.JSONToMap(setting.MFAConfig)
		}
	}

	policy, err := secpolicy.ResolveEffectiveSessionPolicy(sessionConfig, mfaConfig, clientSecurityOverrides(client))
	if err != nil {
		policy, _ = secpolicy.ResolveEffectiveSessionPolicy(nil, nil, clientSecurityOverrides(client))
	}
	return policy
}

func resolveEffectiveTokenPolicy(repo secpolicy.SecuritySettingRepository, client *Client) secpolicy.EffectiveTokenPolicy {
	var tokenConfig map[string]any

	tenantID := clientTenantID(client)
	if repo != nil && tenantID > 0 {
		if setting, err := repo.FindByTenantID(tenantID); err == nil && setting != nil {
			tokenConfig = jsonutil.JSONToMap(setting.TokenConfig)
		}
	}

	policy, err := secpolicy.ResolveEffectiveTokenPolicy(tokenConfig, clientSecurityOverrides(client))
	if err != nil {
		policy, _ = secpolicy.ResolveEffectiveTokenPolicy(nil, clientSecurityOverrides(client))
	}
	platformjwt.SetTokenLeeway(policy.ClockSkewLeewaySeconds)
	return policy
}

func clientTenantID(client *Client) int64 {
	if client == nil {
		return 0
	}
	if client.TenantID > 0 {
		return client.TenantID
	}
	if client.IdentityProvider != nil {
		if client.IdentityProvider.TenantID > 0 {
			return client.IdentityProvider.TenantID
		}
		if client.IdentityProvider.Tenant != nil {
			return client.IdentityProvider.Tenant.TenantID
		}
	}
	return 0
}

func clientIdentityProviderIDPtr(client *Client) *int64 {
	if client == nil || client.IdentityProviderID == 0 {
		return nil
	}
	id := client.IdentityProviderID
	return &id
}

func clientSecurityOverrides(client *Client) secpolicy.SecuritySettingClientOverrides {
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

func tokenAuthContextWithPolicy(amr []string, acr, sessionID string, sessionPolicy secpolicy.EffectiveSessionPolicy, tokenPolicy secpolicy.EffectiveTokenPolicy) tokenAuthContext {
	ctx := tokenAuthContext{
		AMR:                    amr,
		ACR:                    acr,
		SessionID:              sessionID,
		AccessTokenTTLSeconds:  sessionPolicy.AccessTokenTTLSeconds,
		RefreshTokenTTLSeconds: sessionPolicy.RefreshTokenTTLSeconds,
		CookieSecure:           sessionPolicy.CookieSecure,
		CookieHTTPOnly:         sessionPolicy.CookieHTTPOnly,
		CookieSameSite:         sessionPolicy.CookieSameSite,
		CookieRefreshMaxAge:    sessionPolicy.RefreshTokenTTLSeconds,
		HasCookiePolicy:        true,
		SigningAlgorithm:       tokenPolicy.SigningAlgorithm,
	}
	if len(tokenPolicy.AdditionalAccessTokenClaims) > 0 {
		ctx.ExtraAccessClaims = make(map[string]any)
		for _, claim := range tokenPolicy.AdditionalAccessTokenClaims {
			switch claim {
			case "tenant_id":
				ctx.ExtraAccessClaims["tenant_id"] = 0
			}
		}
	}
	return ctx
}

func tokenAuthContextWithPolicyAndRefreshFamily(amr []string, acr, sessionID string, sessionPolicy secpolicy.EffectiveSessionPolicy, tokenPolicy secpolicy.EffectiveTokenPolicy, familyID string) tokenAuthContext {
	ctx := tokenAuthContextWithPolicy(amr, acr, sessionID, sessionPolicy, tokenPolicy)
	ctx.RefreshTokenFamilyID = familyID
	return ctx
}

type policyAwareSessionCreator interface {
	CreateSessionWithPolicy(ctx context.Context, userID int64, ipAddress, userAgent string, policy secpolicy.EffectiveSessionPolicy) (*UserToken, error)
}

type policyAwareConcurrentLimiter interface {
	EnforceConcurrentLimitWithPolicy(ctx context.Context, userUUID uuid.UUID, userID int64, policy secpolicy.EffectiveSessionPolicy) error
}

func createSessionWithPolicy(ctx context.Context, sessionService SessionService, userID int64, ipAddress, userAgent string, policy secpolicy.EffectiveSessionPolicy) (*UserToken, error) {
	if svc, ok := sessionService.(policyAwareSessionCreator); ok {
		return svc.CreateSessionWithPolicy(ctx, userID, ipAddress, userAgent, policy)
	}
	return sessionService.CreateSession(ctx, userID, ipAddress, userAgent)
}

func enforceConcurrentLimitWithPolicy(ctx context.Context, sessionService SessionService, userUUID uuid.UUID, userID int64, policy secpolicy.EffectiveSessionPolicy) error {
	if svc, ok := sessionService.(policyAwareConcurrentLimiter); ok {
		return svc.EnforceConcurrentLimitWithPolicy(ctx, userUUID, userID, policy)
	}
	return sessionService.EnforceConcurrentLimit(ctx, userUUID, userID)
}
