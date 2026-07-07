package scim

import (
	"encoding/json"
	"net/http"
)

func HandleSchemas(w http.ResponseWriter, r *http.Request) {
	userSchema := SCIMSchema{
		ID:          SCIMUserSchema,
		Name:        "User",
		Description: "User Account",
		Attributes:  json.RawMessage(`[{"name":"userName","type":"string","multiValued":false,"required":true,"mutability":"readWrite","uniqueness":"server"},{"name":"name","type":"complex","multiValued":false,"required":false,"mutability":"readWrite","subAttributes":[{"name":"givenName","type":"string","multiValued":false,"required":false},{"name":"familyName","type":"string","multiValued":false,"required":false},{"name":"middleName","type":"string","multiValued":false,"required":false}]},{"name":"displayName","type":"string","multiValued":false,"required":false,"mutability":"readWrite"},{"name":"emails","type":"complex","multiValued":true,"required":false,"mutability":"readWrite","subAttributes":[{"name":"value","type":"string","multiValued":false,"required":false},{"name":"type","type":"string","multiValued":false,"required":false},{"name":"primary","type":"boolean","multiValued":false,"required":false}]},{"name":"phoneNumbers","type":"complex","multiValued":true,"required":false,"mutability":"readWrite","subAttributes":[{"name":"value","type":"string","multiValued":false,"required":false},{"name":"type","type":"string","multiValued":false,"required":false},{"name":"primary","type":"boolean","multiValued":false,"required":false}]},{"name":"externalId","type":"string","multiValued":false,"required":false,"mutability":"readWrite"},{"name":"active","type":"boolean","multiValued":false,"required":false,"mutability":"readWrite"},{"name":"meta","type":"complex","multiValued":false,"required":false,"mutability":"readOnly","subAttributes":[{"name":"resourceType","type":"string"},{"name":"created","type":"datetime"},{"name":"lastModified","type":"datetime"},{"name":"location","type":"string"}]}]`),
	}

	schemas := []SCIMSchema{userSchema}

	response := struct {
		Schemas      []string      `json:"schemas"`
		TotalResults int           `json:"totalResults"`
		Resources    []SCIMSchema  `json:"Resources"`
	}{
		Schemas:      []string{SCIMListResponseSchema},
		TotalResults: len(schemas),
		Resources:    schemas,
	}

	w.Header().Set("Content-Type", "application/scim+json")
	_ = json.NewEncoder(w).Encode(response)
}
