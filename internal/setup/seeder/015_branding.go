package seeder

import (
	"encoding/json"
	"errors"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/maintainerd/maintainerd-auth/internal/shared"
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
// the active tenant baseline, while light and dark are switchable presets. The
// metadata includes both core palette tokens and component-level design tokens;
// the hardcoded app design remains the fallback when no custom theme metadata is
// applied.
var (
	//nolint:unused
	defaultBrandingMetadata = mustBrandingMetadata(themePalette{
		Primary:             "#2563eb",
		PrimaryHover:        "#1d4ed8",
		Secondary:           "#eef1f5",
		Accent:              "#e9edf3",
		AppBackground:       "#f6f7f9",
		TopPanelBackground:  "#0f172a",
		TopPanelBorder:      "#1e293b",
		TopPanelText:        "#ffffff",
		AuthPageBackground:  "#f6f7f9",
		AuthFormBg:          "#ffffff",
		AuthFormBorder:      "#dce1e8",
		AuthFormText:        "#1f252e",
		AuthFormShadow:      "0 1px 2px rgba(15,23,42,0.04), 0 16px 40px -20px rgba(15,23,42,0.25)",
		AuthVisualBg:        "#2563eb",
		AuthVisualText:      "#ffffff",
		AuthVisualOverlay:   "#0f172a",
		AuthDecorativeLight: "#ffffff",
		AuthDecorativeDark:  "#000000",
		AuthProgressBg:      "#ffffff",
		AuthSecurityBg:      "#ffffff",
		SidePanelBackground: "#f6f7f9",
		SidePanelBorder:     "#cfd6e0",
		SideSectionText:     "#667085",
		SideItemIcon:        "#647084",
		SideItemIconHover:   "#2d3748",
		SideItemActiveIcon:  "#2563eb",
		SideItemHoverText:   "#111827",
		SideChevron:         "#7b8797",
		CardBackground:      "#ffffff",
		TextPrimary:         "#1f252e",
		TextMuted:           "#647084",
		Border:              "#dce1e8",
		InputBorder:         "#d1d8e2",
		TopControlBg:        "rgba(255,255,255,0.05)",
		TopControlHover:     "rgba(255,255,255,0.10)",
		TopControlBorder:    "transparent",
		TopControlText:      "#cbd5e1",
		TopDropdownBg:       "rgba(255,255,255,0.05)",
		TopDropdownHover:    "rgba(255,255,255,0.10)",
		TopDropdownBorder:   "#334155",
		TopDropdownText:     "#cbd5e1",
		TopProfileBg:        "rgba(255,255,255,0.05)",
		TopProfileHover:     "rgba(255,255,255,0.10)",
		TopProfileText:      "#ffffff",
		SwitchThumbColor:    "#f6f7f9",
		SideItemText:        "#475569",
		SideItemHover:       "#edf1f6",
		SideItemActiveBg:    "#e4eaf2",
		SideItemActiveText:  "#111827",
		SideSubItemText:     "#5b677a",
	})
	lightBrandingMetadata = mustBrandingMetadata(themePalette{
		Primary:             "#2563eb",
		PrimaryHover:        "#1d4ed8",
		Secondary:           "#eef1f5",
		Accent:              "#e9edf3",
		AppBackground:       "#f8fafc",
		TopPanelBackground:  "#ffffff",
		TopPanelBorder:      "#e2e8f0",
		TopPanelText:        "#0f172a",
		AuthPageBackground:  "#f8fafc",
		AuthFormBg:          "#ffffff",
		AuthFormBorder:      "#e2e8f0",
		AuthFormText:        "#0f172a",
		AuthFormShadow:      "0 1px 2px rgba(15,23,42,0.04), 0 16px 40px -20px rgba(15,23,42,0.22)",
		AuthVisualBg:        "#2563eb",
		AuthVisualText:      "#ffffff",
		AuthVisualOverlay:   "#0f172a",
		AuthDecorativeLight: "#ffffff",
		AuthDecorativeDark:  "#000000",
		AuthProgressBg:      "#ffffff",
		AuthSecurityBg:      "#ffffff",
		SidePanelBackground: "#f6f7f9",
		SidePanelBorder:     "#e2e8f0",
		SideSectionText:     "#667085",
		SideItemIcon:        "#647084",
		SideItemIconHover:   "#2d3748",
		SideItemActiveIcon:  "#2563eb",
		SideItemHoverText:   "#111827",
		SideChevron:         "#7b8797",
		CardBackground:      "#ffffff",
		TextPrimary:         "#0f172a",
		TextMuted:           "#64748b",
		Border:              "#e2e8f0",
		InputBorder:         "#cbd5e1",
		TopControlBg:        "rgba(15,23,42,0.04)",
		TopControlHover:     "rgba(15,23,42,0.08)",
		TopControlBorder:    "transparent",
		TopControlText:      "#334155",
		TopDropdownBg:       "rgba(15,23,42,0.04)",
		TopDropdownHover:    "rgba(15,23,42,0.08)",
		TopDropdownBorder:   "#e2e8f0",
		TopDropdownText:     "#334155",
		TopProfileBg:        "rgba(15,23,42,0.04)",
		TopProfileHover:     "rgba(15,23,42,0.08)",
		TopProfileText:      "#0f172a",
		SwitchThumbColor:    "#f8fafc",
		SideItemText:        "#475569",
		SideItemHover:       "#edf1f6",
		SideItemActiveBg:    "#e4eaf2",
		SideItemActiveText:  "#111827",
		SideSubItemText:     "#5b677a",
	})
	darkBrandingMetadata = mustBrandingMetadata(themePalette{
		Primary:             "#3b82f6",
		PrimaryHover:        "#2563eb",
		Secondary:           "#1f2937",
		Accent:              "#1f2a3a",
		AppBackground:       "#0d1117",
		TopPanelBackground:  "#111827",
		TopPanelBorder:      "#1f2937",
		TopPanelText:        "#ffffff",
		AuthPageBackground:  "#0d1117",
		AuthFormBg:          "#171e2c",
		AuthFormBorder:      "#1f2937",
		AuthFormText:        "#f1f5f9",
		AuthFormShadow:      "0 1px 2px rgba(0,0,0,0.18), 0 18px 48px -24px rgba(0,0,0,0.72)",
		AuthVisualBg:        "#111827",
		AuthVisualText:      "#f1f5f9",
		AuthVisualOverlay:   "#020617",
		AuthDecorativeLight: "#233149",
		AuthDecorativeDark:  "#050914",
		AuthProgressBg:      "#171e2c",
		AuthSecurityBg:      "#171e2c",
		SidePanelBackground: "#111827",
		SidePanelBorder:     "#1f2937",
		SideSectionText:     "#94a3b8",
		SideItemIcon:        "#94a3b8",
		SideItemIconHover:   "#e2e8f0",
		SideItemActiveIcon:  "#3b82f6",
		SideItemHoverText:   "#f8fafc",
		SideChevron:         "#94a3b8",
		CardBackground:      "#171e2c",
		TextPrimary:         "#f1f5f9",
		TextMuted:           "#94a3b8",
		Border:              "#1f2937",
		InputBorder:         "#334155",
		TopControlBg:        "rgba(255,255,255,0.05)",
		TopControlHover:     "rgba(255,255,255,0.10)",
		TopControlBorder:    "transparent",
		TopControlText:      "#cbd5e1",
		TopDropdownBg:       "rgba(255,255,255,0.05)",
		TopDropdownHover:    "rgba(255,255,255,0.10)",
		TopDropdownBorder:   "#334155",
		TopDropdownText:     "#cbd5e1",
		TopProfileBg:        "rgba(255,255,255,0.05)",
		TopProfileHover:     "rgba(255,255,255,0.10)",
		TopProfileText:      "#ffffff",
		SwitchThumbColor:    "#cbd5e1",
		SideItemText:        "#cbd5e1",
		SideItemHover:       "#172033",
		SideItemActiveBg:    "#1f2a3a",
		SideItemActiveText:  "#f8fafc",
		SideSubItemText:     "#cbd5e1",
	})
)

type themePalette struct {
	Primary             string
	PrimaryHover        string
	Secondary           string
	Accent              string
	AppBackground       string
	TopPanelBackground  string
	TopPanelBorder      string
	TopPanelText        string
	AuthPageBackground  string
	AuthFormBg          string
	AuthFormBorder      string
	AuthFormText        string
	AuthFormShadow      string
	AuthVisualBg        string
	AuthVisualText      string
	AuthVisualOverlay   string
	AuthDecorativeLight string
	AuthDecorativeDark  string
	AuthProgressBg      string
	AuthSecurityBg      string
	SidePanelBackground string
	SidePanelBorder     string
	SideSectionText     string
	SideItemIcon        string
	SideItemIconHover   string
	SideItemActiveIcon  string
	SideItemHoverText   string
	SideChevron         string
	CardBackground      string
	TextPrimary         string
	TextMuted           string
	Border              string
	InputBorder         string
	TopControlBg        string
	TopControlHover     string
	TopControlBorder    string
	TopControlText      string
	TopDropdownBg       string
	TopDropdownHover    string
	TopDropdownBorder   string
	TopDropdownText     string
	TopProfileBg        string
	TopProfileHover     string
	TopProfileText      string
	SwitchThumbColor    string
	SideItemText        string
	SideItemHover       string
	SideItemActiveBg    string
	SideItemActiveText  string
	SideSubItemText     string
}

func mustBrandingMetadata(p themePalette) string {
	payload := map[string]any{
		// Branding preferences (no dedicated columns — metadata owns them).
		"layout":                   "centered",
		"logo_label":               "Maintainerd-IAM",
		"logo_detail":              "Identity and Access Management",
		"show_logo_label":          true,
		"identity_logo_label":      "Maintainerd",
		"identity_show_logo_label": true,
		"login_form_logo_detail":   "Open-source Cloud Platform",
		// Brand sits above the card by default, so the logo reads as the page's
		// identity rather than as a row inside the form.
		"login_form_logo_placement": "above-form",
		"colors": map[string]string{
			"primary":                     p.Primary,
			"secondary":                   p.Secondary,
			"accent":                      p.Accent,
			"appBackground":               p.AppBackground,
			"topPanelBackground":          p.TopPanelBackground,
			"topPanelBorder":              p.TopPanelBorder,
			"topPanelText":                p.TopPanelText,
			"authPageBackground":          p.AuthPageBackground,
			"authFormPanelBackground":     p.AuthFormBg,
			"authFormPanelBorder":         p.AuthFormBorder,
			"authFormPanelText":           p.AuthFormText,
			"authVisualPanelBackground":   p.AuthVisualBg,
			"authVisualPanelText":         p.AuthVisualText,
			"authVisualPanelOverlay":      p.AuthVisualOverlay,
			"authDecorativeLight":         p.AuthDecorativeLight,
			"authDecorativeDark":          p.AuthDecorativeDark,
			"authProgressPanelBackground": p.AuthProgressBg,
			"authSecurityPanelBackground": p.AuthSecurityBg,
			"sidePanelBackground":         p.SidePanelBackground,
			"sidePanelBorder":             p.SidePanelBorder,
			"sidePanelSectionText":        p.SideSectionText,
			"sidePanelItemIcon":           p.SideItemIcon,
			"sidePanelItemIconHover":      p.SideItemIconHover,
			"sidePanelItemActiveIcon":     p.SideItemActiveIcon,
			"sidePanelItemHoverText":      p.SideItemHoverText,
			"sidePanelChevron":            p.SideChevron,
			"cardBackground":              p.CardBackground,
			"textPrimary":                 p.TextPrimary,
			"textMuted":                   p.TextMuted,
			"border":                      p.Border,
		},
		"font": map[string]string{
			"family": "Inter, system-ui, sans-serif",
		},
		"effects": map[string]string{
			"authFormPanelShadow": p.AuthFormShadow,
		},
		"components": map[string]any{
			"topPanelControl":         componentTheme(p.TopControlBg, p.TopControlHover, p.TopControlBorder, "0px", "3px", p.TopControlText, "md"),
			"topPanelDropdownTrigger": componentTheme(p.TopDropdownBg, p.TopDropdownHover, p.TopDropdownBorder, "1px", "3px", p.TopDropdownText, "md"),
			"topPanelProfileTrigger":  componentTheme(p.TopProfileBg, p.TopProfileHover, "transparent", "0px", "3px", p.TopProfileText, "md"),
			"topPanelCreateButton":    componentTheme(p.Primary, p.PrimaryHover, "transparent", "0px", "3px", "#ffffff", "md"),
			"sidePanelSectionLabel":   componentTheme("transparent", "transparent", "transparent", "0px", "3px", p.SideSectionText, "sm"),
			"sidePanelItem":           componentTheme("transparent", p.SideItemHover, "transparent", "0px", "3px", p.SideItemText, "md"),
			"sidePanelItemActive":     componentTheme(p.SideItemActiveBg, p.SideItemActiveBg, "transparent", "0px", "3px", p.SideItemActiveText, "md"),
			"sidePanelSubItem":        componentTheme("transparent", p.SideItemHover, "transparent", "0px", "3px", p.SideSubItemText, "md"),
			"sidePanelSubItemActive":  componentTheme(p.SideItemActiveBg, p.SideItemActiveBg, "transparent", "0px", "3px", p.SideItemActiveText, "md"),
			"primaryButton":           componentTheme(p.Primary, p.PrimaryHover, "transparent", "0px", "3px", "#ffffff", "md"),
			"secondaryButton":         componentTheme(p.Secondary, p.Accent, "transparent", "0px", "3px", p.TextPrimary, "md"),
			"outlineButton":           componentTheme(p.CardBackground, p.Accent, p.InputBorder, "1px", "3px", p.TextPrimary, "md"),
			"destructiveButton": componentTheme(
				"#dc2626",
				"#b91c1c",
				"transparent",
				"0px",
				"3px",
				"#ffffff",
				"md",
			),
			"ghostButton":          componentTheme("transparent", p.Accent, "transparent", "0px", "3px", p.TextPrimary, "md"),
			"tableContainer":       componentTheme(p.CardBackground, p.CardBackground, p.Border, "1px", "3px", p.TextPrimary, "md"),
			"tableHeader":          componentTheme(p.AppBackground, p.AppBackground, p.Border, "1px", "0px", p.TextPrimary, "md"),
			"tableRow":             componentTheme(p.CardBackground, p.AppBackground, p.Border, "1px", "0px", p.TextPrimary, "md"),
			"tableCell":            componentTheme("transparent", "transparent", "transparent", "0px", "0px", p.TextPrimary, "md"),
			"iconContainer":        componentTheme(p.Secondary, p.Accent, "transparent", "0px", "3px", p.TextMuted, "md"),
			"listingItem":          componentTheme(p.CardBackground, p.AppBackground, p.Border, "1px", "3px", p.TextPrimary, "md"),
			"listingItemIcon":      componentTheme(p.Secondary, p.Accent, "transparent", "0px", "3px", p.TextMuted, "md"),
			"listingItemMeta":      componentTheme("transparent", "transparent", "transparent", "0px", "0px", p.TextMuted, "sm"),
			"listingSubContainer":  componentTheme(p.AppBackground, p.AppBackground, p.Border, "1px", "3px", p.TextPrimary, "md"),
			"optionCard":           componentTheme(p.CardBackground, p.Accent, p.Border, "1px", "3px", p.TextPrimary, "md"),
			"card":                 componentTheme(p.CardBackground, p.AppBackground, p.Border, "1px", "3px", p.TextPrimary, "md"),
			"alert":                componentTheme(p.CardBackground, p.CardBackground, p.Border, "1px", "0.5rem", p.TextPrimary, "md"),
			"input":                componentTheme(p.CardBackground, p.AppBackground, p.InputBorder, "1px", "3px", p.TextPrimary, "md"),
			"textarea":             componentTheme(p.CardBackground, p.AppBackground, p.InputBorder, "1px", "3px", p.TextPrimary, "md"),
			"datePicker":           componentTheme(p.CardBackground, p.AppBackground, p.InputBorder, "1px", "3px", p.TextPrimary, "md"),
			"switch":               switchTheme(p.Primary, p.PrimaryHover, p.InputBorder, p.SwitchThumbColor, "transparent", "0px", "999px", "#ffffff", "md"),
			"switchSubContainer":   componentTheme(p.CardBackground, p.AppBackground, p.Border, "1px", "3px", p.TextPrimary, "md"),
			"checkboxSubContainer": componentTheme(p.CardBackground, p.AppBackground, p.Border, "1px", "3px", p.TextPrimary, "md"),
			"badges":               badgeThemes(p.AppBackground, p.AppBackground, p.Border, p.TextPrimary),
		},
	}

	data, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		panic(err)
	}
	return string(data)
}

