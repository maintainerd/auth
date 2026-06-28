package branding

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/datatypes"
)

func newBrandingSvc(repo *mockBrandingRepo) BrandingService {
	return NewBrandingService(repo)
}

// ---------------------------------------------------------------------------
// Get
// ---------------------------------------------------------------------------

func TestBrandingService_Get(t *testing.T) {
	t.Run("finds active record", func(t *testing.T) {
		id := uuid.New()
		svc := newBrandingSvc(&mockBrandingRepo{
			findActiveFn: func(tid int64) (*Branding, error) {
				return &Branding{BrandingUUID: id, TenantID: tid, CompanyName: "Acme", IsActive: true}, nil
			},
		})
		res, err := svc.Get(context.Background(), 1)
		require.NoError(t, err)
		assert.Equal(t, id, res.BrandingUUID)
		assert.Equal(t, "Acme", res.CompanyName)
	})

	t.Run("FindActive error", func(t *testing.T) {
		svc := newBrandingSvc(&mockBrandingRepo{
			findActiveFn: func(_ int64) (*Branding, error) { return nil, errors.New("db err") },
		})
		_, err := svc.Get(context.Background(), 1)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "db err")
	})
}

// ---------------------------------------------------------------------------
// Update
// ---------------------------------------------------------------------------

func TestBrandingService_Update(t *testing.T) {
	t.Run("success with existing record", func(t *testing.T) {
		existing := &Branding{BrandingUUID: uuid.New(), TenantID: 1}
		svc := newBrandingSvc(&mockBrandingRepo{
			findByTenantIDFn: func(_ int64) (*Branding, error) { return existing, nil },
			createOrUpdateFn: func(e *Branding) (*Branding, error) { return e, nil },
		})
		res, err := svc.Update(context.Background(), 1,
			"", "Acme", "https://logo.png", "https://favicon.ico",
			datatypes.JSON([]byte(`{"colors":{"primary":"#111"}}`)),
			"https://support", "https://privacy", "https://terms",
		)
		require.NoError(t, err)
		assert.Equal(t, "Acme", res.CompanyName)
		assert.Equal(t, "https://logo.png", res.LogoURL)
		assert.JSONEq(t, `{"colors":{"primary":"#111"}}`, string(res.Metadata))
		assert.Equal(t, "https://support", res.SupportURL)
		assert.Equal(t, "centered", res.Layout)
	})

	t.Run("auto-creates then updates", func(t *testing.T) {
		svc := newBrandingSvc(&mockBrandingRepo{
			findByTenantIDFn: func(_ int64) (*Branding, error) { return nil, nil },
			createFn: func(e *Branding) (*Branding, error) {
				e.BrandingUUID = uuid.New()
				return e, nil
			},
			createOrUpdateFn: func(e *Branding) (*Branding, error) { return e, nil },
		})
		res, err := svc.Update(context.Background(), 1, "", "X", "", "", datatypes.JSON(nil), "", "", "")
		require.NoError(t, err)
		assert.Equal(t, "X", res.CompanyName)
	})

	t.Run("getOrCreate error", func(t *testing.T) {
		svc := newBrandingSvc(&mockBrandingRepo{
			findByTenantIDFn: func(_ int64) (*Branding, error) { return nil, errors.New("db") },
		})
		_, err := svc.Update(context.Background(), 1, "", "", "", "", datatypes.JSON(nil), "", "", "")
		require.Error(t, err)
	})

	t.Run("CreateOrUpdate error", func(t *testing.T) {
		svc := newBrandingSvc(&mockBrandingRepo{
			findByTenantIDFn: func(_ int64) (*Branding, error) {
				return &Branding{BrandingUUID: uuid.New(), TenantID: 1}, nil
			},
			createOrUpdateFn: func(_ *Branding) (*Branding, error) {
				return nil, errors.New("save err")
			},
		})
		_, err := svc.Update(context.Background(), 1, "", "", "", "", datatypes.JSON(nil), "", "", "")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "save err")
	})
}

// ---------------------------------------------------------------------------
// Create
// ---------------------------------------------------------------------------

func TestBrandingService_Create(t *testing.T) {
	t.Run("stores selected layout", func(t *testing.T) {
		svc := newBrandingSvc(&mockBrandingRepo{
			createFn: func(e *Branding) (*Branding, error) {
				e.BrandingUUID = uuid.New()
				return e, nil
			},
		})

		res, err := svc.Create(context.Background(), 1, "Acme", "split", "Acme", "", "", nil, "", "", "")
		require.NoError(t, err)
		assert.Equal(t, "split", res.Layout)
	})

	t.Run("defaults omitted layout to centered", func(t *testing.T) {
		svc := newBrandingSvc(&mockBrandingRepo{
			createFn: func(e *Branding) (*Branding, error) { return e, nil },
		})

		res, err := svc.Create(context.Background(), 1, "Acme", "", "Acme", "", "", nil, "", "", "")
		require.NoError(t, err)
		assert.Equal(t, "centered", res.Layout)
	})

	t.Run("rejects unsupported layout", func(t *testing.T) {
		svc := newBrandingSvc(&mockBrandingRepo{})
		_, err := svc.Create(context.Background(), 1, "Acme", "sidebar", "Acme", "", "", nil, "", "", "")
		require.Error(t, err)
	})
}

// ---------------------------------------------------------------------------
// UpdateByUUID
// ---------------------------------------------------------------------------

func TestBrandingService_UpdateByUUID_Layout(t *testing.T) {
	id := uuid.New()

	t.Run("updates selected layout", func(t *testing.T) {
		svc := newBrandingSvc(&mockBrandingRepo{
			findByUUIDFn: func(_ uuid.UUID) (*Branding, error) {
				return &Branding{BrandingUUID: id, TenantID: 1, Layout: "centered"}, nil
			},
			createOrUpdateFn: func(e *Branding) (*Branding, error) { return e, nil },
		})

		res, err := svc.UpdateByUUID(context.Background(), id, 1, "Acme", "full_page", "Acme", "", "", nil, "", "", "")
		require.NoError(t, err)
		assert.Equal(t, "full_page", res.Layout)
	})

	t.Run("preserves layout when omitted", func(t *testing.T) {
		svc := newBrandingSvc(&mockBrandingRepo{
			findByUUIDFn: func(_ uuid.UUID) (*Branding, error) {
				return &Branding{BrandingUUID: id, TenantID: 1, Layout: "split"}, nil
			},
			createOrUpdateFn: func(e *Branding) (*Branding, error) { return e, nil },
		})

		res, err := svc.UpdateByUUID(context.Background(), id, 1, "Acme", "", "Acme", "", "", nil, "", "", "")
		require.NoError(t, err)
		assert.Equal(t, "split", res.Layout)
	})

	t.Run("rejects unsupported layout", func(t *testing.T) {
		svc := newBrandingSvc(&mockBrandingRepo{
			findByUUIDFn: func(_ uuid.UUID) (*Branding, error) {
				return &Branding{BrandingUUID: id, TenantID: 1}, nil
			},
		})

		_, err := svc.UpdateByUUID(context.Background(), id, 1, "Acme", "sidebar", "Acme", "", "", nil, "", "", "")
		require.Error(t, err)
	})
}
