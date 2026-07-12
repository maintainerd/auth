package user

import (
	"context"
	"log/slog"

	"gorm.io/gorm"
)

type UserConsentService interface {
	Record(ctx context.Context, tx *gorm.DB, userID, tenantID int64, consentType, policyVersion, ipAddress, userAgent string) error
	FindByUserID(ctx context.Context, userID int64) ([]UserConsent, error)
	Withdraw(ctx context.Context, userID, tenantID int64, consentType, ipAddress, userAgent string) error
}

type userConsentService struct {
	repo UserConsentRepository
}

func NewUserConsentService(repo UserConsentRepository) UserConsentService {
	return &userConsentService{repo: repo}
}

func (s *userConsentService) Record(ctx context.Context, tx *gorm.DB, userID, tenantID int64, consentType, policyVersion, ipAddress, userAgent string) error {
	consent := &UserConsent{
		UserID:        userID,
		TenantID:      tenantID,
		ConsentType:   consentType,
		PolicyVersion: policyVersion,
		Accepted:      true,
		IPAddress:     &ipAddress,
		UserAgent:     &userAgent,
	}

	var err error
	if tx != nil {
		err = tx.WithContext(ctx).Create(consent).Error
	} else {
		err = s.repo.CreateConsent(consent)
	}

	if err != nil {
		slog.ErrorContext(ctx, "failed to record user consent",
			"user_id", userID,
			"consent_type", consentType,
			"error", err,
		)
		return err
	}

	slog.InfoContext(ctx, "user consent recorded",
		"user_id", userID,
		"consent_type", consentType,
		"policy_version", policyVersion,
	)
	return nil
}

func (s *userConsentService) FindByUserID(ctx context.Context, userID int64) ([]UserConsent, error) {
	return s.repo.FindByUserID(userID)
}

// Withdraw records a withdrawal of a previously-granted consent. Per GDPR
// Art. 7(3), withdrawal must be as easy as giving consent and is logged rather
// than erased — so this appends a new row with Accepted=false (preserving the
// full consent history), carrying the version of the most recent matching grant.
func (s *userConsentService) Withdraw(ctx context.Context, userID, tenantID int64, consentType, ipAddress, userAgent string) error {
	version := "n/a"
	if existing, err := s.repo.FindByUserID(userID); err == nil {
		for _, c := range existing {
			if c.ConsentType == consentType && c.PolicyVersion != "" {
				version = c.PolicyVersion
			}
		}
	}

	consent := &UserConsent{
		UserID:        userID,
		TenantID:      tenantID,
		ConsentType:   consentType,
		PolicyVersion: version,
		Accepted:      false,
		IPAddress:     &ipAddress,
		UserAgent:     &userAgent,
	}
	if err := s.repo.CreateConsent(consent); err != nil {
		slog.ErrorContext(ctx, "failed to withdraw user consent",
			"user_id", userID, "consent_type", consentType, "error", err)
		return err
	}
	slog.InfoContext(ctx, "user consent withdrawn",
		"user_id", userID, "consent_type", consentType)
	return nil
}
