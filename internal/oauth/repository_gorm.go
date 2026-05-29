package oauth

import (
	"errors"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// OAuthAuthorizationCodeRepository defines data access operations for
// authorization codes.
type OAuthAuthorizationCodeRepository interface {
	BaseRepositoryMethods[OAuthAuthorizationCode]
	WithTx(tx *gorm.DB) OAuthAuthorizationCodeRepository
	FindByCodeHash(codeHash string) (*OAuthAuthorizationCode, error)
	MarkUsed(codeID int64) error
	DeleteExpired(before time.Time) (int64, error)
}

type oauthAuthorizationCodeRepository struct {
	*BaseRepository[OAuthAuthorizationCode]
}

// NewOAuthAuthorizationCodeRepository creates a new OAuthAuthorizationCodeRepository.
func NewOAuthAuthorizationCodeRepository(db *gorm.DB) OAuthAuthorizationCodeRepository {
	return &oauthAuthorizationCodeRepository{
		BaseRepository: NewBaseRepository[OAuthAuthorizationCode](db, "oauth_authorization_code_uuid", "oauth_authorization_code_id"),
	}
}

func (r *oauthAuthorizationCodeRepository) WithTx(tx *gorm.DB) OAuthAuthorizationCodeRepository {
	return &oauthAuthorizationCodeRepository{
		BaseRepository: r.BaseRepository.WithTx(tx),
	}
}

// FindByCodeHash looks up an authorization code by its SHA-256 hash.
// Returns nil, nil when no matching code exists.
func (r *oauthAuthorizationCodeRepository) FindByCodeHash(codeHash string) (*OAuthAuthorizationCode, error) {
	var code OAuthAuthorizationCode
	err := r.DB().
		Preload("Client").
		Preload("Client.IdentityProvider").
		Where("code_hash = ?", codeHash).
		First(&code).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &code, nil
}

// MarkUsed marks an authorization code as consumed so it cannot be reused.
func (r *oauthAuthorizationCodeRepository) MarkUsed(codeID int64) error {
	now := time.Now()
	return r.DB().Model(&OAuthAuthorizationCode{}).
		Where("oauth_authorization_code_id = ?", codeID).
		Updates(map[string]any{
			"is_used": true,
			"used_at": now,
		}).Error
}

// DeleteExpired removes authorization codes that expired before the given
// cutoff time. Returns the number of rows deleted.
func (r *oauthAuthorizationCodeRepository) DeleteExpired(before time.Time) (int64, error) {
	result := r.DB().
		Where("expires_at < ?", before).
		Delete(&OAuthAuthorizationCode{})
	return result.RowsAffected, result.Error
}

// OAuthCIBARequestRepository defines data access operations for
// Client-Initiated Backchannel Authentication requests.
type OAuthCIBARequestRepository interface {
	BaseRepositoryMethods[OAuthCIBARequest]
	WithTx(tx *gorm.DB) OAuthCIBARequestRepository
	FindByAuthReqIDHash(hash string) (*OAuthCIBARequest, error)
	UpdateStatus(id int64, status string) error
	UpdateApproval(id int64, userID int64) error
	UpdateLastPollAt(id int64) error
	MarkNotificationSent(id int64) error
	DeleteExpired(before time.Time) (int64, error)
}

type oauthCIBARequestRepository struct {
	*BaseRepository[OAuthCIBARequest]
}

// NewOAuthCIBARequestRepository creates a new OAuthCIBARequestRepository.
func NewOAuthCIBARequestRepository(db *gorm.DB) OAuthCIBARequestRepository {
	return &oauthCIBARequestRepository{
		BaseRepository: NewBaseRepository[OAuthCIBARequest](db, "oauth_ciba_request_uuid", "oauth_ciba_request_id"),
	}
}

func (r *oauthCIBARequestRepository) WithTx(tx *gorm.DB) OAuthCIBARequestRepository {
	return &oauthCIBARequestRepository{
		BaseRepository: r.BaseRepository.WithTx(tx),
	}
}

// FindByAuthReqIDHash looks up a CIBA request by the SHA-256 hash of the
// auth_req_id. Returns nil, nil when not found.
func (r *oauthCIBARequestRepository) FindByAuthReqIDHash(hash string) (*OAuthCIBARequest, error) {
	var req OAuthCIBARequest
	err := r.DB().
		Preload("Client").
		Preload("Client.IdentityProvider").
		Preload("User").
		Where("auth_req_id_hash = ?", hash).
		First(&req).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &req, nil
}

// UpdateStatus sets the status on a CIBA request.
func (r *oauthCIBARequestRepository) UpdateStatus(id int64, status string) error {
	return r.DB().Model(&OAuthCIBARequest{}).
		Where("oauth_ciba_request_id = ?", id).
		Update("status", status).Error
}

// UpdateApproval sets status=approved and records the approving user.
func (r *oauthCIBARequestRepository) UpdateApproval(id int64, userID int64) error {
	return r.DB().Model(&OAuthCIBARequest{}).
		Where("oauth_ciba_request_id = ?", id).
		Updates(map[string]any{
			"status":  CIBAStatusApproved,
			"user_id": userID,
		}).Error
}

// UpdateLastPollAt records when the client last polled.
func (r *oauthCIBARequestRepository) UpdateLastPollAt(id int64) error {
	now := time.Now()
	return r.DB().Model(&OAuthCIBARequest{}).
		Where("oauth_ciba_request_id = ?", id).
		Update("last_poll_at", now).Error
}

// MarkNotificationSent sets the notification_sent_at timestamp.
func (r *oauthCIBARequestRepository) MarkNotificationSent(id int64) error {
	now := time.Now()
	return r.DB().Model(&OAuthCIBARequest{}).
		Where("oauth_ciba_request_id = ?", id).
		Update("notification_sent_at", now).Error
}

// DeleteExpired removes CIBA requests that expired before the given cutoff.
func (r *oauthCIBARequestRepository) DeleteExpired(before time.Time) (int64, error) {
	result := r.DB().
		Where("expires_at < ?", before).
		Delete(&OAuthCIBARequest{})
	return result.RowsAffected, result.Error
}

// OAuthConsentChallengeRepository defines data access operations for pending
// consent challenges.
type OAuthConsentChallengeRepository interface {
	BaseRepositoryMethods[OAuthConsentChallenge]
	WithTx(tx *gorm.DB) OAuthConsentChallengeRepository
	FindChallengeByUUID(challengeUUID uuid.UUID) (*OAuthConsentChallenge, error)
	DeleteChallengeByUUID(challengeUUID uuid.UUID) error
	DeleteExpired(before time.Time) (int64, error)
}

type oauthConsentChallengeRepository struct {
	*BaseRepository[OAuthConsentChallenge]
}

// NewOAuthConsentChallengeRepository creates a new OAuthConsentChallengeRepository.
func NewOAuthConsentChallengeRepository(db *gorm.DB) OAuthConsentChallengeRepository {
	return &oauthConsentChallengeRepository{
		BaseRepository: NewBaseRepository[OAuthConsentChallenge](db, "oauth_consent_challenge_uuid", "oauth_consent_challenge_id"),
	}
}

func (r *oauthConsentChallengeRepository) WithTx(tx *gorm.DB) OAuthConsentChallengeRepository {
	return &oauthConsentChallengeRepository{
		BaseRepository: r.BaseRepository.WithTx(tx),
	}
}

// FindChallengeByUUID looks up a consent challenge by its UUID. Returns nil, nil when
// no matching challenge exists.
func (r *oauthConsentChallengeRepository) FindChallengeByUUID(challengeUUID uuid.UUID) (*OAuthConsentChallenge, error) {
	var challenge OAuthConsentChallenge
	err := r.DB().
		Preload("Client").
		Preload("Client.IdentityProvider").
		Where("oauth_consent_challenge_uuid = ?", challengeUUID).
		First(&challenge).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &challenge, nil
}

// DeleteChallengeByUUID removes a consent challenge after it has been resolved.
func (r *oauthConsentChallengeRepository) DeleteChallengeByUUID(challengeUUID uuid.UUID) error {
	return r.DB().
		Where("oauth_consent_challenge_uuid = ?", challengeUUID).
		Delete(&OAuthConsentChallenge{}).Error
}

// DeleteExpired removes consent challenges that expired before the given
// cutoff time. Returns the number of rows deleted.
func (r *oauthConsentChallengeRepository) DeleteExpired(before time.Time) (int64, error) {
	result := r.DB().
		Where("expires_at < ?", before).
		Delete(&OAuthConsentChallenge{})
	return result.RowsAffected, result.Error
}

// OAuthConsentGrantRepository defines data access operations for user consent
// grants per client.
type OAuthConsentGrantRepository interface {
	BaseRepositoryMethods[OAuthConsentGrant]
	WithTx(tx *gorm.DB) OAuthConsentGrantRepository
	FindByUserAndClient(userID, clientID int64) (*OAuthConsentGrant, error)
	Upsert(grant *OAuthConsentGrant) (*OAuthConsentGrant, error)
	DeleteByUserAndClient(userID, clientID int64) error
	FindByUserID(userID int64) ([]OAuthConsentGrant, error)
}

type oauthConsentGrantRepository struct {
	*BaseRepository[OAuthConsentGrant]
}

// NewOAuthConsentGrantRepository creates a new OAuthConsentGrantRepository.
func NewOAuthConsentGrantRepository(db *gorm.DB) OAuthConsentGrantRepository {
	return &oauthConsentGrantRepository{
		BaseRepository: NewBaseRepository[OAuthConsentGrant](db, "oauth_consent_grant_uuid", "oauth_consent_grant_id"),
	}
}

func (r *oauthConsentGrantRepository) WithTx(tx *gorm.DB) OAuthConsentGrantRepository {
	return &oauthConsentGrantRepository{
		BaseRepository: r.BaseRepository.WithTx(tx),
	}
}

// FindByUserAndClient looks up the consent grant for a user-client pair.
// Returns nil, nil when no consent exists.
func (r *oauthConsentGrantRepository) FindByUserAndClient(userID, clientID int64) (*OAuthConsentGrant, error) {
	var grant OAuthConsentGrant
	err := r.DB().
		Where("user_id = ? AND client_id = ?", userID, clientID).
		First(&grant).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &grant, nil
}

// Upsert creates a new consent grant or updates the scopes if one already
// exists for the user-client pair.
func (r *oauthConsentGrantRepository) Upsert(grant *OAuthConsentGrant) (*OAuthConsentGrant, error) {
	existing, err := r.FindByUserAndClient(grant.UserID, grant.ClientID)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		existing.Scopes = grant.Scopes
		if err := r.DB().Save(existing).Error; err != nil {
			return nil, err
		}
		return existing, nil
	}
	return r.Create(grant)
}

