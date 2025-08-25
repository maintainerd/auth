package dto

import (
	"errors"
	"net/url"

	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/maintainerd/auth/internal/util"
)

// Login and register request payload structure
type AuthRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

func (r AuthRequest) Validate() error {
	return validation.ValidateStruct(&r,
		validation.Field(&r.Username,
			validation.Required.Error("Username is required"),
			validation.Length(3, 50).Error("Username must be between 3 and 50 characters"),
		),
		validation.Field(&r.Password,
			validation.Required.Error("Password is required"),
			validation.Length(8, 100).Error("Password must be between 8 and 100 characters"),
		),
	)
}

// Normal registration query parameters
// Client user registration requires client ID
type RegisterPublicQuery struct {
	ClientID string `json:"client_id"`
}

func (q RegisterPublicQuery) Validate() error {
	return validation.ValidateStruct(&q,
		validation.Field(&q.ClientID,
			validation.Required.Error("Client ID is required"),
			validation.Length(1, 100).Error("Client ID must not exceed 100 characters"),
		),
	)
}

// Invite based registration query parameters
// This is used for client app for invite-based registration
type RegisterPublicInviteQuery struct {
	ClientID    string `json:"client_id"`
	InviteToken string `json:"invite_token"`
	Expires     string `json:"expires"`
	Sig         string `json:"sig"`
}

func (q RegisterPublicInviteQuery) Validate() error {
	if err := validation.ValidateStruct(&q,
		validation.Field(&q.ClientID,
			validation.Required.Error("Client ID is required"),
			validation.Length(1, 100).Error("Client ID must not exceed 100 characters"),
		),
		validation.Field(&q.InviteToken,
			validation.Length(6, 100).Error("Invite code must be between 6 and 100 characters"),
		),
		validation.Field(&q.Expires,
			validation.Required.Error("Expiration is required"),
		),
		validation.Field(&q.Sig,
			validation.Required.Error("Signature is required"),
		),
	); err != nil {
		return err
	}

	// Check the cryptographic signature
	values := url.Values{}
	values.Set("client_id", q.ClientID)
	values.Set("invite_token", q.InviteToken)
	values.Set("expires", q.Expires)
	values.Set("sig", q.Sig)

	if _, err := util.ValidateSignedURL(values); err != nil {
		return errors.New("invalid or expired signed URL")
	}

	return nil
}

// Register query parameters for internal users (invite based)
// Internal user does not require client ID
type RegisterPrivateQuery struct {
	InviteToken string `json:"invite_token"`
	Expires     string `json:"expires"`
	Sig         string `json:"sig"`
}

func (q RegisterPrivateQuery) Validate() error {
	// Validate required fields
	if err := validation.ValidateStruct(&q,
		validation.Field(&q.InviteToken,
			validation.Required.Error("Invite code is required"),
			validation.Length(6, 100).Error("Invite token must be between 6 and 100 characters"),
		),
		validation.Field(&q.Expires,
			validation.Required.Error("Expiration is required"),
		),
		validation.Field(&q.Sig,
			validation.Required.Error("Signature is required"),
		),
	); err != nil {
		return err
	}

	// Check the cryptographic signature
	values := url.Values{}
	values.Set("invite_token", q.InviteToken)
	values.Set("expires", q.Expires)
	values.Set("sig", q.Sig)

	if _, err := util.ValidateSignedURL(values); err != nil {
		return errors.New("invalid or expired signed URL")
	}

	return nil
}

// Normal registration query parameters
// Client user registration requires client ID
type LoginPublicQuery struct {
	ClientID string `json:"client_id"`
}

func (q LoginPublicQuery) Validate() error {
	return validation.ValidateStruct(&q,
		validation.Field(&q.ClientID,
			validation.Required.Error("Client ID is required"),
			validation.Length(1, 100).Error("Client ID must not exceed 100 characters"),
		),
	)
}

// AuthResponse is the response structure for authentication operations
type AuthResponse struct {
	AccessToken  string `json:"access_token"`
	IDToken      string `json:"id_token"`
	RefreshToken string `json:"refresh_token,omitempty"`
	ExpiresIn    int64  `json:"expires_in"`
	TokenType    string `json:"token_type"`
	IssuedAt     int64  `json:"issued_at"`
}
