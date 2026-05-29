package app

import (
	"github.com/google/uuid"
	"github.com/maintainerd/auth/internal/tenant"
	"github.com/maintainerd/auth/internal/user"
)

type tenantUserReader struct {
	repo user.UserRepository
}

func newTenantUserReader(repo user.UserRepository) tenant.UserReader {
	return &tenantUserReader{repo: repo}
}

func (r *tenantUserReader) FindByUUID(id uuid.UUID) (*tenant.MemberUser, error) {
	u, err := r.repo.FindByUUID(id)
	if err != nil || u == nil {
		return nil, err
	}
	return toTenantMemberUser(u), nil
}

func (r *tenantUserReader) FindByID(id int64) (*tenant.MemberUser, error) {
	u, err := r.repo.FindByID(id)
	if err != nil || u == nil {
		return nil, err
	}
	return toTenantMemberUser(u), nil
}

func toTenantMemberUser(u *user.User) *tenant.MemberUser {
	return &tenant.MemberUser{
		UserID:             u.UserID,
		UserUUID:           u.UserUUID,
		Username:           u.Username,
		Fullname:           u.Fullname,
		Email:              u.Email,
		Phone:              u.Phone,
		IsEmailVerified:    u.IsEmailVerified,
		IsPhoneVerified:    u.IsPhoneVerified,
		IsProfileCompleted: u.IsProfileCompleted,
		IsAccountCompleted: u.IsAccountCompleted,
		Status:             u.Status,
		Metadata:           u.Metadata,
		CreatedAt:          u.CreatedAt,
		UpdatedAt:          u.UpdatedAt,
	}
}
