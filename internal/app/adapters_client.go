package app

import (
	"github.com/maintainerd/auth/internal/client"
	"github.com/maintainerd/auth/internal/platform/database"
	"gorm.io/gorm"
)

type clientAPIRepo struct {
	*database.BaseRepository[client.API]
}

func newClientAPIRepo(db *gorm.DB) client.APIRepository {
	return &clientAPIRepo{database.NewBaseRepository[client.API](db, "api_uuid", "api_id")}
}

func (r *clientAPIRepo) WithTx(tx *gorm.DB) client.APIRepository {
	return &clientAPIRepo{r.BaseRepository.WithTx(tx)}
}

type clientPermissionRepo struct {
	*database.BaseRepository[client.Permission]
}

func newClientPermissionRepo(db *gorm.DB) client.PermissionRepository {
	return &clientPermissionRepo{database.NewBaseRepository[client.Permission](db, "permission_uuid", "permission_id")}
}

func (r *clientPermissionRepo) WithTx(tx *gorm.DB) client.PermissionRepository {
	return &clientPermissionRepo{r.BaseRepository.WithTx(tx)}
}

type clientIDPRepo struct {
	*database.BaseRepository[client.IdentityProvider]
}

func newClientIDPRepo(db *gorm.DB) client.IdentityProviderRepository {
	return &clientIDPRepo{database.NewBaseRepository[client.IdentityProvider](db, "identity_provider_uuid", "identity_provider_id")}
}

func (r *clientIDPRepo) WithTx(tx *gorm.DB) client.IdentityProviderRepository {
	return &clientIDPRepo{r.BaseRepository.WithTx(tx)}
}

type clientTenantRepo struct {
	*database.BaseRepository[client.Tenant]
}

func newClientTenantRepo(db *gorm.DB) client.TenantRepository {
	return &clientTenantRepo{database.NewBaseRepository[client.Tenant](db, "tenant_uuid", "tenant_id")}
}

func (r *clientTenantRepo) WithTx(tx *gorm.DB) client.TenantRepository {
	return &clientTenantRepo{r.BaseRepository.WithTx(tx)}
}

type clientUserRepo struct {
	*database.BaseRepository[client.User]
}

func newClientUserRepo(db *gorm.DB) client.UserRepository {
	return &clientUserRepo{database.NewBaseRepository[client.User](db, "user_uuid", "user_id")}
}

func (r *clientUserRepo) WithTx(tx *gorm.DB) client.UserRepository {
	return &clientUserRepo{r.BaseRepository.WithTx(tx)}
}
