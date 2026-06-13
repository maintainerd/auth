package authn

import (
	"context"

	"github.com/google/uuid"
	"github.com/maintainerd/auth/internal/platform/jsonutil"
	"github.com/maintainerd/auth/internal/secpolicy"
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

func clientTenantID(client *Client) int64 {
	if client == nil {
		return 0
	}
	if client.TenantID > 0 {
		return client.TenantID
	}
	if client.IdentityProvider != nil {
		return client.IdentityProvider.TenantID
	}
	return 0
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

func tokenAuthContextWithPolicy(amr []string, acr, sessionID string, policy secpolicy.EffectiveSessionPolicy) tokenAuthContext {
	return tokenAuthContext{
		AMR:                    amr,
		ACR:                    acr,
		SessionID:              sessionID,
		AccessTokenTTLSeconds:  policy.AccessTokenTTLSeconds,
		RefreshTokenTTLSeconds: policy.RefreshTokenTTLSeconds,
		CookieSecure:           policy.CookieSecure,
		CookieHTTPOnly:         policy.CookieHTTPOnly,
		CookieSameSite:         policy.CookieSameSite,
		CookieRefreshMaxAge:    policy.RefreshTokenTTLSeconds,
		HasCookiePolicy:        true,
	}
}

func tokenAuthContextWithPolicyAndRefreshFamily(amr []string, acr, sessionID string, policy secpolicy.EffectiveSessionPolicy, familyID string) tokenAuthContext {
	ctx := tokenAuthContextWithPolicy(amr, acr, sessionID, policy)
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
