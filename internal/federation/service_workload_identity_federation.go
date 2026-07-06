package federation

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
	"github.com/maintainerd/maintainerd-auth/internal/platform/apperror"
	"github.com/maintainerd/maintainerd-auth/internal/platform/ptr"
	"github.com/maintainerd/maintainerd-auth/internal/shared"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// WorkloadIdentityFederationServiceDataResult is the service-layer view of a
// workload identity federation, enriched with the mapped client's UUID.
type WorkloadIdentityFederationServiceDataResult struct {
	WorkloadIdentityFederationUUID uuid.UUID
	ClientUUID                     uuid.UUID
	Name                           string
	Description                    string
	IssuerURL                      string
	Audience                       string
	SubjectClaim                   string
	SubjectPattern                 string
	AllowedScopes                  []string
	AttributeMapping               map[string]string
	IsActive                       bool
	CreatedAt                      time.Time
	UpdatedAt                      time.Time
}

// WorkloadIdentityFederationServiceListResult holds a paginated list.
type WorkloadIdentityFederationServiceListResult struct {
	Data       []WorkloadIdentityFederationServiceDataResult
	Total      int64
	Page       int
	Limit      int
	TotalPages int
}

// WorkloadIdentityFederationCreateInput carries the resolved create parameters.
type WorkloadIdentityFederationCreateInput struct {
	ClientUUID       uuid.UUID
	Name             string
	Description      string
	IssuerURL        string
	Audience         string
	SubjectClaim     string
	SubjectPattern   string
	AllowedScopes    []string
	AttributeMapping map[string]string
	IsActive         bool
	ActorUserID      *int64
}

// WorkloadIdentityFederationUpdateInput carries the resolved update parameters.
type WorkloadIdentityFederationUpdateInput struct {
	Name             string
	Description      string
	IssuerURL        string
	Audience         string
	SubjectClaim     string
	SubjectPattern   string
	AllowedScopes    []string
	AttributeMapping map[string]string
	IsActive         bool
	ActorUserID      *int64
}

// WorkloadIdentityFederationService defines business operations on workload
// identity federations: tenant-scoped CRUD plus the token-exchange flow that
// converts an external OIDC token into a platform access token.
type WorkloadIdentityFederationService interface {
	GetAll(ctx context.Context, tenantID int64, page, limit int, sortBy, sortOrder string) (*WorkloadIdentityFederationServiceListResult, error)
	GetByUUID(ctx context.Context, tenantID int64, federationUUID uuid.UUID) (*WorkloadIdentityFederationServiceDataResult, error)
	Create(ctx context.Context, tenantID int64, in WorkloadIdentityFederationCreateInput) (*WorkloadIdentityFederationServiceDataResult, error)
	Update(ctx context.Context, tenantID int64, federationUUID uuid.UUID, in WorkloadIdentityFederationUpdateInput) (*WorkloadIdentityFederationServiceDataResult, error)
	Delete(ctx context.Context, tenantID int64, federationUUID uuid.UUID) (*WorkloadIdentityFederationServiceDataResult, error)

	// ExchangeWorkloadToken validates an external OIDC subject token against a
	// configured federation and issues a platform access token. It returns
	// (nil, nil) when no federation trusts the token's issuer, signalling the
	// caller to fall back to the standard RFC 8693 token-exchange path.
	ExchangeWorkloadToken(ctx context.Context, in WorkloadExchangeInput) (*WorkloadExchangeResult, *apperror.OAuthError)
}

type workloadIdentityFederationService struct {
	db       *gorm.DB
	repo     WorkloadIdentityFederationRepository
	auditor  ExchangeAuditor
	provider *providerCache
}

// NewWorkloadIdentityFederationService creates a new service. auditor may be nil.
func NewWorkloadIdentityFederationService(
	db *gorm.DB,
	repo WorkloadIdentityFederationRepository,
	auditor ExchangeAuditor,
) WorkloadIdentityFederationService {
	return &workloadIdentityFederationService{
		db:       db,
		repo:     repo,
		auditor:  auditor,
		provider: newProviderCache(),
	}
}

func toServiceResult(f *WorkloadIdentityFederation, clientUUID uuid.UUID) WorkloadIdentityFederationServiceDataResult {
	return WorkloadIdentityFederationServiceDataResult{
		WorkloadIdentityFederationUUID: f.WorkloadIdentityFederationUUID,
		ClientUUID:                     clientUUID,
		Name:                           f.Name,
		Description:                    derefString(f.Description),
		IssuerURL:                      f.IssuerURL,
		Audience:                       f.Audience,
		SubjectClaim:                   f.SubjectClaim,
		SubjectPattern:                 f.SubjectPattern,
		AllowedScopes:                  []string(f.AllowedScopes),
		AttributeMapping:               decodeAttributeMapping(f.AttributeMapping),
		IsActive:                       f.IsActive,
		CreatedAt:                      f.CreatedAt,
		UpdatedAt:                      f.UpdatedAt,
	}
}

