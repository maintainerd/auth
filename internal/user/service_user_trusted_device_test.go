package user

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// recordingTrustedDeviceRepo notes which finder the service reached for.
type recordingTrustedDeviceRepo struct {
	mockBaseRepo[UserTrustedDevice]
	all    []UserTrustedDevice
	active []UserTrustedDevice
	calls  []string
}

func (r *recordingTrustedDeviceRepo) WithTx(*gorm.DB) UserTrustedDeviceRepository { return r }

func (r *recordingTrustedDeviceRepo) FindByUserID(int64) ([]UserTrustedDevice, error) {
	r.calls = append(r.calls, "FindByUserID")
	return r.all, nil
}

func (r *recordingTrustedDeviceRepo) FindActiveByUserID(int64) ([]UserTrustedDevice, error) {
	r.calls = append(r.calls, "FindActiveByUserID")
	return r.active, nil
}

func (r *recordingTrustedDeviceRepo) FindByUUID(any, ...string) (*UserTrustedDevice, error) {
	return nil, nil
}
func (r *recordingTrustedDeviceRepo) DeleteByUUID(any) error                { return nil }
func (r *recordingTrustedDeviceRepo) CreateDevice(*UserTrustedDevice) error { return nil }
func (r *recordingTrustedDeviceRepo) DeleteExpired() (int64, error)         { return 0, nil }
func (r *recordingTrustedDeviceRepo) UpdateLastSeen(int64, time.Time) error { return nil }

// The "devices that skip MFA" screen used to call FindByUserID, which has no
// expiry predicate — so it listed devices that no longer skip MFA. The page said
// the opposite of the truth about the user's own security posture.
func TestUserTrustedDeviceService_ListDevices_ExcludesExpired(t *testing.T) {
	active := UserTrustedDevice{UserTrustedDeviceID: 1, TrustedUntil: time.Now().Add(24 * time.Hour)}
	expired := UserTrustedDevice{UserTrustedDeviceID: 2, TrustedUntil: time.Now().Add(-24 * time.Hour)}

	repo := &recordingTrustedDeviceRepo{
		all:    []UserTrustedDevice{active, expired},
		active: []UserTrustedDevice{active},
	}
	svc := NewUserTrustedDeviceService(repo)

	got, err := svc.ListDevices(context.Background(), 42)

	require.NoError(t, err)
	assert.Equal(t, []string{"FindActiveByUserID"}, repo.calls,
		"listing must use the expiry-filtered finder")
	require.Len(t, got, 1)
	assert.Equal(t, int64(1), got[0].UserTrustedDeviceID)
}
