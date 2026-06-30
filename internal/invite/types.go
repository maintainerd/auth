package invite

import "time"

// SendInviteRequest represents the payload to invite a user.
type SendInviteRequest struct {
	Email                string  `json:"email"`
	RegistrationFlowUUID *string `json:"registration_flow_uuid,omitempty"`
	CallbackURL          *string `json:"callback_url,omitempty"`
}

// InviteResponseDTO is the JSON representation of an invitation (admin list view).
type InviteResponseDTO struct {
	InviteID     string     `json:"invite_id"`
	InvitedEmail string     `json:"invited_email"`
	Status       string     `json:"status"`
	ExpiresAt    *time.Time `json:"expires_at"`
	UsedAt       *time.Time `json:"used_at"`
	CreatedAt    time.Time  `json:"created_at"`
	// Optional registration flow attached to the invite (nil = default registration).
	RegistrationFlowID   *string `json:"registration_flow_id,omitempty"`
	RegistrationFlowName string  `json:"registration_flow_name,omitempty"`
}

func toInviteResponseDTO(i Invite) InviteResponseDTO {
	dto := InviteResponseDTO{
		InviteID:     i.InviteUUID.String(),
		InvitedEmail: i.InvitedEmail,
		Status:       i.Status,
		ExpiresAt:    i.ExpiresAt,
		UsedAt:       i.UsedAt,
		CreatedAt:    i.CreatedAt,
	}
	if i.RegistrationFlow != nil {
		uuidStr := i.RegistrationFlow.RegistrationFlowUUID.String()
		dto.RegistrationFlowID = &uuidStr
		dto.RegistrationFlowName = i.RegistrationFlow.Name
	}
	return dto
}

type InviteContextResponseDTO struct {
	InviteToken string     `json:"invite_token"`
	Email       string     `json:"email"`
	CallbackURL *string    `json:"callback_url,omitempty"`
	ExpiresAt   *time.Time `json:"expires_at,omitempty"`
	Status      string     `json:"status"`
}

func toInviteContextResponseDTO(i Invite) InviteContextResponseDTO {
	return InviteContextResponseDTO{
		InviteToken: i.InviteToken,
		Email:       i.InvitedEmail,
		CallbackURL: i.CallbackURL,
		ExpiresAt:   i.ExpiresAt,
		Status:      i.Status,
	}
}
