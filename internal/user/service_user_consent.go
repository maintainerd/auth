package user

import (
	"context"
	"log/slog"

	"gorm.io/gorm"
)

type UserConsentService interface {
	Record(ctx context.Context, tx *gorm.DB, userID, tenantID int64, consentType, policyVersion, ipAddress, userAgent string) error
	FindByUserID(ctx context.Context, userID int64) ([]UserConsent, error)
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
