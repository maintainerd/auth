package main

import (
	"github.com/gin-gonic/gin"
	"github.com/maintainerd/auth/config"
	handler "github.com/maintainerd/auth/internal/handlers"
	repository "github.com/maintainerd/auth/internal/repositories"
	service "github.com/maintainerd/auth/internal/services"
)

func main() {
	config.InitDB()

	// Seed initial data (dev only)
	// seed.SeedTasks()

	r := gin.Default()

	taskRepo := repository.NewTaskRepository()
	taskService := service.NewTaskService(taskRepo)
	taskHandler := handler.NewTaskHandler(taskService)

	r.GET("/tasks", taskHandler.GetTasks)
	r.POST("/tasks", taskHandler.AddTask)

	r.Run(":8080")
}
