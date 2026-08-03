package branding

import (
	"context"
	"encoding/json"
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

func testBrandingMetadata(values map[string]any) datatypes.JSON {
	data, err := json.Marshal(values)
	if err != nil {
		panic(err)
	}
	return datatypes.JSON(data)
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

func TestBrandingService_GetPublicByID(t *testing.T) {
	t.Run("returns tenant-owned public branding", func(t *testing.T) {
		id := uuid.New()
		svc := newBrandingSvc(&mockBrandingRepo{
			findByIDFn: func(got any) (*Branding, error) {
				assert.Equal(t, int64(7), got)
				return &Branding{
					BrandingUUID: id,
					TenantID:     1,
					Name:         "client",
					CompanyName:  "Client App",
					Metadata: testBrandingMetadata(map[string]any{
						BrandingMetadataLogoLabel:     "Client",
						BrandingMetadataShowLogoLabel: true,
					}),
				}, nil
			},
		})

		res, err := svc.GetPublicByID(context.Background(), 1, 7)
		require.NoError(t, err)
		require.NotNil(t, res)
		assert.Equal(t, id, res.BrandingUUID)
		assert.Equal(t, "Client App", res.CompanyName)
		assert.Equal(t, "Client", res.LogoLabel)
	})

	t.Run("rejects branding from another tenant", func(t *testing.T) {
		svc := newBrandingSvc(&mockBrandingRepo{
			findByIDFn: func(any) (*Branding, error) {
				return &Branding{BrandingUUID: uuid.New(), TenantID: 2, CompanyName: "Other"}, nil
			},
		})

		_, err := svc.GetPublicByID(context.Background(), 1, 7)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "branding not found")
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
			testBrandingMetadata(map[string]any{
				BrandingMetadataLayout:         "centered",
				BrandingMetadataLogoLabel:      "Acme IAM",
				BrandingMetadataShowLogoLabel:  true,
				"colors":                       map[string]string{"primary": "#111"},
			}),
			"https://support", "https://privacy", "https://terms",
		)
		require.NoError(t, err)
		assert.Equal(t, "Acme", res.CompanyName)
		assert.Equal(t, "Acme IAM", res.LogoLabel)
		assert.True(t, res.ShowLogoLabel)
		assert.Equal(t, "https://logo.png", res.LogoURL)
		assert.JSONEq(t, `{"layout":"centered","logo_label":"Acme IAM","show_logo_label":true,"colors":{"primary":"#111"}}`, string(res.Metadata))
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
		assert.Equal(t, "X", res.LogoLabel)
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

		res, err := svc.Create(context.Background(), 1, "Acme", "Acme", "", "",
			testBrandingMetadata(map[string]any{
				BrandingMetadataLayout:        "split",
				BrandingMetadataLogoLabel:     "Acme IAM",
				BrandingMetadataShowLogoLabel: true,
			}),
			"", "", "")
		require.NoError(t, err)
		assert.Equal(t, "split", res.Layout)
		assert.Equal(t, "Acme IAM", res.LogoLabel)
		assert.True(t, res.ShowLogoLabel)
	})

	t.Run("defaults omitted layout to centered", func(t *testing.T) {
		svc := newBrandingSvc(&mockBrandingRepo{
			createFn: func(e *Branding) (*Branding, error) { return e, nil },
		})

		res, err := svc.Create(context.Background(), 1, "Acme", "Acme", "", "", nil, "", "", "")
		require.NoError(t, err)
		assert.Equal(t, "centered", res.Layout)
		assert.Equal(t, "Acme", res.LogoLabel)
	})

	t.Run("rejects unsupported layout", func(t *testing.T) {
		svc := newBrandingSvc(&mockBrandingRepo{})
		_, err := svc.Create(context.Background(), 1, "Acme", "Acme", "", "",
			testBrandingMetadata(map[string]any{BrandingMetadataLayout: "sidebar"}),
			"", "", "")
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
				return &Branding{BrandingUUID: id, TenantID: 1, Metadata: testBrandingMetadata(map[string]any{BrandingMetadataLayout: "centered"})}, nil
			},
			createOrUpdateFn: func(e *Branding) (*Branding, error) { return e, nil },
		})

		res, err := svc.UpdateByUUID(context.Background(), id, 1, "Acme", "Acme", "", "",
			testBrandingMetadata(map[string]any{
				BrandingMetadataLayout:        "full_page",
				BrandingMetadataLogoLabel:     "Acme Console",
				BrandingMetadataShowLogoLabel: false,
			}),
			"", "", "")
		require.NoError(t, err)
		assert.Equal(t, "full_page", res.Layout)
		assert.Equal(t, "Acme Console", res.LogoLabel)
		assert.False(t, res.ShowLogoLabel)
	})

	t.Run("preserves layout when omitted", func(t *testing.T) {
		svc := newBrandingSvc(&mockBrandingRepo{
			findByUUIDFn: func(_ uuid.UUID) (*Branding, error) {
				return &Branding{BrandingUUID: id, TenantID: 1, Metadata: testBrandingMetadata(map[string]any{BrandingMetadataLayout: "split"})}, nil
			},
			createOrUpdateFn: func(e *Branding) (*Branding, error) { return e, nil },
		})

		res, err := svc.UpdateByUUID(context.Background(), id, 1, "Acme", "Acme", "", "", nil, "", "", "")
		require.NoError(t, err)
		assert.Equal(t, "split", res.Layout)
	})

	t.Run("preserves system theme name", func(t *testing.T) {
		svc := newBrandingSvc(&mockBrandingRepo{
			findByUUIDFn: func(_ uuid.UUID) (*Branding, error) {
				return &Branding{BrandingUUID: id, TenantID: 1, Name: "default", IsSystem: true}, nil
			},
			createOrUpdateFn: func(e *Branding) (*Branding, error) { return e, nil },
		})

		res, err := svc.UpdateByUUID(context.Background(), id, 1, "renamed-default", "Acme", "", "", nil, "", "", "")
		require.NoError(t, err)
		assert.Equal(t, "default", res.Name)
	})

	t.Run("rejects unsupported layout", func(t *testing.T) {
		svc := newBrandingSvc(&mockBrandingRepo{
			findByUUIDFn: func(_ uuid.UUID) (*Branding, error) {
				return &Branding{BrandingUUID: id, TenantID: 1}, nil
			},
		})

		_, err := svc.UpdateByUUID(context.Background(), id, 1, "Acme", "Acme", "", "",
			testBrandingMetadata(map[string]any{BrandingMetadataLayout: "sidebar"}),
			"", "", "")
		require.Error(t, err)
	})
}

