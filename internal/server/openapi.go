package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"time"

	"github.com/maintainerd/auth/docs"
	"github.com/maintainerd/auth/internal/shared"
	"gopkg.in/yaml.v3"
)

var zeroTime = time.Time{}

// ServeOpenAPISpec serves the OpenAPI 3.1 spec as JSON at /openapi.json.
// The YAML source is embedded at build time so no filesystem access is needed at runtime.
func ServeOpenAPISpec(w http.ResponseWriter, r *http.Request) {
	var doc any
	if err := yaml.Unmarshal(docs.OpenAPISpec, &doc); err != nil {
		http.Error(w, "openapi spec parse error", http.StatusInternalServerError)
		return
	}

	out, err := json.Marshal(normalizeYAMLDoc(doc))
	if err != nil {
		http.Error(w, "openapi spec encode error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", shared.DefaultDiscoveryCacheMaxAge)
	http.ServeContent(w, r, "openapi.json", zeroTime, bytes.NewReader(out))
}

// normalizeYAMLDoc converts map[interface{}]interface{} nodes into map[string]interface{}
// so that encoding/json can marshal them. gopkg.in/yaml.v3 already produces
// map[string]interface{}, but we keep this for safety across decoder versions.
func normalizeYAMLDoc(v any) any {
	switch val := v.(type) {
	case map[string]any:
		out := make(map[string]any, len(val))
		for k, v2 := range val {
			out[k] = normalizeYAMLDoc(v2)
		}
		return out
	case map[interface{}]interface{}:
		out := make(map[string]any, len(val))
		for k, v2 := range val {
			out[k.(string)] = normalizeYAMLDoc(v2)
		}
		return out
	case []any:
		for i, item := range val {
			val[i] = normalizeYAMLDoc(item)
		}
		return val
	default:
		return v
	}
}
