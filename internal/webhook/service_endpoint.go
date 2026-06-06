package webhook

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/maintainerd/auth/internal/platform/apperror"
	"github.com/maintainerd/auth/internal/platform/crypto"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
)

// WebhookEndpointServiceDataResult is the service-layer representation of a
// webhook_endpoints record.
type WebhookEndpointServiceDataResult struct {
	WebhookEndpointUUID uuid.UUID
	TenantID            int64
	URL                 string
	SigningSecret       string // plaintext, only populated on create
	SubscribeAll        bool
	MaxRetries          int
	TimeoutSeconds      int
	Status              string
	Description         string
	LastTriggeredAt     *time.Time
	CreatedAt           time.Time
	UpdatedAt           time.Time
}

// WebhookEndpointServiceListResult holds a paginated list of webhook endpoints.
type WebhookEndpointServiceListResult struct {
	Data       []WebhookEndpointServiceDataResult
	Total      int64
	Page       int
	Limit      int
	TotalPages int
}

// WebhookEndpointService defines business operations on webhook endpoints.
type WebhookEndpointService interface {
	GetAll(ctx context.Context, tenantID int64, status []string, page, limit int, sortBy, sortOrder string) (*WebhookEndpointServiceListResult, error)
	GetByUUID(ctx context.Context, tenantID int64, webhookEndpointUUID uuid.UUID) (*WebhookEndpointServiceDataResult, error)
	Create(ctx context.Context, tenantID int64, url string, subscribeAll bool, maxRetries, timeoutSeconds *int, description, status string) (*WebhookEndpointServiceDataResult, error)
	Update(ctx context.Context, tenantID int64, webhookEndpointUUID uuid.UUID, url string, rotateSecret bool, subscribeAll bool, maxRetries, timeoutSeconds *int, description, status string) (*WebhookEndpointServiceDataResult, error)
	UpdateStatus(ctx context.Context, tenantID int64, webhookEndpointUUID uuid.UUID, status string) (*WebhookEndpointServiceDataResult, error)
	Delete(ctx context.Context, tenantID int64, webhookEndpointUUID uuid.UUID) (*WebhookEndpointServiceDataResult, error)
}

type webhookEndpointService struct {
	webhookEndpointRepo WebhookEndpointRepository
}

// NewWebhookEndpointService creates a new WebhookEndpointService.
func NewWebhookEndpointService(webhookEndpointRepo WebhookEndpointRepository) WebhookEndpointService {
	return &webhookEndpointService{webhookEndpointRepo: webhookEndpointRepo}
}

func toWebhookEndpointServiceDataResult(we *WebhookEndpoint) WebhookEndpointServiceDataResult {
	return WebhookEndpointServiceDataResult{
		WebhookEndpointUUID: we.WebhookEndpointUUID,
		TenantID:            we.TenantID,
		URL:                 we.URL,
		SubscribeAll:        we.SubscribeAll,
		MaxRetries:          we.MaxRetries,
		TimeoutSeconds:      we.TimeoutSeconds,
		Status:              we.Status,
		Description:         we.Description,
		LastTriggeredAt:     we.LastTriggeredAt,
		CreatedAt:           we.CreatedAt,
		UpdatedAt:           we.UpdatedAt,
	}
}

// GetAll retrieves a paginated list of webhook endpoints for a tenant.
func (s *webhookEndpointService) GetAll(ctx context.Context, tenantID int64, status []string, page, limit int, sortBy, sortOrder string) (*WebhookEndpointServiceListResult, error) {
	_, span := otel.Tracer("service").Start(ctx, "webhookEndpoint.list")
	defer span.End()
	span.SetAttributes(attribute.Int64("tenant.id", tenantID))

	result, err := s.webhookEndpointRepo.FindPaginated(WebhookEndpointRepositoryGetFilter{
		TenantID:  &tenantID,
		Status:    status,
		Page:      page,
		Limit:     limit,
		SortBy:    sortBy,
		SortOrder: sortOrder,
	})
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "list webhook endpoints failed")
		return nil, err
	}

	data := make([]WebhookEndpointServiceDataResult, len(result.Data))
	for i, ep := range result.Data {
		data[i] = toWebhookEndpointServiceDataResult(&ep)
	}

	span.SetStatus(codes.Ok, "")
	return &WebhookEndpointServiceListResult{
		Data:       data,
		Total:      result.Total,
		Page:       result.Page,
		Limit:      result.Limit,
		TotalPages: result.TotalPages,
	}, nil
}

