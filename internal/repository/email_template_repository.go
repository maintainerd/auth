package repository

import (
	"github.com/maintainerd/auth/internal/model"
	"gorm.io/gorm"
)

type EmailTemplateRepository interface {
	BaseRepositoryMethods[model.EmailTemplate]
	FindByName(name string) (*model.EmailTemplate, error)
}

type emailTemplateRepository struct {
	*BaseRepository[model.EmailTemplate]
	db *gorm.DB
}

func NewEmailTemplateRepository(db *gorm.DB) EmailTemplateRepository {
	return &emailTemplateRepository{
		BaseRepository: NewBaseRepository[model.EmailTemplate](db, "template_uuid", "template_id"),
		db:             db,
	}
}

// FindByName retrieves an active email template by its name
func (r *emailTemplateRepository) FindByName(name string) (*model.EmailTemplate, error) {
	var template model.EmailTemplate
	err := r.db.
		Where("name = ? AND is_active = TRUE", name).
		First(&template).Error
	if err != nil {
		return nil, err
	}
	return &template, nil
}
