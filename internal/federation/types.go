package federation

import (
	"time"

	"github.com/google/uuid"
)

// WorkloadIdentityFederationResponseDTO is the JSON representation of a
// workload identity federation configuration.
type WorkloadIdentityFederationResponseDTO struct {
	WorkloadIdentityFederationUUID uuid.UUID         `json:"workload_identity_federation_uuid"`
	ClientUUID                     string            `json:"client_uuid"`
	Name                           string            `json:"name"`
	Description                    string            `json:"description"`
	IssuerURL                      string            `json:"issuer_url"`
	Audience                       string            `json:"audience"`
	SubjectClaim                   string            `json:"subject_claim"`
	SubjectPattern                 string            `json:"subject_pattern"`
	AllowedScopes                  []string          `json:"allowed_scopes"`
	AttributeMapping               map[string]string `json:"attribute_mapping"`
	IsActive                       bool              `json:"is_active"`
	CreatedAt                      time.Time         `json:"created_at"`
	UpdatedAt                      time.Time         `json:"updated_at"`
}

// WorkloadIdentityFederationCreateRequestDTO is the request body for creating a
// workload identity federation. The client is referenced by its public UUID.
type WorkloadIdentityFederationCreateRequestDTO struct {
	ClientUUID       string            `json:"client_uuid"`
	Name             string            `json:"name"`
	Description      string            `json:"description"`
	IssuerURL        string            `json:"issuer_url"`
	Audience         string            `json:"audience"`
	SubjectClaim     string            `json:"subject_claim"`
	SubjectPattern   string            `json:"subject_pattern"`
	AllowedScopes    []string          `json:"allowed_scopes"`
	AttributeMapping map[string]string `json:"attribute_mapping"`
	IsActive         *bool             `json:"is_active"`
}

// WorkloadIdentityFederationUpdateRequestDTO is the request body for updating a
// workload identity federation. The mapped client cannot be changed after
// creation, so client_uuid is intentionally absent.
type WorkloadIdentityFederationUpdateRequestDTO struct {
	Name             string            `json:"name"`
	Description      string            `json:"description"`
	IssuerURL        string            `json:"issuer_url"`
	Audience         string            `json:"audience"`
	SubjectClaim     string            `json:"subject_claim"`
	SubjectPattern   string            `json:"subject_pattern"`
	AllowedScopes    []string          `json:"allowed_scopes"`
	AttributeMapping map[string]string `json:"attribute_mapping"`
	IsActive         *bool             `json:"is_active"`
}

// WorkloadIdentityFederationFilterDTO holds filter parameters for listing
// workload identity federations.
type WorkloadIdentityFederationFilterDTO struct {
	PaginationRequestDTO
}