// GetByUUID retrieves a single webhook endpoint by UUID, verifying tenant ownership.
func (s *webhookEndpointService) GetByUUID(ctx context.Context, tenantID int64, webhookEndpointUUID uuid.UUID) (*WebhookEndpointServiceDataResult, error) {
	_, span := otel.Tracer("service").Start(ctx, "webhookEndpoint.get")
	defer span.End()
	span.SetAttributes(
		attribute.String("webhook.uuid", webhookEndpointUUID.String()),
		attribute.Int64("tenant.id", tenantID),
	)

	ep, err := s.webhookEndpointRepo.FindByUUIDAndTenantID(webhookEndpointUUID, tenantID)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "get webhook endpoint failed")
		return nil, err
	}
	if ep == nil {
		span.SetStatus(codes.Error, "webhook endpoint not found")
		return nil, apperror.NewNotFoundWithReason("webhook endpoint not found")
	}

	span.SetStatus(codes.Ok, "")
	result := toWebhookEndpointServiceDataResult(ep)
	return &result, nil
}

// Create creates a new webhook endpoint for a tenant.
// The signing secret is generated server-side and returned in the result.
func (s *webhookEndpointService) Create(ctx context.Context, tenantID int64, url string, subscribeAll bool, maxRetries, timeoutSeconds *int, description, status string) (*WebhookEndpointServiceDataResult, error) {
	_, span := otel.Tracer("service").Start(ctx, "webhookEndpoint.create")
	defer span.End()
	span.SetAttributes(attribute.Int64("tenant.id", tenantID))

	rawSecret, secretErr := crypto.GenerateRandomString(48)
	if secretErr != nil {
		span.RecordError(secretErr)
		span.SetStatus(codes.Error, "generate webhook secret failed")
		return nil, apperror.NewInternal("failed to generate webhook signing secret", secretErr)
	}
	encrypted, encErr := crypto.EncryptAtRest(rawSecret)
	if encErr != nil {
		span.RecordError(encErr)
		span.SetStatus(codes.Error, "encrypt webhook secret failed")
		return nil, encErr
	}

	ep := &WebhookEndpoint{
		TenantID:        tenantID,
		URL:             url,
		SecretEncrypted: encrypted,
		SubscribeAll:    subscribeAll,
		Status:          status,
		Description:     description,
		MaxRetries:      3,
		TimeoutSeconds:  30,
	}
	if maxRetries != nil {
		ep.MaxRetries = *maxRetries
	}
	if timeoutSeconds != nil {
		ep.TimeoutSeconds = *timeoutSeconds
	}

	created, err := s.webhookEndpointRepo.Create(ep)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "create webhook endpoint failed")
		return nil, err
	}

	result := toWebhookEndpointServiceDataResult(created)
	result.SigningSecret = rawSecret

	span.SetStatus(codes.Ok, "")
	return &result, nil
}

