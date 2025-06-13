package main

import (
	"github.com/maintainerd/auth/internal/app"
	"github.com/maintainerd/auth/internal/config"
	grpcserver "github.com/maintainerd/auth/internal/grpc"
	restserver "github.com/maintainerd/auth/internal/rest"
	"github.com/maintainerd/auth/internal/startup"
)

func main() {
	db := config.InitDB()

	// 🔁 App startup routines (migrations, seeding, etc.)
	startup.RunAppStartUp(db)

	// ⚙️ App wiring (handlers, services, etc.)
	application := app.NewApp(db)

	// 🚀 gRPC server (background)
	go grpcserver.StartGRPCServer(application)

	// 🌐 REST server (main)
	restserver.StartRESTServer(application)
}
