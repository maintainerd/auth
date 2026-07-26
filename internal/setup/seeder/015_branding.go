package seeder

import (
	"errors"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type Branding struct {
	BrandingID        int64          `gorm:"column:branding_id;primaryKey;autoIncrement"`
	BrandingUUID      uuid.UUID      `gorm:"column:branding_uuid;type:uuid;uniqueIndex;not null"`
	TenantID          int64          `gorm:"column:tenant_id;not null"`
	Name              string         `gorm:"column:name;type:varchar(100)"`
	IsSystem          bool           `gorm:"column:is_system;not null;default:false"`
	IsActive          bool           `gorm:"column:is_active;not null;default:false"`
	Layout            string         `gorm:"column:layout;type:varchar(32);not null;default:centered"`
	CompanyName       string         `gorm:"column:company_name;type:varchar(255)"`
	LogoURL           string         `gorm:"column:logo_url;type:text"`
	FaviconURL        string         `gorm:"column:favicon_url;type:text"`
	SupportURL        string         `gorm:"column:support_url;type:text"`
	PrivacyPolicyURL  string         `gorm:"column:privacy_policy_url;type:text"`
	TermsOfServiceURL string         `gorm:"column:terms_of_service_url;type:text"`
	Metadata          datatypes.JSON `gorm:"column:metadata;type:jsonb;default:'{}'"`
	CreatedAt         time.Time      `gorm:"column:created_at;autoCreateTime"`
	UpdatedAt         time.Time      `gorm:"column:updated_at;autoUpdateTime"`
}

func (Branding) TableName() string { return "branding" }

// Three system themes are seeded. They are is_system (undeletable); default is
// the active tenant baseline, while light and dark are switchable presets.
// Wired but not yet consumed by the login page; the active one is the loaded
// style and admins can switch/extend via the branding API.
const defaultBrandingMetadata = `{
  "colors": {
    "primary": "#2563eb",
    "secondary": "#525252",
    "accent": "#0ea5e9",
    "appBackground": "#f4f4f4",
    "topPanelBackground": "#161616",
    "sidePanelBackground": "#161616",
    "cardBackground": "#ffffff",
    "textPrimary": "#171717",
    "textMuted": "#737373",
    "border": "#e5e5e5"
  },
  "font": { "family": "Inter, system-ui, sans-serif" }
}`

const lightBrandingMetadata = `{
  "colors": {
    "primary": "#2563eb",
    "secondary": "#64748b",
    "accent": "#0ea5e9",
    "appBackground": "#f8fafc",
    "topPanelBackground": "#ffffff",
    "sidePanelBackground": "#0f172a",
    "cardBackground": "#ffffff",
    "textPrimary": "#0f172a",
    "textMuted": "#64748b",
    "border": "#e2e8f0"
  },
  "font": { "family": "Inter, system-ui, sans-serif" }
}`

const darkBrandingMetadata = `{
  "colors": {
    "primary": "#3b82f6",
    "secondary": "#94a3b8",
    "accent": "#38bdf8",
    "appBackground": "#0b1220",
    "topPanelBackground": "#111827",
    "sidePanelBackground": "#0f172a",
    "cardBackground": "#111827",
    "textPrimary": "#f8fafc",
    "textMuted": "#94a3b8",
    "border": "#1f2937"
  },
  "font": { "family": "Inter, system-ui, sans-serif" }
}`

type systemBrandingTheme struct {
	name     string
	metadata string
	active   bool
}

type legacySystemBrandingTheme struct {
	name            string
	replacementName string
	metadata        string
}

func systemBrandingThemes() []systemBrandingTheme {
	return []systemBrandingTheme{
		{name: "default", metadata: defaultBrandingMetadata, active: true},
		{name: "light", metadata: lightBrandingMetadata, active: false},
		{name: "dark", metadata: darkBrandingMetadata, active: false},
	}
}

func legacySystemBrandingThemes() []legacySystemBrandingTheme {
	return []legacySystemBrandingTheme{
		{name: "maintainerd-light", replacementName: "light", metadata: lightBrandingMetadata},
		{name: "maintainerd-dark", replacementName: "dark", metadata: darkBrandingMetadata},
	}
}

func SeedBranding(db *gorm.DB, tenantID int64) error {
	// Immutable system themes: default (active), light, and dark. The active one
	// is the loaded style; admins can switch between them or add their own.
	// Idempotent — seeded by name.
	if err := normalizeLegacySystemBranding(db, tenantID); err != nil {
		return err
	}

	hasActive, err := tenantHasActiveBranding(db, tenantID)
	if err != nil {
		return err
	}

	for _, t := range systemBrandingThemes() {
		var existing Branding
		err := db.Where("tenant_id = ? AND name = ?", tenantID, t.name).First(&existing).Error
		if err == nil {
			if t.active && !hasActive && !existing.IsActive {
				if err := db.Model(&existing).Update("is_active", true).Error; err != nil {
					return err
				}
				hasActive = true
			}
			slog.Info("System branding already seeded", "tenant_id", tenantID, "name", t.name)
			continue
		}
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}

		isActive := t.active && !hasActive
		b := Branding{
			BrandingUUID: uuid.New(),
			TenantID:     tenantID,
			Name:         t.name,
			Layout:       "centered",
			CompanyName:  "Maintainerd-Auth",
			IsSystem:     true,
			IsActive:     isActive,
			Metadata:     datatypes.JSON([]byte(t.metadata)),
		}
		if err := db.Create(&b).Error; err != nil {
			return err
		}
		if isActive {
			hasActive = true
		}
		slog.Info("Seeded system branding", "tenant_id", tenantID, "name", t.name)
	}

	return nil
}

func normalizeLegacySystemBranding(db *gorm.DB, tenantID int64) error {
	for _, legacy := range legacySystemBrandingThemes() {
		var existing Branding
		err := db.Where(
			"tenant_id = ? AND name = ? AND is_system = ?",
			tenantID,
			legacy.name,
			true,
		).First(&existing).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			continue
		}
		if err != nil {
			return err
		}

		var replacement Branding
		err = db.Where(
			"tenant_id = ? AND name = ? AND is_system = ?",
			tenantID,
			legacy.replacementName,
			true,
		).First(&replacement).Error
		if err == nil {
			if existing.IsActive {
				return db.Model(&existing).Update("is_active", false).Error
			}
			continue
		}
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}

		if err := db.Model(&existing).Updates(map[string]any{
			"name":      legacy.replacementName,
			"is_active": false,
			"metadata":  datatypes.JSON([]byte(legacy.metadata)),
		}).Error; err != nil {
			return err
		}
		slog.Info(
			"Normalized legacy system branding",
			"tenant_id",
			tenantID,
			"name",
			legacy.name,
			"replacement",
			legacy.replacementName,
		)
	}
	return nil
}

func tenantHasActiveBranding(db *gorm.DB, tenantID int64) (bool, error) {
	var count int64
	if err := db.Model(&Branding{}).
		Where("tenant_id = ? AND is_active = ?", tenantID, true).
		Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}
