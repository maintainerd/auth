package dto

import (
	validation "github.com/go-ozzo/ozzo-validation/v4"
)

type AuthRequest struct {
	Username           string `json:"username"`
	Password           string `json:"password"`
	ClientID           string `json:"client_id"`
	IdentityProviderID string `json:"identity_provider_id"`
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
		validation.Field(&r.ClientID,
			validation.Required.Error("Client ID is required"),
		),
		validation.Field(&r.IdentityProviderID,
			validation.Required.Error("Identity Provider ID is required"),
		),
	)
}

type AuthResponse struct {
	AccessToken  string `json:"access_token"`
	IDToken      string `json:"id_token"`
	RefreshToken string `json:"refresh_token,omitempty"`
	ExpiresIn    int64  `json:"expires_in"`
	TokenType    string `json:"token_type"`
	IssuedAt     int64  `json:"issued_at"`
}
