package dto

import (
	"time"

	"github.com/google/uuid"
	"github.com/maintainerd/auth/internal/model"
	"gorm.io/datatypes"
)

type ServiceDTO struct {
	ServiceUUID uuid.UUID      `json:"service_uuid"`
	ServiceName string         `json:"service_name"`
	DisplayName string         `json:"display_name"`
	ServiceType string         `json:"service_type"`
	Description string         `json:"description"`
	Version     string         `json:"version"`
	Config      datatypes.JSON `json:"config,omitempty"`
	IsActive    bool           `json:"is_active"`
	IsDefault   bool           `json:"is_default"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   *time.Time     `json:"updated_at,omitempty"`
}

func ToServiceDTO(service *model.Service) ServiceDTO {
	var updatedAt *time.Time
	if !service.UpdatedAt.IsZero() {
		updatedAt = &service.UpdatedAt
	}

	return ServiceDTO{
		ServiceUUID: service.ServiceUUID,
		ServiceName: service.ServiceName,
		DisplayName: service.DisplayName,
		ServiceType: service.ServiceType,
		Description: service.Description,
		Version:     service.Version,
		Config:      service.Config,
		IsActive:    service.IsActive,
		IsDefault:   service.IsDefault,
		CreatedAt:   service.CreatedAt,
		UpdatedAt:   updatedAt,
	}
}
