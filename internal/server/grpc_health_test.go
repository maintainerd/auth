package server

import (
	"context"
	"errors"
	"testing"
	"time"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/health"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func TestGRPCOverallHealthStatus(t *testing.T) {
	t.Run("not serving without application", func(t *testing.T) {
		assert.Equal(t, healthpb.HealthCheckResponse_NOT_SERVING, grpcOverallHealthStatus(context.Background(), nil))
	})

	t.Run("not serving without db", func(t *testing.T) {
		assert.Equal(t, healthpb.HealthCheckResponse_NOT_SERVING, grpcOverallHealthStatus(context.Background(), &Application{}))
	})

	t.Run("not serving when db ping fails", func(t *testing.T) {
		initServerTestJWTKeys(t)
		db, mock := newGRPCHealthMockDB(t)
		mock.ExpectPing().WillReturnError(errors.New("db down"))

		assert.Equal(t, healthpb.HealthCheckResponse_NOT_SERVING, grpcOverallHealthStatus(context.Background(), &Application{DB: db}))
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("not serving without jwks", func(t *testing.T) {
		db, mock := newGRPCHealthMockDB(t)
		mock.ExpectPing()

		assert.Equal(t, healthpb.HealthCheckResponse_NOT_SERVING, grpcOverallHealthStatus(context.Background(), &Application{DB: db}))
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("not serving when redis ping fails", func(t *testing.T) {
		initServerTestJWTKeys(t)
		db, mock := newGRPCHealthMockDB(t)
		mock.ExpectPing()
		redisClient := redis.NewClient(&redis.Options{Addr: "127.0.0.1:1"})
		defer redisClient.Close()

		assert.Equal(t, healthpb.HealthCheckResponse_NOT_SERVING, grpcOverallHealthStatus(context.Background(), &Application{DB: db, RedisClient: redisClient}))
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("serving when dependencies are ready", func(t *testing.T) {
		initServerTestJWTKeys(t)
		db, mock := newGRPCHealthMockDB(t)
		mock.ExpectPing()
		redisServer := miniredis.RunT(t)
		redisClient := redis.NewClient(&redis.Options{Addr: redisServer.Addr()})
		defer redisClient.Close()

		assert.Equal(t, healthpb.HealthCheckResponse_SERVING, grpcOverallHealthStatus(context.Background(), &Application{DB: db, RedisClient: redisClient}))
		require.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestUpdateGRPCOverallHealth(t *testing.T) {
	healthServer := health.NewServer()

	updateGRPCOverallHealth(context.Background(), healthServer, &Application{})

	resp, err := healthServer.Check(context.Background(), &healthpb.HealthCheckRequest{})
	require.NoError(t, err)
	assert.Equal(t, healthpb.HealthCheckResponse_NOT_SERVING, resp.Status)
}

func TestRefreshGRPCOverallHealth(t *testing.T) {
	t.Run("returns when context is already canceled", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		refreshGRPCOverallHealth(ctx, health.NewServer(), &Application{}, time.Hour)
	})

	t.Run("updates on ticker until canceled", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		healthServer := health.NewServer()
		done := make(chan struct{})

		go func() {
			defer close(done)
			refreshGRPCOverallHealth(ctx, healthServer, &Application{}, time.Millisecond)
		}()

		require.Eventually(t, func() bool {
			resp, err := healthServer.Check(context.Background(), &healthpb.HealthCheckRequest{})
			return err == nil && resp.Status == healthpb.HealthCheckResponse_NOT_SERVING
		}, time.Second, time.Millisecond)
		cancel()
		require.Eventually(t, func() bool {
			select {
			case <-done:
				return true
			default:
				return false
			}
		}, time.Second, time.Millisecond)
	})
}

func TestSetGRPCServiceHealth(t *testing.T) {
	healthServer := health.NewServer()

	setGRPCServiceHealth(healthServer, healthpb.HealthCheckResponse_SERVING)

	resp, err := healthServer.Check(context.Background(), &healthpb.HealthCheckRequest{Service: "maintainerd.auth.v1.ServiceService"})
	require.NoError(t, err)
	assert.Equal(t, healthpb.HealthCheckResponse_SERVING, resp.Status)
}

func newGRPCHealthMockDB(t *testing.T) (*gorm.DB, sqlmock.Sqlmock) {
	t.Helper()
	sqlDB, mock, err := sqlmock.New(sqlmock.MonitorPingsOption(true), sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	require.NoError(t, err)
	t.Cleanup(func() { _ = sqlDB.Close() })

	db, err := gorm.Open(postgres.New(postgres.Config{Conn: sqlDB, PreferSimpleProtocol: true}), &gorm.Config{DisableAutomaticPing: true})
	require.NoError(t, err)
	return db, mock
}
