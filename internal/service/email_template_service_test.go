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

func newEmailTemplateSvc(repo *mockEmailTemplateRepo) EmailTemplateService {
	return NewEmailTemplateService(nil, repo)
}

func TestEmailTemplateService_GetByUUID(t *testing.T) {
	tid := int64(1)
	id := uuid.New()

	cases := []struct {
		name    string
		repoFn  func(uuid.UUID, int64, ...string) (*model.EmailTemplate, error)
		wantErr string
	}{
		{
			name:    "not found (nil)",
			repoFn:  func(_ uuid.UUID, _ int64, _ ...string) (*model.EmailTemplate, error) { return nil, nil },
			wantErr: "email template not found",
		},
		{
			name:    "repo error",
			repoFn:  func(_ uuid.UUID, _ int64, _ ...string) (*model.EmailTemplate, error) { return nil, errors.New("db err") },
			wantErr: "db err",
		},
		{
			name: "success",
			repoFn: func(id uuid.UUID, _ int64, _ ...string) (*model.EmailTemplate, error) {
				return &model.EmailTemplate{EmailTemplateUUID: id, Name: "Welcome"}, nil
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			svc := newEmailTemplateSvc(&mockEmailTemplateRepo{findByUUIDAndTenantIDFn: tc.repoFn})
			res, err := svc.GetByUUID(id, tid)
			if tc.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tc.wantErr)
			} else {
				require.NoError(t, err)
				assert.Equal(t, id, res.EmailTemplateUUID)
			}
		})
	}
}

func TestEmailTemplateService_GetAll(t *testing.T) {
	svc := newEmailTemplateSvc(&mockEmailTemplateRepo{
		findPaginatedFn: func(f repository.EmailTemplateRepositoryGetFilter) (*repository.PaginationResult[model.EmailTemplate], error) {
			return &repository.PaginationResult[model.EmailTemplate]{
				Data:  []model.EmailTemplate{{Name: "T1"}},
				Total: 1, Page: 1, Limit: 10, TotalPages: 1,
			}, nil
		},
	})

	res, err := svc.GetAll(1, nil, nil, nil, nil, 1, 10, "created_at", "asc")
	require.NoError(t, err)
	assert.Equal(t, int64(1), res.Total)
	assert.Len(t, res.Data, 1)
}

func TestEmailTemplateService_Create(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		svc := newEmailTemplateSvc(&mockEmailTemplateRepo{
			createFn: func(e *model.EmailTemplate) (*model.EmailTemplate, error) { return e, nil },
		})
		res, err := svc.Create(1, "Welcome", "Hi there", "<b>hi</b>", nil, "active", false)
		require.NoError(t, err)
		assert.Equal(t, "Welcome", res.Name)
	})

	t.Run("repo error", func(t *testing.T) {
		svc := newEmailTemplateSvc(&mockEmailTemplateRepo{
			createFn: func(_ *model.EmailTemplate) (*model.EmailTemplate, error) { return nil, errors.New("db fail") },
		})
		_, err := svc.Create(1, "Welcome", "Hi there", "<b>hi</b>", nil, "active", false)
		require.Error(t, err)
	})
}

func TestEmailTemplateService_Update(t *testing.T) {
	id := uuid.New()
	tid := int64(1)

	t.Run("not found", func(t *testing.T) {
		svc := newEmailTemplateSvc(&mockEmailTemplateRepo{
			findByUUIDAndTenantIDFn: func(_ uuid.UUID, _ int64, _ ...string) (*model.EmailTemplate, error) { return nil, nil },
		})
		_, err := svc.Update(id, tid, "N", "S", "<b>b</b>", nil, "active")
		require.Error(t, err)
	})

	t.Run("system template blocked", func(t *testing.T) {
		svc := newEmailTemplateSvc(&mockEmailTemplateRepo{
			findByUUIDAndTenantIDFn: func(i uuid.UUID, _ int64, _ ...string) (*model.EmailTemplate, error) {
				return &model.EmailTemplate{EmailTemplateUUID: i, IsSystem: true}, nil
			},
		})
		_, err := svc.Update(id, tid, "N", "S", "<b>b</b>", nil, "active")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "system")
	})

	t.Run("success", func(t *testing.T) {
		svc := newEmailTemplateSvc(&mockEmailTemplateRepo{
			findByUUIDAndTenantIDFn: func(i uuid.UUID, _ int64, _ ...string) (*model.EmailTemplate, error) {
				return &model.EmailTemplate{EmailTemplateUUID: i, IsSystem: false, Name: "Old"}, nil
			},
			updateByUUIDFn: func(i, data any) (*model.EmailTemplate, error) {
				return &model.EmailTemplate{Name: "New"}, nil
			},
		})
		res, err := svc.Update(id, tid, "New", "S", "<b>b</b>", nil, "active")
		require.NoError(t, err)
		assert.Equal(t, "New", res.Name)
	})
}

func TestEmailTemplateService_Delete(t *testing.T) {
	id := uuid.New()

	t.Run("not found", func(t *testing.T) {
		svc := newEmailTemplateSvc(&mockEmailTemplateRepo{
			findByUUIDAndTenantIDFn: func(_ uuid.UUID, _ int64, _ ...string) (*model.EmailTemplate, error) { return nil, nil },
		})
		_, err := svc.Delete(id, 1)
		require.Error(t, err)
	})

	t.Run("success", func(t *testing.T) {
		svc := newEmailTemplateSvc(&mockEmailTemplateRepo{
			findByUUIDAndTenantIDFn: func(i uuid.UUID, _ int64, _ ...string) (*model.EmailTemplate, error) {
				return &model.EmailTemplate{EmailTemplateUUID: i, Name: "T"}, nil
			},
			deleteByUUIDFn: func(_ any) error { return nil },
		})
		res, err := svc.Delete(id, 1)
		require.NoError(t, err)
		assert.Equal(t, id, res.EmailTemplateUUID)
	})
}

