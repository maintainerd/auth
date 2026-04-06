package service

import (
	"errors"
	"math"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/maintainerd/auth/internal/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newUserSettingSvc(settingRepo *mockUserSettingRepo, userRepo *mockUserRepo) UserSettingService {
	return NewUserSettingService(nil, settingRepo, userRepo)
}

func TestUserSettingService_GetByUUID(t *testing.T) {
	id := uuid.New()

	t.Run("not found", func(t *testing.T) {
		svc := newUserSettingSvc(&mockUserSettingRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.UserSetting, error) { return nil, nil },
		}, &mockUserRepo{})
		_, err := svc.GetByUUID(id)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})

	t.Run("success", func(t *testing.T) {
		svc := newUserSettingSvc(&mockUserSettingRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.UserSetting, error) {
				return &model.UserSetting{UserSettingUUID: id}, nil
			},
		}, &mockUserRepo{})
		res, err := svc.GetByUUID(id)
		require.NoError(t, err)
		assert.Equal(t, id, res.UserSettingUUID)
	})
}

func TestUserSettingService_GetByUserUUID(t *testing.T) {
	userUUID := uuid.New()

	t.Run("user not found", func(t *testing.T) {
		svc := newUserSettingSvc(&mockUserSettingRepo{}, &mockUserRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.User, error) { return nil, nil },
		})
		_, err := svc.GetByUserUUID(userUUID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "user not found")
	})

	t.Run("setting not found", func(t *testing.T) {
		svc := newUserSettingSvc(&mockUserSettingRepo{
			findByUserIDFn: func(_ int64) (*model.UserSetting, error) { return nil, nil },
		}, &mockUserRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.User, error) {
				return &model.User{UserID: 1}, nil
			},
		})
		_, err := svc.GetByUserUUID(userUUID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})

	t.Run("success", func(t *testing.T) {
		sid := uuid.New()
		svc := newUserSettingSvc(&mockUserSettingRepo{
			findByUserIDFn: func(_ int64) (*model.UserSetting, error) {
				return &model.UserSetting{UserSettingUUID: sid}, nil
			},
		}, &mockUserRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.User, error) {
				return &model.User{UserID: 1}, nil
			},
		})
		res, err := svc.GetByUserUUID(userUUID)
		require.NoError(t, err)
		assert.Equal(t, sid, res.UserSettingUUID)
	})
}

func TestUserSettingService_DeleteByUUID(t *testing.T) {
	id := uuid.New()

	t.Run("not found", func(t *testing.T) {
		svc := newUserSettingSvc(&mockUserSettingRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.UserSetting, error) { return nil, nil },
		}, &mockUserRepo{})
		_, err := svc.DeleteByUUID(id)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})

	t.Run("delete error", func(t *testing.T) {
		svc := newUserSettingSvc(&mockUserSettingRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.UserSetting, error) {
				return &model.UserSetting{UserSettingUUID: id}, nil
			},
			deleteByUUIDFn: func(_ any) error { return errors.New("delete failed") },
		}, &mockUserRepo{})
		_, err := svc.DeleteByUUID(id)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "delete failed")
	})

	t.Run("success", func(t *testing.T) {
		svc := newUserSettingSvc(&mockUserSettingRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.UserSetting, error) {
				return &model.UserSetting{UserSettingUUID: id}, nil
			},
			deleteByUUIDFn: func(_ any) error { return nil },
		}, &mockUserRepo{})
		res, err := svc.DeleteByUUID(id)
		require.NoError(t, err)
		assert.Equal(t, id, res.UserSettingUUID)
	})
}

