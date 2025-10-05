package startup

import (
	"github.com/maintainerd/auth/internal/runner"
	"gorm.io/gorm"
)

func RunAppStartUp(db *gorm.DB) {
	runner.RunMigrations(db)
	// Note: Seeders are now triggered manually via /setup/create_organization endpoint
	// runner.RunSeeders(db, "v0.1.0")
}
