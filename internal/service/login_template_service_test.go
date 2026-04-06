package service

import (
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/maintainerd/auth/internal/model"
	"github.com/maintainerd/auth/internal/repository"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/datatypes"
)

func newLoginTemplateSvc(repo *mockLoginTemplateRepo) LoginTemplateService {
	return NewLoginTemplateService(repo)
}

func TestLoginTemplateService_GetByUUID(t *testing.T) {
	id := uuid.New()
	tid := int64(1)

	cases := []struct {
		name    string
		repoFn  func(uuid.UUID, int64, ...string) (*model.LoginTemplate, error)
		wantErr string
	}{
		{
			name:    "not found",
			repoFn:  func(_ uuid.UUID, _ int64, _ ...string) (*model.LoginTemplate, error) { return nil, nil },
			wantErr: "not found",
		},
		{
			name:    "repo error",
			repoFn:  func(_ uuid.UUID, _ int64, _ ...string) (*model.LoginTemplate, error) { return nil, errors.New("db") },
			wantErr: "db",
		},
		{
			name: "success",
			repoFn: func(i uuid.UUID, _ int64, _ ...string) (*model.LoginTemplate, error) {
				return &model.LoginTemplate{LoginTemplateUUID: i, Name: "Default"}, nil
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			svc := newLoginTemplateSvc(&mockLoginTemplateRepo{findByUUIDAndTenantIDFn: tc.repoFn})
			res, err := svc.GetByUUID(id, tid)
			if tc.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tc.wantErr)
			} else {
				require.NoError(t, err)
				assert.Equal(t, id, res.LoginTemplateUUID)
			}
		})
	}
}

func TestLoginTemplateService_GetAll(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		svc := newLoginTemplateSvc(&mockLoginTemplateRepo{
			findPaginatedFn: func(_ repository.LoginTemplateRepositoryGetFilter) (*repository.PaginationResult[model.LoginTemplate], error) {
				return &repository.PaginationResult[model.LoginTemplate]{
					Data:  []model.LoginTemplate{{Name: "T1"}},
					Total: 1, Page: 1, Limit: 10, TotalPages: 1,
				}, nil
			},
		})

		res, err := svc.GetAll(1, nil, nil, nil, nil, nil, 1, 10, "created_at", "asc")
		require.NoError(t, err)
		assert.Equal(t, int64(1), res.Total)
	})

	t.Run("repo error", func(t *testing.T) {
		svc := newLoginTemplateSvc(&mockLoginTemplateRepo{
			findPaginatedFn: func(_ repository.LoginTemplateRepositoryGetFilter) (*repository.PaginationResult[model.LoginTemplate], error) {
				return nil, errors.New("db error")
			},
		})

		_, err := svc.GetAll(1, nil, nil, nil, nil, nil, 1, 10, "created_at", "asc")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "db error")
	})
}

func TestLoginTemplateService_Create(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		svc := newLoginTemplateSvc(&mockLoginTemplateRepo{
			createFn: func(e *model.LoginTemplate) (*model.LoginTemplate, error) { return e, nil },
		})
		res, err := svc.Create(1, "Login", nil, "<html></html>", nil, "active")
		require.NoError(t, err)
		assert.Equal(t, "Login", res.Name)
	})

	t.Run("success with metadata", func(t *testing.T) {
		svc := newLoginTemplateSvc(&mockLoginTemplateRepo{
			createFn: func(e *model.LoginTemplate) (*model.LoginTemplate, error) { return e, nil },
		})
		meta := map[string]any{"theme": "dark"}
		res, err := svc.Create(1, "Login", nil, "<html></html>", meta, "active")
		require.NoError(t, err)
		assert.Equal(t, "Login", res.Name)
	})

	t.Run("metadata marshal error", func(t *testing.T) {
		svc := newLoginTemplateSvc(&mockLoginTemplateRepo{})
		badMeta := map[string]any{"bad": make(chan int)}
		_, err := svc.Create(1, "Login", nil, "<html></html>", badMeta, "active")
		require.Error(t, err)
	})

	t.Run("repo error", func(t *testing.T) {
		svc := newLoginTemplateSvc(&mockLoginTemplateRepo{
			createFn: func(_ *model.LoginTemplate) (*model.LoginTemplate, error) { return nil, errors.New("db fail") },
		})
		_, err := svc.Create(1, "Login", nil, "<html></html>", nil, "active")
		require.Error(t, err)
	})
}

