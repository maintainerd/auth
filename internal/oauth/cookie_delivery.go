package oauth

import (
	"bytes"
	"encoding/json"
	"net/http"

	"github.com/maintainerd/maintainerd-auth/internal/platform/cookie"
)

// cookieDeliveryRecorder buffers a wrapped handler's status code and body so
// auth cookies (which must be set BEFORE WriteHeader) can be derived from the
// JSON token payload and added before the original response is flushed through.
// Headers written by the wrapped handler land directly on the embedded
// ResponseWriter's header map, so only the status line and body are buffered.
type cookieDeliveryRecorder struct {
	http.ResponseWriter
	status      int
	body        bytes.Buffer
	wroteHeader bool
}

func (rec *cookieDeliveryRecorder) WriteHeader(status int) {
	if !rec.wroteHeader {
		rec.status = status
		rec.wroteHeader = true
	}
}

func (rec *cookieDeliveryRecorder) Write(b []byte) (int, error) {
	if !rec.wroteHeader {
		rec.status = http.StatusOK
		rec.wroteHeader = true
	}
	return rec.body.Write(b)
}

// deliverAuthCookies invokes next and, when the client asked for cookie-based
// delivery (X-Token-Delivery: cookie) and next returns a 2xx JSON token
// response, ALSO sets the tokens as httpOnly cookies — mirroring the login and
// refresh handlers. This gives a cookie-based client (the admin console) a real
// browser session from the OAuth token endpoint, which otherwise only returns
// body tokens. Body-token API clients (which never send the header) are
// unaffected: next runs and its response passes straight through.
func deliverAuthCookies(w http.ResponseWriter, r *http.Request, next http.HandlerFunc) {
	if r.Header.Get("X-Token-Delivery") != "cookie" {
		next(w, r)
		return
	}

	rec := &cookieDeliveryRecorder{ResponseWriter: w, status: http.StatusOK}
	next(rec, r)

	if rec.status >= 200 && rec.status < 300 {
		var payload map[string]any
		if err := json.Unmarshal(rec.body.Bytes(), &payload); err == nil {
			// The token endpoint may return the tokens at the top level (raw
			// OAuth response) or wrapped in a {success, data:{...}} envelope.
			tokens := payload
			if data, ok := payload["data"].(map[string]any); ok {
				tokens = data
			}
			if _, ok := tokens["access_token"].(string); ok {
				// SetAuthCookies reads expires_in as int64 from a map; JSON
				// numbers decode to float64, so normalise it for the correct
				// cookie max-age (it falls back to a default otherwise).
				if ei, ok := tokens["expires_in"].(float64); ok {
					tokens["expires_in"] = int64(ei)
				}
				cookie.SetAuthCookies(w, tokens)
			}
		}
	}

	w.WriteHeader(rec.status)
	_, _ = w.Write(rec.body.Bytes())
}
