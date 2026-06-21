package mfa

import (
	"context"
	"encoding/base64"
	"fmt"
	"strings"
	"time"

	"github.com/go-webauthn/webauthn/protocol"
	"github.com/go-webauthn/webauthn/webauthn"
	"github.com/google/uuid"
	"github.com/maintainerd/auth/internal/authevent"
	"github.com/maintainerd/auth/internal/platform/apperror"
	"github.com/maintainerd/auth/internal/platform/cache"
	"github.com/maintainerd/auth/internal/platform/config"
	"github.com/maintainerd/auth/internal/platform/middleware"
	"github.com/maintainerd/auth/internal/platform/ptr"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"gorm.io/gorm"
)

const webAuthnSessionPrefix = "webauthn:session:"
const webAuthnSessionTTL = 5 * time.Minute

var (
	beginWebAuthnRegistration = func(wa *webauthn.WebAuthn, user webauthn.User) (*protocol.CredentialCreation, *webauthn.SessionData, error) {
		return wa.BeginRegistration(user)
	}
	createWebAuthnCredential = func(wa *webauthn.WebAuthn, user webauthn.User, session webauthn.SessionData, response *protocol.ParsedCredentialCreationData) (*webauthn.Credential, error) {
		return wa.CreateCredential(user, session, response)
	}
	beginWebAuthnLogin = func(wa *webauthn.WebAuthn, user webauthn.User) (*protocol.CredentialAssertion, *webauthn.SessionData, error) {
		return wa.BeginLogin(user)
	}
	validateWebAuthnLogin = func(wa *webauthn.WebAuthn, user webauthn.User, session webauthn.SessionData, response *protocol.ParsedCredentialAssertionData) (*webauthn.Credential, error) {
		return wa.ValidateLogin(user, session, response)
	}
)

// WebAuthnService handles FIDO2 / WebAuthn passkey registration and authentication.
type WebAuthnService interface {
	// BeginRegistration initiates a credential registration ceremony.
	BeginRegistration(ctx context.Context, userID int64) (*protocol.CredentialCreation, error)
	// FinishRegistration completes registration, persists the credential, and
	// enables WebAuthn on the user.
	FinishRegistration(ctx context.Context, userID int64, credName string, response *protocol.ParsedCredentialCreationData) (*UserMFAWebAuthnCredential, error)
	// BeginAuthentication initiates a credential assertion ceremony.
	BeginAuthentication(ctx context.Context, userID int64) (*protocol.CredentialAssertion, error)
	// FinishAuthentication verifies the assertion and updates the sign counter.
	FinishAuthentication(ctx context.Context, userID int64, response *protocol.ParsedCredentialAssertionData) (*UserMFAWebAuthnCredential, error)
	// DeleteCredential removes a single registered credential.
	DeleteCredential(ctx context.Context, credentialUUIDStr string, userID int64) error
	// DownloadCredential returns a downloadable representation of a credential.
	DownloadCredential(ctx context.Context, credentialUUIDStr string, userID int64) (*WebAuthnCredentialDownloadDTO, error)
}

type webAuthnService struct {
	db               *gorm.DB
	wa               *webauthn.WebAuthn
	userRepo         UserRepository
	mfaWebAuthnCredRepo UserMFAWebAuthnCredentialRepository
	sessionStore     cache.WebAuthnSessionStore
	authEventService authevent.AuthEventService
}

// NewWebAuthnService constructs a WebAuthnService.
func NewWebAuthnService(
	db *gorm.DB,
	userRepo UserRepository,
	mfaWebAuthnCredRepo UserMFAWebAuthnCredentialRepository,
	sessionStore cache.WebAuthnSessionStore,
	authEventService authevent.AuthEventService,
) (WebAuthnService, error) {
	rpID := rpIDFromHostname(config.AppPublicHostname)
	rpOrigins := []string{config.AppPublicHostname}
	if extraOrigins := config.GetEnvOrDefault("WEBAUTHN_EXTRA_ORIGINS", ""); extraOrigins != "" {
		for _, o := range strings.Split(extraOrigins, ",") {
			if o = strings.TrimSpace(o); o != "" {
				rpOrigins = append(rpOrigins, o)
			}
		}
	}
	if overrideID := config.GetEnvOrDefault("WEBAUTHN_RP_ID", ""); overrideID != "" {
		rpID = overrideID
	}

	wa, err := webauthn.New(&webauthn.Config{
		RPDisplayName: "maintainerd",
		RPID:          rpID,
		RPOrigins:     rpOrigins,
	})
	if err != nil {
		return nil, fmt.Errorf("webauthn init: %w", err)
	}

	return &webAuthnService{
		db:               db,
		wa:               wa,
		userRepo:         userRepo,
		mfaWebAuthnCredRepo: mfaWebAuthnCredRepo,
		sessionStore:     sessionStore,
		authEventService: authEventService,
	}, nil
}

