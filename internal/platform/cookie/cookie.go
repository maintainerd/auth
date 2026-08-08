package cookie

import (
	"net/http"
	"reflect"
	"strings"

	"github.com/maintainerd/maintainerd-auth/internal/platform/config"
	"github.com/maintainerd/maintainerd-auth/internal/shared"
)

// cookieSecure returns whether cookies should set Secure=true.
// Defaults true; can be disabled for local dev via COOKIE_SECURE=false.
func cookieSecure() bool {
	return config.CookieSecure
}

// cookieSameSite maps the COOKIE_SAMESITE config value to an http.SameSite constant.
func cookieSameSite() http.SameSite {
	switch strings.ToLower(config.CookieSameSite) {
	case "lax":
		return http.SameSiteLaxMode
	case "none":
		return http.SameSiteNoneMode
	default:
		return http.SameSiteStrictMode
	}
}

func secureForCookieName(name string) bool {
	if strings.HasPrefix(name, "__Host-") || strings.HasPrefix(name, "__Secure-") {
		return true
	}
	return cookieSecure()
}

func sameSiteString(value string) http.SameSite {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "lax":
		return http.SameSiteLaxMode
	case "none":
		return http.SameSiteNoneMode
	case "strict":
		return http.SameSiteStrictMode
	default:
		return cookieSameSite()
	}
}

func sameSiteForCookie(secure bool, override string) http.SameSite {
	sameSite := sameSiteString(override)
	if sameSite == http.SameSiteNoneMode && !secure {
		return http.SameSiteLaxMode
	}
	return sameSite
}

func setAuthCookie(w http.ResponseWriter, name, value, path string, maxAge int, opts authCookieOptions) {
	secure := secureForCookieName(name)
	if opts.Secure != nil && !strings.HasPrefix(name, "__Host-") && !strings.HasPrefix(name, "__Secure-") {
		secure = *opts.Secure
	}
	httpOnly := true
	if opts.HTTPOnly != nil {
		httpOnly = *opts.HTTPOnly
	}
	// __Host- forbids a Domain attribute; only __Secure-/unprefixed names may
	// carry one. This applies to deletes too: a Domain-scoped cookie can ONLY be
	// removed by a Set-Cookie carrying the same Domain, so never make this
	// conditional on maxAge.
	domain := ""
	if !strings.HasPrefix(name, "__Host-") {
		domain = sharedCookieDomain()
	}

	http.SetCookie(w, &http.Cookie{ // #nosec G124 -- cookie attributes set per-name via helpers nosemgrep
		Name:     name,
		Value:    value,
		Path:     path,
		Domain:   domain,
		MaxAge:   maxAge,
		HttpOnly: httpOnly,
		Secure:   secure,
		SameSite: sameSiteForCookie(secure, opts.SameSite),
	})
}

// Auth cookies always use the __Host-/__Secure- prefixes. These prefixes are a
// browser-enforced hardening (host-only, Secure-required) and are honored on
// localhost, which browsers treat as a secure context. secureForCookieName
// forces Secure=true for any prefixed name regardless of COOKIE_SECURE.
// sharedCookieDomain reports the configured parent domain for first-party
// single sign-on/out, normalized without a leading dot (Go adds host matching).
// Empty = host-only cookies, i.e. every surface keeps its own session.
//
// The domain IS the first-party boundary: anything the operator hosts under it
// shares the session, anything on another domain is a third-party relying party
// with its own. No registration or per-client flag is involved.
func sharedCookieDomain() string {
	return strings.TrimPrefix(strings.TrimSpace(config.CookieDomain), ".")
}

// authCookiePrefix picks the strongest prefix compatible with the deployment.
// __Host- is preferred (host-only, Path=/, Secure) but forbids a Domain
// attribute by definition, so a shared-domain deployment must use __Secure-.
func authCookiePrefix() string {
	if sharedCookieDomain() != "" {
		return "__Secure-"
	}
	return "__Host-"
}

func accessTokenCookieName() string { return authCookiePrefix() + "access_token" }

func idTokenCookieName() string { return authCookiePrefix() + "id_token" }

func refreshTokenCookieName() string { return "__Secure-refresh_token" }

// refreshTokenCookiePath scopes the refresh-token cookie to the refresh
// endpoint only, so it is never sent on ordinary API requests. It must match the
// mounted route path (POST /api/v1/refresh-token).
const refreshTokenCookiePath = "/api/v1/refresh-token"

type authCookieOptions struct {
	Secure        *bool
	HTTPOnly      *bool
	SameSite      string
	RefreshMaxAge int
}

