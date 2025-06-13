package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/maintainerd/auth/internal/dtos"
	"github.com/maintainerd/auth/internal/models"
	"github.com/maintainerd/auth/internal/services"
)

type TaskHandler struct {
	service services.TaskService
}

func NewTaskHandler(s services.TaskService) *TaskHandler {
	return &TaskHandler{service: s}
}

func (h *TaskHandler) GetTasks(c *gin.Context) {
	tasks, err := h.service.GetTasks()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch tasks"})
		return
	}

	var response []dtos.TaskResponse
	for _, task := range tasks {
		response = append(response, dtos.TaskResponse{
			ID:      task.ID,
			Name:    task.Name,
			Content: task.Content,
		})
	}

	c.JSON(http.StatusOK, response)
}

func (h *TaskHandler) AddTask(c *gin.Context) {
	var req dtos.CreateTaskRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input"})
		return
	}

	task := &models.Task{
		Name:    req.Name,
		Content: req.Content,
		// Note: TaskStatusID is not part of the Task model yet
	}

	if err := h.service.AddTask(task); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create task"})
		return
	}

	response := dtos.TaskResponse{
		ID:      task.ID,
		Name:    task.Name,
		Content: task.Content,
	}

	c.JSON(http.StatusCreated, response)
}
