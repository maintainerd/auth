package oauth

import (
	"context"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/maintainerd/maintainerd-auth/internal/platform/config"
	"github.com/maintainerd/maintainerd-auth/internal/platform/jwt"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/codes"
	"gorm.io/gorm"
)

// BackchannelLogoutNotifier tells every relying party in a session that the
// session has ended (OpenID Connect Back-Channel Logout 1.0).
//
// This is what makes "log out of one app, log out of all of them in this
// browser" work. Cookies cannot do it: every auth cookie is __Host- prefixed
// and therefore host-only, so no logout response can clear another app's
// cookie. The authorization server has to tell each RP out-of-band instead.
//
// Scope is deliberately per-SESSION, not per-user: other browsers and mobile
// devices have their own sessions and are untouched. "Sign out everywhere" is a
// separate, explicit action.
type BackchannelLogoutNotifier interface {
	// Notify fans out a logout token to the given clients.
	//
	// clientIdentifiers is REQUIRED and must be the set of relying parties that
	// THIS session actually authenticated to. It is a parameter rather than a
	// tenant-wide query on purpose: notifying every client in the tenant would
	// log a user out of unrelated third-party applications — including ones on
	// other devices — which is the opposite of the intended scope. An empty set
	// is a no-op, never "all".
	//
	// Best-effort and non-blocking for the caller's response: a slow or dead RP
	// must not stall the user's logout.
	Notify(ctx context.Context, tenantID int64, subject, sessionID string, clientIdentifiers []string)
}

type backchannelLogoutNotifier struct {
	db     *gorm.DB
	client *http.Client
}

// NewBackchannelLogoutNotifier builds a notifier with a short per-RP timeout.
func NewBackchannelLogoutNotifier(db *gorm.DB) BackchannelLogoutNotifier {
	return &backchannelLogoutNotifier{
		db: db,
		client: &http.Client{
			Timeout: 5 * time.Second,
		},
	}
}

// backchannelTarget is one relying party to notify.
type backchannelTarget struct {
	Identifier           string
	BackchannelLogoutURI string
}

func (n *backchannelLogoutNotifier) Notify(ctx context.Context, tenantID int64, subject, sessionID string, clientIdentifiers []string) {
	_, span := otel.Tracer("service").Start(ctx, "oauth.backchannel_logout.notify")
	defer span.End()

	if n.db == nil || (subject == "" && sessionID == "") || len(clientIdentifiers) == 0 {
		return
	}

	var targets []backchannelTarget
	if err := n.db.
		Table("clients").
		Select("identifier, backchannel_logout_uri").
		Where("tenant_id = ?", tenantID).
		// Scoped to the session's own relying parties — see the interface doc.
		Where("identifier IN ?", clientIdentifiers).
		Where("backchannel_logout_uri IS NOT NULL AND backchannel_logout_uri <> ''").
		Where("status = ?", "active").
		Where("deleted_at IS NULL").
		Scan(&targets).Error; err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "backchannel target lookup failed")
		return
	}
	if len(targets) == 0 {
		return
	}

	issuer := strings.TrimRight(config.AppPublicHostname, "/")

	// Detached from the request context on purpose: the user's logout response
	// should not wait on third-party endpoints, and cancelling it when the
	// response is written would abort deliveries mid-flight.
	go func() {
		var wg sync.WaitGroup
		for _, target := range targets {
			if !isDeliverableLogoutURI(target.BackchannelLogoutURI) {
				slog.Warn("backchannel logout: skipping non-deliverable uri",
					"client_id", target.Identifier, "uri", target.BackchannelLogoutURI)
				continue
			}

			wg.Add(1)
			go func(t backchannelTarget) {
				defer wg.Done()

				// One token per RP: `aud` must be that client, so tokens are not
				// interchangeable between relying parties.
				token, err := jwt.GenerateLogoutToken(issuer, t.Identifier, subject, sessionID)
				if err != nil {
					slog.Error("backchannel logout: token minting failed",
						"client_id", t.Identifier, "error", err)
					return
				}
				n.deliver(t, token)
			}(target)
		}
		wg.Wait()
	}()
}

// deliver POSTs the logout token as application/x-www-form-urlencoded, per
// §2.5. Failures are logged, not retried: a retry queue is the right home for
// that, and silently dropping is worse than a visible log line.
func (n *backchannelLogoutNotifier) deliver(target backchannelTarget, token string) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	form := url.Values{}
	form.Set("logout_token", token)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, target.BackchannelLogoutURI,
		strings.NewReader(form.Encode()))
	if err != nil {
		slog.Error("backchannel logout: request build failed",
			"client_id", target.Identifier, "error", err)
		return
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Cache-Control", "no-store")

	resp, err := n.client.Do(req)
	if err != nil {
		slog.Warn("backchannel logout: delivery failed",
			"client_id", target.Identifier, "error", err)
		return
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		slog.Warn("backchannel logout: relying party rejected the token",
			"client_id", target.Identifier, "status", resp.StatusCode)
		return
	}
	slog.Info("backchannel logout delivered", "client_id", target.Identifier)
}

// isDeliverableLogoutURI keeps the fan-out from being used as an SSRF primitive.
// The URI is operator-configured, but it is still attacker-influenced if the
// console is compromised, so require an absolute https URL with a host.
func isDeliverableLogoutURI(raw string) bool {
	u, err := url.Parse(strings.TrimSpace(raw))
	if err != nil {
		return false
	}
	if u.Host == "" {
		return false
	}
	// http is permitted only for loopback, so local development still works.
	switch u.Scheme {
	case "https":
		return true
	case "http":
		host := u.Hostname()
		return host == "localhost" || host == "127.0.0.1" || host == "::1"
	default:
		return false
	}
}
