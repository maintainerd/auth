package restserver

import (
	"log"

	"github.com/gin-gonic/gin"
	"github.com/maintainerd/auth/internal/app"
	"github.com/maintainerd/auth/internal/route"
)

func StartRESTServer(application *app.App) {
	r := gin.Default()

	api := r.Group("/api/v1")
	route.RegisterAuthroute(api, application.AuthHandler)
	route.RegisterRoleroute(api, application.RoleHandler)

	log.Println("REST server running on port 8080")
	if err := r.Run(":8080"); err != nil {
		log.Fatal("REST server failed:", err)
	}
}
