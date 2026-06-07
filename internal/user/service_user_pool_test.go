package user

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/maintainerd/auth/internal/platform/crypto"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func newUserPoolSvc(db *gorm.DB, repo *mockUserPoolRepo) UserPoolService {
	return NewUserPoolService(db, repo)
}

// ---------------------------------------------------------------------------
// List
// ---------------------------------------------------------------------------

func TestUserPoolService_List(t *testing.T) {
	t.Run("repo error", func(t *testing.T) {
		svc := newUserPoolSvc(nil, &mockUserPoolRepo{
			findAllByTenantIDFn: func(_ int64) ([]UserPool, error) { return nil, errors.New("db error") },
		})
		_, err := svc.List(context.Background(), tenantID)
		require.Error(t, err)
	})

	t.Run("success", func(t *testing.T) {
		svc := newUserPoolSvc(nil, &mockUserPoolRepo{
			findAllByTenantIDFn: func(_ int64) ([]UserPool, error) {
				return []UserPool{
					{UserPoolUUID: uuid.New(), Name: "a", TenantID: tenantID},
					{UserPoolUUID: uuid.New(), Name: "b", TenantID: tenantID},
				}, nil
			},
		})
		res, err := svc.List(context.Background(), tenantID)
		require.NoError(t, err)
		assert.Len(t, res, 2)
		assert.Equal(t, "a", res[0].Name)
	})
}

// ---------------------------------------------------------------------------
// GetByUUID
// ---------------------------------------------------------------------------

func TestUserPoolService_GetByUUID(t *testing.T) {
	id := uuid.New()

	t.Run("repo error", func(t *testing.T) {
		svc := newUserPoolSvc(nil, &mockUserPoolRepo{
			findByUUIDFn: func(_ any, _ ...string) (*UserPool, error) { return nil, errors.New("db error") },
		})
		_, err := svc.GetByUUID(context.Background(), id, tenantID)
		require.Error(t, err)
	})

	t.Run("not found", func(t *testing.T) {
		svc := newUserPoolSvc(nil, &mockUserPoolRepo{
			findByUUIDFn: func(_ any, _ ...string) (*UserPool, error) { return nil, nil },
		})
		_, err := svc.GetByUUID(context.Background(), id, tenantID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})

	t.Run("cross-tenant is hidden", func(t *testing.T) {
		svc := newUserPoolSvc(nil, &mockUserPoolRepo{
			findByUUIDFn: func(_ any, _ ...string) (*UserPool, error) {
				return &UserPool{UserPoolUUID: id, TenantID: tenantID + 99}, nil
			},
		})
		_, err := svc.GetByUUID(context.Background(), id, tenantID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})

	t.Run("success", func(t *testing.T) {
		svc := newUserPoolSvc(nil, &mockUserPoolRepo{
			findByUUIDFn: func(_ any, _ ...string) (*UserPool, error) {
				return &UserPool{UserPoolUUID: id, TenantID: tenantID, Name: "pool"}, nil
			},
		})
		res, err := svc.GetByUUID(context.Background(), id, tenantID)
		require.NoError(t, err)
		assert.Equal(t, "pool", res.Name)
		assert.Equal(t, id, res.UserPoolUUID)
	})
}

// ---------------------------------------------------------------------------
// Create
// ---------------------------------------------------------------------------

func TestUserPoolService_Create(t *testing.T) {
	t.Run("defaults status to active and stamps creator", func(t *testing.T) {
		actor := int64(7)
		var captured *UserPool
		svc := newUserPoolSvc(nil, &mockUserPoolRepo{
			createFn: func(p *UserPool) (*UserPool, error) { captured = p; return p, nil },
		})
		res, err := svc.Create(context.Background(), tenantID, "Customers", "Customer Pool", "", nil, &actor)
		require.NoError(t, err)
		assert.Equal(t, "active", res.Status)
		assert.Equal(t, "Customers", res.Name)
		require.NotNil(t, captured)
		assert.Equal(t, tenantID, captured.TenantID)
		assert.False(t, captured.IsSystem)
		assert.NotEmpty(t, captured.Identifier)
		require.NotNil(t, captured.CreatedBy)
		assert.Equal(t, actor, *captured.CreatedBy)
	})

	t.Run("identifier generation failure", func(t *testing.T) {
		orig := crypto.GenerateIdentifier
		crypto.GenerateIdentifier = func(int) (string, error) { return "", errors.New("rng failure") }
		defer func() { crypto.GenerateIdentifier = orig }()

		svc := newUserPoolSvc(nil, &mockUserPoolRepo{})
		_, err := svc.Create(context.Background(), tenantID, "Customers", "", "active", nil, nil)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "identifier")
	})

	t.Run("repo create error", func(t *testing.T) {
		svc := newUserPoolSvc(nil, &mockUserPoolRepo{
			createFn: func(_ *UserPool) (*UserPool, error) { return nil, errors.New("insert failed") },
		})
		_, err := svc.Create(context.Background(), tenantID, "Customers", "", "active", nil, nil)
		require.Error(t, err)
	})
}

// ---------------------------------------------------------------------------
// Update
// ---------------------------------------------------------------------------

func TestUserPoolService_Update(t *testing.T) {
	id := uuid.New()

	t.Run("not found", func(t *testing.T) {
		svc := newUserPoolSvc(nil, &mockUserPoolRepo{
			findByUUIDFn: func(_ any, _ ...string) (*UserPool, error) { return nil, nil },
		})
		_, err := svc.Update(context.Background(), id, tenantID, "n", "d", "active", nil, nil)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})

	t.Run("system pool is immutable", func(t *testing.T) {
		svc := newUserPoolSvc(nil, &mockUserPoolRepo{
			findByUUIDFn: func(_ any, _ ...string) (*UserPool, error) {
				return &UserPool{UserPoolUUID: id, TenantID: tenantID, IsSystem: true}, nil
			},
		})
		_, err := svc.Update(context.Background(), id, tenantID, "n", "d", "active", nil, nil)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "system")
	})

	t.Run("repo update error", func(t *testing.T) {
		svc := newUserPoolSvc(nil, &mockUserPoolRepo{
			findByUUIDFn: func(_ any, _ ...string) (*UserPool, error) {
				return &UserPool{UserPoolUUID: id, TenantID: tenantID}, nil
			},
			updateByUUIDFn: func(_, _ any) (*UserPool, error) { return nil, errors.New("update failed") },
		})
		_, err := svc.Update(context.Background(), id, tenantID, "n", "d", "active", nil, nil)
		require.Error(t, err)
	})

	t.Run("success", func(t *testing.T) {
		actor := int64(3)
		svc := newUserPoolSvc(nil, &mockUserPoolRepo{
			findByUUIDFn: func(_ any, _ ...string) (*UserPool, error) {
				return &UserPool{UserPoolUUID: id, TenantID: tenantID, Name: "old"}, nil
			},
		})
		res, err := svc.Update(context.Background(), id, tenantID, "new", "New Display", "inactive", nil, &actor)
		require.NoError(t, err)
		assert.Equal(t, "new", res.Name)
		assert.Equal(t, "New Display", res.DisplayName)
		assert.Equal(t, "inactive", res.Status)
	})
}

