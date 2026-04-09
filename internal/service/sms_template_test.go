package service

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/maintainerd/auth/internal/model"
	"github.com/maintainerd/auth/internal/repository"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newSMSTemplateSvc(repo *mockSMSTemplateRepo) SMSTemplateService {
	return NewSMSTemplateService(nil, repo)
}

func TestSMSTemplateService_GetByUUID(t *testing.T) {
	id := uuid.New()
	tid := int64(1)

	cases := []struct {
		name    string
		repoFn  func(string, int64) (*model.SMSTemplate, error)
		wantErr string
	}{
		{
			name:    "not found",
			repoFn:  func(_ string, _ int64) (*model.SMSTemplate, error) { return nil, nil },
			wantErr: "not found",
		},
		{
			name:    "repo error",
			repoFn:  func(_ string, _ int64) (*model.SMSTemplate, error) { return nil, errors.New("db err") },
			wantErr: "db err",
		},
		{
			name: "success",
			repoFn: func(s string, _ int64) (*model.SMSTemplate, error) {
				uid, _ := uuid.Parse(s)
				return &model.SMSTemplate{SMSTemplateUUID: uid, Name: "OTP"}, nil
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			svc := newSMSTemplateSvc(&mockSMSTemplateRepo{findByUUIDAndTenantIDFn: tc.repoFn})
			res, err := svc.GetByUUID(context.Background(), id, tid)
			if tc.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tc.wantErr)
			} else {
				require.NoError(t, err)
				assert.Equal(t, id, res.SMSTemplateUUID)
			}
		})
	}
}

func TestSMSTemplateService_GetAll(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		svc := newSMSTemplateSvc(&mockSMSTemplateRepo{
			findPaginatedFn: func(_ repository.SMSTemplateRepositoryGetFilter) (*repository.PaginationResult[model.SMSTemplate], error) {
				return &repository.PaginationResult[model.SMSTemplate]{
					Data:  []model.SMSTemplate{{Name: "OTP"}},
					Total: 1, Page: 1, Limit: 10, TotalPages: 1,
				}, nil
			},
		})
		res, err := svc.GetAll(context.Background(), 1, nil, nil, nil, nil, 1, 10, "created_at", "asc")
		require.NoError(t, err)
		assert.Equal(t, int64(1), res.Total)
		assert.Len(t, res.Data, 1)
	})

	t.Run("repo error", func(t *testing.T) {
		svc := newSMSTemplateSvc(&mockSMSTemplateRepo{
			findPaginatedFn: func(_ repository.SMSTemplateRepositoryGetFilter) (*repository.PaginationResult[model.SMSTemplate], error) {
				return nil, errors.New("db error")
			},
		})
		_, err := svc.GetAll(context.Background(), 1, nil, nil, nil, nil, 1, 10, "created_at", "asc")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "db error")
	})
}

func TestSMSTemplateService_Create(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		svc := newSMSTemplateSvc(&mockSMSTemplateRepo{
			createFn: func(e *model.SMSTemplate) (*model.SMSTemplate, error) { return e, nil },
		})
		res, err := svc.Create(context.Background(), 1, "OTP", nil, "Your code: {{code}}", nil, "active")
		require.NoError(t, err)
		assert.Equal(t, "OTP", res.Name)
	})

	t.Run("repo error", func(t *testing.T) {
		svc := newSMSTemplateSvc(&mockSMSTemplateRepo{
			createFn: func(_ *model.SMSTemplate) (*model.SMSTemplate, error) { return nil, errors.New("fail") },
		})
		_, err := svc.Create(context.Background(), 1, "OTP", nil, "code", nil, "active")
		require.Error(t, err)
	})
}