// DeleteByUserAndClient removes the consent grant for a user-client pair.
func (r *oauthConsentGrantRepository) DeleteByUserAndClient(userID, clientID int64) error {
	return r.DB().
		Where("user_id = ? AND client_id = ?", userID, clientID).
		Delete(&OAuthConsentGrant{}).Error
}

// FindByUserID returns all consent grants for a user.
func (r *oauthConsentGrantRepository) FindByUserID(userID int64) ([]OAuthConsentGrant, error) {
	var grants []OAuthConsentGrant
	err := r.DB().
		Preload("Client").
		Where("user_id = ?", userID).
		Find(&grants).Error
	return grants, err
}

// OAuthDeviceCodeRepository defines data access operations for device
// authorization codes (RFC 8628).
type OAuthDeviceCodeRepository interface {
	BaseRepositoryMethods[OAuthDeviceCode]
	WithTx(tx *gorm.DB) OAuthDeviceCodeRepository
	FindByDeviceCodeHash(hash string) (*OAuthDeviceCode, error)
	FindByUserCode(userCode string) (*OAuthDeviceCode, error)
	UpdateStatus(id int64, status string, userID *int64) error
	UpdateLastPollAt(id int64) error
	DeleteExpired(before time.Time) (int64, error)
}

type oauthDeviceCodeRepository struct {
	*BaseRepository[OAuthDeviceCode]
}

