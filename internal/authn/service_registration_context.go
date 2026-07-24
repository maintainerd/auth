package authn

import (
	"context"
	"strings"

	"github.com/maintainerd/maintainerd-auth/internal/platform/apperror"
	"github.com/maintainerd/maintainerd-auth/internal/secpolicy"
	"github.com/maintainerd/maintainerd-auth/internal/shared"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/codes"
)

// RegistrationContextResult is the presentation contract for a signup form:
// exactly the inputs the server will reject the caller for omitting, and nothing
// else.
type RegistrationContextResult struct {
	// RegistrationFlow echoes the resolved flow name, or "" for flow-less signup.
	RegistrationFlow string
	// RequiredFields is the EFFECTIVE set the form must collect, already merged
	// from the flow's own required_fields and the tenant policy. Never nil.
	RequiredFields []string
	// VerificationRequired is the effective email-verification requirement, so the
	// UI knows to show "check your email" instead of landing the user.
	VerificationRequired bool
}

// RegistrationContextService answers "what must the signup form collect?" for a
// public client, optionally scoped to a registration flow.
//
// It exists because required_fields is enforced server-side at register time
// (enforceRequiredRegistrationFields). Without a way to read the requirement,
// a flow requiring fullname or phone is an unresolvable 400 for every user: the
// hosted form never asks for it. This endpoint is deliberately NOT part of
// /oauth/connections — flow-derived fields were removed from that response so a
// registration flow can never change the login page's options
// (docs/planning/registration-flows.md, D1).
type RegistrationContextService interface {
	Get(ctx context.Context, clientID, tenantID *string, registrationFlowName string) (*RegistrationContextResult, error)
}

type registrationContextService struct {
	clientRepo               ClientRepository
	registrationFlowRoleRepo RegistrationFlowRoleRepository
	securitySettingRepo      secpolicy.SecuritySettingRepository
}

func NewRegistrationContextService(
	clientRepo ClientRepository,
	registrationFlowRoleRepo RegistrationFlowRoleRepository,
	securitySettingRepo secpolicy.SecuritySettingRepository,
) RegistrationContextService {
	return &registrationContextService{
		clientRepo:               clientRepo,
		registrationFlowRoleRepo: registrationFlowRoleRepo,
		securitySettingRepo:      securitySettingRepo,
	}
}

// registrationContextNotFound is the single response for every resolution
// failure — unknown client, inactive client, unknown flow, wrong client, wrong
// tenant, system flow, inactive flow.
//
// The flow name is deliberately guessable, so this endpoint would otherwise be
// the cheapest enumeration primitive in the system. Distinguishing "inactive"
// from "unknown" would additionally leak the operator's kill switch: whoever
// holds a revoked link could poll until it was re-enabled. /oauth/authorize
// collapses these for the same reason.
func registrationContextNotFound() error {
	return apperror.NewNotFoundWithReason("registration flow not found for this client")
}

func (s *registrationContextService) Get(ctx context.Context, clientID, tenantID *string, registrationFlowName string) (*RegistrationContextResult, error) {
	_, span := otel.Tracer("service").Start(ctx, "registrationContext.get")
	defer span.End()

	// resolvePublicClient is the only correct public resolver: it rejects the
	// first-party system clients and binds the request to the tenant the host
	// actually resolves to.
	client, err := resolvePublicClient(ctx, s.clientRepo, clientID, tenantID)
	if err != nil || client == nil {
		span.SetStatus(codes.Error, "client not resolved")
		return nil, registrationContextNotFound()
	}
	if client.Status != shared.StatusActive {
		span.SetStatus(codes.Error, "client inactive")
		return nil, registrationContextNotFound()
	}

	tenantIDValue := clientTenantID(client)
	if tenantIDValue == 0 {
		span.SetStatus(codes.Error, "client tenant not resolved")
		return nil, registrationContextNotFound()
	}

	basePolicy := secpolicy.LoadRegistrationPolicy(s.securitySettingRepo, tenantIDValue)

	// Resolve the flow through the same guards the register path uses, so this
	// endpoint can never describe a flow that /register would refuse. Any failure
	// collapses to the shared not-found.
	var flow *RegistrationFlow
	name := strings.ToLower(strings.TrimSpace(registrationFlowName))
	if name != "" {
		resolved, flowErr := s.resolveFlow(client.ClientID, tenantIDValue, name)
		if flowErr != nil {
			span.SetStatus(codes.Error, "flow not resolved")
			return nil, flowErr
		}
		flow = resolved
	}

	effective := effectiveRegistrationPolicy(basePolicy, flow)

	span.SetStatus(codes.Ok, "")
	return &RegistrationContextResult{
		RegistrationFlow:     name,
		RequiredFields:       effectiveRequiredFields(flow, effective),
		VerificationRequired: effective.RequireEmailVerification,
	}, nil
}

// resolveFlow mirrors registrationFlowByName's guard set — client AND tenant in
// the predicate, system flows refused (invite-only), inactive refused — but
// reports every failure identically.
func (s *registrationContextService) resolveFlow(clientID, tenantID int64, name string) (*RegistrationFlow, error) {
	if s.registrationFlowRoleRepo == nil {
		return nil, apperror.NewInternal("registration flow repository is unavailable", nil)
	}
	flow, err := s.registrationFlowRoleRepo.FindByNameAndClientTenant(name, clientID, tenantID)
	if err != nil {
		return nil, apperror.NewInternal("failed to load registration flow", err)
	}
	if flow == nil || flow.TenantID != tenantID || flow.IsSystem || flow.Status != shared.StatusActive {
		return nil, registrationContextNotFound()
	}
	return flow, nil
}

// effectiveRequiredFields merges the flow's own required_fields with the
// requirements the tenant policy imposes independently, so the form collects
// everything the register call will actually demand.
//
// The two policy-derived additions matter: a flow with an empty required_fields
// but verification_required still needs an email, and a tenant requiring phone
// verification needs a phone even with no flow at all. Omitting either would
// leave a 400 the UI cannot pre-empt.
func effectiveRequiredFields(flow *RegistrationFlow, effective *secpolicy.RegistrationPolicy) []string {
	seen := map[string]bool{}
	ordered := make([]string, 0, 3)
	add := func(field string) {
		if field == "" || seen[field] {
			return
		}
		seen[field] = true
		ordered = append(ordered, field)
	}

	if flow != nil {
		fields, err := parseRequiredRegistrationFields(flow.RequiredFields)
		if err == nil {
			for _, field := range fields {
				switch strings.ToLower(strings.TrimSpace(field)) {
				case "fullname", "email", "phone":
					add(strings.ToLower(strings.TrimSpace(field)))
				}
			}
		}
	}

	if effective != nil {
		if effective.RequireEmailVerification {
			add("email")
		}
		if effective.RequirePhoneVerification {
			add("phone")
		}
	}

	return ordered
}
