package startup

import (
	"os"

	"github.com/maintainerd/auth/internal/config"
	"github.com/maintainerd/auth/internal/repository"
	"github.com/maintainerd/auth/internal/runner"
	"github.com/maintainerd/auth/internal/service"
	"gorm.io/gorm"
)

func RunAppStartUp(db *gorm.DB) {
	appMode := os.Getenv("APP_MODE")
	appVersion := os.Getenv("APP_VERSION")

	connString := config.GetDBConnectionString()

	if appMode == "micro" {
		serviceRepository := repository.NewServiceRepository(db)
		serviceService := service.NewServiceService(serviceRepository)

		_, err := serviceService.GetByName("auth")
		if err != nil && appVersion == "v1" {
			runner.RunMigrations(connString)
			runner.RunDefaultSeeders(db, appVersion)
		}
	}
	if appMode == "mono" {

	}
}
