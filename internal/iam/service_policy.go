package iam

import (
	"context"
	"encoding/json"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/maintainerd/maintainerd-auth/internal/authevent"
	"github.com/maintainerd/maintainerd-auth/internal/event"
	"github.com/maintainerd/maintainerd-auth/internal/platform/apperror"
	"github.com/maintainerd/maintainerd-auth/internal/platform/mrn"
	"github.com/maintainerd/maintainerd-auth/internal/platform/ptr"
	"github.com/maintainerd/maintainerd-auth/internal/tenant"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type PolicyServiceDataResult struct {
	PolicyUUID  uuid.UUID
	Name        string
	Description *string
	Document    datatypes.JSON
	Version     string
	Status      string
	IsSystem    bool
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type PolicyServiceGetFilter struct {
	TenantID    int64
	Name        *string
	Description *string
	Version     *string
	Status      []string
	IsSystem    *bool
	ServiceID   *uuid.UUID
	Page        int
	Limit       int
	SortBy      string
	SortOrder   string
}

type PolicyServiceGetResult struct {
	Data       []PolicyServiceDataResult
	Total      int64
	Page       int
	Limit      int
	TotalPages int
}

type PolicyServiceServiceDataResult struct {
	ServiceUUID uuid.UUID
	Name        string
	DisplayName string
	Description string
	Version     string
	Status      string
	IsSystem    bool
	APICount    int64
	PolicyCount int64
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type PolicyServiceServicesFilter struct {
	Name        *string
	DisplayName *string
	Description *string
	Page        int
	Limit       int
	SortBy      string
	SortOrder   string
}

type PolicyServiceServicesResult struct {
	Data       []PolicyServiceServiceDataResult
	Total      int64
	Page       int
	Limit      int
	TotalPages int
}

// PolicyChangeContext carries WHO changed a policy and WHY into the version
// history. All fields are optional: a change made by a service principal has no
// user, and a reason is only supplied when the caller offers one.
type PolicyChangeContext struct {
	ActorUserID   *int64
	ActorClientID *int64
	Reason        *string
}

type PolicyService interface {
	Get(ctx context.Context, filter PolicyServiceGetFilter) (*PolicyServiceGetResult, error)
	GetByUUID(ctx context.Context, policyUUID uuid.UUID, tenantID int64) (*PolicyServiceDataResult, error)
	GetServicesByPolicyUUID(ctx context.Context, policyUUID uuid.UUID, tenantID int64, filter PolicyServiceServicesFilter) (*PolicyServiceServicesResult, error)
	Create(ctx context.Context, tenantID int64, name string, description *string, document datatypes.JSON, version string, status string, isSystem bool) (*PolicyServiceDataResult, error)
	Update(ctx context.Context, policyUUID uuid.UUID, tenantID int64, name string, description *string, document datatypes.JSON, version string, status string, change PolicyChangeContext) (*PolicyServiceDataResult, error)
	SetStatusByUUID(ctx context.Context, policyUUID uuid.UUID, tenantID int64, status string) (*PolicyServiceDataResult, error)
	DeleteByUUID(ctx context.Context, policyUUID uuid.UUID, tenantID int64) (*PolicyServiceDataResult, error)
	// GetHistory returns a paginated list of version snapshots for a policy.
	GetHistory(ctx context.Context, policyUUID uuid.UUID, tenantID int64, page, limit int) (*PolicyHistoryListResult, error)
	// GetHistoryVersion returns a single version snapshot for a policy.
	GetHistoryVersion(ctx context.Context, policyUUID uuid.UUID, tenantID int64, versionNumber int) (*PolicyHistoryEntryResult, error)
}

// PolicyHistoryEntryResult is the service-layer view of a policy version snapshot.
type PolicyHistoryEntryResult struct {
	UUID          uuid.UUID
	VersionNumber int
	Name          string
	Description   *string
	Document      datatypes.JSON
	PolicyVersion string
	ChangeReason  *string
	SnapshotAt    time.Time
}

// PolicyHistoryListResult is a paginated list of policy version snapshots.
type PolicyHistoryListResult struct {
	Data       []PolicyHistoryEntryResult
	Total      int64
	Page       int
	Limit      int
	TotalPages int
}

type policyService struct {
	db               *gorm.DB
	policyRepo       PolicyRepository
	serviceRepo      ServiceRepository
	apiRepo          APIRepository
	authEventService authevent.AuthEventService
	eventService     event.EventService
	historyRepo      PolicyVersionHistoryRepository // nil → policy version history disabled
	tenantDirectory  PolicyTenantDirectory          // nil → MRN resources are refused (fail closed)
}

// PolicyTenantDirectory resolves tenants for the MRN tenant-boundary check on
// policy writes. Implemented by tenant.TenantService.
type PolicyTenantDirectory interface {
	GetSystem(ctx context.Context) (*tenant.TenantServiceDataResult, error)
	GetByName(ctx context.Context, name string) (*tenant.TenantServiceDataResult, error)
}

// SetPolicyTenantDirectory injects the tenant directory used to enforce the
// MRN tenant boundary on policy Create/Update (setter pattern, mirroring
// SetPolicyVersionHistory, so NewPolicyService callers and tests are
// unaffected). When it is absent, any policy carrying an MRN resource is
// REFUSED — without a directory the boundary cannot be verified, and the only
// safe answer for an unverifiable cross-tenant grant is no.
func SetPolicyTenantDirectory(svc PolicyService, dir PolicyTenantDirectory) {
	if s, ok := svc.(*policyService); ok {
		s.tenantDirectory = dir
	}
}

// SetPolicyVersionHistory injects the append-only policy version history repo
// into an existing PolicyService (setter pattern, so NewPolicyService callers
// and tests are unaffected). When set, every policy Update snapshots the
// before-state; the two history read endpoints also require it.
func SetPolicyVersionHistory(svc PolicyService, repo PolicyVersionHistoryRepository) {
	if s, ok := svc.(*policyService); ok {
		s.historyRepo = repo
	}
}

func NewPolicyService(
	db *gorm.DB,
	policyRepo PolicyRepository,
	serviceRepo ServiceRepository,
	apiRepo APIRepository,
	eventService event.EventService,
	authEventService ...authevent.AuthEventService,
) PolicyService {
	eventSvc := authevent.NoopService()
	if len(authEventService) > 0 && authEventService[0] != nil {
		eventSvc = authEventService[0]
	}
	return &policyService{
		db:               db,
		policyRepo:       policyRepo,
		serviceRepo:      serviceRepo,
		apiRepo:          apiRepo,
		authEventService: eventSvc,
		eventService:     eventService,
	}
}

func (s *policyService) Get(ctx context.Context, filter PolicyServiceGetFilter) (*PolicyServiceGetResult, error) {
	_, span := otel.Tracer("service").Start(ctx, "policy.list")
	defer span.End()
	span.SetAttributes(attribute.Int64("tenant.id", filter.TenantID))
	repoFilter := PolicyRepositoryGetFilter(filter)

	result, err := s.policyRepo.FindPaginated(repoFilter)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "list policies failed")
		return nil, err
	}

	var data []PolicyServiceDataResult
	for _, policy := range result.Data {
		data = append(data, PolicyServiceDataResult{
			PolicyUUID:  policy.PolicyUUID,
			Name:        policy.Name,
			Description: policy.Description,
			Document:    policy.Document,
			Version:     policy.Version,
			Status:      policy.Status,
			IsSystem:    policy.IsSystem,
			CreatedAt:   policy.CreatedAt,
			UpdatedAt:   policy.UpdatedAt,
		})
	}

	return &PolicyServiceGetResult{
		Data:       data,
		Total:      result.Total,
		Page:       result.Page,
		Limit:      result.Limit,
		TotalPages: result.TotalPages,
	}, nil
}

func (s *policyService) GetByUUID(ctx context.Context, policyUUID uuid.UUID, tenantID int64) (*PolicyServiceDataResult, error) {
	_, span := otel.Tracer("service").Start(ctx, "policy.get")
	defer span.End()
	span.SetAttributes(attribute.String("policy.uuid", policyUUID.String()), attribute.Int64("tenant.id", tenantID))
	policy, err := s.policyRepo.FindByUUIDAndTenantID(policyUUID, tenantID)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "get policy failed")
		return nil, err
	}
	if policy == nil {
		return nil, apperror.NewNotFound("policy")
	}

	return &PolicyServiceDataResult{
		PolicyUUID:  policy.PolicyUUID,
		Name:        policy.Name,
		Description: policy.Description,
		Document:    policy.Document,
		Version:     policy.Version,
		Status:      policy.Status,
		IsSystem:    policy.IsSystem,
		CreatedAt:   policy.CreatedAt,
		UpdatedAt:   policy.UpdatedAt,
	}, nil
}

