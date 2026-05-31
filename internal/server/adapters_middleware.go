package server

import (
	"context"

	"github.com/maintainerd/auth/internal/platform/cache"
	"github.com/maintainerd/auth/internal/user"
)

type middlewareUserContextProvider struct {
	userService user.UserService
}

func newMiddlewareUserContextProvider(userService user.UserService) *middlewareUserContextProvider {
	return &middlewareUserContextProvider{userService: userService}
}

func (p *middlewareUserContextProvider) FindBySubAndClientID(ctx context.Context, sub string, clientID string) (*cache.AuthUser, error) {
	u, err := p.userService.FindBySubAndClientID(ctx, sub, clientID)
	if err != nil || u == nil {
		return nil, err
	}
	return toAuthUser(u), nil
}

func toAuthUser(u *user.User) *cache.AuthUser {
	if u == nil {
		return nil
	}

	roles := make([]cache.AuthRole, len(u.Roles))
	for i, role := range u.Roles {
		roles[i] = cache.AuthRole{
			RoleID:   role.RoleID,
			RoleUUID: role.RoleUUID,
			Name:     role.Name,
		}
	}

	var profile *cache.AuthProfile
	if u.Profile != nil {
		profile = &cache.AuthProfile{
			DisplayName: u.Profile.DisplayName,
			FirstName:   u.Profile.FirstName,
			LastName:    u.Profile.LastName,
			ProfileURL:  u.Profile.ProfileURL,
		}
	}

	return &cache.AuthUser{
		UserID:          u.UserID,
		UserUUID:        u.UserUUID,
		Roles:           roles,
		Email:           u.Email,
		IsEmailVerified: u.IsEmailVerified,
		Phone:           u.Phone,
		IsPhoneVerified: u.IsPhoneVerified,
		Fullname:        u.Fullname,
		UpdatedAt:       u.UpdatedAt,
		Profile:         profile,
	}
}
