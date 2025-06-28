package routes

import (
	"github.com/gin-gonic/gin"
	"github.com/maintainerd/auth/internal/handlers"
)

type HandlersCollection struct {
	RoleHandler *handlers.RoleHandler
	AuthHandler *handlers.AuthHandler
}

func RegisterRoutes(r *gin.Engine, h *HandlersCollection) {
	api := r.Group("/api/v1")
	RegisterRoleRoutes(api, h.RoleHandler)
	RegisterAuthRoutes(api, h.AuthHandler)
}
