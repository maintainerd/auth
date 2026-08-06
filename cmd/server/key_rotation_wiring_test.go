package main

import (
	"context"
	"testing"
	"time"

	"gorm.io/gorm"

	"github.com/maintainerd/maintainerd-auth/internal/app"
	"github.com/maintainerd/maintainerd-auth/internal/authevent"
	"github.com/maintainerd/maintainerd-auth/internal/platform/config"
	appserver "github.com/maintainerd/maintainerd-auth/internal/server"
	"github.com/maintainerd/maintainerd-auth/internal/user"
)

// Rotation must go through whichever store owns the signing key.
//
// With JWT_PRIVATE_KEY set the key is process-local and in-memory rotation is
// correct. Without it the signing_keys table owns the key, and rotating in
// memory publishes a JWKS key that exists in no row — leaving the database, the
// key-management API and the served JWKS disagreeing about what signs tokens.
// Getting this branch wrong is silent, so it is pinned here.
func TestKeyRotationRunnerMatchesTheKeyStore(t *testing.T) {
	cases := []struct {
		name       string
		privateKey []byte
		wantRunner string // "" means no rotation runner may start at all
	}{
		// INVERTED: this used to expect in-memory rotation for an env-owned key.
		// That was harmful — the rotated key is never persisted, is not shared
		// with other replicas, and is lost on restart, so in a multi-replica
		// deployment replica A would sign with a key replica B's JWKS never
		// publishes. An env-owned key is rotated by the operator, not by us.
		{"env-owned key is operator-managed, so nothing rotates", []byte("-----BEGIN PRIVATE KEY-----"), ""},
		{"db-owned key rotates through the persisted store", nil, "persisted"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			origKey := config.JWTPrivateKey
			origInMemory := startKeyRotationRunner
			origPersisted := startSigningKeyRotationRunner
			t.Cleanup(func() {
				config.JWTPrivateKey = origKey
				startKeyRotationRunner = origInMemory
				startSigningKeyRotationRunner = origPersisted
			})

			config.JWTPrivateKey = tc.privateKey

			started := make(chan string, 2)
			startKeyRotationRunner = func(context.Context, time.Duration) { started <- "in-memory" }
			startSigningKeyRotationRunner = func(context.Context, *gorm.DB, time.Duration) { started <- "persisted" }

			// Neutralise every other worker so only the rotation choice is exercised.
			quietOtherWorkers(t)

			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			startBackgroundWorkers(ctx, &app.App{}, &appserver.Application{})

			if tc.wantRunner == "" {
				select {
				case got := <-started:
					t.Fatalf("no rotation runner may start for an operator-managed key, got %s", got)
				case <-time.After(300 * time.Millisecond):
				}
				return
			}

			select {
			case got := <-started:
				if got != tc.wantRunner {
					t.Fatalf("expected the %s rotation runner, got %s", tc.wantRunner, got)
				}
			case <-time.After(2 * time.Second):
				t.Fatalf("no rotation runner started; expected the %s one", tc.wantRunner)
			}

			// The other runner must NOT also start — two rotators would race.
			select {
			case other := <-started:
				t.Fatalf("both rotation runners started; the second was %s", other)
			case <-time.After(100 * time.Millisecond):
			}
		})
	}
}

// quietOtherWorkers stubs every background worker except the rotation runners, so
// the test exercises only the rotation-store choice and starts no real goroutine.
func quietOtherWorkers(t *testing.T) {
	t.Helper()
	origRetention := startRetentionRunner
	origTenantRetention := startTenantRetentionRunner
	origSecret := startSecretRefreshRunner
	origCleanup := startCleanupRunner
	origErasure := startDataErasureWorker
	origGRPC := startGRPCServer
	t.Cleanup(func() {
		startRetentionRunner = origRetention
		startTenantRetentionRunner = origTenantRetention
		startSecretRefreshRunner = origSecret
		startCleanupRunner = origCleanup
		startDataErasureWorker = origErasure
		startGRPCServer = origGRPC
	})

	startRetentionRunner = func(context.Context, authevent.RetentionDeleter, *gorm.DB, time.Duration, time.Duration) {}
	startTenantRetentionRunner = func(context.Context, *gorm.DB, time.Duration, time.Duration) {}
	startSecretRefreshRunner = func(context.Context, time.Duration) {}
	startCleanupRunner = func(context.Context, *gorm.DB, time.Duration) {}
	startDataErasureWorker = func(context.Context, user.DataErasureService, time.Duration) {}
	startGRPCServer = func(context.Context, *appserver.Application) error { return nil }
}
