package route

import (
	"github.com/gin-gonic/gin"
	"github.com/maintainerd/auth/internal/handler"
)

func RegisterAuthroute(router *gin.RouterGroup, authHandler *handler.AuthHandler) {
	router.POST("/register", authHandler.Register)
	router.POST("/login", authHandler.Login)
}
