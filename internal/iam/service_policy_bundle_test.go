package iam

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"gorm.io/datatypes"
)

func TestServiceAuthorizationService_PolicyBundle(t *testing.T) {
	now := time.Date(2026, 6, 4, 1, 2, 3, 0, time.UTC)
	serviceRepo := &mockServiceRepo{
		findByNameAndTenantIDFn: func(name string, tenantID int64) (*Service, error) {
			if name != "serviceA" {
				t.Fatalf("service name = %q", name)
			}
			// The lookup must be tenant-scoped: a service name is unique per tenant,
			// not globally, so an unscoped lookup collapsed onto the system tenant.
			if tenantID != 1 {
				t.Fatalf("tenant id = %d, want the identity's tenant", tenantID)
			}
			return &Service{ServiceID: 7, Name: name, TenantID: tenantID, Status: "active"}, nil
		},
	}
	policyRepo := &mockServicePolicyRepo{
		findPoliciesByServiceIDFn: func(serviceID int64) ([]Policy, error) {
			if serviceID != 7 {
				t.Fatalf("service id = %d", serviceID)
			}
			return []Policy{
				{PolicyID: 1, PolicyUUID: testResourceUUID, Status: "active", UpdatedAt: now, Document: datatypes.JSON(`{"version":"v1","statement":[{"effect":"allow","action":["serviceB:*"],"resource":["serviceB:*"]}]}`)},
				{PolicyID: 2, Status: "inactive", UpdatedAt: now, Document: datatypes.JSON(`{"version":"v1","statement":[{"effect":"allow","action":["*"],"resource":["*"]}]}`)},
				// A malformed ACTIVE document now fails the whole bundle (see
				// TestServiceAuthorizationService_PolicyBundle_MalformedDocumentFailsClosed),
				// so the happy path carries only well-formed documents.
			}, nil
		},
	}
	svc := &serviceAuthorizationService{serviceRepo: serviceRepo, servicePolicyRepo: policyRepo, clock: func() time.Time { return now }}

	bundle, etag, err := svc.PolicyBundle(context.Background(), ServiceIdentity{ServiceName: "serviceA", TenantID: 1})
	if err != nil {
		t.Fatalf("PolicyBundle() error = %v", err)
	}
	if bundle.Service != "serviceA" || bundle.GeneratedAt != now || len(bundle.Policies) != 1 {
		t.Fatalf("bundle = %+v", bundle)
	}
	if etag == "" || etag[0] != '"' || etag[len(etag)-1] != '"' {
		t.Fatalf("etag = %q", etag)
	}
	again, againETag, err := svc.PolicyBundle(context.Background(), ServiceIdentity{ServiceName: "serviceA", TenantID: 1})
	if err != nil || again.Version != bundle.Version || againETag != etag {
		t.Fatalf("stable version mismatch: bundle=%q again=%q etag=%q again=%q err=%v", bundle.Version, again.Version, etag, againETag, err)
	}
}

func TestServiceAuthorizationService_PolicyBundle_NotFoundOrInactive(t *testing.T) {
	tests := []struct {
		name    string
		service *Service
	}{
		{"missing service", nil},
		{"inactive service", &Service{Name: "serviceA", Status: "inactive"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := NewServiceAuthorizationService(&mockServiceRepo{findByNameAndTenantIDFn: func(string, int64) (*Service, error) { return tt.service, nil }}, &mockServicePolicyRepo{})
			if _, _, err := svc.PolicyBundle(context.Background(), ServiceIdentity{ServiceName: "serviceA", TenantID: 1}); err == nil {
				t.Fatal("PolicyBundle() error = nil")
			}
		})
	}
}

func TestServiceAuthorizationService_PolicyBundle_RepositoryErrors(t *testing.T) {
	t.Run("service repo error", func(t *testing.T) {
		svc := NewServiceAuthorizationService(&mockServiceRepo{findByNameAndTenantIDFn: func(string, int64) (*Service, error) {
			return nil, assert.AnError
		}}, &mockServicePolicyRepo{})
		if _, _, err := svc.PolicyBundle(context.Background(), ServiceIdentity{ServiceName: "serviceA", TenantID: 1}); err == nil {
			t.Fatal("PolicyBundle() error = nil")
		}
	})
	t.Run("policy repo error", func(t *testing.T) {
		svc := NewServiceAuthorizationService(&mockServiceRepo{findByNameAndTenantIDFn: func(string, int64) (*Service, error) {
			return &Service{ServiceID: 1, Name: "serviceA", Status: "active"}, nil
		}}, &mockServicePolicyRepo{findPoliciesByServiceIDFn: func(int64) ([]Policy, error) {
			return nil, assert.AnError
		}})
		if _, _, err := svc.PolicyBundle(context.Background(), ServiceIdentity{ServiceName: "serviceA", TenantID: 1}); err == nil {
			t.Fatal("PolicyBundle() error = nil")
		}
	})
}

