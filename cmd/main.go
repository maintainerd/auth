package main

import (
	"log"
	"os"

	"github.com/maintainerd/auth/config"
	"github.com/maintainerd/auth/db/runner"
	"github.com/maintainerd/auth/internal/app"
	"github.com/maintainerd/auth/internal/repository"
	"github.com/maintainerd/auth/internal/route"
	"github.com/maintainerd/auth/internal/service"

	"github.com/gin-gonic/gin"
)

func main() {
	db := config.InitDB()
	connString := config.GetDBConnectionString()

	appVersion := os.Getenv("APP_VERSION")
	appMode := os.Getenv("APP_MODE")

	if appMode == "micro" {
		authConfigRepository := repository.NewAuthConfigRepository(db)
		authConfigService := service.NewAuthConfigService(authConfigRepository)

		_, err := authConfigService.GetLatestConfig()
		if err != nil {
			// Run default seeders
			if appVersion == "v0.0.1" {
				runner.RunDefaultMigrations(connString)
				runner.RunDefaultSeeders(db, appVersion)
			}
		}
	}

	// App wiring
	application := app.NewApp(db)

	// Routring
	r := gin.Default()
	route.Registerroute(r, &route.HandlerCollection{
		RoleHandler: application.RoleHandler,
		AuthHandler: application.AuthHandler,
	})

	if err := r.Run(":8080"); err != nil {
		log.Fatal("Server failed:", err)
	}
}
