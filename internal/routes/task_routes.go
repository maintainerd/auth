package routes

import (
	"github.com/gin-gonic/gin"
	"github.com/maintainerd/auth/internal/handlers"
)

func RegisterTaskRoutes(r *gin.Engine, taskHandler *handlers.TaskHandler) {
	tasks := r.Group("/tasks")
	{
		tasks.GET("/", taskHandler.GetTasks)
		tasks.POST("/", taskHandler.AddTask)
	}
}
