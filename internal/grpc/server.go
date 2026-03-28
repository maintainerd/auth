package grpcserver

import (
	"log/slog"
	"net"
	"os"

	"github.com/maintainerd/auth/internal/app"
	authv1 "github.com/maintainerd/auth/internal/gen/go/auth/v1"
	"google.golang.org/grpc"
)

func StartGRPCServer(app *app.App) {
	lis, err := net.Listen("tcp", ":50051")
	if err != nil {
		slog.Error("gRPC failed to listen", "error", err)
		os.Exit(1)
	}

	s := grpc.NewServer()
	authv1.RegisterSeederServiceServer(s, app.SeederHandler)

	slog.Info("gRPC server starting", "addr", ":50051")
	if err := s.Serve(lis); err != nil {
		slog.Error("gRPC server failed", "error", err)
		os.Exit(1)
	}
}
