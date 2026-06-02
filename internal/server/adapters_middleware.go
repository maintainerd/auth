package server

import (
	"context"

	"github.com/maintainerd/auth/internal/authctx"
	"github.com/maintainerd/auth/internal/user"
)

type middlewareUserContextProvider struct {
	userService user.UserService
}

func newMiddlewareUserContextProvider(userService user.UserService) *middlewareUserContextProvider {
	return &middlewareUserContextProvider{userService: userService}
}

func (p *middlewareUserContextProvider) FindBySubAndClientID(ctx context.Context, sub string, clientID string) (*authctx.AuthUser, error) {
	u, err := p.userService.FindBySubAndClientID(ctx, sub, clientID)
	if err != nil || u == nil {
		return nil, err
	}
	return toAuthUser(u), nil
}

func toAuthUser(u *user.User) *authctx.AuthUser {
	if u == nil {
		return nil
	}

	roles := make([]authctx.AuthRole, len(u.Roles))
	for i, role := range u.Roles {
		roles[i] = authctx.AuthRole{
			RoleID:   role.RoleID,
			RoleUUID: role.RoleUUID,
			Name:     role.Name,
		}
	}

	var profile *authctx.AuthProfile
	if u.Profile != nil {
		profile = &authctx.AuthProfile{
			DisplayName: u.Profile.DisplayName,
			FirstName:   u.Profile.FirstName,
			LastName:    u.Profile.LastName,
			ProfileURL:  u.Profile.ProfileURL,
		}
	}

	return &authctx.AuthUser{
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
