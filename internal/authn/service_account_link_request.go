package authn

import (
	"context"
	"time"

	"github.com/maintainerd/maintainerd-auth/internal/platform/apperror"
	"github.com/maintainerd/maintainerd-auth/internal/platform/crypto"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
)

// accountLinkTokenBytes is the size of the cryptographically random confirmation
// token (32 bytes, per the plan). GenerateRandomString base64url-encodes it.
const accountLinkTokenBytes = 32

// accountLinkTTL is the short lifetime of a pending link request (15 minutes).
const accountLinkTTL = 15 * time.Minute

// AccountIdentityLinker is the narrow capability the account-link flow needs to
// inspect and create external identity links. It is satisfied by an app-layer
// adapter over the user_identities repository, so the authn package does not
// import the user domain and can create a client-less (NULL client_id) link.
type AccountIdentityLinker interface {
	// FindLinkedUserID returns the user_id an external (provider, sub) identity
	// is already linked to within a tenant, or found=false when unlinked.
	FindLinkedUserID(tenantID int64, provider, sub string) (userID int64, found bool, err error)
	// LinkIdentity creates a user_identities row attaching the external identity
	// to userID. client_id is left NULL (identity is user data, not client data).
	LinkIdentity(tenantID, userID int64, provider, sub string, claims []byte) error
}

// InitiateAccountLinkInput carries the parameters for creating a pending link
// request when a social login collides with an existing local account.
type InitiateAccountLinkInput struct {
	TenantID        int64
	ExistingUserID  int64
	ProviderName    string
	ProviderSubject string
	ProviderEmail   string
	ProviderClaims  []byte
	IPAddress       string
}

// AccountLinkConfirmResult is returned after a successful confirmation.
type AccountLinkConfirmResult struct {
	UUID           string
	ExistingUserID int64
	ProviderName   string
}

// AccountLinkRequestService manages the social-login account-linking flow.
type AccountLinkRequestService interface {
	// Initiate creates a pending link request with a 32-byte crypto-random
	// confirmation token and a 15-minute TTL. It is called by the social-login
	// provisioning path on an email collision.
	Initiate(ctx context.Context, in InitiateAccountLinkInput) (*AccountLinkRequest, error)
	// Confirm finalizes a link request: it validates the token, requires the
	// caller to be authenticated as the existing account, attaches the external
	// identity, and marks the request confirmed.
	Confirm(ctx context.Context, token string, authUserID, authTenantID int64) (*AccountLinkConfirmResult, error)
	// ExpireStale marks pending requests past their TTL as expired.
	ExpireStale(ctx context.Context) (int64, error)
}

type accountLinkRequestService struct {
	repo     AccountLinkRequestRepository
	userRepo UserRepository
	linker   AccountIdentityLinker
}

// NewAccountLinkRequestService creates a new AccountLinkRequestService.
func NewAccountLinkRequestService(repo AccountLinkRequestRepository, userRepo UserRepository, linker AccountIdentityLinker) AccountLinkRequestService {
	return &accountLinkRequestService{repo: repo, userRepo: userRepo, linker: linker}
}

// Initiate implements AccountLinkRequestService.
func (s *accountLinkRequestService) Initiate(ctx context.Context, in InitiateAccountLinkInput) (*AccountLinkRequest, error) {
	_, span := otel.Tracer("service").Start(ctx, "accountLink.initiate")
	defer span.End()
	span.SetAttributes(attribute.Int64("tenant.id", in.TenantID), attribute.Int64("user.id", in.ExistingUserID))

	token, err := crypto.GenerateRandomString(accountLinkTokenBytes)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "token generation failed")
		return nil, apperror.NewInternal("failed to generate confirmation token", err)
	}

	claims := in.ProviderClaims
	if len(claims) == 0 {
		claims = []byte("{}")
	}

	req := &AccountLinkRequest{
		TenantID:          in.TenantID,
		ExistingUserID:    in.ExistingUserID,
		ProviderName:      in.ProviderName,
		ProviderSubject:   in.ProviderSubject,
		ProviderEmail:     strPtrOrNil(in.ProviderEmail),
		ProviderClaims:    claims,
		Status:            "pending",
		ConfirmationToken: token,
		IPAddress:         strPtrOrNil(in.IPAddress),
		ExpiresAt:         time.Now().Add(accountLinkTTL),
	}
	created, err := s.repo.Create(req)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "create failed")
		return nil, err
	}

	span.SetStatus(codes.Ok, "")
	return created, nil
}

// Confirm implements AccountLinkRequestService.
func (s *accountLinkRequestService) Confirm(ctx context.Context, token string, authUserID, authTenantID int64) (*AccountLinkConfirmResult, error) {
	_, span := otel.Tracer("service").Start(ctx, "accountLink.confirm")
	defer span.End()

	req, err := s.repo.FindByToken(token)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "lookup failed")
		return nil, apperror.NewInternal("failed to look up link request", err)
	}
	if req == nil {
		return nil, apperror.NewNotFoundWithReason("link request not found")
	}

	// Reject already-finalized requests (confirmed / rejected / expired).
	if req.Status != "pending" {
		return nil, apperror.NewConflict("link request is no longer pending")
	}

	// Reject expired requests (and mark them so the cleanup worker is not needed
	// to observe the transition).
	if req.IsExpired() {
		_ = s.repo.MarkExpired(req.AccountLinkRequestID, time.Now())
		return nil, apperror.NewConflict("link request has expired")
	}

	// The caller must be authenticated as the existing account (re-auth gate).
	if req.TenantID != authTenantID || req.ExistingUserID != authUserID {
		return nil, apperror.NewForbidden("you must be signed in to the account being linked")
	}

	// Reject if the existing user has been deleted.
	existing, err := s.userRepo.FindByID(req.ExistingUserID)
	if err != nil || existing == nil {
		return nil, apperror.NewNotFoundWithReason("the account to link no longer exists")
	}

	// Reject if the provider identity is already linked to a different user.
	linkedUserID, found, err := s.linker.FindLinkedUserID(req.TenantID, req.ProviderName, req.ProviderSubject)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "identity lookup failed")
		return nil, apperror.NewInternal("failed to check existing identity link", err)
	}
	if found && linkedUserID != req.ExistingUserID {
		return nil, apperror.NewConflict("this provider identity is already linked to a different account")
	}

	// Attach the external identity (idempotent when it already points at this user).
	if !found {
		if err := s.linker.LinkIdentity(req.TenantID, req.ExistingUserID, req.ProviderName, req.ProviderSubject, []byte(req.ProviderClaims)); err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, "link creation failed")
			return nil, apperror.NewInternal("failed to link identity", err)
		}
	}

	now := time.Now()
	req.Status = "confirmed"
	req.ConfirmedAt = &now
	if err := s.repo.MarkConfirmed(req.AccountLinkRequestID, now); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "mark confirmed failed")
		return nil, err
	}

	span.SetStatus(codes.Ok, "")
	return &AccountLinkConfirmResult{
		UUID:           req.AccountLinkRequestUUID.String(),
		ExistingUserID: req.ExistingUserID,
		ProviderName:   req.ProviderName,
	}, nil
}

// ExpireStale implements AccountLinkRequestService.
func (s *accountLinkRequestService) ExpireStale(ctx context.Context) (int64, error) {
	_, span := otel.Tracer("service").Start(ctx, "accountLink.expireStale")
	defer span.End()
	return s.repo.ExpireStale(time.Now())
}

func strPtrOrNil(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}