// NewOAuthDeviceCodeRepository creates a new OAuthDeviceCodeRepository.
func NewOAuthDeviceCodeRepository(db *gorm.DB) OAuthDeviceCodeRepository {
	return &oauthDeviceCodeRepository{
		BaseRepository: NewBaseRepository[OAuthDeviceCode](db, "oauth_device_code_uuid", "oauth_device_code_id"),
	}
}

func (r *oauthDeviceCodeRepository) WithTx(tx *gorm.DB) OAuthDeviceCodeRepository {
	return &oauthDeviceCodeRepository{
		BaseRepository: r.BaseRepository.WithTx(tx),
	}
}

// FindByDeviceCodeHash looks up a device code record by the SHA-256 hash of the
// raw device_code. Returns nil, nil when not found.
func (r *oauthDeviceCodeRepository) FindByDeviceCodeHash(hash string) (*OAuthDeviceCode, error) {
	var code OAuthDeviceCode
	err := r.DB().
		Preload("Client").
		Preload("Client.IdentityProvider").
		Preload("User").
		Where("device_code_hash = ?", hash).
		First(&code).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &code, nil
}

// FindByUserCode looks up a pending device code by the human-readable user_code.
// Returns nil, nil when not found.
func (r *oauthDeviceCodeRepository) FindByUserCode(userCode string) (*OAuthDeviceCode, error) {
	var code OAuthDeviceCode
	err := r.DB().
		Preload("Client").
		Where("user_code = ? AND status = 'pending'", userCode).
		First(&code).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &code, nil
}