func (s *policyService) GetServicesByPolicyUUID(ctx context.Context, policyUUID uuid.UUID, tenantID int64, filter PolicyServiceServicesFilter) (*PolicyServiceServicesResult, error) {
	_, span := otel.Tracer("service").Start(ctx, "policy.getServices")
	defer span.End()
	span.SetAttributes(attribute.String("policy.uuid", policyUUID.String()), attribute.Int64("tenant.id", tenantID))
	// First check if policy exists and belongs to tenant. The returned policy must
	// be inspected, not just err: FindByUUIDAndTenantID reports not-found as
	// (nil, nil), so checking err alone let a foreign policy UUID fall straight
	// through to the service listing below.
	policy, err := s.policyRepo.FindByUUIDAndTenantID(policyUUID, tenantID)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "get services by policy failed")
		return nil, err
	}
	if policy == nil {
		return nil, apperror.NewNotFoundWithReason("policy not found or access denied")
	}

	// Convert filter to repository filter. TenantID is carried into the repository
	// so the join is scoped there too — the policy check above proves the caller
	// owns the POLICY, not that every service linked to it is theirs.
	repoFilter := ServiceRepositoryGetFilter{
		Name:        filter.Name,
		DisplayName: filter.DisplayName,
		Description: filter.Description,
		TenantID:    &tenantID,
		Page:        filter.Page,
		Limit:       filter.Limit,
		SortBy:      filter.SortBy,
		SortOrder:   filter.SortOrder,
	}

	// Get services that use this policy
	result, err := s.serviceRepo.FindServicesByPolicyUUID(policyUUID, repoFilter)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "get services by policy failed")
		return nil, err
	}

	// Convert to service data results
	var data []PolicyServiceServiceDataResult
	for _, service := range result.Data {
		// Get API count and policy count for each service, scoped to the caller's tenant
		apiCount, _ := s.apiRepo.CountByServiceID(service.ServiceID, tenantID)
		policyCount, _ := s.serviceRepo.CountPoliciesByServiceID(service.ServiceID)

		data = append(data, PolicyServiceServiceDataResult{
			ServiceUUID: service.ServiceUUID,
			Name:        service.Name,
			DisplayName: service.DisplayName,
			Description: service.Description,
			Version:     service.Version,
			Status:      service.Status,
			IsSystem:    service.IsSystem,
			APICount:    apiCount,
			PolicyCount: policyCount,
			CreatedAt:   service.CreatedAt,
			UpdatedAt:   service.UpdatedAt,
		})
	}

	return &PolicyServiceServicesResult{
		Data:       data,
		Total:      result.Total,
		Page:       result.Page,
		Limit:      result.Limit,
		TotalPages: result.TotalPages,
	}, nil
}