// ──────────────────────────────────────────────────────────────────────────────
// webauthn.User adapter
// ──────────────────────────────────────────────────────────────────────────────

// webAuthnUser wraps User + its stored credentials to satisfy the
// webauthn.User interface.
type webAuthnUser struct {
	user  *User
	creds []webauthn.Credential
}

func (u *webAuthnUser) WebAuthnID() []byte                         { return []byte(fmt.Sprintf("%d", u.user.UserID)) }
func (u *webAuthnUser) WebAuthnName() string                       { return u.user.Email }
func (u *webAuthnUser) WebAuthnDisplayName() string                { return u.user.Email }
func (u *webAuthnUser) WebAuthnCredentials() []webauthn.Credential { return u.creds }

// ──────────────────────────────────────────────────────────────────────────────
// Registration
// ──────────────────────────────────────────────────────────────────────────────

func (s *webAuthnService) BeginRegistration(ctx context.Context, userID int64) (*protocol.CredentialCreation, error) {
	_, span := otel.Tracer("service").Start(ctx, "webauthn.begin_registration")
	defer span.End()
	span.SetAttributes(attribute.Int64("user.id", userID))

	wu, err := s.loadWebAuthnUser(userID)
	if err != nil {
		span.RecordError(err)
		return nil, err
	}

	creation, session, err := beginWebAuthnRegistration(s.wa, wu)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "begin registration failed")
		return nil, apperror.NewInternal("WebAuthn registration initiation failed", err)
	}

	if err := s.storeSession(ctx, userID, "reg", session); err != nil {
		return nil, err
	}

	span.SetStatus(codes.Ok, "")
	return creation, nil
}

func (s *webAuthnService) FinishRegistration(ctx context.Context, userID int64, credName string, response *protocol.ParsedCredentialCreationData) (*UserMFAWebAuthnCredential, error) {
	_, span := otel.Tracer("service").Start(ctx, "webauthn.finish_registration")
	defer span.End()
	span.SetAttributes(attribute.Int64("user.id", userID))

	wu, err := s.loadWebAuthnUser(userID)
	if err != nil {
		return nil, err
	}

	session, err := s.loadSession(ctx, userID, "reg")
	if err != nil {
		return nil, err
	}

	cred, err := createWebAuthnCredential(s.wa, wu, *session, response)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "credential creation failed")
		return nil, apperror.NewValidation(fmt.Sprintf("WebAuthn registration failed: %s", err.Error()))
	}

	name := credName
	if strings.TrimSpace(name) == "" {
		name = "Security Key"
	}

	transport := joinTransports(cred.Transport)
	storedCred := &UserMFAWebAuthnCredential{
		UserID:           userID,
		CredentialKeyID:  base64.RawURLEncoding.EncodeToString(cred.ID),
		PublicKey:        cred.PublicKey,
		SignCount:        int64(cred.Authenticator.SignCount),
		Transport:        transport,
		IsBackupEligible: cred.Flags.BackupEligible,
		IsBackupState:    cred.Flags.BackupState,
		Name:             name,
	}

	if err := s.mfaWebAuthnCredRepo.CreateCredential(storedCred); err != nil {
		span.RecordError(err)
		return nil, apperror.NewInternal("failed to persist WebAuthn credential", err)
	}

	now := time.Now()
	if err := s.db.Model(&User{}).Where("user_id = ?", userID).
		Updates(map[string]any{
			"is_webauthn_enabled": true,
			"mfa_enabled_at":      now,
		}).Error; err != nil {
		span.RecordError(err)
		return nil, apperror.NewInternal("failed to update user WebAuthn state", err)
	}

	if err := s.deleteSession(ctx, userID, "reg"); err != nil {
		span.RecordError(err)
		return nil, apperror.NewInternal("failed to clear WebAuthn registration session", err)
	}

	s.authEventService.Log(ctx, authevent.AuthEventInput{
		ActorUserID: &userID,
		IPAddress:   middleware.ClientIPFromContext(ctx),
		UserAgent:   ptr.PtrOrNil(middleware.UserAgentFromContext(ctx)),
		Category:    authevent.AuthEventCategoryAuthn,
		EventType:   authevent.AuthEventTypeTokenCreated,
		Severity:    authevent.AuthEventSeverityInfo,
		Result:      authevent.AuthEventResultSuccess,
		Description: ptr.Ptr("WebAuthn credential registered"),
	})

	span.SetStatus(codes.Ok, "")
	return storedCred, nil
}

// ──────────────────────────────────────────────────────────────────────────────
// Authentication
// ──────────────────────────────────────────────────────────────────────────────

