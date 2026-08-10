// Package webui serves the bundled admin console + hosted identity SPAs from the
// Go process itself, so the all-in-one release image is a single binary with no
// nginx and no process supervisor.
//
// Each SPA is served on its own port with its API mounted SAME-ORIGIN in the
// same process: the browser talks to one origin, so the __Host- prefixed auth
// cookies (host-only) are always sent back with the app's own API calls. This is
// the exact same-origin contract the old bundled nginx provided — reimplemented
// in ~one file of stdlib http — which is why deleting nginx is safe.
//
// The static assets are compiled into the binary via go:embed, but ONLY under
// the `embedassets` build tag (see assets_embed.go / assets_stub.go). Dev builds
// and `go test` omit the tag: no assets are embedded, Enabled is false, and the
// SPA listeners never start — dev keeps serving the frontends via vite.
package webui

import (
	"fmt"
	"io/fs"
	"net/http"
	"path"
	"strings"
)

// mount attaches an in-process API handler under a URL prefix on an SPA server.
// When strip is non-empty it is trimmed from the path before the handler sees it
// (e.g. the console reaches the data plane at /public-api/... which must arrive
// at the router as /api/...), mirroring the old nginx `rewrite` rules.
type mount struct {
	pattern string // e.g. "/api/", "/public-api/", "/.well-known/"
	strip   string // "" for none, else the prefix to strip (e.g. "/public-api")
	handler http.Handler
}

// buildSPA composes one SPA server: the API mounts, a dynamic /config.js, and
// the embedded static files with client-side-routing fallback to index.html.
func buildSPA(static fs.FS, configJS []byte, mounts []mount) http.Handler {
	mux := http.NewServeMux()

	for _, m := range mounts {
		h := m.handler
		if m.strip != "" {
			h = http.StripPrefix(m.strip, h)
		}
		mux.Handle(m.pattern, h)
	}

	// Runtime config, evaluated once at start from env and served no-store so a
	// single built image targets any deployment without a rebuild. Replaces the
	// old entrypoint.sh writing config.js into the nginx docroot.
	mux.HandleFunc("/config.js", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/javascript; charset=utf-8")
		w.Header().Set("Cache-Control", "no-store")
		// configJS is built server-side from env (consoleConfigJS/identityConfigJS,
		// %q-escaped) and served as application/javascript — no user input, no HTML.
		_, _ = w.Write(configJS) // nosemgrep: go.lang.security.audit.xss.no-direct-write-to-responsewriter.no-direct-write-to-responsewriter
	})

	mux.Handle("/", spaFiles(static))
	return mux
}

// spaFiles serves embedded static assets, falling back to index.html for any
// path that is not a real file so client-side routes resolve. Long-lived,
// immutable caching for hashed /assets/; no-store for the HTML shell.
func spaFiles(fsys fs.FS) http.Handler {
	fileServer := http.FileServer(http.FS(fsys))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		setStaticSecurityHeaders(w)

		// path.Clean on an absolute URL path cannot escape the root, the assets are
		// a read-only go:embed FS, and fs.ValidPath rejects any non-clean/traversing
		// path before it is opened — defense in depth.
		upath := strings.TrimPrefix(path.Clean(r.URL.Path), "/") // nosemgrep: go.lang.security.filepath-clean-misuse.filepath-clean-misuse
		if upath == "" || !fs.ValidPath(upath) {
			serveIndex(w, fsys)
			return
		}
		if f, err := fsys.Open(upath); err != nil {
			// Not a real asset — hand the SPA its shell for client-side routing.
			serveIndex(w, fsys)
			return
		} else {
			_ = f.Close()
		}

		if strings.HasPrefix(r.URL.Path, "/assets/") {
			w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
		}
		fileServer.ServeHTTP(w, r)
	})
}

func serveIndex(w http.ResponseWriter, fsys fs.FS) {
	b, err := fs.ReadFile(fsys, "index.html")
	if err != nil {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	// b is the embedded index.html shell (static, no user input); Content-Type is
	// text/html as set above.
	_, _ = w.Write(b) // nosemgrep: go.lang.security.audit.xss.no-direct-write-to-responsewriter.no-direct-write-to-responsewriter
}

// setStaticSecurityHeaders mirrors the CSP + hardening headers the bundled nginx
// applied to the SPA responses. The API mounts carry their own headers via the
// router middleware, so these are set only on the static/HTML path.
func setStaticSecurityHeaders(w http.ResponseWriter) {
	w.Header().Set("Content-Security-Policy", "default-src 'self'; script-src 'self'; style-src 'self' 'unsafe-inline'; img-src 'self' data: blob: https:; font-src 'self' data:; connect-src 'self'; frame-ancestors 'none'; base-uri 'self'; form-action 'self'")
	w.Header().Set("X-Frame-Options", "DENY")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
}

// consoleConfigJS / identityConfigJS build the window.__ENV__ payload each SPA
// reads at boot. The URLs are empty by default so the apps use their same-origin
// /api/v1 (and /public-api/api/v1) defaults, which this process serves. The only
// cross-app link — the console pointing at the hosted identity UI — comes from
// the backend's own APP_FRONTEND_IDENTITY_HOSTNAME.
func consoleConfigJS(identityURL string) []byte {
	return []byte(fmt.Sprintf(`window.__ENV__ = {
  VITE_AUTH_API_BASE_URL: "",
  VITE_AUTH_PUBLIC_API_BASE_URL: "",
  VITE_AUTH_IDENTITY_BASE_URL: %q
};
`, identityURL))
}

func identityConfigJS() []byte {
	return []byte(`window.__ENV__ = {
  VITE_AUTH_API_BASE_URL: ""
};
`)
}
