package user

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	authv1 "github.com/maintainerd/auth/internal/platform/gen/go/maintainerd/auth"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gorm.io/datatypes"
)

type testUserTenantResolver struct {
	getByUUIDFn func(ctx context.Context, tenantUUID uuid.UUID) (*TenantServiceDataResult, error)
}

func (m *testUserTenantResolver) GetByUUID(ctx context.Context, tenantUUID uuid.UUID) (*TenantServiceDataResult, error) {
	if m.getByUUIDFn != nil {
		return m.getByUUIDFn(ctx, tenantUUID)
	}
	return &TenantServiceDataResult{TenantID: 1, TenantUUID: tenantUUID}, nil
}

type testUserService struct {
	getFn              func(ctx context.Context, filter UserServiceGetFilter) (*UserServiceGetResult, error)
	getByUUIDFn        func(ctx context.Context, userUUID uuid.UUID, tenantID int64) (*UserServiceDataResult, error)
	createFn           func(ctx context.Context, username, fullname string, email, phone *string, password, status string, metadata datatypes.JSON, tenantUUID string, creatorUserUUID uuid.UUID) (*UserServiceDataResult, error)
	updateFn           func(ctx context.Context, userUUID uuid.UUID, tenantID int64, username, fullname string, email, phone *string, status string, metadata datatypes.JSON, updaterUserUUID uuid.UUID) (*UserServiceDataResult, error)
	setStatusFn        func(ctx context.Context, userUUID uuid.UUID, tenantID int64, status string, updaterUserUUID uuid.UUID) (*UserServiceDataResult, error)
	verifyEmailFn      func(ctx context.Context, userUUID uuid.UUID, tenantID int64) (*UserServiceDataResult, error)
	verifyPhoneFn      func(ctx context.Context, userUUID uuid.UUID, tenantID int64) (*UserServiceDataResult, error)
	completeAccountFn  func(ctx context.Context, userUUID uuid.UUID, tenantID int64) (*UserServiceDataResult, error)
	deleteByUUIDFn     func(ctx context.Context, userUUID uuid.UUID, tenantID int64, deleterUserUUID uuid.UUID) (*UserServiceDataResult, error)
	assignUserRolesFn  func(ctx context.Context, userUUID uuid.UUID, roleUUIDs []uuid.UUID, tenantID int64) (*UserServiceDataResult, error)
	removeUserRoleFn   func(ctx context.Context, userUUID uuid.UUID, roleUUID uuid.UUID, tenantID int64) (*UserServiceDataResult, error)
	getUserRolesFn     func(ctx context.Context, userUUID uuid.UUID, tenantID int64, filter GetUserRolesFilter) ([]RoleServiceDataResult, int64, error)
	getUserIdentitiesFn func(ctx context.Context, userUUID uuid.UUID, tenantID int64, filter GetUserIdentitiesFilter) ([]UserIdentityServiceDataResult, int64, error)

	forcePasswordChangeFn func(ctx context.Context, userUUID uuid.UUID, force bool) error
}

