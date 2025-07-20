package dto

import "github.com/maintainerd/auth/internal/validator"

type AuthRequest struct {
	Username           string `json:"username"`
	Password           string `json:"password"`
	ClientID           string `json:"client_id"`
	IdentityProviderID string `json:"identity_provider_id"`
}

func (r AuthRequest) Validate() error {
	return validator.ValidateStruct(&r,
		validator.Field(&r.Username,
			validator.Required().Error("Username is required"),
			validator.MinLength(3).Error("At least 3 characters"),
			validator.MaxLength(50).Error("At most 50 characters"),
		),
		validator.Field(&r.Password,
			validator.Required().Error("Password is required"),
			validator.MinLength(8).Error("At least 8 characters"),
			validator.MaxLength(100).Error("At most 100 characters"),
		),
		validator.Field(&r.ClientID,
			validator.Required().Error("Client ID is required"),
		),
		validator.Field(&r.IdentityProviderID,
			validator.Required().Error("Identity Provider ID is required"),
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
