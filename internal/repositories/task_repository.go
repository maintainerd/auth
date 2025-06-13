package repositories

import (
	"github.com/maintainerd/auth/config"
	"github.com/maintainerd/auth/internal/models"
)

type TaskRepository interface {
	FindAll() ([]models.Task, error)
	Create(task *models.Task) error
}

type taskRepo struct{}

func NewTaskRepository() TaskRepository {
	return &taskRepo{}
}

func (r *taskRepo) FindAll() ([]models.Task, error) {
	var tasks []models.Task
	result := config.DB.Find(&tasks)
	return tasks, result.Error
}

func (r *taskRepo) Create(task *models.Task) error {
	return config.DB.Create(task).Error
}
