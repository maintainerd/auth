package iam

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"sort"
	"time"

	"github.com/maintainerd/maintainerd-auth/internal/platform/apperror"
	"github.com/maintainerd/maintainerd-auth/internal/shared"
)

// PolicyBundle is the service-account policy bundle distributed to callers.
type PolicyBundle struct {
	Service     string           `json:"service"`
	Version     string           `json:"version"`
	Policies    []PolicyDocument `json:"policies"`
	GeneratedAt time.Time        `json:"generated_at"`
}

// ServiceIdentity identifies the service principal resolved from an access token.
type ServiceIdentity struct {
	ServiceName string
	ClientID    string
	TenantID    int64
}

// ServiceAuthorizationService resolves service policy bundles and evaluates
// service-to-service authorization questions.
type ServiceAuthorizationService interface {
	PolicyBundle(ctx context.Context, identity ServiceIdentity) (*PolicyBundle, string, error)
	Authorize(ctx context.Context, req AuthzRequest) Decision
}

type serviceAuthorizationService struct {
	serviceRepo       ServiceRepository
	servicePolicyRepo ServicePolicyRepository
	clock             func() time.Time
}

// NewServiceAuthorizationService creates the IAM enforcement service.
func NewServiceAuthorizationService(serviceRepo ServiceRepository, servicePolicyRepo ServicePolicyRepository) ServiceAuthorizationService {
	return &serviceAuthorizationService{
		serviceRepo:       serviceRepo,
		servicePolicyRepo: servicePolicyRepo,
		clock:             time.Now,
	}
}

func (s *serviceAuthorizationService) PolicyBundle(ctx context.Context, identity ServiceIdentity) (*PolicyBundle, string, error) {
	// A service name is unique PER TENANT, not globally — the seeder creates a
	// service named "auth" in every tenant. Resolving without a tenant fell back to
	// FindByName, whose First() returns the lowest service_id, i.e. the oldest
	// (system) tenant. Every tenant-less principal therefore collapsed onto the
	// system tenant's service and received its policy bundle.
	if identity.TenantID <= 0 {
		return nil, "", apperror.NewForbidden("this token is not bound to a tenant")
	}
	service, err := s.serviceRepo.FindByNameAndTenantID(identity.ServiceName, identity.TenantID)
	if err != nil {
		return nil, "", err
	}
	if service == nil || service.Status != shared.StatusActive {
		return nil, "", apperror.NewNotFoundWithReason("service principal not found or inactive")
	}

	policies, err := s.servicePolicyRepo.FindPoliciesByServiceID(service.ServiceID)
	if err != nil {
		return nil, "", err
	}

	docs := make([]PolicyDocument, 0, len(policies))
	versionInputs := make([]string, 0, len(policies))
	for _, policy := range policies {
		if policy.Status != shared.StatusActive {
			continue
		}
		var doc PolicyDocument
		if err := json.Unmarshal(policy.Document, &doc); err != nil {
			// Fail the whole bundle rather than silently dropping one document: an
			// unparseable document may have carried a deny, and serving the
			// remaining allows without it would widen access.
			return nil, "", apperror.NewInternal(
				"a policy attached to this service could not be parsed", err)
		}
		docs = append(docs, doc)
		versionInputs = append(versionInputs, policy.PolicyUUID.String()+":"+policy.UpdatedAt.UTC().Format(time.RFC3339Nano)+":"+string(policy.Document))
	}

	version := policyBundleVersion(versionInputs)
	return &PolicyBundle{
		Service:     service.Name,
		Version:     version,
		Policies:    docs,
		GeneratedAt: s.clock().UTC(),
	}, `"` + version + `"`, nil
}

func (s *serviceAuthorizationService) Authorize(ctx context.Context, req AuthzRequest) Decision {
	bundle, _, err := s.PolicyBundle(ctx, ServiceIdentity{ServiceName: req.Principal, TenantID: req.TenantID})
	if err != nil {
		return Decision{Allowed: false, Reason: "principal bundle unavailable"}
	}
	return Evaluate(bundle.Policies, req)
}

func policyBundleVersion(inputs []string) string {
	sort.Strings(inputs)
	h := sha256.New()
	for _, input := range inputs {
		_, _ = h.Write([]byte(input))
		_, _ = h.Write([]byte{0})
	}
	return "v" + hex.EncodeToString(h.Sum(nil))[:12]
}
