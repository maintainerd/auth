package grpctest

import (
	"context"
	"errors"
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/test/bufconn"
)

const defaultBufferSize = 1024 * 1024

type Harness struct {
	Server    *grpc.Server
	Conn      *grpc.ClientConn
	Listener  *bufconn.Listener
	closeConn func() error
}

func New(t testing.TB, register func(*grpc.Server), opts ...grpc.ServerOption) *Harness {
	t.Helper()

	listener := bufconn.Listen(defaultBufferSize)
	server := grpc.NewServer(opts...)
	if register != nil {
		register(server)
	}

	errCh := make(chan error, 1)
	go func() {
		errCh <- server.Serve(listener)
	}()

	conn, err := grpc.NewClient(
		"passthrough:///bufnet",
		grpc.WithContextDialer(func(ctx context.Context, _ string) (net.Conn, error) {
			return listener.DialContext(ctx)
		}),
		grpc.WithTransportCredentials(insecure.NewCredentials()), // nosemgrep
	)
	require.NoError(t, err)

	h := &Harness{
		Server:    server,
		Conn:      conn,
		Listener:  listener,
		closeConn: conn.Close,
	}
	t.Cleanup(func() {
		require.NoError(t, h.Close())
		require.NoError(t, waitForServeStop(errCh, time.After(time.Second)))
	})

	return h
}

func (h *Harness) Close() error {
	if h == nil {
		return nil
	}
	if h.Conn != nil {
		closeConn := h.Conn.Close
		if h.closeConn != nil {
			closeConn = h.closeConn
		}
		if err := closeConn(); err != nil {
			return err
		}
	}
	if h.Server != nil {
		h.Server.Stop()
	}
	return nil
}

func waitForServeStop(errCh <-chan error, timeout <-chan time.Time) error {
	select {
	case err := <-errCh:
		if err != nil && !errors.Is(err, grpc.ErrServerStopped) {
			return err
		}
		return nil
	case <-timeout:
		return errors.New("gRPC bufconn server did not stop")
	}
}
