package invite

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	authv1 "github.com/maintainerd/auth/internal/platform/gen/go/maintainerd/auth"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type testInviteTenantResolver struct {
	getFn func(ctx context.Context, tenantUUID uuid.UUID) (int64, error)
}

func (m *testInviteTenantResolver) GetTenantIDByUUID(ctx context.Context, tenantUUID uuid.UUID) (int64, error) {
	if m.getFn != nil {
		return m.getFn(ctx, tenantUUID)
	}
	return 1, nil
}

type testInviteService struct {
	sendFn func(ctx context.Context, tenantID int64, email string, userID int64, roleUUIDs []string) (*Invite, error)
}

func (m *testInviteService) SendInvite(ctx context.Context, tenantID int64, email string, userID int64, roleUUIDs []string) (*Invite, error) {
	return m.sendFn(ctx, tenantID, email, userID, roleUUIDs)
}

func TestInviteGRPCHandler_RPCS(t *testing.T) {
	ctx := context.Background()
	tenantUUID := uuid.New()
	roleUUID := uuid.New()
	resolver := &testInviteTenantResolver{}

	t.Run("send success", func(t *testing.T) {
		svc := &testInviteService{
			sendFn: func(ctx context.Context, tenantID int64, email string, userID int64, roleUUIDs []string) (*Invite, error) {
				return &Invite{}, nil
			},
		}
		h := NewInviteGRPCHandler(resolver, svc)
		res, err := h.SendInvite(ctx, &authv1.SendInviteRequest{TenantUuid: tenantUUID.String(), Email: "test@example.com", RoleUuids: []string{roleUUID.String()}, ActorUserUuid: uuid.New().String()})
		if err != nil {
			t.Fatal(err)
		}
		if res.Message == "" {
			t.Error("expected message")
		}
	})

	t.Run("validation errors", func(t *testing.T) {
		svc := &testInviteService{}
		h := NewInviteGRPCHandler(resolver, svc)
		_, err := h.SendInvite(ctx, &authv1.SendInviteRequest{TenantUuid: "bad", Email: "test@example.com"})
		if code := status.Code(err); code != codes.InvalidArgument {
			t.Errorf("expected InvalidArgument, got %v", code)
		}
	})

	t.Run("service errors", func(t *testing.T) {
		svc := &testInviteService{
			sendFn: func(ctx context.Context, tenantID int64, email string, userID int64, roleUUIDs []string) (*Invite, error) {
				return nil, errors.New("db error")
			},
		}
		h := NewInviteGRPCHandler(resolver, svc)
		_, err := h.SendInvite(ctx, &authv1.SendInviteRequest{TenantUuid: tenantUUID.String(), Email: "test@example.com", RoleUuids: []string{roleUUID.String()}, ActorUserUuid: uuid.New().String()})
		if code := status.Code(err); code != codes.Internal {
			t.Errorf("expected Internal, got %v", code)
		}
	})

	t.Run("missing tenant UUID", func(t *testing.T) {
		svc := &testInviteService{}
		h := NewInviteGRPCHandler(resolver, svc)
		_, err := h.SendInvite(ctx, &authv1.SendInviteRequest{Email: "test@example.com"})
		if code := status.Code(err); code != codes.InvalidArgument {
			t.Errorf("expected InvalidArgument, got %v", code)
		}
	})

	t.Run("parseUUID empty", func(t *testing.T) {
		_, err := parseUUID("", "test")
		if code := status.Code(err); code != codes.InvalidArgument {
			t.Errorf("expected InvalidArgument, got %v", code)
		}
	})

	t.Run("tenant resolver error", func(t *testing.T) {
		errResolver := &testInviteTenantResolver{getFn: func(ctx context.Context, tuuid uuid.UUID) (int64, error) { return 0, errors.New("tenant") }}
		svc := &testInviteService{}
		h := NewInviteGRPCHandler(errResolver, svc)
		_, err := h.SendInvite(ctx, &authv1.SendInviteRequest{TenantUuid: tenantUUID.String(), Email: "test@example.com"})
		if code := status.Code(err); code != codes.Internal { t.Errorf("expected Internal, got %v", code) }
	})
}