// UpdateStatus sets the status and optionally the approving user on a device code.
func (r *oauthDeviceCodeRepository) UpdateStatus(id int64, status string, userID *int64) error {
	updates := map[string]any{"status": status}
	if userID != nil {
		updates["user_id"] = *userID
	}
	return r.DB().Model(&OAuthDeviceCode{}).
		Where("oauth_device_code_id = ?", id).
		Updates(updates).Error
}

// UpdateLastPollAt records when the device last polled to enforce the minimum
// polling interval.
func (r *oauthDeviceCodeRepository) UpdateLastPollAt(id int64) error {
	now := time.Now()
	return r.DB().Model(&OAuthDeviceCode{}).
		Where("oauth_device_code_id = ?", id).
		Update("last_poll_at", now).Error
}

// DeleteExpired removes device codes that expired before the given cutoff.
func (r *oauthDeviceCodeRepository) DeleteExpired(before time.Time) (int64, error) {
	result := r.DB().
		Where("expires_at < ?", before).
		Delete(&OAuthDeviceCode{})
	return result.RowsAffected, result.Error
}

// OAuthPARRequestRepository defines data access operations for Pushed
// Authorization Requests (RFC 9126).
type OAuthPARRequestRepository interface {
	BaseRepositoryMethods[OAuthPARRequest]
	WithTx(tx *gorm.DB) OAuthPARRequestRepository
	FindByRequestURIHash(hash string) (*OAuthPARRequest, error)
	MarkUsed(id int64) error
	DeleteExpired(before time.Time) (int64, error)
}

type oauthPARRequestRepository struct {
	*BaseRepository[OAuthPARRequest]
}

// NewOAuthPARRequestRepository creates a new OAuthPARRequestRepository.
func NewOAuthPARRequestRepository(db *gorm.DB) OAuthPARRequestRepository {
	return &oauthPARRequestRepository{
		BaseRepository: NewBaseRepository[OAuthPARRequest](db, "oauth_par_request_uuid", "oauth_par_request_id"),
	}
}

func (r *oauthPARRequestRepository) WithTx(tx *gorm.DB) OAuthPARRequestRepository {
	return &oauthPARRequestRepository{
		BaseRepository: r.BaseRepository.WithTx(tx),
	}
}

// FindByRequestURIHash looks up a PAR request by the SHA-256 hash of its
// request_uri token. Returns nil, nil when no matching record exists.
func (r *oauthPARRequestRepository) FindByRequestURIHash(hash string) (*OAuthPARRequest, error) {
	var req OAuthPARRequest
	err := r.DB().
		Preload("Client").
		Preload("Client.ClientURIs").
		Where("request_uri_hash = ? AND is_used = false", hash).
		First(&req).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &req, nil
}

// MarkUsed marks a PAR request as consumed so it cannot be replayed.
func (r *oauthPARRequestRepository) MarkUsed(id int64) error {
	return r.DB().Model(&OAuthPARRequest{}).
		Where("oauth_par_request_id = ?", id).
		Update("is_used", true).Error
}

// DeleteExpired removes PAR requests that expired before the given cutoff.
func (r *oauthPARRequestRepository) DeleteExpired(before time.Time) (int64, error) {
	result := r.DB().
		Where("expires_at < ?", before).
		Delete(&OAuthPARRequest{})
	return result.RowsAffected, result.Error
}

// OAuthRefreshTokenRepository defines data access operations for OAuth refresh
// tokens with family-based rotation tracking.
type OAuthRefreshTokenRepository interface {
	BaseRepositoryMethods[OAuthRefreshToken]
	WithTx(tx *gorm.DB) OAuthRefreshTokenRepository
	FindByTokenHash(tokenHash string) (*OAuthRefreshToken, error)
	FindActiveByUserAndClient(userID, clientID int64) ([]OAuthRefreshToken, error)
	RevokeByID(tokenID int64) error
	RevokeByFamily(familyID uuid.UUID) (int64, error)
	RevokeByUserAndClient(userID, clientID int64) (int64, error)
	RevokeByUserID(userID int64) (int64, error)
	UpdateLastUsed(tokenID int64) error
	DeleteExpired(before time.Time) (int64, error)
	CountByUserAndClient(userID, clientID int64) (int64, error)
}

type oauthRefreshTokenRepository struct {
	*BaseRepository[OAuthRefreshToken]
}

// NewOAuthRefreshTokenRepository creates a new OAuthRefreshTokenRepository.
func NewOAuthRefreshTokenRepository(db *gorm.DB) OAuthRefreshTokenRepository {
	return &oauthRefreshTokenRepository{
		BaseRepository: NewBaseRepository[OAuthRefreshToken](db, "oauth_refresh_token_uuid", "oauth_refresh_token_id"),
	}
}