func (m *testUserService) Get(ctx context.Context, filter UserServiceGetFilter) (*UserServiceGetResult, error) {
	return m.getFn(ctx, filter)
}
func (m *testUserService) GetByUUID(ctx context.Context, userUUID uuid.UUID, tenantID int64) (*UserServiceDataResult, error) {
	return m.getByUUIDFn(ctx, userUUID, tenantID)
}
func (m *testUserService) Create(ctx context.Context, username, fullname string, email, phone *string, password, status string, metadata datatypes.JSON, tenantUUID string, creatorUserUUID uuid.UUID) (*UserServiceDataResult, error) {
	return m.createFn(ctx, username, fullname, email, phone, password, status, metadata, tenantUUID, creatorUserUUID)
}
func (m *testUserService) Update(ctx context.Context, userUUID uuid.UUID, tenantID int64, username, fullname string, email, phone *string, status string, metadata datatypes.JSON, updaterUserUUID uuid.UUID) (*UserServiceDataResult, error) {
	return m.updateFn(ctx, userUUID, tenantID, username, fullname, email, phone, status, metadata, updaterUserUUID)
}
func (m *testUserService) SetStatus(ctx context.Context, userUUID uuid.UUID, tenantID int64, status string, updaterUserUUID uuid.UUID) (*UserServiceDataResult, error) {
	return m.setStatusFn(ctx, userUUID, tenantID, status, updaterUserUUID)
}
func (m *testUserService) VerifyEmail(ctx context.Context, userUUID uuid.UUID, tenantID int64) (*UserServiceDataResult, error) {
	return m.verifyEmailFn(ctx, userUUID, tenantID)
}
func (m *testUserService) VerifyPhone(ctx context.Context, userUUID uuid.UUID, tenantID int64) (*UserServiceDataResult, error) {
	return m.verifyPhoneFn(ctx, userUUID, tenantID)
}
func (m *testUserService) CompleteAccount(ctx context.Context, userUUID uuid.UUID, tenantID int64) (*UserServiceDataResult, error) {
	return m.completeAccountFn(ctx, userUUID, tenantID)
}
func (m *testUserService) DeleteByUUID(ctx context.Context, userUUID uuid.UUID, tenantID int64, deleterUserUUID uuid.UUID) (*UserServiceDataResult, error) {
	return m.deleteByUUIDFn(ctx, userUUID, tenantID, deleterUserUUID)
}
func (m *testUserService) AssignUserRoles(ctx context.Context, userUUID uuid.UUID, roleUUIDs []uuid.UUID, tenantID int64) (*UserServiceDataResult, error) {
	return m.assignUserRolesFn(ctx, userUUID, roleUUIDs, tenantID)
}
func (m *testUserService) RemoveUserRole(ctx context.Context, userUUID uuid.UUID, roleUUID uuid.UUID, tenantID int64) (*UserServiceDataResult, error) {
	return m.removeUserRoleFn(ctx, userUUID, roleUUID, tenantID)
}
func (m *testUserService) GetUserRoles(ctx context.Context, userUUID uuid.UUID, tenantID int64, filter GetUserRolesFilter) ([]RoleServiceDataResult, int64, error) {
	return m.getUserRolesFn(ctx, userUUID, tenantID, filter)
}
func (m *testUserService) GetUserIdentities(ctx context.Context, userUUID uuid.UUID, tenantID int64, filter GetUserIdentitiesFilter) ([]UserIdentityServiceDataResult, int64, error) {
	return m.getUserIdentitiesFn(ctx, userUUID, tenantID, filter)
}
func (m *testUserService) FindBySubAndClientID(ctx context.Context, sub string, clientID string) (*User, error) {
	return nil, nil
}
func (m *testUserService) ForcePasswordChange(ctx context.Context, userUUID uuid.UUID, force bool) error {
	return m.forcePasswordChangeFn(ctx, userUUID, force)
}