func (s *policyService) Create(ctx context.Context, tenantID int64, name string, description *string, document datatypes.JSON, version string, status string, isSystem bool) (*PolicyServiceDataResult, error) {
	_, span := otel.Tracer("service").Start(ctx, "policy.create")
	defer span.End()
	span.SetAttributes(attribute.Int64("tenant.id", tenantID))
	var createdPolicy *Policy

	// Enforced HERE — the service layer both transports (REST and gRPC) funnel
	// through — so neither surface can drift out of parity.
	if err := s.enforceMRNTenantBoundary(ctx, tenantID, document); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "create policy failed")
		return nil, err
	}

	err := s.db.Transaction(func(tx *gorm.DB) error {
		txPolicyRepo := s.policyRepo.WithTx(tx)

		// Check if policy with same name and version already exists
		existingPolicy, err := txPolicyRepo.FindByNameAndVersion(name, version, tenantID)
		if err != nil {
			return err
		}
		if existingPolicy != nil {
			return apperror.NewConflict("policy with name '" + name + "' and version '" + version + "' already exists")
		}

		// Create new policy
		policy := &Policy{
			PolicyUUID:  uuid.New(),
			TenantID:    tenantID,
			Name:        name,
			Description: description,
			Document:    document,
			Version:     version,
			Status:      status,
			IsSystem:    isSystem,
		}

		createdPolicy, err = txPolicyRepo.Create(policy)
		if err != nil {
			return err
		}

		createdPolicy = policy

		// Emit policy.created integration event inside the transaction
		if s.eventService != nil {
			if _, emitErr := s.eventService.Emit(ctx, tx, event.NewIntegrationEvent(
				event.EventTypePolicyCreated, 1, tenantID,
			).SetSubject(&createdPolicy.PolicyUUID, "policy")); emitErr != nil {
				return emitErr
			}
		}

		return nil
	})

	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "create policy failed")
		return nil, err
	}

	span.SetStatus(codes.Ok, "")
	return &PolicyServiceDataResult{
		PolicyUUID:  createdPolicy.PolicyUUID,
		Name:        createdPolicy.Name,
		Description: createdPolicy.Description,
		Document:    createdPolicy.Document,
		Version:     createdPolicy.Version,
		Status:      createdPolicy.Status,
		IsSystem:    createdPolicy.IsSystem,
		CreatedAt:   createdPolicy.CreatedAt,
		UpdatedAt:   createdPolicy.UpdatedAt,
	}, nil
}

