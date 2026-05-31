package runner

import (
	"github.com/maintainerd/auth/internal/setup/seeder"
	"gorm.io/gorm"
)

var RunSeeders = runSeeders

func runSeeders(db *gorm.DB, appVersion string) error {
	return seeder.RunAll(db, appVersion)
}
