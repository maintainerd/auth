package route

import (
	"github.com/gin-gonic/gin"
	"github.com/maintainerd/auth/internal/handler/resthandler"
)

func RegisterAuthroute(router *gin.RouterGroup, authHandler *resthandler.AuthHandler) {
	router.POST("/register", authHandler.Register)
	router.POST("/login", authHandler.Login)
}
