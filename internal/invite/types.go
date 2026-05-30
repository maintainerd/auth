package invite

import (
	"github.com/google/uuid"
)

// SendInviteRequest represents the payload to invite a user.
type SendInviteRequest struct {
	Email string      `json:"email"` // Email address of the user to invite
	Roles []uuid.UUID `json:"roles"` // List of role UUIDs to assign to the invited user
}
