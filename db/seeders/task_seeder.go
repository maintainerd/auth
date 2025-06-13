package seed

import (
	"github.com/maintainerd/auth/config"
	"github.com/maintainerd/auth/internal/models"
)

func SeedTasks() {
	tasks := []models.Task{
		{Name: "Sample Task 1", Content: "This is a sample task"},
		{Name: "Sample Task 2", Content: "Another task here"},
	}
	config.DB.Create(&tasks)
}