func (s *policyService) Update(ctx context.Context, policyUUID uuid.UUID, tenantID int64, name string, description *string, document datatypes.JSON, version string, status string, change PolicyChangeContext) (*PolicyServiceDataResult, error) {
	_, span := otel.Tracer("service").Start(ctx, "policy.update")
	defer span.End()
	span.SetAttributes(attribute.String("policy.uuid", policyUUID.String()), attribute.Int64("tenant.id", tenantID))
	var updatedPolicy *Policy

	// Same boundary as Create: an update is just another write path for smuggling
	// a cross-tenant MRN into an already-accepted policy.
	if err := s.enforceMRNTenantBoundary(ctx, tenantID, document); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "update policy failed")
		return nil, err
	}

	err := s.db.Transaction(func(tx *gorm.DB) error {
		txPolicyRepo := s.policyRepo.WithTx(tx)

		// Check if policy exists and belongs to tenant
		policy, err := txPolicyRepo.FindByUUIDAndTenantID(policyUUID, tenantID)
		if err != nil {
			return err
		}
		if policy == nil {
			return apperror.NewNotFoundWithReason("policy not found or access denied")
		}

		// Check if policy is a system record (critical for app functionality)
		if policy.IsSystem {
			return apperror.NewValidation("system policy cannot be updated")
		}

		// Check if another policy with same name and version exists (excluding current policy)
		if policy.Name != name || policy.Version != version {
			existingPolicy, err := txPolicyRepo.FindByNameAndVersion(name, version, tenantID)
			if err != nil {
				return err
			}
			if existingPolicy != nil && existingPolicy.PolicyUUID != policyUUID {
				return apperror.NewConflict("policy with name '" + name + "' and version '" + version + "' already exists")
			}
		}

		// Track changed fields
		var changed []string
		if policy.Name != name {
			changed = append(changed, "name")
		}
		if policy.Description != description && (policy.Description == nil || description == nil || *policy.Description != *description) {
			changed = append(changed, "description")
		}
		if policy.Version != version {
			changed = append(changed, "version")
		}
		if policy.Status != status {
			changed = append(changed, "status")
		}
		if string(policy.Document) != string(document) {
			changed = append(changed, "document")
		}

		// Snapshot the before-state into policy_version_history (append-only,
		// same transaction) so the prior version can be audited or rolled back.
		// Attribution is recorded HERE as well as in management_audit_log. "What did
		// this policy look like" and "who changed it and why" are the same question
		// for an IAM auditor, and correlating two tables by timestamp is not an
		// answer — especially for the gRPC control plane, which writes no audit row
		// at all. The columns already existed and were never populated.
		if s.historyRepo != nil {
			txHistoryRepo := s.historyRepo.WithTx(tx)
			nextVersion, verr := txHistoryRepo.NextVersionNumber(policy.PolicyID)
			if verr != nil {
				return verr
			}
			beforeDoc := policy.Document
			if len(beforeDoc) == 0 {
				beforeDoc = datatypes.JSON([]byte("{}"))
			}
			if _, herr := txHistoryRepo.Create(&PolicyVersionHistory{
				TenantID:          policy.TenantID,
				PolicyID:          policy.PolicyID,
				VersionNumber:     nextVersion,
				Name:              policy.Name,
				Description:       policy.Description,
				Document:          beforeDoc,
				PolicyVersion:     policy.Version,
				ChangedByUserID:   change.ActorUserID,
				ChangedByClientID: change.ActorClientID,
				ChangeReason:      change.Reason,
			}); herr != nil {
				return herr
			}
		}

		// Update policy
		policy.Name = name
		policy.Description = description
		policy.Document = document
		policy.Version = version
		policy.Status = status

		updatedPolicy, err = txPolicyRepo.UpdateByUUID(policy.PolicyUUID, policy)
		if err != nil {
			return err
		}

		// Emit policy.updated integration event inside the transaction
		if s.eventService != nil && len(changed) > 0 {
			if _, emitErr := s.eventService.Emit(ctx, tx, event.NewIntegrationEvent(
				event.EventTypeIAMPolicyUpdated, 1, tenantID,
			).SetSubject(&updatedPolicy.PolicyUUID, "policy").
				SetChangedFields(changed...)); emitErr != nil {
				return emitErr
			}
		}

		return nil
	})

	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "update policy failed")
		return nil, err
	}

	span.SetStatus(codes.Ok, "")
	s.authEventService.Log(ctx, authevent.AuthEventInput{
		TenantID:    tenantID,
		Category:    authevent.AuthEventCategoryAuthz,
		EventType:   authevent.AuthEventTypeIAMPolicyUpdated,
		Severity:    authevent.AuthEventSeverityInfo,
		Result:      authevent.AuthEventResultSuccess,
		Description: ptr.Ptr("IAM policy updated"),
	})
	return &PolicyServiceDataResult{
		PolicyUUID:  updatedPolicy.PolicyUUID,
		Name:        updatedPolicy.Name,
		Description: updatedPolicy.Description,
		Document:    updatedPolicy.Document,
		Version:     updatedPolicy.Version,
		Status:      updatedPolicy.Status,
		IsSystem:    updatedPolicy.IsSystem,
		CreatedAt:   updatedPolicy.CreatedAt,
		UpdatedAt:   updatedPolicy.UpdatedAt,
	}, nil
}

