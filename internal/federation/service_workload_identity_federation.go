package federation

import (
	"context"
	"errors"
	"log/slog"
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

// WorkloadIdentityFederationListFilter carries the resolved list parameters.
type WorkloadIdentityFederationListFilter struct {
	Name      *string
	IsActive  *bool
	Page      int
	Limit     int
	SortBy    string
	SortOrder string
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
	IsActive         *bool
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
	// IsActive nil means "leave unchanged".
	IsActive    *bool
	ActorUserID *int64
}

// WorkloadIdentityFederationService defines business operations on workload
// identity federations: tenant-scoped CRUD plus the token-exchange flow that
// converts an external OIDC token into a platform access token.
type WorkloadIdentityFederationService interface {
	GetAll(ctx context.Context, tenantID int64, filter WorkloadIdentityFederationListFilter) (*WorkloadIdentityFederationServiceListResult, error)
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
		IsActive:                       f.IsActive == nil || *f.IsActive,
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

// resolveClientUUIDs maps every distinct client_id in the page to its public UUID in
// a single query. A client that no longer resolves is absent from the map rather than
// an error: a soft-deleted client is a legitimate state, and the response says so by
// carrying the zero UUID for that row only.
func (s *workloadIdentityFederationService) resolveClientUUIDs(
	federations []WorkloadIdentityFederation,
) (map[int64]uuid.UUID, error) {
	if len(federations) == 0 {
		return map[int64]uuid.UUID{}, nil
	}

	ids := make([]int64, 0, len(federations))
	seen := map[int64]struct{}{}
	for i := range federations {
		id := federations[i].ClientID
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		ids = append(ids, id)
	}

	var clients []Client
	if err := s.db.Where("client_id IN ?", ids).Find(&clients).Error; err != nil {
		return nil, err
	}

	out := make(map[int64]uuid.UUID, len(clients))
	for i := range clients {
		out[clients[i].ClientID] = clients[i].ClientUUID
	}
	return out, nil
}

// GetAll retrieves a paginated list of federations for a tenant.
func (s *workloadIdentityFederationService) GetAll(ctx context.Context, tenantID int64, filter WorkloadIdentityFederationListFilter) (*WorkloadIdentityFederationServiceListResult, error) {
	_, span := otel.Tracer("service").Start(ctx, "workloadIdentityFederation.list")
	defer span.End()
	span.SetAttributes(attribute.Int64("tenant.id", tenantID))

	result, err := s.repo.FindPaginated(WorkloadIdentityFederationRepositoryGetFilter{
		TenantID:  &tenantID,
		Name:      filter.Name,
		IsActive:  filter.IsActive,
		Page:      filter.Page,
		Limit:     filter.Limit,
		SortBy:    filter.SortBy,
		SortOrder: filter.SortOrder,
	})
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "list failed")
		return nil, err
	}

	// One batched lookup for the whole page. This was a query per row, and each
	// failure was swallowed — leaving a zero UUID in the response, so a consumer
	// could not tell "client was deleted" from "the database errored".
	clientUUIDs, err := s.resolveClientUUIDs(result.Data)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "client lookup failed")
		return nil, apperror.NewInternal("failed to resolve the clients for these federations", err)
	}

	data := make([]WorkloadIdentityFederationServiceDataResult, len(result.Data))
	for i := range result.Data {
		f := result.Data[i]
		data[i] = toServiceResult(&f, clientUUIDs[f.ClientID])
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

	clientUUID, err := s.clientUUIDFor(f.ClientID)
	if err != nil {
		span.RecordError(err)
		return nil, apperror.NewInternal("failed to resolve the client for this federation", err)
	}

	result := toServiceResult(f, clientUUID)
	span.SetStatus(codes.Ok, "")
	return &result, nil
}

// clientUUIDFor returns the client's public UUID, or the zero UUID when the client no
// longer resolves (soft-deleted). A genuine DB error is returned rather than
// flattened into a zero UUID, which used to make the two indistinguishable.
func (s *workloadIdentityFederationService) clientUUIDFor(clientID int64) (uuid.UUID, error) {
	client, err := s.resolveClientByID(clientID)
	if err != nil {
		return uuid.Nil, err
	}
	if client == nil {
		return uuid.Nil, nil
	}
	return client.ClientUUID, nil
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
		// The underlying go-oidc error embeds the RAW HTTP RESPONSE BODY of the
		// fetched URL, which turned this into an arbitrary-GET-with-body-echo
		// primitive from the server's network position. Log it, never return it.
		slog.Warn("workload identity federation issuer probe failed",
			"issuer_url", in.IssuerURL, "error", err)
		return nil, apperror.NewValidation("issuer_url is not a reachable OIDC issuer")
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
			// The underlying go-oidc error embeds the RAW HTTP RESPONSE BODY of the
			// fetched URL, which turned this into an arbitrary-GET-with-body-echo
			// primitive from the server's network position. Log it, never return it.
			slog.Warn("workload identity federation issuer probe failed",
				"issuer_url", in.IssuerURL, "error", err)
			return nil, apperror.NewValidation("issuer_url is not a reachable OIDC issuer")
		}
	}

	subjectClaim := in.SubjectClaim
	if subjectClaim == "" {
		subjectClaim = "sub"
	}

	// An explicit column map, not the loaded struct. GORM's struct-based Updates
	// skips every zero-valued field, which made `is_active = false` and an emptied
	// description unwritable — so a live trust rule could not be disabled or have its
	// description cleared, and a PUT silently behaved differently per field.
	updates := map[string]any{
		"name":              in.Name,
		"description":       ptr.PtrOrNil(in.Description),
		"issuer_url":        in.IssuerURL,
		"audience":          in.Audience,
		"subject_claim":     subjectClaim,
		"subject_pattern":   in.SubjectPattern,
		"allowed_scopes":    pq.StringArray(in.AllowedScopes),
		"attribute_mapping": encodeAttributeMapping(in.AttributeMapping),
		"updated_by":        in.ActorUserID,
	}
	if in.IsActive != nil {
		updates["is_active"] = *in.IsActive
	}

	updated, err := s.repo.UpdateByUUID(federationUUID, updates)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "update failed")
		if isUniqueViolation(err) {
			return nil, apperror.NewConflict("a workload identity federation with this name already exists")
		}
		return nil, err
	}

	clientUUID, err := s.clientUUIDFor(updated.ClientID)
	if err != nil {
		span.RecordError(err)
		return nil, apperror.NewInternal("failed to resolve the client for this federation", err)
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

	clientUUID, err := s.clientUUIDFor(f.ClientID)
	if err != nil {
		span.RecordError(err)
		return nil, apperror.NewInternal("failed to resolve the client for this federation", err)
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
