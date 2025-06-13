package grpcserver

import (
	"log"
	"net"

	"github.com/maintainerd/auth/internal/app"
	authv1 "github.com/maintainerd/auth/internal/gen/go/auth/v1"
	"google.golang.org/grpc"
)

func StartGRPCServer(app *app.App) {
	lis, err := net.Listen("tcp", ":50051")
	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}

	s := grpc.NewServer()
	authv1.RegisterSeederServiceServer(s, app.SeederHandler)

	log.Println("gRPC server running on port 50051")
	if err := s.Serve(lis); err != nil {
		log.Fatalf("Failed to serve: %v", err)
	}
}
