package mfa

import "gorm.io/gorm"

type UserMFAEmailRepository interface {
	FindByUserID(userID int64) (*UserMFAEmail, error)
	Create(record *UserMFAEmail) (*UserMFAEmail, error)
	Save(record *UserMFAEmail) error
	DeleteByUserID(userID int64) error
}

type userMFAEmailRepository struct {
	db *gorm.DB
}

func NewUserMFAEmailRepository(db *gorm.DB) UserMFAEmailRepository {
	return &userMFAEmailRepository{db: db}
}

func (r *userMFAEmailRepository) FindByUserID(userID int64) (*UserMFAEmail, error) {
	var record UserMFAEmail
	err := r.db.Where("user_id = ?", userID).First(&record).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &record, nil
}

func (r *userMFAEmailRepository) Create(record *UserMFAEmail) (*UserMFAEmail, error) {
	if err := r.db.Create(record).Error; err != nil {
		return nil, err
	}
	return record, nil
}

func (r *userMFAEmailRepository) Save(record *UserMFAEmail) error {
	return r.db.Save(record).Error
}

func (r *userMFAEmailRepository) DeleteByUserID(userID int64) error {
	return r.db.Where("user_id = ?", userID).Delete(&UserMFAEmail{}).Error
}
