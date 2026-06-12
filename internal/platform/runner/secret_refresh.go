package runner

import (
	"bytes"
	"context"
	"log/slog"
	"time"

	"github.com/maintainerd/auth/internal/platform/config"
	"github.com/maintainerd/auth/internal/platform/jwt"
)

// StartSecretRefreshRunner periodically re-fetches secrets from the active
// provider and applies any changes without requiring a restart. JWT key changes
// trigger a call to jwt.InitJWTKeys so the new material takes effect
// immediately; SMTP credentials are updated in-place.
//
// The runner exits cleanly when ctx is cancelled.
func StartSecretRefreshRunner(ctx context.Context, period time.Duration) {
	ticker := time.NewTicker(period)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			slog.Info("secret_refresh: runner stopped")
			return
		case <-ticker.C:
			refreshSecrets()
		}
	}
}

func refreshSecrets() {
	refreshJWTKeys()
}

func refreshJWTKeys() {
	priv, err := config.LoadSecret("JWT_PRIVATE_KEY")
	if err != nil {
		slog.Warn("secret_refresh: failed to reload JWT_PRIVATE_KEY", "err", err)
		return
	}
	pub, err := config.LoadSecret("JWT_PUBLIC_KEY")
	if err != nil {
		slog.Warn("secret_refresh: failed to reload JWT_PUBLIC_KEY", "err", err)
		return
	}

	if bytes.Equal(priv, config.JWTPrivateKey) && bytes.Equal(pub, config.JWTPublicKey) {
		return // no change — skip re-init
	}

	config.JWTPrivateKey = priv
	config.JWTPublicKey = pub

	if err := jwt.InitJWTKeys(); err != nil {
		slog.Warn("secret_refresh: JWT key re-init failed", "err", err)
		return
	}
	slog.Info("secret_refresh: JWT keys refreshed")
}
