package user

import (
	"context"
	"log/slog"
)

type UserTrustedDeviceService interface {
	ListDevices(ctx context.Context, userID int64) ([]UserTrustedDevice, error)
	DeleteDevice(ctx context.Context, deviceUUID string) error
}

type userTrustedDeviceService struct {
	repo UserTrustedDeviceRepository
}

func NewUserTrustedDeviceService(repo UserTrustedDeviceRepository) UserTrustedDeviceService {
	return &userTrustedDeviceService{repo: repo}
}

func (s *userTrustedDeviceService) ListDevices(ctx context.Context, userID int64) ([]UserTrustedDevice, error) {
	return s.repo.FindByUserID(userID)
}

func (s *userTrustedDeviceService) DeleteDevice(ctx context.Context, deviceUUID string) error {
	if err := s.repo.DeleteByUUID(deviceUUID); err != nil {
		slog.ErrorContext(ctx, "failed to delete trusted device", "uuid", deviceUUID, "error", err)
		return err
	}
	return nil
}