// ---------------------------------------------------------------------------
// RestoreSystem
// ---------------------------------------------------------------------------

func TestBrandingService_RestoreSystem(t *testing.T) {
	id := uuid.New()

	t.Run("restores seeded system theme without changing active state", func(t *testing.T) {
		svc := newBrandingSvc(&mockBrandingRepo{
			findByUUIDFn: func(_ uuid.UUID) (*Branding, error) {
				return &Branding{
					BrandingUUID:     id,
					TenantID:         1,
					Name:             "dark",
					IsSystem:         true,
					IsActive:         true,
					CompanyName:      "Changed",
					LogoURL:          "https://example.com/logo.png",
					LogoData:         []byte("logo"),
					LogoContentType:  "image/png",
					FaviconURL:       "https://example.com/favicon.ico",
					SupportURL:       "https://example.com/support",
					PrivacyPolicyURL: "https://example.com/privacy",
					Metadata:         testBrandingMetadata(map[string]any{BrandingMetadataLayout: "split", BrandingMetadataLogoLabel: "Changed"}),
				}, nil
			},
			createOrUpdateFn: func(e *Branding) (*Branding, error) {
				assert.Equal(t, "centered", metadataString(e.Metadata, BrandingMetadataLayout))
				assert.Equal(t, "Maintainerd-Auth", e.CompanyName)
				assert.Equal(t, "Maintainerd-IAM", metadataString(e.Metadata, BrandingMetadataLogoLabel))
				assert.Equal(t, "Identity and Access Management", metadataString(e.Metadata, BrandingMetadataLogoDetail))
				assert.Equal(t, "Maintainerd", metadataString(e.Metadata, BrandingMetadataIdentityLogoLabel))
				assert.True(t, metadataBool(e.Metadata, BrandingMetadataShowLogoLabel, false))
				assert.True(t, metadataBool(e.Metadata, BrandingMetadataIdentityShowLogoLabel, false))
				assert.Empty(t, e.LogoURL)
				assert.Empty(t, e.LogoData)
				assert.Empty(t, e.LogoContentType)
				assert.Empty(t, e.FaviconURL)
				assert.Empty(t, e.SupportURL)
				assert.Empty(t, e.PrivacyPolicyURL)
				assert.True(t, e.IsActive)
				assert.Contains(t, string(e.Metadata), `"authPageBackground"`)
				assert.NotContains(t, string(e.Metadata), `"dropdownMenu"`)
				return e, nil
			},
		})

		res, err := svc.RestoreSystem(context.Background(), id, 1)
		require.NoError(t, err)
		assert.Equal(t, "centered", res.Layout)
		assert.Equal(t, "Maintainerd-Auth", res.CompanyName)
		assert.True(t, res.IsActive)
		assert.Contains(t, string(res.Metadata), `"authPageBackground"`)
	})

	t.Run("rejects custom theme", func(t *testing.T) {
		svc := newBrandingSvc(&mockBrandingRepo{
			findByUUIDFn: func(_ uuid.UUID) (*Branding, error) {
				return &Branding{BrandingUUID: id, TenantID: 1, Name: "Acme", IsSystem: false}, nil
			},
		})

		_, err := svc.RestoreSystem(context.Background(), id, 1)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "only system branding themes")
	})

	t.Run("rejects unknown system theme", func(t *testing.T) {
		svc := newBrandingSvc(&mockBrandingRepo{
			findByUUIDFn: func(_ uuid.UUID) (*Branding, error) {
				return &Branding{BrandingUUID: id, TenantID: 1, Name: "custom-system", IsSystem: true}, nil
			},
		})

		_, err := svc.RestoreSystem(context.Background(), id, 1)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "seeded default")
	})
}