// ---------------------------------------------------------------------------
// SetStatus
// ---------------------------------------------------------------------------

func TestUserPoolService_SetStatus(t *testing.T) {
	id := uuid.New()

	t.Run("not found", func(t *testing.T) {
		svc := newUserPoolSvc(nil, &mockUserPoolRepo{
			findByUUIDFn: func(_ any, _ ...string) (*UserPool, error) { return nil, nil },
		})
		_, err := svc.SetStatus(context.Background(), id, tenantID, "inactive", nil)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})

	t.Run("system pool status cannot be changed", func(t *testing.T) {
		svc := newUserPoolSvc(nil, &mockUserPoolRepo{
			findByUUIDFn: func(_ any, _ ...string) (*UserPool, error) {
				return &UserPool{UserPoolUUID: id, TenantID: tenantID, IsSystem: true}, nil
			},
		})
		_, err := svc.SetStatus(context.Background(), id, tenantID, "inactive", nil)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "system")
	})

	t.Run("repo update error", func(t *testing.T) {
		svc := newUserPoolSvc(nil, &mockUserPoolRepo{
			findByUUIDFn: func(_ any, _ ...string) (*UserPool, error) {
				return &UserPool{UserPoolUUID: id, TenantID: tenantID, Status: "active"}, nil
			},
			updateByUUIDFn: func(_, _ any) (*UserPool, error) { return nil, errors.New("update failed") },
		})
		_, err := svc.SetStatus(context.Background(), id, tenantID, "inactive", nil)
		require.Error(t, err)
	})

	t.Run("success", func(t *testing.T) {
		actor := int64(5)
		var captured *UserPool
		svc := newUserPoolSvc(nil, &mockUserPoolRepo{
			findByUUIDFn: func(_ any, _ ...string) (*UserPool, error) {
				return &UserPool{UserPoolUUID: id, TenantID: tenantID, Status: "active"}, nil
			},
			updateByUUIDFn: func(_, data any) (*UserPool, error) { captured = data.(*UserPool); return captured, nil },
		})
		res, err := svc.SetStatus(context.Background(), id, tenantID, "inactive", &actor)
		require.NoError(t, err)
		assert.Equal(t, "inactive", res.Status)
		require.NotNil(t, captured)
		assert.Equal(t, "inactive", captured.Status)
		require.NotNil(t, captured.UpdatedBy)
		assert.Equal(t, actor, *captured.UpdatedBy)
	})
}

// ---------------------------------------------------------------------------
// Delete
// ---------------------------------------------------------------------------

func TestUserPoolService_Delete(t *testing.T) {
	id := uuid.New()

	t.Run("not found", func(t *testing.T) {
		svc := newUserPoolSvc(nil, &mockUserPoolRepo{
			findByUUIDFn: func(_ any, _ ...string) (*UserPool, error) { return nil, nil },
		})
		_, err := svc.Delete(context.Background(), id, tenantID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})

	t.Run("system pool cannot be deleted", func(t *testing.T) {
		svc := newUserPoolSvc(nil, &mockUserPoolRepo{
			findByUUIDFn: func(_ any, _ ...string) (*UserPool, error) {
				return &UserPool{UserPoolUUID: id, TenantID: tenantID, IsSystem: true}, nil
			},
		})
		_, err := svc.Delete(context.Background(), id, tenantID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "system")
	})

	t.Run("repo delete error", func(t *testing.T) {
		svc := newUserPoolSvc(nil, &mockUserPoolRepo{
			findByUUIDFn: func(_ any, _ ...string) (*UserPool, error) {
				return &UserPool{UserPoolUUID: id, TenantID: tenantID}, nil
			},
			deleteByUUIDFn: func(_ any) error { return errors.New("delete failed") },
		})
		_, err := svc.Delete(context.Background(), id, tenantID)
		require.Error(t, err)
	})

	t.Run("success", func(t *testing.T) {
		svc := newUserPoolSvc(nil, &mockUserPoolRepo{
			findByUUIDFn: func(_ any, _ ...string) (*UserPool, error) {
				return &UserPool{UserPoolUUID: id, TenantID: tenantID, Name: "gone"}, nil
			},
		})
		res, err := svc.Delete(context.Background(), id, tenantID)
		require.NoError(t, err)
		assert.Equal(t, "gone", res.Name)
	})
}
