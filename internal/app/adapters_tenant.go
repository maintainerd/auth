package app

import (
	"context"

	"github.com/google/uuid"
	"github.com/maintainerd/maintainerd-auth/internal/tenant"
	"github.com/maintainerd/maintainerd-auth/internal/user"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type tenantUserReader struct {
	repo user.UserRepository
}

func newTenantUserReader(repo user.UserRepository) tenant.UserReader {
	return &tenantUserReader{repo: repo}
}

type tenantUserProvisioner struct {
	svc user.UserService
}

func newTenantUserProvisioner(svc user.UserService) tenant.UserProvisioner {
	return &tenantUserProvisioner{svc: svc}
}

func (p *tenantUserProvisioner) EnsureUserInTenant(ctx context.Context, userUUID uuid.UUID, targetTenantID int64) (int64, error) {
	return p.svc.EnsureUserInTenant(ctx, userUUID, targetTenantID)
}

func (p *tenantUserProvisioner) GrantRoleByName(ctx context.Context, tx *gorm.DB, userID, tenantID int64, roleName string) error {
	var role user.Role
	if err := tx.WithContext(ctx).
		Where("tenant_id = ? AND name = ?", tenantID, roleName).
		First(&role).Error; err != nil {
		return err
	}
	return tx.WithContext(ctx).Clauses(clause.OnConflict{DoNothing: true}).Create(&user.UserRole{
		UserID: userID,
		RoleID: role.RoleID,
	}).Error
}

func (p *tenantUserProvisioner) RevokeRoleByName(ctx context.Context, tx *gorm.DB, userID, tenantID int64, roleName string) error {
	return tx.WithContext(ctx).
		Where("user_id = ? AND role_id IN (?)", userID,
			tx.Model(&user.Role{}).Select("role_id").Where("tenant_id = ? AND name = ?", tenantID, roleName)).
		Delete(&user.UserRole{}).Error
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
		IsEmailVerified: u.IsEmailVerified,
		IsPhoneVerified: u.IsPhoneVerified,
		Status:          u.Status,
		Metadata:           u.Metadata,
		CreatedAt:          u.CreatedAt,
		UpdatedAt:          u.UpdatedAt,
	}
}