func TestUserSettingService_CreateOrUpdateUserSetting(t *testing.T) {
	userUUID := uuid.New()

	t.Run("user not found → rollback", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		svc := NewUserSettingService(db, &mockUserSettingRepo{}, &mockUserRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.User, error) { return nil, nil },
		})
		_, err := svc.CreateOrUpdateUserSetting(userUUID, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "user not found")
	})

	t.Run("new setting → create → commit", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectCommit()
		sid := uuid.New()
		svc := NewUserSettingService(db, &mockUserSettingRepo{
			findByUserIDFn: func(_ int64) (*model.UserSetting, error) { return nil, nil },
			createFn: func(e *model.UserSetting) (*model.UserSetting, error) {
				e.UserSettingUUID = sid
				return e, nil
			},
		}, &mockUserRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.User, error) {
				return &model.User{UserID: 1}, nil
			},
		})
		res, err := svc.CreateOrUpdateUserSetting(userUUID, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
		require.NoError(t, err)
		assert.Equal(t, sid, res.UserSettingUUID)
	})

	t.Run("FindByUserID error → rollback", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		svc := NewUserSettingService(db, &mockUserSettingRepo{
			findByUserIDFn: func(_ int64) (*model.UserSetting, error) {
				return nil, errors.New("db error")
			},
		}, &mockUserRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.User, error) {
				return &model.User{UserID: 1}, nil
			},
		})
		_, err := svc.CreateOrUpdateUserSetting(userUUID, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "db error")
	})

	t.Run("social links marshal error → rollback", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		badLinks := map[string]any{"bad": math.Inf(1)}
		svc := NewUserSettingService(db, &mockUserSettingRepo{
			findByUserIDFn: func(_ int64) (*model.UserSetting, error) { return nil, nil },
		}, &mockUserRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.User, error) {
				return &model.User{UserID: 1}, nil
			},
		})
		_, err := svc.CreateOrUpdateUserSetting(userUUID, nil, nil, nil, badLinks, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid social links")
	})

	t.Run("new setting → Create error → rollback", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		svc := NewUserSettingService(db, &mockUserSettingRepo{
			findByUserIDFn: func(_ int64) (*model.UserSetting, error) { return nil, nil },
			createFn: func(_ *model.UserSetting) (*model.UserSetting, error) {
				return nil, errors.New("create failed")
			},
		}, &mockUserRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.User, error) {
				return &model.User{UserID: 1}, nil
			},
		})
		_, err := svc.CreateOrUpdateUserSetting(userUUID, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "create failed")
	})

	t.Run("update existing → success → commit", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectCommit()
		sid := uuid.New()
		tz := "America/New_York"
		svc := NewUserSettingService(db, &mockUserSettingRepo{
			findByUserIDFn: func(_ int64) (*model.UserSetting, error) {
				return &model.UserSetting{UserSettingID: 10, UserSettingUUID: sid, UserID: 1}, nil
			},
		}, &mockUserRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.User, error) {
				return &model.User{UserID: 1}, nil
			},
		})
		res, err := svc.CreateOrUpdateUserSetting(userUUID, &tz, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
		require.NoError(t, err)
		assert.Equal(t, sid, res.UserSettingUUID)
		assert.Equal(t, &tz, res.Timezone)
	})

	t.Run("update existing → UpdateByUserID error → rollback", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		svc := NewUserSettingService(db, &mockUserSettingRepo{
			findByUserIDFn: func(_ int64) (*model.UserSetting, error) {
				return &model.UserSetting{UserSettingID: 10, UserSettingUUID: uuid.New(), UserID: 1}, nil
			},
			updateByUserIDFn: func(_ int64, _ *model.UserSetting) error {
				return errors.New("update failed")
			},
		}, &mockUserRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.User, error) {
				return &model.User{UserID: 1}, nil
			},
		})
		_, err := svc.CreateOrUpdateUserSetting(userUUID, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "update failed")
	})

	t.Run("create with all optional fields", func(t *testing.T) {
		db, mock := newMockGormDB(t)
		mock.ExpectBegin()
		mock.ExpectCommit()
		tz := "UTC"
		lang := "en"
		locale := "en-US"
		contact := "email"
		vis := "public"
		mktg := true
		sms := true
		push := false
		consent := true
		now := time.Now()
		ecName := "Jane"
		ecPhone := "555-1234"
		ecEmail := "jane@x.com"
		ecRel := "spouse"
		links := map[string]any{"twitter": "@user"}
		svc := NewUserSettingService(db, &mockUserSettingRepo{
			findByUserIDFn: func(_ int64) (*model.UserSetting, error) { return nil, nil },
		}, &mockUserRepo{
			findByUUIDFn: func(_ any, _ ...string) (*model.User, error) {
				return &model.User{UserID: 1}, nil
			},
		})
		res, err := svc.CreateOrUpdateUserSetting(userUUID, &tz, &lang, &locale, links, &contact, &mktg, &sms, &push, &vis, &consent, &now, &now, &ecName, &ecPhone, &ecEmail, &ecRel)
		require.NoError(t, err)
		assert.Equal(t, &tz, res.Timezone)
		assert.Equal(t, &lang, res.PreferredLanguage)
		assert.Equal(t, &locale, res.Locale)
		assert.Equal(t, &contact, res.PreferredContactMethod)
		assert.True(t, res.MarketingEmailConsent)
		assert.True(t, res.SMSNotificationsConsent)
		assert.False(t, res.PushNotificationsConsent)
		assert.Equal(t, &vis, res.ProfileVisibility)
		assert.True(t, res.DataProcessingConsent)
		assert.Equal(t, &ecName, res.EmergencyContactName)
	})
}

// ---------------------------------------------------------------------------
// toUserSettingServiceDataResult
// ---------------------------------------------------------------------------

func TestToUserSettingServiceDataResult(t *testing.T) {
	t.Run("nil input → nil", func(t *testing.T) {
		assert.Nil(t, toUserSettingServiceDataResult(nil))
	})

	t.Run("full fields", func(t *testing.T) {
		sid := uuid.New()
		tz := "UTC"
		us := &model.UserSetting{
			UserSettingUUID: sid,
			Timezone:        &tz,
		}
		res := toUserSettingServiceDataResult(us)
		require.NotNil(t, res)
		assert.Equal(t, sid, res.UserSettingUUID)
		assert.Equal(t, &tz, res.Timezone)
	})
}
