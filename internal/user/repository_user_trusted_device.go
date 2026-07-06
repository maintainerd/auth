package user

import (
	"time"

	"github.com/maintainerd/maintainerd-auth/internal/platform/database"
	"gorm.io/gorm"
)

type UserTrustedDeviceRepository interface {
	BaseRepositoryMethods[UserTrustedDevice]
	WithTx(tx *gorm.DB) UserTrustedDeviceRepository
	FindByUserID(userID int64) ([]UserTrustedDevice, error)
	FindActiveByUserID(userID int64) ([]UserTrustedDevice, error)
	FindByUUID(uuid any, preloads ...string) (*UserTrustedDevice, error)
	DeleteByUUID(uuid any) error
	CreateDevice(device *UserTrustedDevice) error
	DeleteExpired() (int64, error)
	UpdateLastSeen(deviceID int64, seenAt time.Time) error
}

type userTrustedDeviceRepository struct {
	*BaseRepository[UserTrustedDevice]
}

func NewUserTrustedDeviceRepository(db *gorm.DB) UserTrustedDeviceRepository {
	return &userTrustedDeviceRepository{
		BaseRepository: database.NewBaseRepository[UserTrustedDevice](db, "user_trusted_device_uuid", "user_trusted_device_id"),
	}
}

func (r *userTrustedDeviceRepository) WithTx(tx *gorm.DB) UserTrustedDeviceRepository {
	return &userTrustedDeviceRepository{
		BaseRepository: r.BaseRepository.WithTx(tx),
	}
}

func (r *userTrustedDeviceRepository) FindByUserID(userID int64) ([]UserTrustedDevice, error) {
	var devices []UserTrustedDevice
	err := r.DB().Where("user_id = ?", userID).Order("created_at DESC").Find(&devices).Error
	return devices, err
}

func (r *userTrustedDeviceRepository) FindActiveByUserID(userID int64) ([]UserTrustedDevice, error) {
	var devices []UserTrustedDevice
	err := r.DB().
		Where("user_id = ? AND trusted_until > ?", userID, time.Now()).
		Order("created_at DESC").
		Find(&devices).Error
	return devices, err
}

func (r *userTrustedDeviceRepository) CreateDevice(device *UserTrustedDevice) error {
	return r.DB().Create(device).Error
}

func (r *userTrustedDeviceRepository) DeleteExpired() (int64, error) {
	result := r.DB().
		Where("trusted_until < ?", time.Now()).
		Delete(&UserTrustedDevice{})
	return result.RowsAffected, result.Error
}

func (r *userTrustedDeviceRepository) UpdateLastSeen(deviceID int64, seenAt time.Time) error {
	return r.DB().Model(&UserTrustedDevice{}).
		Where("user_trusted_device_id = ?", deviceID).
		Update("last_seen_at", seenAt).Error
}
