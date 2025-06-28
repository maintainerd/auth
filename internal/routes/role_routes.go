package routes

import (
	"github.com/gin-gonic/gin"
	"github.com/maintainerd/auth/internal/handlers"
)

func RegisterRoleRoutes(router *gin.RouterGroup, roleHandler *handlers.RoleHandler) {
	router.POST("/roles", roleHandler.Create)
	router.GET("/roles", roleHandler.GetAll)
	router.GET("/roles/:role_uuid", roleHandler.GetByUUID)
	router.PUT("/roles/:role_uuid", roleHandler.Update)
	router.DELETE("/roles/:role_uuid", roleHandler.Delete)
}