func TestUserGRPCHandler_RPCS(t *testing.T) {
	ctx := context.Background()
	tenantUUID := uuid.New()
	userUUID := uuid.New()
	roleUUID := uuid.New()
	now := time.Now()
	resolver := &testUserTenantResolver{}

	userResult := UserServiceDataResult{
		UserUUID: userUUID, Username: "testuser", Fullname: "Test User", Email: "test@example.com",
		Status: "active", CreatedAt: now, UpdatedAt: now,
	}
	roleResult := RoleServiceDataResult{
		RoleUUID: roleUUID, Name: "admin", Description: "Admin role", Status: "active",
		CreatedAt: now, UpdatedAt: now,
	}

	t.Run("list success", func(t *testing.T) {
		svc := &testUserService{
			getFn: func(ctx context.Context, filter UserServiceGetFilter) (*UserServiceGetResult, error) {
				return &UserServiceGetResult{Data: []UserServiceDataResult{userResult}, Total: 1, Page: 1, Limit: 10, TotalPages: 1}, nil
			},
		}
		h := NewUserGRPCHandler(resolver, svc)
		res, err := h.ListUsers(ctx, &authv1.ListUsersRequest{TenantUuid: tenantUUID.String()})
		if err != nil {
			t.Fatal(err)
		}
		if len(res.Users) != 1 {
			t.Fatalf("expected 1 user, got %d", len(res.Users))
		}
	})

	t.Run("get success", func(t *testing.T) {
		svc := &testUserService{
			getByUUIDFn: func(ctx context.Context, id uuid.UUID, tenantID int64) (*UserServiceDataResult, error) {
				return &userResult, nil
			},
		}
		h := NewUserGRPCHandler(resolver, svc)
		_, err := h.GetUser(ctx, &authv1.GetUserRequest{TenantUuid: tenantUUID.String(), UserUuid: userUUID.String()})
		if err != nil {
			t.Fatal(err)
		}
	})

	t.Run("create success", func(t *testing.T) {
		svc := &testUserService{
			createFn: func(ctx context.Context, username, fullname string, email, phone *string, password, status string, metadata datatypes.JSON, tUUID string, cu uuid.UUID) (*UserServiceDataResult, error) {
				return &userResult, nil
			},
		}
		h := NewUserGRPCHandler(resolver, svc)
		_, err := h.CreateUser(ctx, &authv1.CreateUserRequest{TenantUuid: tenantUUID.String(), Username: "testuser", Fullname: "Test User", Password: "pass", Status: "active"})
		if err != nil {
			t.Fatal(err)
		}
	})

	t.Run("listRoles success", func(t *testing.T) {
		svc := &testUserService{
			getUserRolesFn: func(ctx context.Context, id uuid.UUID, tenantID int64, filter GetUserRolesFilter) ([]RoleServiceDataResult, int64, error) {
				return []RoleServiceDataResult{roleResult}, 1, nil
			},
		}
		h := NewUserGRPCHandler(resolver, svc)
		res, err := h.ListUserRoles(ctx, &authv1.ListUserRolesRequest{TenantUuid: tenantUUID.String(), UserUuid: userUUID.String()})
		if err != nil {
			t.Fatal(err)
		}
		if len(res.Roles) != 1 {
			t.Fatalf("expected 1 role, got %d", len(res.Roles))
		}
	})

	t.Run("assignRoles success", func(t *testing.T) {
		svc := &testUserService{
			assignUserRolesFn: func(ctx context.Context, id uuid.UUID, roleUUIDs []uuid.UUID, tenantID int64) (*UserServiceDataResult, error) {
				return &userResult, nil
			},
		}
		h := NewUserGRPCHandler(resolver, svc)
		_, err := h.AssignUserRoles(ctx, &authv1.AssignUserRolesRequest{TenantUuid: tenantUUID.String(), UserUuid: userUUID.String(), RoleUuids: []string{roleUUID.String()}})
		if err != nil {
			t.Fatal(err)
		}
	})

	t.Run("validation errors", func(t *testing.T) {
		svc := &testUserService{
			getFn: func(ctx context.Context, filter UserServiceGetFilter) (*UserServiceGetResult, error) { return &UserServiceGetResult{}, nil },
		}
		h := NewUserGRPCHandler(resolver, svc)
		_, err := h.ListUsers(ctx, &authv1.ListUsersRequest{TenantUuid: "bad"})
		if code := status.Code(err); code != codes.InvalidArgument {
			t.Errorf("expected InvalidArgument, got %v", code)
		}
	})

	t.Run("service errors", func(t *testing.T) {
		svcErr := errors.New("db error")
		svc := &testUserService{
			getFn: func(ctx context.Context, filter UserServiceGetFilter) (*UserServiceGetResult, error) { return nil, svcErr },
		}
		h := NewUserGRPCHandler(resolver, svc)
		_, err := h.ListUsers(ctx, &authv1.ListUsersRequest{TenantUuid: tenantUUID.String()})
		if code := status.Code(err); code != codes.Internal {
			t.Errorf("expected Internal, got %v", code)
		}
	})
}
