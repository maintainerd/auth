package scim

import (
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/maintainerd/maintainerd-auth/internal/platform/security"
)

func (dto *SCIMConfigurationCreateDTO) Validate() error {
	return validation.ValidateStruct(dto,
		validation.Field(&dto.DisplayName, validation.Required, validation.Length(1, 255)),
		validation.Field(&dto.BaseURL, validation.When(dto.BaseURL != nil, validation.Length(0, 2048))),
		validation.Field(&dto.BearerToken, validation.When(dto.BearerToken != nil, validation.Length(1, 512))),
		validation.Field(&dto.SyncDirection, validation.When(dto.SyncDirection != nil,
			validation.In("inbound", "outbound", "bidirectional"),
		)),
	)
}

func (dto *SCIMConfigurationUpdateDTO) Validate() error {
	return validation.ValidateStruct(dto,
		validation.Field(&dto.DisplayName, validation.When(dto.DisplayName != nil, validation.Length(1, 255))),
		validation.Field(&dto.BaseURL, validation.When(dto.BaseURL != nil, validation.Length(0, 2048))),
		validation.Field(&dto.BearerToken, validation.When(dto.BearerToken != nil, validation.Length(1, 512))),
		validation.Field(&dto.SyncDirection, validation.When(dto.SyncDirection != nil,
			validation.In("inbound", "outbound", "bidirectional"),
		)),
	)
}

func (dto *SCIMConfigurationCreateDTO) Sanitize() {
	dto.DisplayName = security.SanitizeInput(dto.DisplayName)
	if dto.BaseURL != nil {
		s := security.SanitizeInput(*dto.BaseURL)
		dto.BaseURL = &s
	}
}

func (dto *SCIMConfigurationUpdateDTO) Sanitize() {
	if dto.DisplayName != nil {
		s := security.SanitizeInput(*dto.DisplayName)
		dto.DisplayName = &s
	}
	if dto.BaseURL != nil {
		s := security.SanitizeInput(*dto.BaseURL)
		dto.BaseURL = &s
	}
}
