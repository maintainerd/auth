package invite

import "time"

// SendInviteRequest represents the payload to invite a user.
type SendInviteRequest struct {
	Email        string  `json:"email"`
	AuthFlowUUID *string `json:"auth_flow_uuid,omitempty"`
}

// InviteResponseDTO is the JSON representation of an invitation (admin list view).
type InviteResponseDTO struct {
	InviteID     string     `json:"invite_id"`
	InvitedEmail string     `json:"invited_email"`
	Status       string     `json:"status"`
	ExpiresAt    *time.Time `json:"expires_at"`
	UsedAt       *time.Time `json:"used_at"`
	CreatedAt    time.Time  `json:"created_at"`
	// Optional auth flow attached to the invite (nil = default registration).
	AuthFlowID   *string `json:"auth_flow_id,omitempty"`
	AuthFlowName string  `json:"auth_flow_name,omitempty"`
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
	if i.AuthFlow != nil {
		uuidStr := i.AuthFlow.AuthFlowUUID.String()
		dto.AuthFlowID = &uuidStr
		dto.AuthFlowName = i.AuthFlow.Name
	}
	return dto
}
