package docs

import _ "embed"

// OpenAPISpec is the OpenAPI 3.1 spec embedded at build time.
//
//go:embed openapi.yaml
var OpenAPISpec []byte
