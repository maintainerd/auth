package main

import (
	"log"

	"github.com/maintainerd/auth/internal/app"
	"github.com/maintainerd/auth/internal/config"
	grpcserver "github.com/maintainerd/auth/internal/grpc"
	restserver "github.com/maintainerd/auth/internal/rest"
	"github.com/maintainerd/auth/internal/startup"
	"github.com/maintainerd/auth/internal/util"
)

func main() {
	// ⚙️ Load configurations
	config.Init()

	// ⚙️ Parse RSA keys (required for token signing)
	if err := util.InitJWTKeys(); err != nil {
		log.Fatalf("❌ Failed to initialize JWT keys: %v", err)
	}

	// ⚙️ Load database
	db := config.InitDB()

	// ⚙️ Load Redis
	redisClient := config.NewRedisClient()

	// 🔁 App startup routines (migrations, seeding, etc.)
	startup.RunAppStartUp(db)

	// ⚙️ App wiring (handlers, services, etc.)
	application := app.NewApp(db, redisClient)

	// 🚀 gRPC server (background)
	go grpcserver.StartGRPCServer(application)

	// 🌐 REST server (main)
	restserver.StartRESTServer(application)
}
