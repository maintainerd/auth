package startup

import (
	"github.com/maintainerd/auth/internal/runner"
	"gorm.io/gorm"
)

func RunAppStartUp(db *gorm.DB) {
	runner.RunMigrations(db)
	runner.RunSeeders(db, "v0.1.0")
}
