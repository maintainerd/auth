package services

import (
	"github.com/maintainerd/auth/internal/models"
	"github.com/maintainerd/auth/internal/repositories"
)

type TaskService interface {
	GetTasks() ([]models.Task, error)
	AddTask(task *models.Task) error
}

type taskService struct {
	repo repositories.TaskRepository
}

func NewTaskService(r repositories.TaskRepository) TaskService {
	return &taskService{repo: r}
}

func (s *taskService) GetTasks() ([]models.Task, error) {
	return s.repo.FindAll()
}

func (s *taskService) AddTask(task *models.Task) error {
	return s.repo.Create(task)
}