func (s *policyService) SetStatusByUUID(ctx context.Context, policyUUID uuid.UUID, tenantID int64, status string) (*PolicyServiceDataResult, error) {
	_, span := otel.Tracer("service").Start(ctx, "policy.setStatus")
	defer span.End()
	span.SetAttributes(attribute.String("policy.uuid", policyUUID.String()), attribute.Int64("tenant.id", tenantID))
	var updatedPolicy *Policy

	err := s.db.Transaction(func(tx *gorm.DB) error {
		txPolicyRepo := s.policyRepo.WithTx(tx)

		// Check if policy exists and belongs to tenant
		policy, err := txPolicyRepo.FindByUUIDAndTenantID(policyUUID, tenantID)
		if err != nil {
			return err
		}
		if policy == nil {
			return apperror.NewNotFoundWithReason("policy not found or access denied")
		}

		// Check if policy is a system record (critical for app functionality)
		if policy.IsSystem {
			return apperror.NewValidation("system policy status cannot be updated")
		}

		statusChanged := policy.Status != status

		// Update status
		if err := txPolicyRepo.SetStatusByUUID(policyUUID, tenantID, status); err != nil {
			return err
		}

		// Get updated policy
		updatedPolicy, err = txPolicyRepo.FindByUUIDAndTenantID(policyUUID, tenantID)
		if err != nil {
			return err
		}

		// Create/Update/Delete all emit; this path did not, so deactivating a policy —
		// the revocation path, the change downstream bundle consumers most need to
		// hear about — was invisible to both the outbox and the audit trail.
		if s.eventService != nil && statusChanged {
			if _, emitErr := s.eventService.Emit(ctx, tx, event.NewIntegrationEvent(
				event.EventTypeIAMPolicyUpdated, 1, tenantID,
			).SetSubject(&updatedPolicy.PolicyUUID, "policy").
				SetChangedFields("status")); emitErr != nil {
				return emitErr
			}
		}

		return nil
	})

	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "set policy status failed")
		return nil, err
	}

	span.SetStatus(codes.Ok, "")
	s.authEventService.Log(ctx, authevent.AuthEventInput{
		TenantID:    tenantID,
		Category:    authevent.AuthEventCategoryAuthz,
		EventType:   authevent.AuthEventTypeIAMPolicyUpdated,
		Severity:    authevent.AuthEventSeverityInfo,
		Result:      authevent.AuthEventResultSuccess,
		Description: ptr.Ptr("IAM policy status set to " + updatedPolicy.Status),
	})
	return &PolicyServiceDataResult{
		PolicyUUID:  updatedPolicy.PolicyUUID,
		Name:        updatedPolicy.Name,
		Description: updatedPolicy.Description,
		Document:    updatedPolicy.Document,
		Version:     updatedPolicy.Version,
		Status:      updatedPolicy.Status,
		IsSystem:    updatedPolicy.IsSystem,
		CreatedAt:   updatedPolicy.CreatedAt,
		UpdatedAt:   updatedPolicy.UpdatedAt,
	}, nil
}

