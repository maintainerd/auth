package scim

import (
	"encoding/json"
	"net/http"
)

func HandleServiceProviderConfig(w http.ResponseWriter, r *http.Request) {
	cfg := SCIMServiceProviderConfig{
		Schemas: []string{SCIMServiceProviderConfigSchema},
		DocumentationURI: "https://docs.maintainerd.com/scim",
		Patch: SCIMSupported{
			Supported: true,
		},
		Bulk: SCIMBulkSupported{
			Supported:      false,
			MaxOperations:  0,
			MaxPayloadSize: 0,
		},
		Filter: SCIMFilterSupported{
			Supported:  false,
			MaxResults: 100,
		},
		ChangePassword: SCIMSupported{
			Supported: false,
		},
		Sort: SCIMSupported{
			Supported: false,
		},
		Etag: SCIMSupported{
			Supported: false,
		},
		AuthenticationSchemes: []SCIMAuthScheme{
			{
				Type:        "oauthbearertoken",
				Name:        "OAuth Bearer Token",
				Description: "Authentication using a SCIM bearer token",
			},
		},
	}

	w.Header().Set("Content-Type", "application/scim+json")
	json.NewEncoder(w).Encode(cfg)
}