// SetAuthCookies sets authentication tokens as secure HTTP-only cookies.
//
// Cookie naming conventions:
//   - __Host-access_token  / __Host-id_token  — __Host- prefix (Secure, Path=/, no Domain)
//   - __Secure-refresh_token                  — __Secure- prefix (Secure, narrow path)
//
// The __Host- prefix prevents subdomain fixation attacks; __Secure- is used for
// the refresh token because it has a non-root path (/api/v1/refresh-token) so it
// is only sent to the refresh endpoint.
func SetAuthCookies(w http.ResponseWriter, authResponse interface{}) {
	var accessToken, idToken, refreshToken string
	var expiresIn int64 = shared.DefaultAccessTokenExpiresIn
	var accessTokenCookieMaxAge int64
	opts := authCookieOptions{RefreshMaxAge: 7 * 24 * 60 * 60}

	if response, ok := authResponse.(map[string]interface{}); ok {
		if at, exists := response["access_token"]; exists {
			if s, ok := at.(string); ok {
				accessToken = s
			}
		}
		if it, exists := response["id_token"]; exists {
			if s, ok := it.(string); ok {
				idToken = s
			}
		}
		if rt, exists := response["refresh_token"]; exists {
			if s, ok := rt.(string); ok {
				refreshToken = s
			}
		}
		if ei, exists := response["expires_in"]; exists {
			if v, ok := ei.(int64); ok {
				expiresIn = v
			}
		}
	} else {
		v := reflect.ValueOf(authResponse)
		if v.Kind() == reflect.Ptr {
			v = v.Elem()
		}
		if v.Kind() == reflect.Struct {
			if f := v.FieldByName("AccessToken"); f.IsValid() && f.Kind() == reflect.String {
				accessToken = f.String()
			}
			if f := v.FieldByName("IDToken"); f.IsValid() && f.Kind() == reflect.String {
				idToken = f.String()
			}
			if f := v.FieldByName("RefreshToken"); f.IsValid() && f.Kind() == reflect.String {
				refreshToken = f.String()
			}
			if f := v.FieldByName("ExpiresIn"); f.IsValid() && f.Kind() == reflect.Int64 {
				expiresIn = f.Int()
			}
			if f := v.FieldByName("AccessTokenCookieMaxAge"); f.IsValid() && f.Kind() == reflect.Int64 {
				accessTokenCookieMaxAge = f.Int()
			}
			if f := v.FieldByName("CookieSecure"); f.IsValid() && !f.IsNil() {
				b := f.Elem().Bool()
				opts.Secure = &b
			}
			if f := v.FieldByName("CookieHTTPOnly"); f.IsValid() && !f.IsNil() {
				b := f.Elem().Bool()
				opts.HTTPOnly = &b
			}
			if f := v.FieldByName("CookieSameSite"); f.IsValid() && f.Kind() == reflect.String {
				opts.SameSite = f.String()
			}
			if f := v.FieldByName("RefreshTokenMaxAge"); f.IsValid() && f.Kind() == reflect.Int && f.Int() > 0 {
				opts.RefreshMaxAge = int(f.Int())
			}
		}
	}

	atCookieMaxAge := int(expiresIn)
	if accessTokenCookieMaxAge > 0 {
		atCookieMaxAge = int(accessTokenCookieMaxAge)
	}

	if accessToken != "" {
		setAuthCookie(w, accessTokenCookieName(), accessToken, "/", atCookieMaxAge, opts)
	}

	if idToken != "" {
		setAuthCookie(w, idTokenCookieName(), idToken, "/", atCookieMaxAge, opts)
	}

	if refreshToken != "" {
		setAuthCookie(w, refreshTokenCookieName(), refreshToken, refreshTokenCookiePath, opts.RefreshMaxAge, opts)
	}
}

// authorizeBindingCookieName is the httpOnly cookie that binds a pending OAuth
// authorize request to the browser that initiated it. The __Host- prefix forces
// Secure + host-only + Path=/ so it cannot be fixated from a subdomain.
func authorizeBindingCookieName() string { return "__Host-authz_binding" }

// authorizeBindingMaxAge matches the authorize-request TTL (10 minutes); the
// cookie need not outlive the pending request it binds.
const authorizeBindingMaxAge = 10 * 60

// SetAuthorizeBindingCookie stores the raw browser-binding secret for a pending
// OAuth authorize request. Only its hash is persisted server-side, so possession
// of a request_id without this cookie cannot continue the flow.
func SetAuthorizeBindingCookie(w http.ResponseWriter, secret string) {
	setAuthCookie(w, authorizeBindingCookieName(), secret, "/", authorizeBindingMaxAge, authCookieOptions{})
}