// resolveClientByUUID loads the mapped client scoped to the tenant.
func (s *workloadIdentityFederationService) resolveClientByUUID(tenantID int64, clientUUID uuid.UUID) (*Client, error) {
	var client Client
	err := s.db.Where("client_uuid = ? AND tenant_id = ?", clientUUID, tenantID).First(&client).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &client, nil
}

// resolveClientByID loads a client by internal id (used to enrich responses).
func (s *workloadIdentityFederationService) resolveClientByID(clientID int64) (*Client, error) {
	var client Client
	err := s.db.Where("client_id = ?", clientID).First(&client).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &client, nil
}

// GetAll retrieves a paginated list of federations for a tenant.
func (s *workloadIdentityFederationService) GetAll(ctx context.Context, tenantID int64, page, limit int, sortBy, sortOrder string) (*WorkloadIdentityFederationServiceListResult, error) {
	_, span := otel.Tracer("service").Start(ctx, "workloadIdentityFederation.list")
	defer span.End()
	span.SetAttributes(attribute.Int64("tenant.id", tenantID))

	result, err := s.repo.FindPaginated(WorkloadIdentityFederationRepositoryGetFilter{
		TenantID:  &tenantID,
		Page:      page,
		Limit:     limit,
		SortBy:    sortBy,
		SortOrder: sortOrder,
	})
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "list failed")
		return nil, err
	}

	// Cache client UUID lookups within this page.
	clientUUIDs := map[int64]uuid.UUID{}
	data := make([]WorkloadIdentityFederationServiceDataResult, len(result.Data))
	for i := range result.Data {
		f := result.Data[i]
		cu, ok := clientUUIDs[f.ClientID]
		if !ok {
			if client, cerr := s.resolveClientByID(f.ClientID); cerr == nil && client != nil {
				cu = client.ClientUUID
			}
			clientUUIDs[f.ClientID] = cu
		}
		data[i] = toServiceResult(&f, cu)
	}

	span.SetStatus(codes.Ok, "")
	return &WorkloadIdentityFederationServiceListResult{
		Data:       data,
		Total:      result.Total,
		Page:       result.Page,
		Limit:      result.Limit,
		TotalPages: result.TotalPages,
	}, nil
}

// GetByUUID retrieves a single federation by UUID, verifying tenant ownership.
func (s *workloadIdentityFederationService) GetByUUID(ctx context.Context, tenantID int64, federationUUID uuid.UUID) (*WorkloadIdentityFederationServiceDataResult, error) {
	_, span := otel.Tracer("service").Start(ctx, "workloadIdentityFederation.get")
	defer span.End()
	span.SetAttributes(attribute.Int64("tenant.id", tenantID))

	f, err := s.repo.FindByUUIDAndTenantID(federationUUID, tenantID)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "get failed")
		return nil, err
	}
	if f == nil {
		return nil, apperror.NewNotFoundWithReason("workload identity federation not found")
	}

	var clientUUID uuid.UUID
	if client, cerr := s.resolveClientByID(f.ClientID); cerr == nil && client != nil {
		clientUUID = client.ClientUUID
	}

	result := toServiceResult(f, clientUUID)
	span.SetStatus(codes.Ok, "")
	return &result, nil
}

// Create validates the issuer via OIDC discovery, then persists the federation.
func (s *workloadIdentityFederationService) Create(ctx context.Context, tenantID int64, in WorkloadIdentityFederationCreateInput) (*WorkloadIdentityFederationServiceDataResult, error) {
	_, span := otel.Tracer("service").Start(ctx, "workloadIdentityFederation.create")
	defer span.End()
	span.SetAttributes(attribute.Int64("tenant.id", tenantID))

	client, err := s.resolveClientByUUID(tenantID, in.ClientUUID)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "client lookup failed")
		return nil, apperror.NewInternal("failed to resolve client", err)
	}
	if client == nil {
		return nil, apperror.NewNotFoundWithReason("client not found")
	}
	if client.Status != shared.StatusActive {
		return nil, apperror.NewValidation("client is not active")
	}

	// Validate the issuer is a reachable OIDC issuer (extracts the JWKS URI).
	if err := s.probeIssuer(ctx, in.IssuerURL); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "issuer probe failed")
		return nil, apperror.NewValidation("issuer_url is not a reachable OIDC issuer: " + err.Error())
	}

	subjectClaim := in.SubjectClaim
	if subjectClaim == "" {
		subjectClaim = "sub"
	}

	f := &WorkloadIdentityFederation{
		TenantID:         tenantID,
		ClientID:         client.ClientID,
		Name:             in.Name,
		Description:      ptr.PtrOrNil(in.Description),
		IssuerURL:        in.IssuerURL,
		Audience:         in.Audience,
		SubjectClaim:     subjectClaim,
		SubjectPattern:   in.SubjectPattern,
		AllowedScopes:    pq.StringArray(in.AllowedScopes),
		AttributeMapping: encodeAttributeMapping(in.AttributeMapping),
		IsActive:         in.IsActive,
		CreatedBy:        in.ActorUserID,
		UpdatedBy:        in.ActorUserID,
	}

	created, err := s.repo.Create(f)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "create failed")
		if isUniqueViolation(err) {
			return nil, apperror.NewConflict("a workload identity federation with this name already exists")
		}
		return nil, err
	}

	result := toServiceResult(created, client.ClientUUID)
	span.SetStatus(codes.Ok, "")
	return &result, nil
}