func (s *policyService) DeleteByUUID(ctx context.Context, policyUUID uuid.UUID, tenantID int64) (*PolicyServiceDataResult, error) {
	_, span := otel.Tracer("service").Start(ctx, "policy.delete")
	defer span.End()
	span.SetAttributes(attribute.String("policy.uuid", policyUUID.String()), attribute.Int64("tenant.id", tenantID))
	var deletedPolicy *Policy

	err := s.db.Transaction(func(tx *gorm.DB) error {
		txPolicyRepo := s.policyRepo.WithTx(tx)

		// Check if policy exists and belongs to tenant
		policy, err := txPolicyRepo.FindByUUIDAndTenantID(policyUUID, tenantID)
		if err != nil {
			return err
		}
		if policy == nil {
			return apperror.NewNotFoundWithReason("policy not found or access denied")
		}

		// Check if policy is system policy (cannot be deleted)
		if policy.IsSystem {
			return apperror.NewValidation("system policies cannot be deleted")
		}

		deletedPolicy = policy

		// Delete policy
		if err := txPolicyRepo.DeleteByUUIDAndTenantID(policyUUID, tenantID); err != nil {
			return err
		}

		if s.eventService != nil {
			if _, emitErr := s.eventService.Emit(ctx, tx, event.NewIntegrationEvent(
				event.EventTypePolicyDeleted, 1, tenantID,
			).SetSubject(&policy.PolicyUUID, "policy")); emitErr != nil {
				return emitErr
			}
		}
		return nil
	})

	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "delete policy failed")
		return nil, err
	}

	span.SetStatus(codes.Ok, "")
	return &PolicyServiceDataResult{
		PolicyUUID:  deletedPolicy.PolicyUUID,
		Name:        deletedPolicy.Name,
		Description: deletedPolicy.Description,
		Document:    deletedPolicy.Document,
		Version:     deletedPolicy.Version,
		Status:      deletedPolicy.Status,
		IsSystem:    deletedPolicy.IsSystem,
		CreatedAt:   deletedPolicy.CreatedAt,
		UpdatedAt:   deletedPolicy.UpdatedAt,
	}, nil
}

