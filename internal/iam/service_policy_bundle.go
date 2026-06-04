package iam

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"sort"
	"time"

	"github.com/maintainerd/auth/internal/platform/apperror"
	"github.com/maintainerd/auth/internal/shared"
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
	_ = ctx
	service, err := s.serviceRepo.FindByName(identity.ServiceName)
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
			continue
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
	bundle, _, err := s.PolicyBundle(ctx, ServiceIdentity{ServiceName: req.Principal})
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
