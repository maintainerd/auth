package routes

import (
	"github.com/gin-gonic/gin"
	"github.com/maintainerd/auth/internal/handlers"
	"github.com/maintainerd/auth/internal/middlewares"
)

func RegisterRoleRoutes(router *gin.RouterGroup, roleHandler *handlers.RoleHandler) {
	protected := router.Group("/roles")
	protected.Use(middlewares.JWTAuthMiddleware())

	protected.POST("", roleHandler.Create)
	protected.GET("", roleHandler.GetAll)
	protected.GET("/:role_uuid", roleHandler.GetByUUID)
	protected.PUT("/:role_uuid", roleHandler.Update)
	protected.DELETE("/:role_uuid", roleHandler.Delete)
}
