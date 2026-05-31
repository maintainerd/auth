package setup

import (
	authv1 "github.com/maintainerd/auth/internal/platform/gen/go/maintainerd/auth"
)

type SeederHandler struct {
	authv1.UnimplementedSeederServiceServer
}