func TestSMSTemplateService_Update(t *testing.T) {
	id := uuid.New()

	t.Run("not found", func(t *testing.T) {
		svc := newSMSTemplateSvc(&mockSMSTemplateRepo{
			findByUUIDAndTenantIDFn: func(_ string, _ int64) (*model.SMSTemplate, error) { return nil, nil },
		})
		_, err := svc.Update(context.Background(), id, 1, "N", nil, "M", nil, "active")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})

	t.Run("find error", func(t *testing.T) {
		svc := newSMSTemplateSvc(&mockSMSTemplateRepo{
			findByUUIDAndTenantIDFn: func(_ string, _ int64) (*model.SMSTemplate, error) {
				return nil, errors.New("db error")
			},
		})
		_, err := svc.Update(context.Background(), id, 1, "N", nil, "M", nil, "active")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "db error")
	})

	t.Run("system template blocked", func(t *testing.T) {
		svc := newSMSTemplateSvc(&mockSMSTemplateRepo{
			findByUUIDAndTenantIDFn: func(s string, _ int64) (*model.SMSTemplate, error) {
				uid, _ := uuid.Parse(s)
				return &model.SMSTemplate{SMSTemplateUUID: uid, IsSystem: true}, nil
			},
		})
		_, err := svc.Update(context.Background(), id, 1, "N", nil, "M", nil, "active")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "system")
	})

	t.Run("UpdateByUUID error", func(t *testing.T) {
		svc := newSMSTemplateSvc(&mockSMSTemplateRepo{
			findByUUIDAndTenantIDFn: func(s string, _ int64) (*model.SMSTemplate, error) {
				uid, _ := uuid.Parse(s)
				return &model.SMSTemplate{SMSTemplateUUID: uid, IsSystem: false}, nil
			},
			updateByUUIDFn: func(_, _ any) (*model.SMSTemplate, error) {
				return nil, errors.New("update failed")
			},
		})
		_, err := svc.Update(context.Background(), id, 1, "Updated", nil, "M", nil, "active")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "update failed")
	})

	t.Run("success", func(t *testing.T) {
		svc := newSMSTemplateSvc(&mockSMSTemplateRepo{
			findByUUIDAndTenantIDFn: func(s string, _ int64) (*model.SMSTemplate, error) {
				uid, _ := uuid.Parse(s)
				return &model.SMSTemplate{SMSTemplateUUID: uid, IsSystem: false}, nil
			},
			updateByUUIDFn: func(_, _ any) (*model.SMSTemplate, error) {
				return &model.SMSTemplate{Name: "Updated"}, nil
			},
		})
		res, err := svc.Update(context.Background(), id, 1, "Updated", nil, "M", nil, "active")
		require.NoError(t, err)
		assert.Equal(t, "Updated", res.Name)
	})
}

func TestSMSTemplateService_Delete(t *testing.T) {
	id := uuid.New()

	t.Run("not found", func(t *testing.T) {
		svc := newSMSTemplateSvc(&mockSMSTemplateRepo{
			findByUUIDAndTenantIDFn: func(_ string, _ int64) (*model.SMSTemplate, error) { return nil, nil },
		})
		_, err := svc.Delete(context.Background(), id, 1)
		require.Error(t, err)
	})

	t.Run("find error", func(t *testing.T) {
		svc := newSMSTemplateSvc(&mockSMSTemplateRepo{
			findByUUIDAndTenantIDFn: func(_ string, _ int64) (*model.SMSTemplate, error) {
				return nil, errors.New("db error")
			},
		})
		_, err := svc.Delete(context.Background(), id, 1)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "db error")
	})

	t.Run("system template blocked", func(t *testing.T) {
		svc := newSMSTemplateSvc(&mockSMSTemplateRepo{
			findByUUIDAndTenantIDFn: func(s string, _ int64) (*model.SMSTemplate, error) {
				uid, _ := uuid.Parse(s)
				return &model.SMSTemplate{SMSTemplateUUID: uid, IsSystem: true}, nil
			},
		})
		_, err := svc.Delete(context.Background(), id, 1)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "system")
	})

	t.Run("delete error", func(t *testing.T) {
		svc := newSMSTemplateSvc(&mockSMSTemplateRepo{
			findByUUIDAndTenantIDFn: func(s string, _ int64) (*model.SMSTemplate, error) {
				uid, _ := uuid.Parse(s)
				return &model.SMSTemplate{SMSTemplateUUID: uid, Name: "OTP"}, nil
			},
			deleteByUUIDFn: func(_ any) error { return errors.New("delete failed") },
		})
		_, err := svc.Delete(context.Background(), id, 1)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "delete failed")
	})

	t.Run("success", func(t *testing.T) {
		svc := newSMSTemplateSvc(&mockSMSTemplateRepo{
			findByUUIDAndTenantIDFn: func(s string, _ int64) (*model.SMSTemplate, error) {
				uid, _ := uuid.Parse(s)
				return &model.SMSTemplate{SMSTemplateUUID: uid, Name: "OTP"}, nil
			},
			deleteByUUIDFn: func(_ any) error { return nil },
		})
		res, err := svc.Delete(context.Background(), id, 1)
		require.NoError(t, err)
		assert.Equal(t, id, res.SMSTemplateUUID)
	})
}