func (s *webAuthnService) BeginAuthentication(ctx context.Context, userID int64) (*protocol.CredentialAssertion, error) {
	_, span := otel.Tracer("service").Start(ctx, "webauthn.begin_authentication")
	defer span.End()
	span.SetAttributes(attribute.Int64("user.id", userID))

	wu, err := s.loadWebAuthnUser(userID)
	if err != nil {
		return nil, err
	}

	assertion, session, err := beginWebAuthnLogin(s.wa, wu)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "begin login failed")
		return nil, apperror.NewInternal("WebAuthn authentication initiation failed", err)
	}

	if err := s.storeSession(ctx, userID, "auth", session); err != nil {
		return nil, err
	}

	span.SetStatus(codes.Ok, "")
	return assertion, nil
}

func (s *webAuthnService) FinishAuthentication(ctx context.Context, userID int64, response *protocol.ParsedCredentialAssertionData) (*UserMFAWebAuthnCredential, error) {
	_, span := otel.Tracer("service").Start(ctx, "webauthn.finish_authentication")
	defer span.End()
	span.SetAttributes(attribute.Int64("user.id", userID))

	wu, err := s.loadWebAuthnUser(userID)
	if err != nil {
		return nil, err
	}

	session, err := s.loadSession(ctx, userID, "auth")
	if err != nil {
		return nil, err
	}

	cred, err := validateWebAuthnLogin(s.wa, wu, *session, response)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "login validation failed")
		return nil, apperror.NewUnauthorized("WebAuthn authentication failed")
	}

	// Update sign counter and last used timestamp.
	credKeyID := base64.RawURLEncoding.EncodeToString(cred.ID)
	stored, err := s.mfaWebAuthnCredRepo.FindByCredentialKeyID(credKeyID)
	if err != nil || stored == nil {
		return nil, apperror.NewInternal("credential not found after validation", err)
	}
	newSignCount := int64(cred.Authenticator.SignCount)
	if stored.SignCount > 0 && newSignCount <= stored.SignCount {
		span.SetStatus(codes.Error, "sign count regression")
		return nil, apperror.NewUnauthorized("WebAuthn authentication failed: sign count regression detected")
	}
	if err := s.mfaWebAuthnCredRepo.UpdateSignCount(stored.CredentialID, newSignCount); err != nil {
		span.RecordError(err)
		return nil, apperror.NewInternal("failed to update WebAuthn sign count", err)
	}
	if err := s.mfaWebAuthnCredRepo.UpdateLastUsed(stored.CredentialID); err != nil {
		span.RecordError(err)
		return nil, apperror.NewInternal("failed to update WebAuthn last-used timestamp", err)
	}

	if err := s.deleteSession(ctx, userID, "auth"); err != nil {
		span.RecordError(err)
		return nil, apperror.NewInternal("failed to clear WebAuthn authentication session", err)
	}

	s.authEventService.Log(ctx, authevent.AuthEventInput{
		ActorUserID: &userID,
		IPAddress:   middleware.ClientIPFromContext(ctx),
		UserAgent:   ptr.PtrOrNil(middleware.UserAgentFromContext(ctx)),
		Category:    authevent.AuthEventCategoryAuthn,
		EventType:   authevent.AuthEventTypeTokenCreated,
		Severity:    authevent.AuthEventSeverityInfo,
		Result:      authevent.AuthEventResultSuccess,
		Description: ptr.Ptr("WebAuthn authentication succeeded"),
	})

	span.SetStatus(codes.Ok, "")
	return stored, nil
}

func (s *webAuthnService) DeleteCredential(ctx context.Context, credentialUUIDStr string, userID int64) error {
	_, span := otel.Tracer("service").Start(ctx, "webauthn.delete_credential")
	defer span.End()

	creds, err := s.mfaWebAuthnCredRepo.FindByUserID(userID)
	if err != nil {
		return apperror.NewInternal("credential lookup failed", err)
	}

	var target *UserMFAWebAuthnCredential
	for i := range creds {
		if creds[i].CredentialUUID.String() == credentialUUIDStr {
			target = &creds[i]
			break
		}
	}
	if target == nil {
		return apperror.NewNotFound("credential not found")
	}

	if err := s.mfaWebAuthnCredRepo.DeleteCredentialByID(target.CredentialID, userID); err != nil {
		return apperror.NewInternal("failed to delete credential", err)
	}

	// Disable WebAuthn on user if no credentials remain.
	remaining, err := s.mfaWebAuthnCredRepo.FindByUserID(userID)
	if err != nil {
		span.RecordError(err)
		return apperror.NewInternal("failed to list remaining WebAuthn credentials", err)
	}
	if len(remaining) == 0 {
		if err := s.db.Model(&User{}).Where("user_id = ?", userID).
			Updates(map[string]any{"is_webauthn_enabled": false}).Error; err != nil {
			span.RecordError(err)
			return apperror.NewInternal("failed to update user WebAuthn state", err)
		}
	}

	span.SetStatus(codes.Ok, "")
	return nil
}

