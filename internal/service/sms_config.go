package service

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/maintainerd/auth/internal/apperror"
	"github.com/maintainerd/auth/internal/model"
	"github.com/maintainerd/auth/internal/repository"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
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
	smsConfigRepo repository.SMSConfigRepository
}

// NewSMSConfigService creates a new SMSConfigService.
func NewSMSConfigService(smsConfigRepo repository.SMSConfigRepository) SMSConfigService {
	return &smsConfigService{smsConfigRepo: smsConfigRepo}
}

func toSMSConfigServiceDataResult(sc *model.SMSConfig) *SMSConfigServiceDataResult {
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
		config = &model.SMSConfig{TenantID: tenantID, Status: model.StatusActive}
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
