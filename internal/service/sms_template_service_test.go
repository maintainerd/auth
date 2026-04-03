package service

import (
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/maintainerd/auth/internal/model"
	"github.com/maintainerd/auth/internal/repository"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newSmsTemplateSvc(repo *mockSmsTemplateRepo) SmsTemplateService {
	return NewSmsTemplateService(nil, repo)
}

func TestSmsTemplateService_GetByUUID(t *testing.T) {
	id := uuid.New()
	tid := int64(1)

	cases := []struct {
		name    string
		repoFn  func(string, int64) (*model.SmsTemplate, error)
		wantErr string
	}{
		{
			name:    "not found",
			repoFn:  func(_ string, _ int64) (*model.SmsTemplate, error) { return nil, nil },
			wantErr: "not found",
		},
		{
			name:    "repo error",
			repoFn:  func(_ string, _ int64) (*model.SmsTemplate, error) { return nil, errors.New("db err") },
			wantErr: "db err",
		},
		{
			name: "success",
			repoFn: func(s string, _ int64) (*model.SmsTemplate, error) {
				uid, _ := uuid.Parse(s)
				return &model.SmsTemplate{SmsTemplateUUID: uid, Name: "OTP"}, nil
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			svc := newSmsTemplateSvc(&mockSmsTemplateRepo{findByUUIDAndTenantIDFn: tc.repoFn})
			res, err := svc.GetByUUID(id, tid)
			if tc.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tc.wantErr)
			} else {
				require.NoError(t, err)
				assert.Equal(t, id, res.SmsTemplateUUID)
			}
		})
	}
}

func TestSmsTemplateService_GetAll(t *testing.T) {
	svc := newSmsTemplateSvc(&mockSmsTemplateRepo{
		findPaginatedFn: func(_ repository.SmsTemplateRepositoryGetFilter) (*repository.PaginationResult[model.SmsTemplate], error) {
			return &repository.PaginationResult[model.SmsTemplate]{
				Data:  []model.SmsTemplate{{Name: "OTP"}},
				Total: 1, Page: 1, Limit: 10, TotalPages: 1,
			}, nil
		},
	})
	res, err := svc.GetAll(1, nil, nil, nil, nil, 1, 10, "created_at", "asc")
	require.NoError(t, err)
	assert.Equal(t, int64(1), res.Total)
	assert.Len(t, res.Data, 1)
}

func TestSmsTemplateService_Create(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		svc := newSmsTemplateSvc(&mockSmsTemplateRepo{
			createFn: func(e *model.SmsTemplate) (*model.SmsTemplate, error) { return e, nil },
		})
		res, err := svc.Create(1, "OTP", nil, "Your code: {{code}}", nil, "active")
		require.NoError(t, err)
		assert.Equal(t, "OTP", res.Name)
	})

	t.Run("repo error", func(t *testing.T) {
		svc := newSmsTemplateSvc(&mockSmsTemplateRepo{
			createFn: func(_ *model.SmsTemplate) (*model.SmsTemplate, error) { return nil, errors.New("fail") },
		})
		_, err := svc.Create(1, "OTP", nil, "code", nil, "active")
		require.Error(t, err)
	})
}

func TestSmsTemplateService_Update(t *testing.T) {
	id := uuid.New()

	t.Run("not found", func(t *testing.T) {
		svc := newSmsTemplateSvc(&mockSmsTemplateRepo{
			findByUUIDAndTenantIDFn: func(_ string, _ int64) (*model.SmsTemplate, error) { return nil, nil },
		})
		_, err := svc.Update(id, 1, "N", nil, "M", nil, "active")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})

	t.Run("system template blocked", func(t *testing.T) {
		svc := newSmsTemplateSvc(&mockSmsTemplateRepo{
			findByUUIDAndTenantIDFn: func(s string, _ int64) (*model.SmsTemplate, error) {
				uid, _ := uuid.Parse(s)
				return &model.SmsTemplate{SmsTemplateUUID: uid, IsSystem: true}, nil
			},
		})
		_, err := svc.Update(id, 1, "N", nil, "M", nil, "active")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "system")
	})

	t.Run("success", func(t *testing.T) {
		svc := newSmsTemplateSvc(&mockSmsTemplateRepo{
			findByUUIDAndTenantIDFn: func(s string, _ int64) (*model.SmsTemplate, error) {
				uid, _ := uuid.Parse(s)
				return &model.SmsTemplate{SmsTemplateUUID: uid, IsSystem: false}, nil
			},
			updateByUUIDFn: func(_, _ any) (*model.SmsTemplate, error) {
				return &model.SmsTemplate{Name: "Updated"}, nil
			},
		})
		res, err := svc.Update(id, 1, "Updated", nil, "M", nil, "active")
		require.NoError(t, err)
		assert.Equal(t, "Updated", res.Name)
	})
}

func TestSmsTemplateService_Delete(t *testing.T) {
	id := uuid.New()

	t.Run("not found", func(t *testing.T) {
		svc := newSmsTemplateSvc(&mockSmsTemplateRepo{
			findByUUIDAndTenantIDFn: func(_ string, _ int64) (*model.SmsTemplate, error) { return nil, nil },
		})
		_, err := svc.Delete(id, 1)
		require.Error(t, err)
	})

	t.Run("system template blocked", func(t *testing.T) {
		svc := newSmsTemplateSvc(&mockSmsTemplateRepo{
			findByUUIDAndTenantIDFn: func(s string, _ int64) (*model.SmsTemplate, error) {
				uid, _ := uuid.Parse(s)
				return &model.SmsTemplate{SmsTemplateUUID: uid, IsSystem: true}, nil
			},
		})
		_, err := svc.Delete(id, 1)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "system")
	})

	t.Run("success", func(t *testing.T) {
		svc := newSmsTemplateSvc(&mockSmsTemplateRepo{
			findByUUIDAndTenantIDFn: func(s string, _ int64) (*model.SmsTemplate, error) {
				uid, _ := uuid.Parse(s)
				return &model.SmsTemplate{SmsTemplateUUID: uid, Name: "OTP"}, nil
			},
			deleteByUUIDFn: func(_ any) error { return nil },
		})
		res, err := svc.Delete(id, 1)
		require.NoError(t, err)
		assert.Equal(t, id, res.SmsTemplateUUID)
	})
}