func TestLoginTemplateService_Update(t *testing.T) {
	id := uuid.New()

	t.Run("find error", func(t *testing.T) {
		svc := newLoginTemplateSvc(&mockLoginTemplateRepo{
			findByUUIDAndTenantIDFn: func(_ uuid.UUID, _ int64, _ ...string) (*model.LoginTemplate, error) {
				return nil, errors.New("db err")
			},
		})
		_, err := svc.Update(id, 1, "N", nil, "<html></html>", nil, "active")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "db err")
	})

	t.Run("not found", func(t *testing.T) {
		svc := newLoginTemplateSvc(&mockLoginTemplateRepo{
			findByUUIDAndTenantIDFn: func(_ uuid.UUID, _ int64, _ ...string) (*model.LoginTemplate, error) { return nil, nil },
		})
		_, err := svc.Update(id, 1, "N", nil, "<html></html>", nil, "active")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})

	t.Run("system template blocked", func(t *testing.T) {
		svc := newLoginTemplateSvc(&mockLoginTemplateRepo{
			findByUUIDAndTenantIDFn: func(i uuid.UUID, _ int64, _ ...string) (*model.LoginTemplate, error) {
				return &model.LoginTemplate{LoginTemplateUUID: i, IsSystem: true}, nil
			},
		})
		_, err := svc.Update(id, 1, "N", nil, "<html></html>", nil, "active")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "system")
	})

	t.Run("metadata marshal error", func(t *testing.T) {
		svc := newLoginTemplateSvc(&mockLoginTemplateRepo{
			findByUUIDAndTenantIDFn: func(i uuid.UUID, _ int64, _ ...string) (*model.LoginTemplate, error) {
				return &model.LoginTemplate{LoginTemplateUUID: i}, nil
			},
		})
		badMeta := map[string]any{"bad": make(chan int)}
		_, err := svc.Update(id, 1, "N", nil, "<html></html>", badMeta, "active")
		require.Error(t, err)
	})

	t.Run("update repo error", func(t *testing.T) {
		svc := newLoginTemplateSvc(&mockLoginTemplateRepo{
			findByUUIDAndTenantIDFn: func(i uuid.UUID, _ int64, _ ...string) (*model.LoginTemplate, error) {
				return &model.LoginTemplate{LoginTemplateUUID: i}, nil
			},
			updateByUUIDFn: func(_, _ any) (*model.LoginTemplate, error) {
				return nil, errors.New("update fail")
			},
		})
		_, err := svc.Update(id, 1, "N", nil, "<html></html>", nil, "active")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "update fail")
	})

	t.Run("success with metadata", func(t *testing.T) {
		svc := newLoginTemplateSvc(&mockLoginTemplateRepo{
			findByUUIDAndTenantIDFn: func(i uuid.UUID, _ int64, _ ...string) (*model.LoginTemplate, error) {
				return &model.LoginTemplate{LoginTemplateUUID: i}, nil
			},
			updateByUUIDFn: func(_, _ any) (*model.LoginTemplate, error) {
				return &model.LoginTemplate{Name: "Updated"}, nil
			},
		})
		meta := map[string]any{"theme": "dark"}
		res, err := svc.Update(id, 1, "Updated", nil, "<html></html>", meta, "active")
		require.NoError(t, err)
		assert.Equal(t, "Updated", res.Name)
	})

	t.Run("success", func(t *testing.T) {
		svc := newLoginTemplateSvc(&mockLoginTemplateRepo{
			findByUUIDAndTenantIDFn: func(i uuid.UUID, _ int64, _ ...string) (*model.LoginTemplate, error) {
				return &model.LoginTemplate{LoginTemplateUUID: i}, nil
			},
			updateByUUIDFn: func(_, _ any) (*model.LoginTemplate, error) {
				return &model.LoginTemplate{Name: "Updated"}, nil
			},
		})
		res, err := svc.Update(id, 1, "Updated", nil, "<html></html>", nil, "active")
		require.NoError(t, err)
		assert.Equal(t, "Updated", res.Name)
	})
}

