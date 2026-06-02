package main

import (
	"context"
	"testing"
	"time"

	"github.com/maintainerd/auth/internal/app"
	"github.com/maintainerd/auth/internal/authevent"
	"github.com/maintainerd/auth/internal/platform/config"
	appserver "github.com/maintainerd/auth/internal/server"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStartBackgroundWorkers_StartsSecurityRunners(t *testing.T) {
	origRetention := startRetentionRunner
	origKeyRotation := startKeyRotationRunner
	origSecretRefresh := startSecretRefreshRunner
	origGRPC := startGRPCServer
	origKeyRotationPeriod := config.JWTKeyRotationPeriodSeconds
	origSecretRefreshPeriod := config.SecretRefreshPeriodSeconds
	t.Cleanup(func() {
		startRetentionRunner = origRetention
		startKeyRotationRunner = origKeyRotation
		startSecretRefreshRunner = origSecretRefresh
		startGRPCServer = origGRPC
		config.JWTKeyRotationPeriodSeconds = origKeyRotationPeriod
		config.SecretRefreshPeriodSeconds = origSecretRefreshPeriod
	})

	config.JWTKeyRotationPeriodSeconds = 42
	config.SecretRefreshPeriodSeconds = 7

	type workerCall struct {
		name   string
		period time.Duration
	}
	calls := make(chan workerCall, 4)

	startRetentionRunner = func(context.Context, authevent.RetentionDeleter, time.Duration, time.Duration) {
		calls <- workerCall{name: "retention"}
	}
	startKeyRotationRunner = func(_ context.Context, period time.Duration) {
		calls <- workerCall{name: "key_rotation", period: period}
	}
	startSecretRefreshRunner = func(_ context.Context, period time.Duration) {
		calls <- workerCall{name: "secret_refresh", period: period}
	}
	startGRPCServer = func(context.Context, *appserver.Application) error {
		calls <- workerCall{name: "grpc"}
		return nil
	}

	startBackgroundWorkers(
		context.Background(),
		&app.App{AuthEventService: authevent.NoopService()},
		&appserver.Application{},
	)

	seen := map[string]time.Duration{}
	for range 4 {
		select {
		case call := <-calls:
			seen[call.name] = call.period
		case <-time.After(time.Second):
			t.Fatal("timed out waiting for background worker startup")
		}
	}

	require.Contains(t, seen, "retention")
	require.Contains(t, seen, "grpc")
	assert.Equal(t, 42*time.Second, seen["key_rotation"])
	assert.Equal(t, 7*time.Second, seen["secret_refresh"])
}

func TestStartBackgroundWorkers_UsesDefaultSecurityRunnerPeriods(t *testing.T) {
	origRetention := startRetentionRunner
	origKeyRotation := startKeyRotationRunner
	origSecretRefresh := startSecretRefreshRunner
	origGRPC := startGRPCServer
	origKeyRotationPeriod := config.JWTKeyRotationPeriodSeconds
	origSecretRefreshPeriod := config.SecretRefreshPeriodSeconds
	t.Cleanup(func() {
		startRetentionRunner = origRetention
		startKeyRotationRunner = origKeyRotation
		startSecretRefreshRunner = origSecretRefresh
		startGRPCServer = origGRPC
		config.JWTKeyRotationPeriodSeconds = origKeyRotationPeriod
		config.SecretRefreshPeriodSeconds = origSecretRefreshPeriod
	})

	config.JWTKeyRotationPeriodSeconds = 0
	config.SecretRefreshPeriodSeconds = -1

	keyRotationPeriods := make(chan time.Duration, 1)
	secretRefreshPeriods := make(chan time.Duration, 1)

	startRetentionRunner = func(context.Context, authevent.RetentionDeleter, time.Duration, time.Duration) {}
	startKeyRotationRunner = func(_ context.Context, period time.Duration) {
		keyRotationPeriods <- period
	}
	startSecretRefreshRunner = func(_ context.Context, period time.Duration) {
		secretRefreshPeriods <- period
	}
	startGRPCServer = func(context.Context, *appserver.Application) error { return nil }

	startBackgroundWorkers(
		context.Background(),
		&app.App{AuthEventService: authevent.NoopService()},
		&appserver.Application{},
	)

	select {
	case period := <-keyRotationPeriods:
		assert.Equal(t, 24*time.Hour, period)
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for key rotation runner")
	}

	select {
	case period := <-secretRefreshPeriods:
		assert.Equal(t, 5*time.Minute, period)
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for secret refresh runner")
	}
}
