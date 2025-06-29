package route

import (
	"github.com/gin-gonic/gin"
	"github.com/maintainerd/auth/internal/handler"
)

type HandlerCollection struct {
	RoleHandler *handler.RoleHandler
	AuthHandler *handler.AuthHandler
}

func Registerroute(r *gin.Engine, h *HandlerCollection) {
	api := r.Group("/api/v1")
	RegisterRoleroute(api, h.RoleHandler)
	RegisterAuthroute(api, h.AuthHandler)
}