func (s *policyService) GetHistory(ctx context.Context, policyUUID uuid.UUID, tenantID int64, page, limit int) (*PolicyHistoryListResult, error) {
	if s.historyRepo == nil {
		return nil, apperror.NewNotFoundWithReason("policy version history is not available")
	}
	policy, err := s.policyRepo.FindByUUIDAndTenantID(policyUUID, tenantID)
	if err != nil {
		return nil, err
	}
	if policy == nil {
		return nil, apperror.NewNotFoundWithReason("policy not found or access denied")
	}
	entries, err := s.historyRepo.FindByPolicyIDPaginated(policy.PolicyID, page, limit)
	if err != nil {
		return nil, err
	}
	data := make([]PolicyHistoryEntryResult, len(entries.Data))
	for i, e := range entries.Data {
		data[i] = toHistoryEntry(&e)
	}
	return &PolicyHistoryListResult{
		Data:       data,
		Total:      entries.Total,
		Page:       entries.Page,
		Limit:      entries.Limit,
		TotalPages: entries.TotalPages,
	}, nil
}

func (s *policyService) GetHistoryVersion(ctx context.Context, policyUUID uuid.UUID, tenantID int64, versionNumber int) (*PolicyHistoryEntryResult, error) {
	if s.historyRepo == nil {
		return nil, apperror.NewNotFoundWithReason("policy version history is not available")
	}
	policy, err := s.policyRepo.FindByUUIDAndTenantID(policyUUID, tenantID)
	if err != nil {
		return nil, err
	}
	if policy == nil {
		return nil, apperror.NewNotFoundWithReason("policy not found or access denied")
	}
	entry, err := s.historyRepo.FindByPolicyIDAndVersion(policy.PolicyID, versionNumber)
	if err != nil {
		return nil, err
	}
	if entry == nil {
		return nil, apperror.NewNotFoundWithReason("policy version not found")
	}
	result := toHistoryEntry(entry)
	return &result, nil
}

