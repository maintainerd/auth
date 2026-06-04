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
		findByNameFn: func(name string) (*Service, error) {
			if name != "serviceA" {
				t.Fatalf("service name = %q", name)
			}
			return &Service{ServiceID: 7, Name: name, Status: "active"}, nil
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
				{PolicyID: 3, Status: "active", UpdatedAt: now, Document: datatypes.JSON(`{bad`)},
			}, nil
		},
	}
	svc := &serviceAuthorizationService{serviceRepo: serviceRepo, servicePolicyRepo: policyRepo, clock: func() time.Time { return now }}

	bundle, etag, err := svc.PolicyBundle(context.Background(), ServiceIdentity{ServiceName: "serviceA"})
	if err != nil {
		t.Fatalf("PolicyBundle() error = %v", err)
	}
	if bundle.Service != "serviceA" || bundle.GeneratedAt != now || len(bundle.Policies) != 1 {
		t.Fatalf("bundle = %+v", bundle)
	}
	if etag == "" || etag[0] != '"' || etag[len(etag)-1] != '"' {
		t.Fatalf("etag = %q", etag)
	}
	again, againETag, err := svc.PolicyBundle(context.Background(), ServiceIdentity{ServiceName: "serviceA"})
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
			svc := NewServiceAuthorizationService(&mockServiceRepo{findByNameFn: func(string) (*Service, error) { return tt.service, nil }}, &mockServicePolicyRepo{})
			if _, _, err := svc.PolicyBundle(context.Background(), ServiceIdentity{ServiceName: "serviceA"}); err == nil {
				t.Fatal("PolicyBundle() error = nil")
			}
		})
	}
}

func TestServiceAuthorizationService_PolicyBundle_RepositoryErrors(t *testing.T) {
	t.Run("service repo error", func(t *testing.T) {
		svc := NewServiceAuthorizationService(&mockServiceRepo{findByNameFn: func(string) (*Service, error) {
			return nil, assert.AnError
		}}, &mockServicePolicyRepo{})
		if _, _, err := svc.PolicyBundle(context.Background(), ServiceIdentity{ServiceName: "serviceA"}); err == nil {
			t.Fatal("PolicyBundle() error = nil")
		}
	})
	t.Run("policy repo error", func(t *testing.T) {
		svc := NewServiceAuthorizationService(&mockServiceRepo{findByNameFn: func(string) (*Service, error) {
			return &Service{ServiceID: 1, Name: "serviceA", Status: "active"}, nil
		}}, &mockServicePolicyRepo{findPoliciesByServiceIDFn: func(int64) ([]Policy, error) {
			return nil, assert.AnError
		}})
		if _, _, err := svc.PolicyBundle(context.Background(), ServiceIdentity{ServiceName: "serviceA"}); err == nil {
			t.Fatal("PolicyBundle() error = nil")
		}
	})
}

func TestServiceAuthorizationService_Authorize(t *testing.T) {
	svc := NewServiceAuthorizationService(
		&mockServiceRepo{findByNameFn: func(string) (*Service, error) {
			return &Service{ServiceID: 1, Name: "serviceA", Status: "active"}, nil
		}},
		&mockServicePolicyRepo{findPoliciesByServiceIDFn: func(int64) ([]Policy, error) {
			return []Policy{{Status: "active", Document: datatypes.JSON(`{"version":"v1","statement":[{"effect":"allow","action":["serviceB:invoke"],"resource":["serviceB:grpc"]}]}`)}}, nil
		}},
	)
	decision := svc.Authorize(context.Background(), AuthzRequest{Principal: "serviceA", Action: "serviceB:invoke", Resource: "serviceB:grpc"})
	if !decision.Allowed {
		t.Fatalf("Authorize() = %+v", decision)
	}

	denied := NewServiceAuthorizationService(&mockServiceRepo{findByNameFn: func(string) (*Service, error) {
		return nil, assert.AnError
	}}, &mockServicePolicyRepo{}).Authorize(context.Background(), AuthzRequest{Principal: "serviceA", Action: "x:y", Resource: "z:q"})
	if denied.Allowed || denied.Reason != "principal bundle unavailable" {
		t.Fatalf("Authorize() denied = %+v", denied)
	}
}
