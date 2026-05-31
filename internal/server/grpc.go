package server

import (
	"context"
	"fmt"
	"log/slog"
	"net"

	authv1 "github.com/maintainerd/auth/internal/platform/gen/go/maintainerd/auth"
	"github.com/maintainerd/auth/internal/setup"
	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"google.golang.org/grpc"
)

// StartGRPCServer binds to :50051 and serves until ctx is cancelled, at which
// point it drains in-flight RPCs via GracefulStop. It returns an error for any
// fatal startup failure so that main() can handle it appropriately instead of
// calling os.Exit inside a library function.
func StartGRPCServer(ctx context.Context, application *Application) error {
	lis, err := net.Listen("tcp", ":50051")
	if err != nil {
		return fmt.Errorf("gRPC failed to listen on :50051: %w", err)
	}

	seederHandler := &setup.SeederHandler{}

	s := grpc.NewServer(
		grpc.StatsHandler(otelgrpc.NewServerHandler()),
	)
	authv1.RegisterSeederServiceServer(s, seederHandler)

	// Stop the server when the context is cancelled (e.g. after REST servers drain).
	go func() {
		<-ctx.Done()
		slog.Info("gRPC shutdown signal received, draining connections...")
		s.GracefulStop()
	}()

	slog.Info("gRPC server starting", "addr", ":50051")
	if err := s.Serve(lis); err != nil {
		return fmt.Errorf("gRPC server failed: %w", err)
	}
	return nil
}