// enforceMRNTenantBoundary refuses MRN statement resources that reach outside
// the policy's owning tenant.
//
// The WHY: a policy row is tenant-scoped, but the evaluator matches on the
// resource STRING alone. Without this rule, tenant A's admin writes a policy
// granting themselves "mrn:storage:tenant-b:..." — a perfectly well-formed
// MRN — attaches it to a principal in tenant A, and the evaluator honors it,
// because nothing at evaluation time knows which tenant authored the grant.
// The write path is the only place that knows both the document AND its owner,
// so the boundary must be enforced here.
//
// The rule: a policy owned by a regular tenant may only carry MRN resources
// whose tenant segment is that tenant's own slug. "*" (every tenant) and ""
// (platform scope) are REFUSED for regular tenants — both are broader than the
// tenant itself. Only a policy owned by the singleton SYSTEM tenant (the
// control plane, resolved via the same GetSystem mechanism the gRPC
// cross-tenant guard uses) may use "*", "", or another tenant's literal.
// Legacy flat resource strings are untouched by all of this.
func (s *policyService) enforceMRNTenantBoundary(ctx context.Context, tenantID int64, document datatypes.JSON) error {
	if len(document) == 0 {
		// "Document is required" belongs to the DTO structure validation.
		return nil
	}
	var doc PolicyDocument
	if err := json.Unmarshal(document, &doc); err != nil {
		// Also rejected by DTO validation on both transports; repeated here
		// because a document this chokepoint cannot read cannot be cleared.
		return apperror.NewValidation("policy document must be valid JSON: " + err.Error())
	}

	// Resolved lazily, once, and only when an MRN resource actually appears, so
	// legacy-only policies never touch the tenant directory.
	ownerIsSystem := (*bool)(nil)
	for _, stmt := range doc.Statement {
		for _, raw := range stmt.Resource {
			res := strings.TrimSpace(raw)
			if !mrn.IsMRN(res) {
				continue
			}
			pattern, err := mrn.ParsePattern(res)
			if err != nil {
				return apperror.NewValidation("statement resource " + strconv.Quote(raw) + " is not a valid MRN pattern: " + err.Error())
			}

			if ownerIsSystem == nil {
				isSystem, err := s.ownerIsSystemTenant(ctx, tenantID)
				if err != nil {
					return err
				}
				ownerIsSystem = &isSystem
			}
			if *ownerIsSystem {
				// The control plane writes policies about any tenant by design.
				continue
			}

			switch pattern.Tenant {
			case "*":
				return apperror.NewValidation("statement resource " + strconv.Quote(raw) + " uses a wildcard tenant segment; only the system tenant may grant across tenants")
			case "":
				return apperror.NewValidation("statement resource " + strconv.Quote(raw) + " is platform-scoped (empty tenant segment); only the system tenant may grant platform-scoped resources")
			}

			// The literal must be THIS policy's own tenant. The lookup result is
			// deliberately not distinguished from a mismatch: reporting "no such
			// tenant" vs "not your tenant" would turn this validator into an
			// existence oracle for other tenants' slugs.
			owner, err := s.tenantDirectory.GetByName(ctx, pattern.Tenant)
			if err != nil || owner == nil || owner.TenantID != tenantID {
				return apperror.NewValidation("statement resource " + strconv.Quote(raw) + " names tenant segment " + strconv.Quote(pattern.Tenant) + ", which is not this policy's own tenant")
			}
		}
	}
	return nil
}

// ownerIsSystemTenant reports whether the policy's owning tenant is the
// singleton system tenant. Every failure mode fails CLOSED: if the directory
// is not wired or the system tenant cannot be resolved, no MRN-bearing policy
// is accepted, because an unverifiable tenant boundary is indistinguishable
// from a crossed one.
func (s *policyService) ownerIsSystemTenant(ctx context.Context, tenantID int64) (bool, error) {
	if s.tenantDirectory == nil {
		return false, apperror.NewValidation("MRN resources cannot be accepted: the tenant boundary cannot be verified")
	}
	system, err := s.tenantDirectory.GetSystem(ctx)
	if err != nil || system == nil {
		return false, apperror.NewValidation("MRN resources cannot be accepted: the system tenant could not be resolved")
	}
	return system.TenantID == tenantID, nil
}

func toHistoryEntry(e *PolicyVersionHistory) PolicyHistoryEntryResult {
	return PolicyHistoryEntryResult{
		UUID:          e.PolicyVersionHistoryUUID,
		VersionNumber: e.VersionNumber,
		Name:          e.Name,
		Description:   e.Description,
		Document:      e.Document,
		PolicyVersion: e.PolicyVersion,
		ChangeReason:  e.ChangeReason,
		SnapshotAt:    e.SnapshotAt,
	}
}
