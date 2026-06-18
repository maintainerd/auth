package client

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/maintainerd/auth/internal/authctx"
	"github.com/maintainerd/auth/internal/shared"
)

type APIKeyAuthenticator struct {
	apiKeyRepo    APIKeyRepository
	apiKeyAPIRepo APIKeyAPIRepository
}

func NewAPIKeyAuthenticator(apiKeyRepo APIKeyRepository, apiKeyAPIRepo APIKeyAPIRepository) *APIKeyAuthenticator {
	return &APIKeyAuthenticator{
		apiKeyRepo:    apiKeyRepo,
		apiKeyAPIRepo: apiKeyAPIRepo,
	}
}

func (a *APIKeyAuthenticator) AuthenticateAPIKey(_ context.Context, rawKey string) (*authctx.AuthContext, error) {
	rawKey = strings.TrimSpace(rawKey)
	if rawKey == "" {
		return nil, errors.New("API key is required")
	}
	hash := sha256.Sum256([]byte(rawKey))
	keyHash := hex.EncodeToString(hash[:])

	apiKey, err := a.apiKeyRepo.FindByKeyHash(keyHash)
	if err != nil {
		return nil, err
	}
	if apiKey == nil || apiKey.Status != shared.StatusActive {
		return nil, errors.New("invalid API key")
	}
	if apiKey.ExpiresAt != nil && time.Now().After(*apiKey.ExpiresAt) {
		return nil, errors.New("API key has expired")
	}

	grants, err := a.apiKeyAPIRepo.FindByAPIKeyUUID(apiKey.APIKeyUUID)
	if err != nil {
		return nil, err
	}
	permissions := make([]authctx.AuthPermission, 0)
	seen := map[int64]bool{}
	for _, grant := range grants {
		for _, apiKeyPerm := range grant.Permissions {
			if apiKeyPerm.Permission == nil || seen[apiKeyPerm.Permission.PermissionID] {
				continue
			}
			seen[apiKeyPerm.Permission.PermissionID] = true
			permissions = append(permissions, authctx.AuthPermission{
				PermissionID:   apiKeyPerm.Permission.PermissionID,
				PermissionUUID: apiKeyPerm.Permission.PermissionUUID,
				Name:           apiKeyPerm.Permission.Name,
			})
		}
	}

	userID := int64(0)
	if apiKey.CreatedBy != nil {
		userID = *apiKey.CreatedBy
	}
	return &authctx.AuthContext{
		User: &authctx.AuthUser{
			UserID:   userID,
			UserUUID: uuid.Nil,
			Roles: []authctx.AuthRole{{
				Name:        "api-key",
				Permissions: permissions,
			}},
		},
		Tenant: &authctx.AuthTenant{TenantID: apiKey.TenantID},
	}, nil
}