// ──────────────────────────────────────────────────────────────────────────────
// Internal helpers
// ──────────────────────────────────────────────────────────────────────────────

func (s *webAuthnService) loadWebAuthnUser(userID int64) (*webAuthnUser, error) {
	user, err := s.userRepo.FindByID(userID)
	if err != nil || user == nil {
		return nil, apperror.NewNotFound("user not found")
	}

	storedCreds, err := s.mfaWebAuthnCredRepo.FindByUserID(userID)
	if err != nil {
		return nil, apperror.NewInternal("credential lookup failed", err)
	}

	waCreds := make([]webauthn.Credential, 0, len(storedCreds))
	for _, sc := range storedCreds {
		id, err := base64.RawURLEncoding.DecodeString(sc.CredentialKeyID)
		if err != nil {
			continue
		}
		waCreds = append(waCreds, webauthn.Credential{
			ID:        id,
			PublicKey: sc.PublicKey,
			Authenticator: webauthn.Authenticator{
				// #nosec G115 -- sign count from webauthn library fits uint32
				SignCount: uint32(sc.SignCount),
			},
			Flags: webauthn.CredentialFlags{
				BackupEligible: sc.IsBackupEligible,
				BackupState:    sc.IsBackupState,
			},
		})
	}

	return &webAuthnUser{user: user, creds: waCreds}, nil
}

func (s *webAuthnService) sessionKey(userID int64, kind string) string {
	return fmt.Sprintf("%s%d:%s", webAuthnSessionPrefix, userID, kind)
}

func (s *webAuthnService) storeSession(ctx context.Context, userID int64, kind string, session *webauthn.SessionData) error {
	return s.sessionStore.SetSession(ctx, s.sessionKey(userID, kind), session, webAuthnSessionTTL)
}

func (s *webAuthnService) loadSession(ctx context.Context, userID int64, kind string) (*webauthn.SessionData, error) {
	var session webauthn.SessionData
	if err := s.sessionStore.GetSession(ctx, s.sessionKey(userID, kind), &session); err != nil {
		return nil, apperror.NewValidation("WebAuthn session expired or not found; please restart the ceremony")
	}
	return &session, nil
}

func (s *webAuthnService) deleteSession(ctx context.Context, userID int64, kind string) error {
	return s.sessionStore.DeleteSession(ctx, s.sessionKey(userID, kind))
}

// ──────────────────────────────────────────────────────────────────────────────
// Download
// ──────────────────────────────────────────────────────────────────────────────

func (s *webAuthnService) DownloadCredential(ctx context.Context, credentialUUIDStr string, userID int64) (*WebAuthnCredentialDownloadDTO, error) {
	credUUID, err := uuid.Parse(credentialUUIDStr)
	if err != nil {
		return nil, apperror.NewValidation("invalid credential UUID")
	}
	cred, err := s.mfaWebAuthnCredRepo.FindByUUID(credUUID)
	if err != nil || cred == nil {
		return nil, apperror.NewNotFound("credential not found")
	}
	if cred.UserID != userID {
		return nil, apperror.NewNotFound("credential not found")
	}

	dto := &WebAuthnCredentialDownloadDTO{
		CredentialUUID:   cred.CredentialUUID.String(),
		Name:             cred.Name,
		CredentialKeyID:  cred.CredentialKeyID,
		PublicKeyBase64:  base64.RawStdEncoding.EncodeToString(cred.PublicKey),
		Transport:        cred.Transport,
		IsBackupEligible: cred.IsBackupEligible,
		IsBackupState:    cred.IsBackupState,
		CreatedAt:        cred.CreatedAt.Format(time.RFC3339),
	}
	if cred.AAGUID != nil {
		dto.AAGUID = cred.AAGUID.String()
	}
	return dto, nil
}

// ──────────────────────────────────────────────────────────────────────────────
// Utility helpers
// ──────────────────────────────────────────────────────────────────────────────

func rpIDFromHostname(hostname string) string {
	host := strings.TrimPrefix(hostname, "https://")
	host = strings.TrimPrefix(host, "http://")
	if idx := strings.IndexByte(host, ':'); idx >= 0 {
		host = host[:idx]
	}
	return host
}

func joinTransports(ts []protocol.AuthenticatorTransport) string {
	parts := make([]string, len(ts))
	for i, t := range ts {
		parts[i] = string(t)
	}
	return strings.Join(parts, ",")
}
