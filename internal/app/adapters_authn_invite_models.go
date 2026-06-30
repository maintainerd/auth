package app

import (
	"github.com/maintainerd/auth/internal/authn"
	"github.com/maintainerd/auth/internal/invite"
)

func toAuthnInvite(i *invite.Invite) *authn.Invite {
	if i == nil {
		return nil
	}
	return &authn.Invite{
		InviteID: i.InviteID, InviteUUID: i.InviteUUID, TenantID: i.TenantID,
		InvitedEmail: i.InvitedEmail, RegistrationFlowID: i.RegistrationFlowID,
		Status: i.Status, ExpiresAt: i.ExpiresAt,
		CreatedAt: i.CreatedAt, UpdatedAt: i.UpdatedAt,
	}
}

func toInviteInvite(i *authn.Invite) *invite.Invite {
	if i == nil {
		return nil
	}
	return &invite.Invite{
		InviteID: i.InviteID, InviteUUID: i.InviteUUID, TenantID: i.TenantID,
		InvitedEmail: i.InvitedEmail, RegistrationFlowID: i.RegistrationFlowID,
		Status: i.Status, ExpiresAt: i.ExpiresAt,
		CreatedAt: i.CreatedAt, UpdatedAt: i.UpdatedAt,
	}
}

func mapAuthnInvites(items []invite.Invite) []authn.Invite {
	out := make([]authn.Invite, len(items))
	for i := range items {
		out[i] = *toAuthnInvite(&items[i])
	}
	return out
}