// Update updates an existing webhook endpoint, verifying tenant ownership.
// When rotateSecret is true, a new signing secret is generated server-side.
func (s *webhookEndpointService) Update(ctx context.Context, tenantID int64, webhookEndpointUUID uuid.UUID, url string, rotateSecret bool, subscribeAll bool, maxRetries, timeoutSeconds *int, description, status string) (*WebhookEndpointServiceDataResult, error) {
	_, span := otel.Tracer("service").Start(ctx, "webhookEndpoint.update")
	defer span.End()
	span.SetAttributes(
		attribute.String("webhook.uuid", webhookEndpointUUID.String()),
		attribute.Int64("tenant.id", tenantID),
	)

	ep, err := s.webhookEndpointRepo.FindByUUIDAndTenantID(webhookEndpointUUID, tenantID)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "find webhook endpoint for update failed")
		return nil, err
	}
	if ep == nil {
		span.SetStatus(codes.Error, "webhook endpoint not found")
		return nil, apperror.NewNotFoundWithReason("webhook endpoint not found")
	}

	ep.URL = url
	ep.SubscribeAll = subscribeAll
	ep.Description = description
	ep.Status = status
	if rotateSecret {
		rawSecret, secretErr := crypto.GenerateRandomString(48)
		if secretErr != nil {
			span.RecordError(secretErr)
			span.SetStatus(codes.Error, "generate webhook secret failed")
			return nil, apperror.NewInternal("failed to generate webhook signing secret", secretErr)
		}
		enc, encErr := crypto.EncryptAtRest(rawSecret)
		if encErr != nil {
			span.RecordError(encErr)
			span.SetStatus(codes.Error, "encrypt webhook secret failed")
			return nil, encErr
		}
		ep.SecretEncrypted = enc
	}
	if maxRetries != nil {
		ep.MaxRetries = *maxRetries
	}
	if timeoutSeconds != nil {
		ep.TimeoutSeconds = *timeoutSeconds
	}

	updated, err := s.webhookEndpointRepo.UpdateByUUID(webhookEndpointUUID, ep)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "update webhook endpoint failed")
		return nil, err
	}

	span.SetStatus(codes.Ok, "")
	result := toWebhookEndpointServiceDataResult(updated)
	return &result, nil
}

// UpdateStatus updates only the status field of a webhook endpoint.
func (s *webhookEndpointService) UpdateStatus(ctx context.Context, tenantID int64, webhookEndpointUUID uuid.UUID, status string) (*WebhookEndpointServiceDataResult, error) {
	_, span := otel.Tracer("service").Start(ctx, "webhookEndpoint.updateStatus")
	defer span.End()
	span.SetAttributes(
		attribute.String("webhook.uuid", webhookEndpointUUID.String()),
		attribute.Int64("tenant.id", tenantID),
	)

	ep, err := s.webhookEndpointRepo.FindByUUIDAndTenantID(webhookEndpointUUID, tenantID)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "find webhook endpoint for status update failed")
		return nil, err
	}
	if ep == nil {
		span.SetStatus(codes.Error, "webhook endpoint not found")
		return nil, apperror.NewNotFoundWithReason("webhook endpoint not found")
	}

	ep.Status = status

	updated, err := s.webhookEndpointRepo.UpdateByUUID(webhookEndpointUUID, ep)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "update webhook endpoint status failed")
		return nil, err
	}

	span.SetStatus(codes.Ok, "")
	result := toWebhookEndpointServiceDataResult(updated)
	return &result, nil
}

// Delete deletes a webhook endpoint, verifying tenant ownership first.
func (s *webhookEndpointService) Delete(ctx context.Context, tenantID int64, webhookEndpointUUID uuid.UUID) (*WebhookEndpointServiceDataResult, error) {
	_, span := otel.Tracer("service").Start(ctx, "webhookEndpoint.delete")
	defer span.End()
	span.SetAttributes(
		attribute.String("webhook.uuid", webhookEndpointUUID.String()),
		attribute.Int64("tenant.id", tenantID),
	)

	ep, err := s.webhookEndpointRepo.FindByUUIDAndTenantID(webhookEndpointUUID, tenantID)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "find webhook endpoint for delete failed")
		return nil, err
	}
	if ep == nil {
		span.SetStatus(codes.Error, "webhook endpoint not found")
		return nil, apperror.NewNotFoundWithReason("webhook endpoint not found")
	}

	if err := s.webhookEndpointRepo.DeleteByUUID(webhookEndpointUUID); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "delete webhook endpoint failed")
		return nil, err
	}

	span.SetStatus(codes.Ok, "")
	result := toWebhookEndpointServiceDataResult(ep)
	return &result, nil
}
