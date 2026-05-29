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

// SMSConfigServiceDataResult is the service-layer representation of an
// sms_config record.
type SMSConfigServiceDataResult struct {
	SMSConfigUUID uuid.UUID
	Provider      string
	AccountSID    string
	FromNumber    string
	SenderID      string
	TestMode      bool
	Status        string
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

// SMSConfigService defines business operations on the tenant SMS delivery
// configuration.
type SMSConfigService interface {
	Get(ctx context.Context, tenantID int64) (*SMSConfigServiceDataResult, error)
	Update(ctx context.Context, tenantID int64, provider, accountSID, authToken, fromNumber, senderID string, testMode *bool) (*SMSConfigServiceDataResult, error)
}

type smsConfigService struct {
	smsConfigRepo SMSConfigRepository
}

// NewSMSConfigService creates a new SMSConfigService.
func NewSMSConfigService(smsConfigRepo SMSConfigRepository) SMSConfigService {
	return &smsConfigService{smsConfigRepo: smsConfigRepo}
}

func toSMSConfigServiceDataResult(sc *SMSConfig) *SMSConfigServiceDataResult {
	return &SMSConfigServiceDataResult{
		SMSConfigUUID: sc.SMSConfigUUID,
		Provider:      sc.Provider,
		AccountSID:    sc.AccountSID,
		FromNumber:    sc.FromNumber,
		SenderID:      sc.SenderID,
		TestMode:      sc.TestMode,
		Status:        sc.Status,
		CreatedAt:     sc.CreatedAt,
		UpdatedAt:     sc.UpdatedAt,
	}
}

// Get retrieves the SMS config for a tenant, returning not-found if none
// exists.
func (s *smsConfigService) Get(ctx context.Context, tenantID int64) (*SMSConfigServiceDataResult, error) {
	_, span := otel.Tracer("service").Start(ctx, "smsConfig.get")
	defer span.End()
	span.SetAttributes(attribute.Int64("tenant.id", tenantID))

	config, err := s.smsConfigRepo.FindByTenantID(tenantID)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "get sms config failed")
		return nil, err
	}
	if config == nil {
		span.SetStatus(codes.Error, "sms config not found")
		return nil, apperror.NewNotFoundWithReason("sms configuration not found")
	}
	span.SetStatus(codes.Ok, "")
	return toSMSConfigServiceDataResult(config), nil
}

// Update upserts the SMS config for a tenant. The auth token is only written
// when a non-empty value is provided (preserves existing on blank).
func (s *smsConfigService) Update(ctx context.Context, tenantID int64, provider, accountSID, authToken, fromNumber, senderID string, testMode *bool) (*SMSConfigServiceDataResult, error) {
	_, span := otel.Tracer("service").Start(ctx, "smsConfig.update")
	defer span.End()
	span.SetAttributes(attribute.Int64("tenant.id", tenantID))

	config, err := s.smsConfigRepo.FindByTenantID(tenantID)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "find sms config for update failed")
		return nil, err
	}

	if config == nil {
		config = &SMSConfig{TenantID: tenantID, Status: shared.StatusActive}
	}

	config.Provider = provider
	config.AccountSID = accountSID
	config.FromNumber = fromNumber
	config.SenderID = senderID

	if authToken != "" {
		config.AuthTokenEncrypted = authToken
	}
	if testMode != nil {
		config.TestMode = *testMode
	}

	updated, err := s.smsConfigRepo.CreateOrUpdate(config)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "update sms config failed")
		return nil, err
	}

	span.SetStatus(codes.Ok, "")
	return toSMSConfigServiceDataResult(updated), nil
}

// SMSConfig holds tenant-level SMS delivery configuration (Twilio, SNS,
// Vonage, MessageBird, etc.).
type SMSConfig struct {
	SMSConfigID        int64          `gorm:"column:sms_config_id;primaryKey;autoIncrement" json:"sms_config_id"`
	SMSConfigUUID      uuid.UUID      `gorm:"column:sms_config_uuid;type:uuid;uniqueIndex;not null" json:"sms_config_uuid"`
	TenantID           int64          `gorm:"column:tenant_id;not null" json:"tenant_id"`
	Provider           string         `gorm:"column:provider;type:varchar(50);not null" json:"provider"`
	AccountSID         string         `gorm:"column:account_sid;type:varchar(255)" json:"account_sid"`
	AuthTokenEncrypted string         `gorm:"column:auth_token_encrypted;type:text" json:"-"`
	FromNumber         string         `gorm:"column:from_number;type:varchar(50)" json:"from_number"`
	SenderID           string         `gorm:"column:sender_id;type:varchar(50)" json:"sender_id"`
	TestMode           bool           `gorm:"column:test_mode;not null;default:false" json:"test_mode"`
	Status             string         `gorm:"column:status;type:varchar(20);not null;default:'active'" json:"status"`
	Metadata           datatypes.JSON `gorm:"column:metadata;type:jsonb;default:'{}'" json:"metadata"`
	CreatedBy          *int64         `gorm:"column:created_by" json:"created_by,omitempty"`
	UpdatedBy          *int64         `gorm:"column:updated_by" json:"updated_by,omitempty"`
	CreatedAt          time.Time      `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	UpdatedAt          time.Time      `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
	DeletedAt          gorm.DeletedAt `gorm:"column:deleted_at;index" json:"deleted_at,omitempty"`

	// Relationships
}

// TableName returns the database table name for SMSConfig.
func (SMSConfig) TableName() string {
	return "sms_config"
}

// BeforeCreate sets a new UUID on the SMSConfig before it is inserted into the
// database if one has not already been assigned.
func (sc *SMSConfig) BeforeCreate(tx *gorm.DB) error {
	if sc.SMSConfigUUID == uuid.Nil {
		sc.SMSConfigUUID = uuid.New()
	}
	return nil
}
