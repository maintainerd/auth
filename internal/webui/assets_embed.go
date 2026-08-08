//go:build embedassets

// This file is compiled only for the release image (`go build -tags embedassets`).
// The Docker build populates internal/webui/console and internal/webui/identity
// with each SPA's built dist output before compiling, so the assets are baked
// into the single binary.
package webui

import (
	"embed"
	"io/fs"
	"net/http"
)

//go:embed all:console
var consoleAssets embed.FS

//go:embed all:identity
var identityAssets embed.FS

// Enabled reports that SPA assets are embedded, so rest.go starts the console
// and identity listeners.
const Enabled = true

// Console serves the admin console SPA with its APIs mounted same-origin:
// the control plane at /api and the data plane at /public-api (prefix stripped
// so the router sees its normal /api/v1 path).
func Console(control, data http.Handler, identityURL string) http.Handler {
	static, _ := fs.Sub(consoleAssets, "console")
	return buildSPA(static, consoleConfigJS(identityURL), []mount{
		{pattern: "/api/", handler: control},
		{pattern: "/public-api/", strip: "/public-api", handler: data},
	})
}

// Identity serves the hosted identity SPA. It talks only to the data plane, at
// /api plus the root-level OIDC discovery under /.well-known.
func Identity(data http.Handler) http.Handler {
	static, _ := fs.Sub(identityAssets, "identity")
	return buildSPA(static, identityConfigJS(), []mount{
		{pattern: "/api/", handler: data},
		{pattern: "/.well-known/", handler: data},
	})
}
