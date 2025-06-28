package routes

import (
	"github.com/gin-gonic/gin"
	"github.com/maintainerd/auth/internal/handlers"
)

type HandlersCollection struct {
	RoleHandler *handlers.RoleHandler
}

func RegisterRoutes(r *gin.Engine, h *HandlersCollection) {
	api := r.Group("/api")
	RegisterRoleRoutes(api, h.RoleHandler)
}
