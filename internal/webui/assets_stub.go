//go:build !embedassets

// Default build (dev + `go test`): no assets embedded. The SPAs run under vite
// in dev via maintainerd-dev, so the Go process serves only the API/gRPC ports
// and these constructors are never called.
package webui

import "net/http"

// Enabled reports that no SPA assets are embedded, so rest.go skips the console
// and identity listeners.
const Enabled = false

// Console is a no-op in non-embed builds; rest.go guards calls with Enabled.
func Console(_, _ http.Handler, _ string) http.Handler { return nil }

// Identity is a no-op in non-embed builds; rest.go guards calls with Enabled.
func Identity(_ http.Handler) http.Handler { return nil }
