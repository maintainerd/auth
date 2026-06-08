package notifier

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/maintainerd/auth/internal/platform/crypto"
	"github.com/maintainerd/auth/internal/shared"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
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
	LogoURL         string
	TestMode        bool
	Status          string
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

// EmailConfigService defines business operations on the tenant email delivery
// configuration.
type EmailConfigService interface {
	Get(ctx context.Context, tenantID int64) (*EmailConfigServiceDataResult, error)
	Update(ctx context.Context, tenantID int64, provider, host string, port int, username, password, fromAddress, fromName, replyTo, encryption, logoURL string, testMode *bool) (*EmailConfigServiceDataResult, error)
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
		LogoURL:         ec.LogoURL,
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
		span.SetStatus(codes.Ok, "")
		return &EmailConfigServiceDataResult{
			Provider: "smtp",
			Status:   shared.StatusActive,
		}, nil
	}
	span.SetStatus(codes.Ok, "")
	return toEmailConfigServiceDataResult(config), nil
}

// Update upserts the email config for a tenant. The password field is only
// written when a non-empty value is provided (preserves existing on blank).
func (s *emailConfigService) Update(ctx context.Context, tenantID int64, provider, host string, port int, username, password, fromAddress, fromName, replyTo, encryption, logoURL string, testMode *bool) (*EmailConfigServiceDataResult, error) {
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
	config.LogoURL = logoURL

	if password != "" {
		enc, encErr := crypto.EncryptAtRest(password)
		if encErr != nil {
			span.RecordError(encErr)
			span.SetStatus(codes.Error, "encrypt email password failed")
			return nil, encErr
		}
		config.PasswordEncrypted = enc
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
