package routes

import (
	"github.com/gin-gonic/gin"
	"github.com/maintainerd/auth/internal/handlers"
)

func RegisterAuthRoutes(router *gin.RouterGroup, authHandler *handlers.AuthHandler) {
	router.POST("/register", authHandler.Register)
	router.POST("/login", authHandler.Login)
}
