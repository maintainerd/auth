package seeder

import (
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

// Two system themes are seeded — captures the app's current look (blue primary).
// Both are is_system (undeletable); maintainerd-light is the active default.
// Wired but not yet consumed by the login page; the active one is the loaded
// style and admins can switch/extend via the branding API.
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

func SeedBranding(db *gorm.DB, tenantID int64) error {
	// Two immutable system themes: maintainerd-light (active default) and
	// maintainerd-dark (inactive). The active one is the loaded style; admins
	// can switch between them or add their own. Idempotent — seeded by name.
	systemThemes := []struct {
		name     string
		metadata string
		active   bool
	}{
		{name: "maintainerd-light", metadata: lightBrandingMetadata, active: true},
		{name: "maintainerd-dark", metadata: darkBrandingMetadata, active: false},
	}

	for _, t := range systemThemes {
		var existing Branding
		err := db.Where("tenant_id = ? AND name = ?", tenantID, t.name).First(&existing).Error
		if err == nil {
			slog.Info("System branding already seeded", "tenant_id", tenantID, "name", t.name)
			continue
		}

		b := Branding{
			BrandingUUID: uuid.New(),
			TenantID:     tenantID,
			Name:         t.name,
			Layout:       "centered",
			CompanyName:  "Maintainerd-Auth",
			IsSystem:     true,
			IsActive:     t.active,
			Metadata:     datatypes.JSON([]byte(t.metadata)),
		}
		if err := db.Create(&b).Error; err != nil {
			return err
		}
		slog.Info("Seeded system branding", "tenant_id", tenantID, "name", t.name)
	}

	return nil
}
