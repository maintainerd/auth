package service

import (
	"errors"

	"github.com/maintainerd/auth/internal/model"
)

// ValidateAuthContainerAccess validates if a user can access the target auth container
// Rules:
// - Users from default auth container can access any auth container in the same organization
// - Users from non-default auth container can only access their own auth container
// - Both auth containers must belong to the same organization
func ValidateAuthContainerAccess(actorUser *model.User, targetAuthContainer *model.AuthContainer) error {
	if actorUser.AuthContainer == nil {
		return errors.New("actor user has no auth container")
	}

	// Check if both auth containers belong to the same organization
	if actorUser.AuthContainer.OrganizationID != targetAuthContainer.OrganizationID {
		return errors.New("access denied: auth containers belong to different organizations")
	}

	// If actor is from default auth container, they can access any auth container in the same organization
	if actorUser.AuthContainer.IsDefault {
		return nil
	}

	// If actor is from non-default auth container, they can only access their own auth container
	if actorUser.AuthContainerID == targetAuthContainer.AuthContainerID {
		return nil
	}

	return errors.New("access denied: non-default auth container users can only access their own auth container")
}

// ValidateAuthContainerAccessByID validates auth container access using auth container ID
func ValidateAuthContainerAccessByID(actorUser *model.User, targetAuthContainerID int64) error {
	if actorUser.AuthContainer == nil {
		return errors.New("actor user has no auth container")
	}

	// If actor is from default auth container, they can access any auth container in the same organization
	if actorUser.AuthContainer.IsDefault {
		return nil
	}

	// If actor is from non-default auth container, they can only access their own auth container
	if actorUser.AuthContainerID == targetAuthContainerID {
		return nil
	}

	return errors.New("access denied: non-default auth container users can only access their own auth container")
}
