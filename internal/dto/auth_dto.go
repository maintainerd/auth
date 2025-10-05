package dto

import (
	"errors"
	"net/url"

	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/maintainerd/auth/internal/util"
)

// Login and register request payload structure
type AuthRequestDto struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

func (r AuthRequestDto) Validate() error {
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

// Universal registration query parameters
type RegisterQueryDto struct {
	AuthClientID    string `json:"auth_client_id"`
	AuthContainerID string `json:"auth_container_id"`
}

func (q RegisterQueryDto) Validate() error {
	return validation.ValidateStruct(&q,
		validation.Field(&q.AuthClientID,
			validation.Required.Error("Auth client ID is required"),
			validation.Length(1, 100).Error("Auth client ID must not exceed 100 characters"),
		),
		validation.Field(&q.AuthContainerID,
			validation.Required.Error("Auth container ID is required"),
			validation.Length(1, 100).Error("Auth container ID must not exceed 100 characters"),
		),
	)
}

// Universal invite-based registration query parameters
type RegisterInviteQueryDto struct {
	AuthClientID    string `json:"auth_client_id"`
	AuthContainerID string `json:"auth_container_id"`
	InviteToken     string `json:"invite_token"`
	Expires         string `json:"expires"`
	Sig             string `json:"sig"`
}

func (q RegisterInviteQueryDto) Validate() error {
	if err := validation.ValidateStruct(&q,
		validation.Field(&q.AuthClientID,
			validation.Required.Error("Auth client ID is required"),
			validation.Length(1, 100).Error("Auth client ID must not exceed 100 characters"),
		),
		validation.Field(&q.AuthContainerID,
			validation.Required.Error("Auth container ID is required"),
			validation.Length(1, 100).Error("Auth container ID must not exceed 100 characters"),
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
	values.Set("auth_client_id", q.AuthClientID)
	values.Set("auth_container_id", q.AuthContainerID)
	values.Set("invite_token", q.InviteToken)
	values.Set("expires", q.Expires)
	values.Set("sig", q.Sig)

	if _, err := util.ValidateSignedURL(values); err != nil {
		return errors.New("invalid or expired signed URL")
	}

	return nil
}

// Universal login query parameters
type LoginQueryDto struct {
	AuthClientID    string `json:"auth_client_id"`
	AuthContainerID string `json:"auth_container_id"`
}

func (q LoginQueryDto) Validate() error {
	return validation.ValidateStruct(&q,
		validation.Field(&q.AuthClientID,
			validation.Required.Error("Auth client ID is required"),
			validation.Length(1, 100).Error("Auth client ID must not exceed 100 characters"),
		),
		validation.Field(&q.AuthContainerID,
			validation.Required.Error("Auth container ID is required"),
			validation.Length(1, 100).Error("Auth container ID must not exceed 100 characters"),
		),
	)
}

// AuthResponse is the response structure for authentication operations
type AuthResponseDto struct {
	AccessToken  string `json:"access_token"`
	IDToken      string `json:"id_token"`
	RefreshToken string `json:"refresh_token,omitempty"`
	ExpiresIn    int64  `json:"expires_in"`
	TokenType    string `json:"token_type"`
	IssuedAt     int64  `json:"issued_at"`
}