func componentTheme(background, hoverColor, borderColor, borderThickness, borderRadius, textColor, size string) map[string]string {
	return map[string]string{
		"background":      background,
		"hoverColor":      hoverColor,
		"borderColor":     borderColor,
		"borderThickness": borderThickness,
		"borderRadius":    borderRadius,
		"textColor":       textColor,
		"size":            size,
	}
}

func switchTheme(background, hoverColor, uncheckedBackground, thumbColor, borderColor, borderThickness, borderRadius, textColor, size string) map[string]string {
	theme := componentTheme(background, hoverColor, borderColor, borderThickness, borderRadius, textColor, size)
	theme["uncheckedBackground"] = uncheckedBackground
	theme["thumbColor"] = thumbColor
	return theme
}

func badgeThemes(background, hoverColor, borderColor, textColor string) map[string]any {
	dots := map[string]string{
		"positive":    "#00bc7d",
		"in-progress": "#f0b100",
		"neutral":     "#647084",
		"negative":    "#fb2c36",
	}

	out := make(map[string]any, len(dots))
	for group, dot := range dots {
		theme := componentTheme(background, hoverColor, borderColor, "1px", "3px", textColor, "sm")
		theme["dotColor"] = dot
		out[group] = theme
	}
	return out
}

type systemBrandingTheme struct {
	name     string
	metadata string
	active   bool
}

// SystemBrandingThemeDefault exposes the canonical seeded system branding
// payloads for restore operations.
type SystemBrandingThemeDefault struct {
	Name     string
	Metadata string
	Active   bool
}

type legacySystemBrandingTheme struct {
	name            string
	replacementName string
	metadata        string
}

func systemBrandingThemes() []systemBrandingTheme {
	defaults := shared.SystemBrandingThemeDefaults()
	themes := make([]systemBrandingTheme, 0, len(defaults))
	for _, theme := range defaults {
		themes = append(themes, systemBrandingTheme{
			name:     theme.Name,
			metadata: theme.Metadata,
			active:   theme.Active,
		})
	}
	return themes
}

func SystemBrandingThemeDefaults() []SystemBrandingThemeDefault {
	themes := systemBrandingThemes()
	defaults := make([]SystemBrandingThemeDefault, 0, len(themes))
	for _, theme := range themes {
		defaults = append(defaults, SystemBrandingThemeDefault{
			Name:     theme.name,
			Metadata: theme.metadata,
			Active:   theme.active,
		})
	}
	return defaults
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
