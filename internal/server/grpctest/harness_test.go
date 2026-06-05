package grpctest

import (
	"context"
	"errors"
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/health"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/test/bufconn"
)

func TestNew(t *testing.T) {
	h := New(t, func(server *grpc.Server) {
		healthServer := health.NewServer()
		healthServer.SetServingStatus("", healthpb.HealthCheckResponse_SERVING)
		healthpb.RegisterHealthServer(server, healthServer)
	})

	resp, err := healthpb.NewHealthClient(h.Conn).Check(context.Background(), &healthpb.HealthCheckRequest{})
	require.NoError(t, err)
	assert.Equal(t, healthpb.HealthCheckResponse_SERVING, resp.Status)
	assert.NotNil(t, h.Server)
	assert.NotNil(t, h.Listener)
}

func TestCloseNil(t *testing.T) {
	var h *Harness
	require.NoError(t, h.Close())
}

func TestNewWithoutRegister(t *testing.T) {
	h := New(t, nil)

	assert.NotNil(t, h.Server)
	assert.NotNil(t, h.Conn)
	assert.NotNil(t, h.Listener)
}

func TestClosePartialHarness(t *testing.T) {
	t.Run("empty", func(t *testing.T) {
		require.NoError(t, (&Harness{}).Close())
	})

	t.Run("server only", func(t *testing.T) {
		server := grpc.NewServer()
		require.NoError(t, (&Harness{Server: server}).Close())
	})

	t.Run("conn close error", func(t *testing.T) {
		wantErr := errors.New("close failed")
		h := &Harness{
			Conn:      &grpc.ClientConn{},
			closeConn: func() error { return wantErr },
		}

		require.ErrorIs(t, h.Close(), wantErr)
	})
}

func TestWaitForServeStop(t *testing.T) {
	t.Run("nil error", func(t *testing.T) {
		errCh := make(chan error, 1)
		errCh <- nil

		require.NoError(t, waitForServeStop(errCh, time.After(time.Second)))
	})

	t.Run("grpc server stopped", func(t *testing.T) {
		errCh := make(chan error, 1)
		errCh <- grpc.ErrServerStopped

		require.NoError(t, waitForServeStop(errCh, time.After(time.Second)))
	})

	t.Run("unexpected error", func(t *testing.T) {
		wantErr := errors.New("serve failed")
		errCh := make(chan error, 1)
		errCh <- wantErr

		require.ErrorIs(t, waitForServeStop(errCh, time.After(time.Second)), wantErr)
	})

	t.Run("timeout", func(t *testing.T) {
		errCh := make(chan error)
		timeout := make(chan time.Time, 1)
		timeout <- time.Now()

		require.EqualError(t, waitForServeStop(errCh, timeout), "gRPC bufconn server did not stop")
	})
}

func TestCloseUsesConnCloseWhenNoHook(t *testing.T) {
	listener := bufconn.Listen(1024)
	defer func() { _ = listener.Close() }()

	server := grpc.NewServer()
	defer server.Stop()

	errCh := make(chan error, 1)
	go func() {
		errCh <- server.Serve(listener)
	}()

	conn, err := grpc.NewClient(
		"passthrough:///bufnet",
		grpc.WithContextDialer(func(ctx context.Context, _ string) (net.Conn, error) {
			return listener.DialContext(ctx)
		}),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	require.NoError(t, err)

	require.NoError(t, (&Harness{Conn: conn, Server: server}).Close())
	require.NoError(t, waitForServeStop(errCh, time.After(time.Second)))
}