// ---------------------------------------------------------------------------
// UpdateStatus
// ---------------------------------------------------------------------------

func TestSMSTemplateService_UpdateStatus(t *testing.T) {
	id := uuid.New()

	t.Run("find error", func(t *testing.T) {
		svc := newSMSTemplateSvc(&mockSMSTemplateRepo{
			findByUUIDAndTenantIDFn: func(_ string, _ int64) (*model.SMSTemplate, error) {
				return nil, errors.New("db error")
			},
		})
		_, err := svc.UpdateStatus(context.Background(), id, 1, "active")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "db error")
	})

	t.Run("not found", func(t *testing.T) {
		svc := newSMSTemplateSvc(&mockSMSTemplateRepo{
			findByUUIDAndTenantIDFn: func(_ string, _ int64) (*model.SMSTemplate, error) { return nil, nil },
		})
		_, err := svc.UpdateStatus(context.Background(), id, 1, "active")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})

	t.Run("system template blocked", func(t *testing.T) {
		svc := newSMSTemplateSvc(&mockSMSTemplateRepo{
			findByUUIDAndTenantIDFn: func(s string, _ int64) (*model.SMSTemplate, error) {
				uid, _ := uuid.Parse(s)
				return &model.SMSTemplate{SMSTemplateUUID: uid, IsSystem: true}, nil
			},
		})
		_, err := svc.UpdateStatus(context.Background(), id, 1, "active")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "system")
	})

	t.Run("UpdateByUUID error", func(t *testing.T) {
		svc := newSMSTemplateSvc(&mockSMSTemplateRepo{
			findByUUIDAndTenantIDFn: func(s string, _ int64) (*model.SMSTemplate, error) {
				uid, _ := uuid.Parse(s)
				return &model.SMSTemplate{SMSTemplateUUID: uid, IsSystem: false, Status: "draft"}, nil
			},
			updateByUUIDFn: func(_, _ any) (*model.SMSTemplate, error) {
				return nil, errors.New("update failed")
			},
		})
		_, err := svc.UpdateStatus(context.Background(), id, 1, "active")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "update failed")
	})

	t.Run("success", func(t *testing.T) {
		svc := newSMSTemplateSvc(&mockSMSTemplateRepo{
			findByUUIDAndTenantIDFn: func(s string, _ int64) (*model.SMSTemplate, error) {
				uid, _ := uuid.Parse(s)
				return &model.SMSTemplate{SMSTemplateUUID: uid, IsSystem: false, Status: "draft"}, nil
			},
			updateByUUIDFn: func(_, _ any) (*model.SMSTemplate, error) {
				return &model.SMSTemplate{SMSTemplateUUID: id, Status: "active"}, nil
			},
		})
		res, err := svc.UpdateStatus(context.Background(), id, 1, "active")
		require.NoError(t, err)
		assert.Equal(t, "active", res.Status)
	})
}
