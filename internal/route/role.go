package route

import (
	"github.com/gin-gonic/gin"
	"github.com/maintainerd/auth/internal/handler/resthandler"
	"github.com/maintainerd/auth/internal/middleware"
)

func RegisterRoleroute(router *gin.RouterGroup, roleHandler *resthandler.RoleHandler) {
	protected := router.Group("/roles")
	protected.Use(middleware.JWTAuthMiddleware())

	protected.POST("", roleHandler.Create)
	protected.GET("", roleHandler.GetAll)
	protected.GET("/:role_uuid", roleHandler.GetByUUID)
	protected.PUT("/:role_uuid", roleHandler.Update)
	protected.DELETE("/:role_uuid", roleHandler.Delete)
}