// AuthorizeBindingValue returns the browser-binding secret from the request, or
// an empty string when the cookie is absent.
func AuthorizeBindingValue(r *http.Request) string {
	c, err := r.Cookie(authorizeBindingCookieName())
	if err != nil {
		return ""
	}
	return c.Value
}

// ClearAuthorizeBindingCookie removes the binding cookie once the authorize
// request has been continued (or is being abandoned).
func ClearAuthorizeBindingCookie(w http.ResponseWriter) {
	setAuthCookie(w, authorizeBindingCookieName(), "", "/", -1, authCookieOptions{})
}

// trustedDeviceCookieName is the httpOnly cookie that lets a previously trusted
// browser skip the login MFA step until the tenant's trusted-device window
// expires. The __Host- prefix forces Secure + host-only + Path=/, so it cannot
// be read by JavaScript or fixated from a subdomain. It deliberately outlives a
// logout (see ClearAuthCookies) because device trust is meant to persist across
// sessions on the same browser.
func trustedDeviceCookieName() string { return "__Host-mfa_trusted_device" }

// trustedDeviceDefaultMaxAge (30 days) is used when the caller cannot supply an
// explicit lifetime.
const trustedDeviceDefaultMaxAge = 30 * 24 * 60 * 60

// SetTrustedDeviceCookie stores the opaque trusted-device secret for
// maxAgeSeconds. The secret — never the fingerprint — is the trust credential;
// only its hash is persisted server-side (user_trusted_devices.device_token_hash).
func SetTrustedDeviceCookie(w http.ResponseWriter, token string, maxAgeSeconds int) {
	if maxAgeSeconds <= 0 {
		maxAgeSeconds = trustedDeviceDefaultMaxAge
	}
	setAuthCookie(w, trustedDeviceCookieName(), token, "/", maxAgeSeconds, authCookieOptions{})
}

// TrustedDeviceValue returns the trusted-device secret from the request, or an
// empty string when the cookie is absent.
func TrustedDeviceValue(r *http.Request) string {
	c, err := r.Cookie(trustedDeviceCookieName())
	if err != nil {
		return ""
	}
	return c.Value
}

// ClearTrustedDeviceCookie removes the trusted-device cookie from this browser.
// Not part of ClearAuthCookies: an ordinary logout keeps device trust intact.
func ClearTrustedDeviceCookie(w http.ResponseWriter) {
	setAuthCookie(w, trustedDeviceCookieName(), "", "/", -1, authCookieOptions{})
}

// deviceIDCookieName is a stable, long-lived per-browser identifier used only to
// dedupe trusted-device rows (so two browsers with an identical User-Agent don't
// collide onto one row). It is NOT a secret and never grants trust on its own —
// the opaque trusted-device token is the credential. httpOnly all the same, so
// page scripts can't read or fixate it.
func deviceIDCookieName() string { return "__Host-device_id" }

// deviceIDMaxAge (2 years) lets the identifier outlive individual trust windows.
const deviceIDMaxAge = 2 * 365 * 24 * 60 * 60

// SetDeviceIDCookie persists the per-browser device identifier.
func SetDeviceIDCookie(w http.ResponseWriter, deviceID string) {
	setAuthCookie(w, deviceIDCookieName(), deviceID, "/", deviceIDMaxAge, authCookieOptions{})
}

// DeviceIDValue returns the per-browser device identifier, or an empty string
// when the cookie is absent.
func DeviceIDValue(r *http.Request) string {
	c, err := r.Cookie(deviceIDCookieName())
	if err != nil {
		return ""
	}
	return c.Value
}

// ClearDeviceIDCookie removes the device identifier (used alongside
// ClearTrustedDeviceCookie when a browser explicitly forgets its trust).
func ClearDeviceIDCookie(w http.ResponseWriter) {
	setAuthCookie(w, deviceIDCookieName(), "", "/", -1, authCookieOptions{})
}

// ClearAuthCookies clears all authentication-related cookies.
func ClearAuthCookies(w http.ResponseWriter) {
	// Clear BOTH prefixes, not just the one this deployment currently issues:
	// changing COOKIE_DOMAIN changes the prefix, and a cookie left behind under
	// the old name would keep a "logged out" user authenticated.
	for _, name := range []string{"__Host-access_token", "__Secure-access_token", "access_token"} {
		setAuthCookie(w, name, "", "/", -1, authCookieOptions{})
	}
	for _, name := range []string{"__Host-id_token", "__Secure-id_token", "id_token"} {
		setAuthCookie(w, name, "", "/", -1, authCookieOptions{})
	}
	for _, name := range []string{refreshTokenCookieName(), "refresh_token"} {
		setAuthCookie(w, name, "", refreshTokenCookiePath, -1, authCookieOptions{})
	}
}
