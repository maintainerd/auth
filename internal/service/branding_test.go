package service

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/maintainerd/auth/internal/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newBrandingSvc(repo *mockBrandingRepo) BrandingService {
	return NewBrandingService(repo)
}

// ---------------------------------------------------------------------------
// Get
// ---------------------------------------------------------------------------

func TestBrandingService_Get(t *testing.T) {
	t.Run("existing record", func(t *testing.T) {
		id := uuid.New()
		svc := newBrandingSvc(&mockBrandingRepo{
			findByTenantIDFn: func(tid int64) (*model.Branding, error) {
				return &model.Branding{BrandingUUID: id, TenantID: tid, CompanyName: "Acme"}, nil
			},
		})
		res, err := svc.Get(context.Background(), 1)
		require.NoError(t, err)
		assert.Equal(t, id, res.BrandingUUID)
		assert.Equal(t, "Acme", res.CompanyName)
	})

	t.Run("auto-creates default when not found", func(t *testing.T) {
		svc := newBrandingSvc(&mockBrandingRepo{
			findByTenantIDFn: func(_ int64) (*model.Branding, error) { return nil, nil },
			createFn: func(e *model.Branding) (*model.Branding, error) {
				e.BrandingUUID = uuid.New()
				return e, nil
			},
		})
		res, err := svc.Get(context.Background(), 1)
		require.NoError(t, err)
		assert.NotNil(t, res)
	})

	t.Run("FindByTenantID error", func(t *testing.T) {
		svc := newBrandingSvc(&mockBrandingRepo{
			findByTenantIDFn: func(_ int64) (*model.Branding, error) { return nil, errors.New("db err") },
		})
		_, err := svc.Get(context.Background(), 1)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "db err")
	})

	t.Run("create default error", func(t *testing.T) {
		svc := newBrandingSvc(&mockBrandingRepo{
			findByTenantIDFn: func(_ int64) (*model.Branding, error) { return nil, nil },
			createFn: func(_ *model.Branding) (*model.Branding, error) {
				return nil, errors.New("create fail")
			},
		})
		_, err := svc.Get(context.Background(), 1)
		require.Error(t, err)
	})
}

// ---------------------------------------------------------------------------
// Update
// ---------------------------------------------------------------------------

func TestBrandingService_Update(t *testing.T) {
	t.Run("success with existing record", func(t *testing.T) {
		existing := &model.Branding{BrandingUUID: uuid.New(), TenantID: 1}
		svc := newBrandingSvc(&mockBrandingRepo{
			findByTenantIDFn: func(_ int64) (*model.Branding, error) { return existing, nil },
			createOrUpdateFn: func(e *model.Branding) (*model.Branding, error) { return e, nil },
		})
		res, err := svc.Update(context.Background(), 1,
			"Acme", "https://logo.png", "https://favicon.ico",
			"#111", "#222", "#333", "Inter", "body{}",
			"https://support", "https://privacy", "https://terms",
		)
		require.NoError(t, err)
		assert.Equal(t, "Acme", res.CompanyName)
		assert.Equal(t, "https://logo.png", res.LogoURL)
		assert.Equal(t, "#111", res.PrimaryColor)
		assert.Equal(t, "https://support", res.SupportURL)
	})

	t.Run("auto-creates then updates", func(t *testing.T) {
		svc := newBrandingSvc(&mockBrandingRepo{
			findByTenantIDFn: func(_ int64) (*model.Branding, error) { return nil, nil },
			createFn: func(e *model.Branding) (*model.Branding, error) {
				e.BrandingUUID = uuid.New()
				return e, nil
			},
			createOrUpdateFn: func(e *model.Branding) (*model.Branding, error) { return e, nil },
		})
		res, err := svc.Update(context.Background(), 1, "X", "", "", "", "", "", "", "", "", "", "")
		require.NoError(t, err)
		assert.Equal(t, "X", res.CompanyName)
	})

	t.Run("getOrCreate error", func(t *testing.T) {
		svc := newBrandingSvc(&mockBrandingRepo{
			findByTenantIDFn: func(_ int64) (*model.Branding, error) { return nil, errors.New("db") },
		})
		_, err := svc.Update(context.Background(), 1, "", "", "", "", "", "", "", "", "", "", "")
		require.Error(t, err)
	})

	t.Run("CreateOrUpdate error", func(t *testing.T) {
		svc := newBrandingSvc(&mockBrandingRepo{
			findByTenantIDFn: func(_ int64) (*model.Branding, error) {
				return &model.Branding{BrandingUUID: uuid.New(), TenantID: 1}, nil
			},
			createOrUpdateFn: func(_ *model.Branding) (*model.Branding, error) {
				return nil, errors.New("save err")
			},
		})
		_, err := svc.Update(context.Background(), 1, "", "", "", "", "", "", "", "", "", "", "")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "save err")
	})
}
