package seeder

import (
	"errors"
	"fmt"
	"log/slog"

	"github.com/google/uuid"
	model "github.com/maintainerd/maintainerd-auth/internal/iam"
	"gorm.io/gorm"
)

// SeedAPIPermissions backfills the api_permissions join table so that every
// permission is explicitly linked to the API that owns it.
//
// Ownership is primarily recorded on permissions.api_id (set when each
// permission is seeded), but the api_permissions join table is what downstream
// features (auth-client API permissions, API-key API permissions) layer on top
// of. Without this seeder the table stays empty after tenant creation even
// though every permission already has an owning API, so this mirrors the
// permissions.api_id link into api_permissions.
//
// It is idempotent: an (api_id, permission_id) pair is only inserted when it is
// not already present, so re-running the seeder (or running it for a tenant
// whose permissions were seeded before this seeder existed) is safe.
func SeedAPIPermissions(db *gorm.DB, tenantID int64) error {
	var permissions []model.Permission
	if err := db.Where("tenant_id = ?", tenantID).Find(&permissions).Error; err != nil {
		return fmt.Errorf("failed to fetch permissions for api_permissions seeding: %w", err)
	}

	linked := 0
	for _, perm := range permissions {
		if perm.APIID == 0 {
			slog.Warn("Permission has no owning API, skipping api_permission link", "permission", perm.Name)
			continue
		}

		var existing model.ApiPermission
		err := db.
			Where("api_id = ? AND permission_id = ?", perm.APIID, perm.PermissionID).
			First(&existing).Error
		if err == nil {
			// Link already exists, skip.
			continue
		}
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return fmt.Errorf("failed to check api_permission for %q: %w", perm.Name, err)
		}

		link := model.ApiPermission{
			ApiPermissionUUID: uuid.New(),
			APIID:             perm.APIID,
			PermissionID:      perm.PermissionID,
		}
		if err := db.Create(&link).Error; err != nil {
			return fmt.Errorf("failed to create api_permission for %q: %w", perm.Name, err)
		}
		linked++
	}

	slog.Info("API permissions seeded", "linked", linked, "total", len(permissions))
	return nil
}