// Update mutates an existing federation, verifying tenant ownership.
func (s *workloadIdentityFederationService) Update(ctx context.Context, tenantID int64, federationUUID uuid.UUID, in WorkloadIdentityFederationUpdateInput) (*WorkloadIdentityFederationServiceDataResult, error) {
	_, span := otel.Tracer("service").Start(ctx, "workloadIdentityFederation.update")
	defer span.End()
	span.SetAttributes(attribute.Int64("tenant.id", tenantID))

	f, err := s.repo.FindByUUIDAndTenantID(federationUUID, tenantID)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "find for update failed")
		return nil, err
	}
	if f == nil {
		return nil, apperror.NewNotFoundWithReason("workload identity federation not found")
	}

	// Re-probe only when the issuer changed to avoid an unnecessary network call.
	if in.IssuerURL != f.IssuerURL {
		if err := s.probeIssuer(ctx, in.IssuerURL); err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, "issuer probe failed")
			return nil, apperror.NewValidation("issuer_url is not a reachable OIDC issuer: " + err.Error())
		}
	}

	subjectClaim := in.SubjectClaim
	if subjectClaim == "" {
		subjectClaim = "sub"
	}

	f.Name = in.Name
	f.Description = ptr.PtrOrNil(in.Description)
	f.IssuerURL = in.IssuerURL
	f.Audience = in.Audience
	f.SubjectClaim = subjectClaim
	f.SubjectPattern = in.SubjectPattern
	f.AllowedScopes = pq.StringArray(in.AllowedScopes)
	f.AttributeMapping = encodeAttributeMapping(in.AttributeMapping)
	f.IsActive = in.IsActive
	f.UpdatedBy = in.ActorUserID

	updated, err := s.repo.UpdateByUUID(federationUUID, f)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "update failed")
		if isUniqueViolation(err) {
			return nil, apperror.NewConflict("a workload identity federation with this name already exists")
		}
		return nil, err
	}

	var clientUUID uuid.UUID
	if client, cerr := s.resolveClientByID(updated.ClientID); cerr == nil && client != nil {
		clientUUID = client.ClientUUID
	}

	result := toServiceResult(updated, clientUUID)
	span.SetStatus(codes.Ok, "")
	return &result, nil
}

// Delete soft-deletes a federation, verifying tenant ownership.
func (s *workloadIdentityFederationService) Delete(ctx context.Context, tenantID int64, federationUUID uuid.UUID) (*WorkloadIdentityFederationServiceDataResult, error) {
	_, span := otel.Tracer("service").Start(ctx, "workloadIdentityFederation.delete")
	defer span.End()
	span.SetAttributes(attribute.Int64("tenant.id", tenantID))

	f, err := s.repo.FindByUUIDAndTenantID(federationUUID, tenantID)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "find for delete failed")
		return nil, err
	}
	if f == nil {
		return nil, apperror.NewNotFoundWithReason("workload identity federation not found")
	}

	if err := s.repo.DeleteByUUID(federationUUID); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "delete failed")
		return nil, err
	}

	var clientUUID uuid.UUID
	if client, cerr := s.resolveClientByID(f.ClientID); cerr == nil && client != nil {
		clientUUID = client.ClientUUID
	}

	result := toServiceResult(f, clientUUID)
	span.SetStatus(codes.Ok, "")
	return &result, nil
}

// encodeAttributeMapping serialises a string map to JSONB, defaulting to '{}'.
func encodeAttributeMapping(m map[string]string) datatypes.JSON {
	if len(m) == 0 {
		return datatypes.JSON([]byte(`{}`))
	}
	raw, err := jsonMarshal(m)
	if err != nil {
		return datatypes.JSON([]byte(`{}`))
	}
	return datatypes.JSON(raw)
}

// decodeAttributeMapping parses the JSONB attribute mapping into a string map.
func decodeAttributeMapping(j datatypes.JSON) map[string]string {
	out := map[string]string{}
	if len(j) == 0 {
		return out
	}
	_ = jsonUnmarshal([]byte(j), &out)
	return out
}
