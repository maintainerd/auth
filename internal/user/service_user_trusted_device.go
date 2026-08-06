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

// ListDevices returns only devices that are still inside their trust window.
//
// It used to call FindByUserID, which has no expiry predicate, so the "devices
// that skip MFA" screen listed devices that no longer skip MFA — the page said
// the opposite of the truth about the user's own security posture, and revoking
// an already-expired entry was the only action it offered.
func (s *userTrustedDeviceService) ListDevices(ctx context.Context, userID int64) ([]UserTrustedDevice, error) {
	return s.repo.FindActiveByUserID(userID)
}

func (s *userTrustedDeviceService) DeleteDevice(ctx context.Context, deviceUUID string) error {
	if err := s.repo.DeleteByUUID(deviceUUID); err != nil {
		slog.ErrorContext(ctx, "failed to delete trusted device", "uuid", deviceUUID, "error", err)
		return err
	}
	return nil
}