func (r *oauthRefreshTokenRepository) WithTx(tx *gorm.DB) OAuthRefreshTokenRepository {
	return &oauthRefreshTokenRepository{
		BaseRepository: r.BaseRepository.WithTx(tx),
	}
}

// FindByTokenHash looks up a refresh token by its SHA-256 hash.
// Returns nil, nil when no matching token exists.
func (r *oauthRefreshTokenRepository) FindByTokenHash(tokenHash string) (*OAuthRefreshToken, error) {
	var token OAuthRefreshToken
	err := r.DB().
		Preload("Client").
		Preload("Client.IdentityProvider").
		Where("token_hash = ?", tokenHash).
		First(&token).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &token, nil
}

// FindActiveByUserAndClient returns all non-revoked, non-expired refresh tokens
// for a user-client pair.
func (r *oauthRefreshTokenRepository) FindActiveByUserAndClient(userID, clientID int64) ([]OAuthRefreshToken, error) {
	var tokens []OAuthRefreshToken
	err := r.DB().
		Where("user_id = ? AND client_id = ? AND is_revoked = false AND expires_at > ?", userID, clientID, time.Now()).
		Find(&tokens).Error
	return tokens, err
}

// RevokeByID revokes a single refresh token.
func (r *oauthRefreshTokenRepository) RevokeByID(tokenID int64) error {
	now := time.Now()
	return r.DB().Model(&OAuthRefreshToken{}).
		Where("oauth_refresh_token_id = ? AND is_revoked = false", tokenID).
		Updates(map[string]any{
			"is_revoked": true,
			"revoked_at": now,
		}).Error
}

// RevokeByFamily revokes all refresh tokens in a family. Used for reuse
// detection — when a rotated-out token is presented again, the entire family
// is considered compromised. Returns the number of tokens revoked.
func (r *oauthRefreshTokenRepository) RevokeByFamily(familyID uuid.UUID) (int64, error) {
	now := time.Now()
	result := r.DB().Model(&OAuthRefreshToken{}).
		Where("family_id = ? AND is_revoked = false", familyID).
		Updates(map[string]any{
			"is_revoked": true,
			"revoked_at": now,
		})
	return result.RowsAffected, result.Error
}

// RevokeByUserAndClient revokes all refresh tokens for a user-client pair.
// Returns the number of tokens revoked.
func (r *oauthRefreshTokenRepository) RevokeByUserAndClient(userID, clientID int64) (int64, error) {
	now := time.Now()
	result := r.DB().Model(&OAuthRefreshToken{}).
		Where("user_id = ? AND client_id = ? AND is_revoked = false", userID, clientID).
		Updates(map[string]any{
			"is_revoked": true,
			"revoked_at": now,
		})
	return result.RowsAffected, result.Error
}

// RevokeByUserID revokes all refresh tokens for a user across all clients.
// Returns the number of tokens revoked.
func (r *oauthRefreshTokenRepository) RevokeByUserID(userID int64) (int64, error) {
	now := time.Now()
	result := r.DB().Model(&OAuthRefreshToken{}).
		Where("user_id = ? AND is_revoked = false", userID).
		Updates(map[string]any{
			"is_revoked": true,
			"revoked_at": now,
		})
	return result.RowsAffected, result.Error
}

// UpdateLastUsed records when a refresh token was last used at token exchange.
func (r *oauthRefreshTokenRepository) UpdateLastUsed(tokenID int64) error {
	return r.DB().Model(&OAuthRefreshToken{}).
		Where("oauth_refresh_token_id = ?", tokenID).
		Update("last_used_at", time.Now()).Error
}

// DeleteExpired removes refresh tokens that expired before the given cutoff.
// Returns the number of rows deleted.
func (r *oauthRefreshTokenRepository) DeleteExpired(before time.Time) (int64, error) {
	result := r.DB().
		Where("expires_at < ?", before).
		Delete(&OAuthRefreshToken{})
	return result.RowsAffected, result.Error
}

// CountByUserAndClient returns the total number of active refresh tokens for
// a given user-client pair. Used to enforce token count limits.
func (r *oauthRefreshTokenRepository) CountByUserAndClient(userID, clientID int64) (int64, error) {
	var count int64
	err := r.DB().Model(&OAuthRefreshToken{}).
		Where("user_id = ? AND client_id = ? AND is_revoked = false AND expires_at > ?", userID, clientID, time.Now()).
		Count(&count).Error
	return count, err
}