func TestServiceAuthorizationService_Authorize(t *testing.T) {
	svc := NewServiceAuthorizationService(
		&mockServiceRepo{findByNameAndTenantIDFn: func(string, int64) (*Service, error) {
			return &Service{ServiceID: 1, Name: "serviceA", Status: "active"}, nil
		}},
		&mockServicePolicyRepo{findPoliciesByServiceIDFn: func(int64) ([]Policy, error) {
			return []Policy{{Status: "active", Document: datatypes.JSON(`{"version":"v1","statement":[{"effect":"allow","action":["serviceB:invoke"],"resource":["serviceB:grpc"]}]}`)}}, nil
		}},
	)
	decision := svc.Authorize(context.Background(), AuthzRequest{Principal: "serviceA", Action: "serviceB:invoke", Resource: "serviceB:grpc", TenantID: 1})
	if !decision.Allowed {
		t.Fatalf("Authorize() = %+v", decision)
	}

	denied := NewServiceAuthorizationService(&mockServiceRepo{findByNameAndTenantIDFn: func(string, int64) (*Service, error) {
		return nil, assert.AnError
	}}, &mockServicePolicyRepo{}).Authorize(context.Background(), AuthzRequest{Principal: "serviceA", Action: "x:y", Resource: "z:q", TenantID: 1})
	if denied.Allowed || denied.Reason != "principal bundle unavailable" {
		t.Fatalf("Authorize() denied = %+v", denied)
	}
}

// A principal with no tenant used to fall back to an unscoped FindByName, whose
// First() returns the lowest service_id — always the oldest (system) tenant. Every
// tenant-less principal therefore received the platform's own policy bundle.
func TestServiceAuthorizationService_PolicyBundle_RequiresATenant(t *testing.T) {
	called := false
	svc := NewServiceAuthorizationService(
		&mockServiceRepo{
			findByNameFn: func(string) (*Service, error) {
				called = true
				return &Service{ServiceID: 1, Name: "auth", Status: "active"}, nil
			},
			findByNameAndTenantIDFn: func(string, int64) (*Service, error) {
				called = true
				return &Service{ServiceID: 1, Name: "auth", Status: "active"}, nil
			},
		},
		&mockServicePolicyRepo{},
	)

	_, _, err := svc.PolicyBundle(context.Background(), ServiceIdentity{ServiceName: "auth"})
	if err == nil {
		t.Fatal("a tenant-less principal must be refused")
	}
	if called {
		t.Fatal("no service lookup may happen for a tenant-less principal")
	}
}

// Silently skipping an unparseable document is asymmetric: dropping an allow is
// safe, dropping a DENY serves the remaining allows without its guardrail. The
// bundle is the input to every downstream decision, so it must fail closed.
func TestServiceAuthorizationService_PolicyBundle_MalformedDocumentFailsClosed(t *testing.T) {
	svc := NewServiceAuthorizationService(
		&mockServiceRepo{
			findByNameAndTenantIDFn: func(string, int64) (*Service, error) {
				return &Service{ServiceID: 7, Name: "serviceA", TenantID: 1, Status: "active"}, nil
			},
		},
		&mockServicePolicyRepo{
			findPoliciesByServiceIDFn: func(int64) ([]Policy, error) {
				return []Policy{
					{PolicyID: 1, Status: "active", Document: datatypes.JSON(`{"version":"v1","statement":[]}`)},
					{PolicyID: 2, Status: "active", Document: datatypes.JSON(`{bad`)},
				}, nil
			},
		},
	)

	bundle, _, err := svc.PolicyBundle(context.Background(), ServiceIdentity{ServiceName: "serviceA", TenantID: 1})
	if err == nil {
		t.Fatal("an unparseable policy document must fail the bundle, not be skipped")
	}
	if bundle != nil {
		t.Fatal("no partial bundle may be served")
	}
}
