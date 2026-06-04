package setup

import (
	"context"

	authv1 "github.com/maintainerd/auth/internal/platform/gen/go/maintainerd/auth"
	"github.com/maintainerd/auth/internal/platform/runner"
	"gorm.io/gorm"
)

var setupRunSeeders = runner.RunSeeders

type SeederHandler struct {
	authv1.UnimplementedSeederServiceServer
	db *gorm.DB
}

func NewSeederHandler(db *gorm.DB) *SeederHandler {
	return &SeederHandler{db: db}
}

func (h *SeederHandler) TriggerSeeder(_ context.Context, _ *authv1.TriggerSeederRequest) (*authv1.TriggerSeederResponse, error) {
	if err := setupRunSeeders(h.db, "v0.1.0"); err != nil {
		return nil, err
	}
	return &authv1.TriggerSeederResponse{
		Success: true,
		Message: "seeders completed",
	}, nil
}
