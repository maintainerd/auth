package main

import (
	"log"

	"github.com/maintainerd/auth/config"
	"github.com/maintainerd/auth/db/runner"
	"github.com/maintainerd/auth/internal/app"
	"github.com/maintainerd/auth/internal/route"

	"github.com/gin-gonic/gin"
)

func main() {
	db := config.InitDB()
	connString := config.GetDBConnectionString()

	// Run default seeders
	targetVersion := "v1"
	runner.RunDefaultMigrations(targetVersion, connString)
	runner.RunDefaultSeeders(db, targetVersion)

	application := app.NewApp(db)
	r := gin.Default()

	route.Registerroute(r, &route.HandlerCollection{
		RoleHandler: application.RoleHandler,
		AuthHandler: application.AuthHandler,
	})

	if err := r.Run(":8080"); err != nil {
		log.Fatal("Server failed:", err)
	}
}