func TestLoginTemplateService_Delete(t *testing.T) {
	id := uuid.New()

	t.Run("find error", func(t *testing.T) {
		svc := newLoginTemplateSvc(&mockLoginTemplateRepo{
			findByUUIDAndTenantIDFn: func(_ uuid.UUID, _ int64, _ ...string) (*model.LoginTemplate, error) {
				return nil, errors.New("db err")
			},
		})
		_, err := svc.Delete(id, 1)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "db err")
	})

	t.Run("not found", func(t *testing.T) {
		svc := newLoginTemplateSvc(&mockLoginTemplateRepo{
			findByUUIDAndTenantIDFn: func(_ uuid.UUID, _ int64, _ ...string) (*model.LoginTemplate, error) { return nil, nil },
		})
		_, err := svc.Delete(id, 1)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})

	t.Run("system template blocked", func(t *testing.T) {
		svc := newLoginTemplateSvc(&mockLoginTemplateRepo{
			findByUUIDAndTenantIDFn: func(i uuid.UUID, _ int64, _ ...string) (*model.LoginTemplate, error) {
				return &model.LoginTemplate{LoginTemplateUUID: i, IsSystem: true}, nil
			},
		})
		_, err := svc.Delete(id, 1)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "cannot delete system")
	})

	t.Run("delete repo error", func(t *testing.T) {
		svc := newLoginTemplateSvc(&mockLoginTemplateRepo{
			findByUUIDAndTenantIDFn: func(i uuid.UUID, _ int64, _ ...string) (*model.LoginTemplate, error) {
				return &model.LoginTemplate{LoginTemplateUUID: i, Name: "L"}, nil
			},
			deleteByUUIDFn: func(_ any) error { return errors.New("delete fail") },
		})
		_, err := svc.Delete(id, 1)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "delete fail")
	})

	t.Run("success", func(t *testing.T) {
		svc := newLoginTemplateSvc(&mockLoginTemplateRepo{
			findByUUIDAndTenantIDFn: func(i uuid.UUID, _ int64, _ ...string) (*model.LoginTemplate, error) {
				return &model.LoginTemplate{LoginTemplateUUID: i, Name: "L"}, nil
			},
			deleteByUUIDFn: func(_ any) error { return nil },
		})
		res, err := svc.Delete(id, 1)
		require.NoError(t, err)
		assert.Equal(t, id, res.LoginTemplateUUID)
	})
}

func TestLoginTemplateService_UpdateStatus(t *testing.T) {
	id := uuid.New()

	t.Run("find error", func(t *testing.T) {
		svc := newLoginTemplateSvc(&mockLoginTemplateRepo{
			findByUUIDAndTenantIDFn: func(_ uuid.UUID, _ int64, _ ...string) (*model.LoginTemplate, error) {
				return nil, errors.New("db err")
			},
		})
		_, err := svc.UpdateStatus(id, 1, "inactive")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "db err")
	})

	t.Run("not found", func(t *testing.T) {
		svc := newLoginTemplateSvc(&mockLoginTemplateRepo{
			findByUUIDAndTenantIDFn: func(_ uuid.UUID, _ int64, _ ...string) (*model.LoginTemplate, error) { return nil, nil },
		})
		_, err := svc.UpdateStatus(id, 1, "inactive")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})

	t.Run("system template blocked", func(t *testing.T) {
		svc := newLoginTemplateSvc(&mockLoginTemplateRepo{
			findByUUIDAndTenantIDFn: func(i uuid.UUID, _ int64, _ ...string) (*model.LoginTemplate, error) {
				return &model.LoginTemplate{LoginTemplateUUID: i, IsSystem: true}, nil
			},
		})
		_, err := svc.UpdateStatus(id, 1, "inactive")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "system")
	})

	t.Run("update repo error", func(t *testing.T) {
		svc := newLoginTemplateSvc(&mockLoginTemplateRepo{
			findByUUIDAndTenantIDFn: func(i uuid.UUID, _ int64, _ ...string) (*model.LoginTemplate, error) {
				return &model.LoginTemplate{LoginTemplateUUID: i}, nil
			},
			updateByUUIDFn: func(_, _ any) (*model.LoginTemplate, error) {
				return nil, errors.New("update fail")
			},
		})
		_, err := svc.UpdateStatus(id, 1, "inactive")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "update fail")
	})

	t.Run("success", func(t *testing.T) {
		svc := newLoginTemplateSvc(&mockLoginTemplateRepo{
			findByUUIDAndTenantIDFn: func(i uuid.UUID, _ int64, _ ...string) (*model.LoginTemplate, error) {
				return &model.LoginTemplate{LoginTemplateUUID: i, Status: "active"}, nil
			},
			updateByUUIDFn: func(_, _ any) (*model.LoginTemplate, error) {
				return &model.LoginTemplate{LoginTemplateUUID: id, Status: "inactive"}, nil
			},
		})
		res, err := svc.UpdateStatus(id, 1, "inactive")
		require.NoError(t, err)
		assert.Equal(t, "inactive", res.Status)
	})
}

func TestToLoginTemplateServiceDataResult_InvalidMetadata(t *testing.T) {
	tmpl := &model.LoginTemplate{
		LoginTemplateUUID: uuid.New(),
		Name:              "Test",
		Metadata:          datatypes.JSON([]byte("not-json")),
	}
	result := toLoginTemplateServiceDataResult(tmpl)
	assert.NotNil(t, result.Metadata)
	assert.Empty(t, result.Metadata)
}
