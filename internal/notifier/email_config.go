package notifier

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/maintainerd/auth/internal/platform/apperror"
	"github.com/maintainerd/auth/internal/shared"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// EmailConfigServiceDataResult is the service-layer representation of an
// email_config record.
type EmailConfigServiceDataResult struct {
	EmailConfigUUID uuid.UUID
	Provider        string
	Host            string
	Port            int
	Username        string
	FromAddress     string
	FromName        string
	ReplyTo         string
	Encryption      string
	TestMode        bool
	Status          string
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

// EmailConfigService defines business operations on the tenant email delivery
// configuration.
type EmailConfigService interface {
	Get(ctx context.Context, tenantID int64) (*EmailConfigServiceDataResult, error)
	Update(ctx context.Context, tenantID int64, provider, host string, port int, username, password, fromAddress, fromName, replyTo, encryption string, testMode *bool) (*EmailConfigServiceDataResult, error)
}

type emailConfigService struct {
	emailConfigRepo EmailConfigRepository
}

// NewEmailConfigService creates a new EmailConfigService.
func NewEmailConfigService(emailConfigRepo EmailConfigRepository) EmailConfigService {
	return &emailConfigService{emailConfigRepo: emailConfigRepo}
}

func toEmailConfigServiceDataResult(ec *EmailConfig) *EmailConfigServiceDataResult {
	return &EmailConfigServiceDataResult{
		EmailConfigUUID: ec.EmailConfigUUID,
		Provider:        ec.Provider,
		Host:            ec.Host,
		Port:            ec.Port,
		Username:        ec.Username,
		FromAddress:     ec.FromAddress,
		FromName:        ec.FromName,
		ReplyTo:         ec.ReplyTo,
		Encryption:      ec.Encryption,
		TestMode:        ec.TestMode,
		Status:          ec.Status,
		CreatedAt:       ec.CreatedAt,
		UpdatedAt:       ec.UpdatedAt,
	}
}

// Get retrieves the email config for a tenant, returning not-found if none
// exists.
func (s *emailConfigService) Get(ctx context.Context, tenantID int64) (*EmailConfigServiceDataResult, error) {
	_, span := otel.Tracer("service").Start(ctx, "emailConfig.get")
	defer span.End()
	span.SetAttributes(attribute.Int64("tenant.id", tenantID))

	config, err := s.emailConfigRepo.FindByTenantID(tenantID)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "get email config failed")
		return nil, err
	}
	if config == nil {
		span.SetStatus(codes.Error, "email config not found")
		return nil, apperror.NewNotFoundWithReason("email configuration not found")
	}
	span.SetStatus(codes.Ok, "")
	return toEmailConfigServiceDataResult(config), nil
}

// Update upserts the email config for a tenant. The password field is only
// written when a non-empty value is provided (preserves existing on blank).
func (s *emailConfigService) Update(ctx context.Context, tenantID int64, provider, host string, port int, username, password, fromAddress, fromName, replyTo, encryption string, testMode *bool) (*EmailConfigServiceDataResult, error) {
	_, span := otel.Tracer("service").Start(ctx, "emailConfig.update")
	defer span.End()
	span.SetAttributes(attribute.Int64("tenant.id", tenantID))

	config, err := s.emailConfigRepo.FindByTenantID(tenantID)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "find email config for update failed")
		return nil, err
	}

	if config == nil {
		config = &EmailConfig{TenantID: tenantID, Status: shared.StatusActive}
	}

	config.Provider = provider
	config.Host = host
	config.Port = port
	config.Username = username
	config.FromAddress = fromAddress
	config.FromName = fromName
	config.ReplyTo = replyTo
	config.Encryption = encryption

	if password != "" {
		config.PasswordEncrypted = password
	}
	if testMode != nil {
		config.TestMode = *testMode
	}

	updated, err := s.emailConfigRepo.CreateOrUpdate(config)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "update email config failed")
		return nil, err
	}

	span.SetStatus(codes.Ok, "")
	return toEmailConfigServiceDataResult(updated), nil
}

// EmailConfig holds tenant-level SMTP/SES/SendGrid delivery configuration.
type EmailConfig struct {
	EmailConfigID     int64          `gorm:"column:email_config_id;primaryKey;autoIncrement" json:"email_config_id"`
	EmailConfigUUID   uuid.UUID      `gorm:"column:email_config_uuid;type:uuid;uniqueIndex;not null" json:"email_config_uuid"`
	TenantID          int64          `gorm:"column:tenant_id;not null" json:"tenant_id"`
	Provider          string         `gorm:"column:provider;type:varchar(50);not null" json:"provider"`
	Host              string         `gorm:"column:host;type:varchar(255)" json:"host"`
	Port              int            `gorm:"column:port" json:"port"`
	Username          string         `gorm:"column:username;type:varchar(255)" json:"username"`
	PasswordEncrypted string         `gorm:"column:password_encrypted;type:text" json:"-"`
	FromAddress       string         `gorm:"column:from_address;type:varchar(255);not null" json:"from_address"`
	FromName          string         `gorm:"column:from_name;type:varchar(255)" json:"from_name"`
	ReplyTo           string         `gorm:"column:reply_to;type:varchar(255)" json:"reply_to"`
	Encryption        string         `gorm:"column:encryption;type:varchar(20)" json:"encryption"`
	TestMode          bool           `gorm:"column:test_mode;not null;default:false" json:"test_mode"`
	Status            string         `gorm:"column:status;type:varchar(20);not null;default:'active'" json:"status"`
	Metadata          datatypes.JSON `gorm:"column:metadata;type:jsonb;default:'{}'" json:"metadata"`
	CreatedBy         *int64         `gorm:"column:created_by" json:"created_by,omitempty"`
	UpdatedBy         *int64         `gorm:"column:updated_by" json:"updated_by,omitempty"`
	CreatedAt         time.Time      `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	UpdatedAt         time.Time      `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
	DeletedAt         gorm.DeletedAt `gorm:"column:deleted_at;index" json:"deleted_at,omitempty"`

	// Relationships
}

// TableName returns the database table name for EmailConfig.
func (EmailConfig) TableName() string {
	return "email_config"
}

// BeforeCreate sets a new UUID on the EmailConfig before it is inserted into
// the database if one has not already been assigned.
func (ec *EmailConfig) BeforeCreate(tx *gorm.DB) error {
	if ec.EmailConfigUUID == uuid.Nil {
		ec.EmailConfigUUID = uuid.New()
	}
	return nil
}
