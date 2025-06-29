package main

import (
	"log"

	"github.com/maintainerd/auth/config"
	"github.com/maintainerd/auth/internal/app"
	"github.com/maintainerd/auth/internal/route"

	"github.com/gin-gonic/gin"
)

func main() {
	db := config.InitDB()

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
