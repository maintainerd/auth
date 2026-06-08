package user

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	authv1 "github.com/maintainerd/auth/internal/platform/gen/go/maintainerd/auth"
	"github.com/maintainerd/auth/internal/platform/pagination"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/structpb"
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
	getFn               func(ctx context.Context, filter UserServiceGetFilter) (*UserServiceGetResult, error)
	getByUUIDFn         func(ctx context.Context, userUUID uuid.UUID, tenantID int64) (*UserServiceDataResult, error)
	createFn            func(ctx context.Context, username string, email, phone *string, password, status string, metadata datatypes.JSON, tenantUUID string, creatorUserUUID uuid.UUID) (*UserServiceDataResult, error)
	updateFn            func(ctx context.Context, userUUID uuid.UUID, tenantID int64, username string, email, phone *string, status string, metadata datatypes.JSON, updaterUserUUID uuid.UUID) (*UserServiceDataResult, error)
	setStatusFn         func(ctx context.Context, userUUID uuid.UUID, tenantID int64, status string, updaterUserUUID uuid.UUID) (*UserServiceDataResult, error)
	verifyEmailFn       func(ctx context.Context, userUUID uuid.UUID, tenantID int64) (*UserServiceDataResult, error)
	verifyPhoneFn       func(ctx context.Context, userUUID uuid.UUID, tenantID int64) (*UserServiceDataResult, error)
	completeAccountFn   func(ctx context.Context, userUUID uuid.UUID, tenantID int64) (*UserServiceDataResult, error)
	deleteByUUIDFn      func(ctx context.Context, userUUID uuid.UUID, tenantID int64, deleterUserUUID uuid.UUID) (*UserServiceDataResult, error)
	assignUserRolesFn   func(ctx context.Context, userUUID uuid.UUID, roleUUIDs []uuid.UUID, tenantID int64) (*UserServiceDataResult, error)
	removeUserRoleFn    func(ctx context.Context, userUUID uuid.UUID, roleUUID uuid.UUID, tenantID int64) (*UserServiceDataResult, error)
	getUserRolesFn      func(ctx context.Context, userUUID uuid.UUID, tenantID int64, filter GetUserRolesFilter) ([]RoleServiceDataResult, int64, error)
	getUserIdentitiesFn func(ctx context.Context, userUUID uuid.UUID, tenantID int64, filter GetUserIdentitiesFilter) ([]UserIdentityServiceDataResult, int64, error)

	forcePasswordChangeFn func(ctx context.Context, userUUID uuid.UUID, force bool) error
}

func (m *testUserService) Get(ctx context.Context, filter UserServiceGetFilter) (*UserServiceGetResult, error) {
	return m.getFn(ctx, filter)
}
func (m *testUserService) GetByUUID(ctx context.Context, userUUID uuid.UUID, tenantID int64) (*UserServiceDataResult, error) {
	return m.getByUUIDFn(ctx, userUUID, tenantID)
}
func (m *testUserService) Create(ctx context.Context, username string, email, phone *string, password, status string, metadata datatypes.JSON, tenantUUID string, creatorUserUUID uuid.UUID) (*UserServiceDataResult, error) {
	return m.createFn(ctx, username, email, phone, password, status, metadata, tenantUUID, creatorUserUUID)
}
func (m *testUserService) Update(ctx context.Context, userUUID uuid.UUID, tenantID int64, username string, email, phone *string, status string, metadata datatypes.JSON, updaterUserUUID uuid.UUID) (*UserServiceDataResult, error) {
	return m.updateFn(ctx, userUUID, tenantID, username, email, phone, status, metadata, updaterUserUUID)
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
			createFn: func(ctx context.Context, username string, email, phone *string, password, status string, metadata datatypes.JSON, tUUID string, cu uuid.UUID) (*UserServiceDataResult, error) {
				return &userResult, nil
			},
		}
		h := NewUserGRPCHandler(resolver, svc)
		_, err := h.CreateUser(ctx, &authv1.CreateUserRequest{TenantUuid: tenantUUID.String(), Username: "testuser", Password: "pass", Status: "active"})
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
			getFn: func(ctx context.Context, filter UserServiceGetFilter) (*UserServiceGetResult, error) {
				return &UserServiceGetResult{}, nil
			},
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
			getFn: func(ctx context.Context, filter UserServiceGetFilter) (*UserServiceGetResult, error) {
				return nil, svcErr
			},
		}
		h := NewUserGRPCHandler(resolver, svc)
		_, err := h.ListUsers(ctx, &authv1.ListUsersRequest{TenantUuid: tenantUUID.String()})
		if code := status.Code(err); code != codes.Internal {
			t.Errorf("expected Internal, got %v", code)
		}
	})

	t.Run("update success", func(t *testing.T) {
		svc := &testUserService{
			updateFn: func(ctx context.Context, userUUID uuid.UUID, tenantID int64, username string, email, phone *string, status string, metadata datatypes.JSON, updaterUserUUID uuid.UUID) (*UserServiceDataResult, error) {
				return &userResult, nil
			},
		}
		h := NewUserGRPCHandler(resolver, svc)
		_, err := h.UpdateUser(ctx, &authv1.UpdateUserRequest{TenantUuid: tenantUUID.String(), UserUuid: userUUID.String()})
		if err != nil {
			t.Fatal(err)
		}
	})

	t.Run("update service error", func(t *testing.T) {
		svc := &testUserService{
			updateFn: func(ctx context.Context, userUUID uuid.UUID, tenantID int64, username string, email, phone *string, status string, metadata datatypes.JSON, updaterUserUUID uuid.UUID) (*UserServiceDataResult, error) {
				return nil, errors.New("db")
			},
		}
		h := NewUserGRPCHandler(resolver, svc)
		_, err := h.UpdateUser(ctx, &authv1.UpdateUserRequest{TenantUuid: tenantUUID.String(), UserUuid: userUUID.String()})
		if code := status.Code(err); code != codes.Internal {
			t.Errorf("expected Internal, got %v", code)
		}
	})

	t.Run("setUserStatus success", func(t *testing.T) {
		svc := &testUserService{
			setStatusFn: func(ctx context.Context, userUUID uuid.UUID, tenantID int64, rStatus string, updaterUserUUID uuid.UUID) (*UserServiceDataResult, error) {
				return &userResult, nil
			},
		}
		h := NewUserGRPCHandler(resolver, svc)
		_, err := h.SetUserStatus(ctx, &authv1.SetUserStatusRequest{TenantUuid: tenantUUID.String(), UserUuid: userUUID.String(), Status: "inactive"})
		if err != nil {
			t.Fatal(err)
		}
	})

	t.Run("setUserStatus service error", func(t *testing.T) {
		svc := &testUserService{
			setStatusFn: func(ctx context.Context, userUUID uuid.UUID, tenantID int64, rStatus string, updaterUserUUID uuid.UUID) (*UserServiceDataResult, error) {
				return nil, errors.New("db")
			},
		}
		h := NewUserGRPCHandler(resolver, svc)
		_, err := h.SetUserStatus(ctx, &authv1.SetUserStatusRequest{TenantUuid: tenantUUID.String(), UserUuid: userUUID.String(), Status: "inactive"})
		if code := status.Code(err); code != codes.Internal {
			t.Errorf("expected Internal, got %v", code)
		}
	})

	t.Run("verifyEmail success", func(t *testing.T) {
		svc := &testUserService{
			verifyEmailFn: func(ctx context.Context, userUUID uuid.UUID, tenantID int64) (*UserServiceDataResult, error) {
				return &userResult, nil
			},
		}
		h := NewUserGRPCHandler(resolver, svc)
		_, err := h.VerifyUserEmail(ctx, &authv1.VerifyUserEmailRequest{TenantUuid: tenantUUID.String(), UserUuid: userUUID.String()})
		if err != nil {
			t.Fatal(err)
		}
	})

	t.Run("verifyEmail service error", func(t *testing.T) {
		svc := &testUserService{
			verifyEmailFn: func(ctx context.Context, userUUID uuid.UUID, tenantID int64) (*UserServiceDataResult, error) {
				return nil, errors.New("db")
			},
		}
		h := NewUserGRPCHandler(resolver, svc)
		_, err := h.VerifyUserEmail(ctx, &authv1.VerifyUserEmailRequest{TenantUuid: tenantUUID.String(), UserUuid: userUUID.String()})
		if code := status.Code(err); code != codes.Internal {
			t.Errorf("expected Internal, got %v", code)
		}
	})

	t.Run("verifyPhone success", func(t *testing.T) {
		svc := &testUserService{
			verifyPhoneFn: func(ctx context.Context, userUUID uuid.UUID, tenantID int64) (*UserServiceDataResult, error) {
				return &userResult, nil
			},
		}
		h := NewUserGRPCHandler(resolver, svc)
		_, err := h.VerifyUserPhone(ctx, &authv1.VerifyUserPhoneRequest{TenantUuid: tenantUUID.String(), UserUuid: userUUID.String()})
		if err != nil {
			t.Fatal(err)
		}
	})

	t.Run("verifyPhone service error", func(t *testing.T) {
		svc := &testUserService{
			verifyPhoneFn: func(ctx context.Context, userUUID uuid.UUID, tenantID int64) (*UserServiceDataResult, error) {
				return nil, errors.New("db")
			},
		}
		h := NewUserGRPCHandler(resolver, svc)
		_, err := h.VerifyUserPhone(ctx, &authv1.VerifyUserPhoneRequest{TenantUuid: tenantUUID.String(), UserUuid: userUUID.String()})
		if code := status.Code(err); code != codes.Internal {
			t.Errorf("expected Internal, got %v", code)
		}
	})

	t.Run("completeAccount success", func(t *testing.T) {
		svc := &testUserService{
			completeAccountFn: func(ctx context.Context, userUUID uuid.UUID, tenantID int64) (*UserServiceDataResult, error) {
				return &userResult, nil
			},
		}
		h := NewUserGRPCHandler(resolver, svc)
		_, err := h.CompleteUserAccount(ctx, &authv1.CompleteUserAccountRequest{TenantUuid: tenantUUID.String(), UserUuid: userUUID.String()})
		if err != nil {
			t.Fatal(err)
		}
	})

	t.Run("completeAccount service error", func(t *testing.T) {
		svc := &testUserService{
			completeAccountFn: func(ctx context.Context, userUUID uuid.UUID, tenantID int64) (*UserServiceDataResult, error) {
				return nil, errors.New("db")
			},
		}
		h := NewUserGRPCHandler(resolver, svc)
		_, err := h.CompleteUserAccount(ctx, &authv1.CompleteUserAccountRequest{TenantUuid: tenantUUID.String(), UserUuid: userUUID.String()})
		if code := status.Code(err); code != codes.Internal {
			t.Errorf("expected Internal, got %v", code)
		}
	})

	t.Run("delete success", func(t *testing.T) {
		svc := &testUserService{
			deleteByUUIDFn: func(ctx context.Context, userUUID uuid.UUID, tenantID int64, deleterUserUUID uuid.UUID) (*UserServiceDataResult, error) {
				return &userResult, nil
			},
		}
		h := NewUserGRPCHandler(resolver, svc)
		_, err := h.DeleteUser(ctx, &authv1.DeleteUserRequest{TenantUuid: tenantUUID.String(), UserUuid: userUUID.String()})
		if err != nil {
			t.Fatal(err)
		}
	})

	t.Run("delete service error", func(t *testing.T) {
		svc := &testUserService{
			deleteByUUIDFn: func(ctx context.Context, userUUID uuid.UUID, tenantID int64, deleterUserUUID uuid.UUID) (*UserServiceDataResult, error) {
				return nil, errors.New("db")
			},
		}
		h := NewUserGRPCHandler(resolver, svc)
		_, err := h.DeleteUser(ctx, &authv1.DeleteUserRequest{TenantUuid: tenantUUID.String(), UserUuid: userUUID.String()})
		if code := status.Code(err); code != codes.Internal {
			t.Errorf("expected Internal, got %v", code)
		}
	})

	t.Run("forcePasswordChange success", func(t *testing.T) {
		svc := &testUserService{
			forcePasswordChangeFn: func(ctx context.Context, userUUID uuid.UUID, force bool) error {
				return nil
			},
		}
		h := NewUserGRPCHandler(resolver, svc)
		res, err := h.ForceUserPasswordChange(ctx, &authv1.ForceUserPasswordChangeRequest{TenantUuid: tenantUUID.String(), UserUuid: userUUID.String(), Force: true})
		if err != nil {
			t.Fatal(err)
		}
		if !res.Success {
			t.Error("expected success")
		}
	})

	t.Run("forcePasswordChange service error", func(t *testing.T) {
		svc := &testUserService{
			forcePasswordChangeFn: func(ctx context.Context, userUUID uuid.UUID, force bool) error {
				return errors.New("db")
			},
		}
		h := NewUserGRPCHandler(resolver, svc)
		_, err := h.ForceUserPasswordChange(ctx, &authv1.ForceUserPasswordChangeRequest{TenantUuid: tenantUUID.String(), UserUuid: userUUID.String(), Force: true})
		if code := status.Code(err); code != codes.Internal {
			t.Errorf("expected Internal, got %v", code)
		}
	})

	t.Run("listIdentities success", func(t *testing.T) {
		svc := &testUserService{
			getUserIdentitiesFn: func(ctx context.Context, userUUID uuid.UUID, tenantID int64, filter GetUserIdentitiesFilter) ([]UserIdentityServiceDataResult, int64, error) {
				return []UserIdentityServiceDataResult{{UserIdentityUUID: uuid.New(), Provider: "google"}}, 1, nil
			},
		}
		h := NewUserGRPCHandler(resolver, svc)
		res, err := h.ListUserIdentities(ctx, &authv1.ListUserIdentitiesRequest{TenantUuid: tenantUUID.String(), UserUuid: userUUID.String()})
		if err != nil {
			t.Fatal(err)
		}
		if len(res.Identities) != 1 {
			t.Errorf("expected 1 identity, got %d", len(res.Identities))
		}
	})

	t.Run("listIdentities service error", func(t *testing.T) {
		svc := &testUserService{
			getUserIdentitiesFn: func(ctx context.Context, userUUID uuid.UUID, tenantID int64, filter GetUserIdentitiesFilter) ([]UserIdentityServiceDataResult, int64, error) {
				return nil, 0, errors.New("db")
			},
		}
		h := NewUserGRPCHandler(resolver, svc)
		_, err := h.ListUserIdentities(ctx, &authv1.ListUserIdentitiesRequest{TenantUuid: tenantUUID.String(), UserUuid: userUUID.String()})
		if code := status.Code(err); code != codes.Internal {
			t.Errorf("expected Internal, got %v", code)
		}
	})

	t.Run("removeUserRole success", func(t *testing.T) {
		svc := &testUserService{
			removeUserRoleFn: func(ctx context.Context, userUUID uuid.UUID, roleUUID uuid.UUID, tenantID int64) (*UserServiceDataResult, error) {
				return &userResult, nil
			},
		}
		h := NewUserGRPCHandler(resolver, svc)
		_, err := h.RemoveUserRole(ctx, &authv1.RemoveUserRoleRequest{TenantUuid: tenantUUID.String(), UserUuid: userUUID.String(), RoleUuid: roleUUID.String()})
		if err != nil {
			t.Fatal(err)
		}
	})

	t.Run("removeUserRole service error", func(t *testing.T) {
		svc := &testUserService{
			removeUserRoleFn: func(ctx context.Context, userUUID uuid.UUID, roleUUID uuid.UUID, tenantID int64) (*UserServiceDataResult, error) {
				return nil, errors.New("db")
			},
		}
		h := NewUserGRPCHandler(resolver, svc)
		_, err := h.RemoveUserRole(ctx, &authv1.RemoveUserRoleRequest{TenantUuid: tenantUUID.String(), UserUuid: userUUID.String(), RoleUuid: roleUUID.String()})
		if code := status.Code(err); code != codes.Internal {
			t.Errorf("expected Internal, got %v", code)
		}
	})

	t.Run("get service error", func(t *testing.T) {
		svc := &testUserService{
			getByUUIDFn: func(ctx context.Context, userUUID uuid.UUID, tenantID int64) (*UserServiceDataResult, error) {
				return nil, errors.New("db")
			},
		}
		h := NewUserGRPCHandler(resolver, svc)
		_, err := h.GetUser(ctx, &authv1.GetUserRequest{TenantUuid: tenantUUID.String(), UserUuid: userUUID.String()})
		if code := status.Code(err); code != codes.Internal {
			t.Errorf("expected Internal, got %v", code)
		}
	})

	t.Run("create service error", func(t *testing.T) {
		svc := &testUserService{
			createFn: func(ctx context.Context, username string, email, phone *string, password, rStatus string, metadata datatypes.JSON, tUUID string, cu uuid.UUID) (*UserServiceDataResult, error) {
				return nil, errors.New("db")
			},
		}
		h := NewUserGRPCHandler(resolver, svc)
		_, err := h.CreateUser(ctx, &authv1.CreateUserRequest{TenantUuid: tenantUUID.String(), Username: "test", Password: "pass"})
		if code := status.Code(err); code != codes.Internal {
			t.Errorf("expected Internal, got %v", code)
		}
	})

	t.Run("listRoles service error", func(t *testing.T) {
		svc := &testUserService{
			getUserRolesFn: func(ctx context.Context, userUUID uuid.UUID, tenantID int64, filter GetUserRolesFilter) ([]RoleServiceDataResult, int64, error) {
				return nil, 0, errors.New("db")
			},
		}
		h := NewUserGRPCHandler(resolver, svc)
		_, err := h.ListUserRoles(ctx, &authv1.ListUserRolesRequest{TenantUuid: tenantUUID.String(), UserUuid: userUUID.String()})
		if code := status.Code(err); code != codes.Internal {
			t.Errorf("expected Internal, got %v", code)
		}
	})

	t.Run("assignRoles service error", func(t *testing.T) {
		svc := &testUserService{
			assignUserRolesFn: func(ctx context.Context, userUUID uuid.UUID, roleUUIDs []uuid.UUID, tenantID int64) (*UserServiceDataResult, error) {
				return nil, errors.New("db")
			},
		}
		h := NewUserGRPCHandler(resolver, svc)
		_, err := h.AssignUserRoles(ctx, &authv1.AssignUserRolesRequest{TenantUuid: tenantUUID.String(), UserUuid: userUUID.String(), RoleUuids: []string{roleUUID.String()}})
		if code := status.Code(err); code != codes.Internal {
			t.Errorf("expected Internal, got %v", code)
		}
	})

	t.Run("grpcPagination nil", func(t *testing.T) {
		dto := grpcPagination(nil)
		if dto.Page != 1 || dto.Limit != pagination.DefaultPageSize {
			t.Errorf("expected defaults, got %+v", dto)
		}
	})

	t.Run("toUserProto nil", func(t *testing.T) {
		v := toUserProto(nil)
		if v != nil {
			t.Error("expected nil")
		}
	})

	t.Run("toUserRoleProto nil", func(t *testing.T) {
		v := toUserRoleProto(nil)
		if v != nil {
			t.Error("expected nil")
		}
	})

	t.Run("toUserIdentityProto nil", func(t *testing.T) {
		v := toUserIdentityProto(nil)
		if v != nil {
			t.Error("expected nil")
		}
	})

	t.Run("toUserIdentityProto success", func(t *testing.T) {
		id := uuid.New()
		v := toUserIdentityProto(&UserIdentityServiceDataResult{UserIdentityUUID: id, Provider: "google"})
		if v == nil {
			t.Error("expected non-nil")
		}
	})

	t.Run("structToJSON nil", func(t *testing.T) {
		v := structToJSON(nil)
		if v != nil {
			t.Error("expected nil")
		}
	})

	t.Run("structToJSON non-nil", func(t *testing.T) {
		s, _ := structpb.NewStruct(map[string]any{"key": "val"})
		v := structToJSON(s)
		if v == nil {
			t.Error("expected non-nil")
		}
	})

	t.Run("optionalStr non-empty", func(t *testing.T) {
		v := optionalStr("hello")
		if v == nil || *v != "hello" {
			t.Error("expected non-nil")
		}
	})

	t.Run("optionalEmail non-empty", func(t *testing.T) {
		v := optionalEmail("test@example.com")
		if v == nil || *v != "test@example.com" {
			t.Error("expected non-nil")
		}
	})

	t.Run("optionalPhone non-empty", func(t *testing.T) {
		v := optionalPhone("+123")
		if v == nil || *v != "+123" {
			t.Error("expected non-nil")
		}
	})

	t.Run("assignRoles invalid role UUID", func(t *testing.T) {
		svc := &testUserService{}
		h := NewUserGRPCHandler(resolver, svc)
		_, err := h.AssignUserRoles(ctx, &authv1.AssignUserRolesRequest{TenantUuid: tenantUUID.String(), UserUuid: userUUID.String(), RoleUuids: []string{"bad"}})
		if code := status.Code(err); code != codes.InvalidArgument {
			t.Errorf("expected InvalidArgument, got %v", code)
		}
	})

	t.Run("resolveTenant resolver error", func(t *testing.T) {
		errResolver := &testUserTenantResolver{getByUUIDFn: func(ctx context.Context, tuuid uuid.UUID) (*TenantServiceDataResult, error) {
			return nil, errors.New("tenant")
		}}
		svc := &testUserService{}
		h := NewUserGRPCHandler(errResolver, svc)
		_, err := h.ListUsers(ctx, &authv1.ListUsersRequest{TenantUuid: tenantUUID.String()})
		if code := status.Code(err); code != codes.Internal {
			t.Errorf("expected Internal, got %v", code)
		}
	})

	t.Run("get tenant resolver error", func(t *testing.T) {
		errResolver := &testUserTenantResolver{getByUUIDFn: func(ctx context.Context, tuuid uuid.UUID) (*TenantServiceDataResult, error) {
			return nil, errors.New("tenant")
		}}
		h := NewUserGRPCHandler(errResolver, &testUserService{})
		_, err := h.GetUser(ctx, &authv1.GetUserRequest{TenantUuid: tenantUUID.String(), UserUuid: userUUID.String()})
		if code := status.Code(err); code != codes.Internal {
			t.Errorf("expected Internal, got %v", code)
		}
	})

	t.Run("create tenant resolver error", func(t *testing.T) {
		errResolver := &testUserTenantResolver{getByUUIDFn: func(ctx context.Context, tuuid uuid.UUID) (*TenantServiceDataResult, error) {
			return nil, errors.New("tenant")
		}}
		h := NewUserGRPCHandler(errResolver, &testUserService{})
		_, err := h.CreateUser(ctx, &authv1.CreateUserRequest{TenantUuid: tenantUUID.String(), Username: "test", Password: "pass"})
		if code := status.Code(err); code != codes.Internal {
			t.Errorf("expected Internal, got %v", code)
		}
	})

	t.Run("create invalid actor UUID", func(t *testing.T) {
		svc := &testUserService{
			createFn: func(ctx context.Context, username string, email, phone *string, password, rStatus string, metadata datatypes.JSON, tUUID string, cu uuid.UUID) (*UserServiceDataResult, error) {
				return &userResult, nil
			},
		}
		h := NewUserGRPCHandler(resolver, svc)
		_, err := h.CreateUser(ctx, &authv1.CreateUserRequest{TenantUuid: tenantUUID.String(), Username: "test", Password: "pass", ActorUserUuid: "bad"})
		if code := status.Code(err); code != codes.InvalidArgument {
			t.Errorf("expected InvalidArgument, got %v", code)
		}
	})

	t.Run("update tenant resolver error", func(t *testing.T) {
		errResolver := &testUserTenantResolver{getByUUIDFn: func(ctx context.Context, tuuid uuid.UUID) (*TenantServiceDataResult, error) {
			return nil, errors.New("tenant")
		}}
		h := NewUserGRPCHandler(errResolver, &testUserService{})
		_, err := h.UpdateUser(ctx, &authv1.UpdateUserRequest{TenantUuid: tenantUUID.String(), UserUuid: userUUID.String()})
		if code := status.Code(err); code != codes.Internal {
			t.Errorf("expected Internal, got %v", code)
		}
	})

	t.Run("update invalid actor UUID", func(t *testing.T) {
		svc := &testUserService{
			updateFn: func(ctx context.Context, userUUID uuid.UUID, tenantID int64, username string, email, phone *string, status string, metadata datatypes.JSON, updaterUserUUID uuid.UUID) (*UserServiceDataResult, error) {
				return &userResult, nil
			},
		}
		h := NewUserGRPCHandler(resolver, svc)
		_, err := h.UpdateUser(ctx, &authv1.UpdateUserRequest{TenantUuid: tenantUUID.String(), UserUuid: userUUID.String(), ActorUserUuid: "bad"})
		if code := status.Code(err); code != codes.InvalidArgument {
			t.Errorf("expected InvalidArgument, got %v", code)
		}
	})

	t.Run("setStatus tenant resolver error", func(t *testing.T) {
		errResolver := &testUserTenantResolver{getByUUIDFn: func(ctx context.Context, tuuid uuid.UUID) (*TenantServiceDataResult, error) {
			return nil, errors.New("tenant")
		}}
		h := NewUserGRPCHandler(errResolver, &testUserService{})
		_, err := h.SetUserStatus(ctx, &authv1.SetUserStatusRequest{TenantUuid: tenantUUID.String(), UserUuid: userUUID.String(), Status: "inactive"})
		if code := status.Code(err); code != codes.Internal {
			t.Errorf("expected Internal, got %v", code)
		}
	})

	t.Run("setStatus invalid actor UUID", func(t *testing.T) {
		svc := &testUserService{
			setStatusFn: func(ctx context.Context, userUUID uuid.UUID, tenantID int64, rStatus string, updaterUserUUID uuid.UUID) (*UserServiceDataResult, error) {
				return &userResult, nil
			},
		}
		h := NewUserGRPCHandler(resolver, svc)
		_, err := h.SetUserStatus(ctx, &authv1.SetUserStatusRequest{TenantUuid: tenantUUID.String(), UserUuid: userUUID.String(), Status: "inactive", ActorUserUuid: "bad"})
		if code := status.Code(err); code != codes.InvalidArgument {
			t.Errorf("expected InvalidArgument, got %v", code)
		}
	})

	t.Run("verifyEmail tenant resolver error", func(t *testing.T) {
		errResolver := &testUserTenantResolver{getByUUIDFn: func(ctx context.Context, tuuid uuid.UUID) (*TenantServiceDataResult, error) {
			return nil, errors.New("tenant")
		}}
		h := NewUserGRPCHandler(errResolver, &testUserService{})
		_, err := h.VerifyUserEmail(ctx, &authv1.VerifyUserEmailRequest{TenantUuid: tenantUUID.String(), UserUuid: userUUID.String()})
		if code := status.Code(err); code != codes.Internal {
			t.Errorf("expected Internal, got %v", code)
		}
	})

	t.Run("verifyPhone tenant resolver error", func(t *testing.T) {
		errResolver := &testUserTenantResolver{getByUUIDFn: func(ctx context.Context, tuuid uuid.UUID) (*TenantServiceDataResult, error) {
			return nil, errors.New("tenant")
		}}
		h := NewUserGRPCHandler(errResolver, &testUserService{})
		_, err := h.VerifyUserPhone(ctx, &authv1.VerifyUserPhoneRequest{TenantUuid: tenantUUID.String(), UserUuid: userUUID.String()})
		if code := status.Code(err); code != codes.Internal {
			t.Errorf("expected Internal, got %v", code)
		}
	})

	t.Run("completeAccount tenant resolver error", func(t *testing.T) {
		errResolver := &testUserTenantResolver{getByUUIDFn: func(ctx context.Context, tuuid uuid.UUID) (*TenantServiceDataResult, error) {
			return nil, errors.New("tenant")
		}}
		h := NewUserGRPCHandler(errResolver, &testUserService{})
		_, err := h.CompleteUserAccount(ctx, &authv1.CompleteUserAccountRequest{TenantUuid: tenantUUID.String(), UserUuid: userUUID.String()})
		if code := status.Code(err); code != codes.Internal {
			t.Errorf("expected Internal, got %v", code)
		}
	})

	t.Run("delete tenant resolver error", func(t *testing.T) {
		errResolver := &testUserTenantResolver{getByUUIDFn: func(ctx context.Context, tuuid uuid.UUID) (*TenantServiceDataResult, error) {
			return nil, errors.New("tenant")
		}}
		h := NewUserGRPCHandler(errResolver, &testUserService{})
		_, err := h.DeleteUser(ctx, &authv1.DeleteUserRequest{TenantUuid: tenantUUID.String(), UserUuid: userUUID.String()})
		if code := status.Code(err); code != codes.Internal {
			t.Errorf("expected Internal, got %v", code)
		}
	})

	t.Run("delete invalid actor UUID", func(t *testing.T) {
		svc := &testUserService{
			deleteByUUIDFn: func(ctx context.Context, userUUID uuid.UUID, tenantID int64, deleterUserUUID uuid.UUID) (*UserServiceDataResult, error) {
				return &userResult, nil
			},
		}
		h := NewUserGRPCHandler(resolver, svc)
		_, err := h.DeleteUser(ctx, &authv1.DeleteUserRequest{TenantUuid: tenantUUID.String(), UserUuid: userUUID.String(), ActorUserUuid: "bad"})
		if code := status.Code(err); code != codes.InvalidArgument {
			t.Errorf("expected InvalidArgument, got %v", code)
		}
	})

	t.Run("forcePasswordChange tenant resolver error", func(t *testing.T) {
		errResolver := &testUserTenantResolver{getByUUIDFn: func(ctx context.Context, tuuid uuid.UUID) (*TenantServiceDataResult, error) {
			return nil, errors.New("tenant")
		}}
		h := NewUserGRPCHandler(errResolver, &testUserService{})
		_, err := h.ForceUserPasswordChange(ctx, &authv1.ForceUserPasswordChangeRequest{TenantUuid: tenantUUID.String(), UserUuid: userUUID.String(), Force: true})
		if code := status.Code(err); code != codes.Internal {
			t.Errorf("expected Internal, got %v", code)
		}
	})

	t.Run("forcePasswordChange invalid user UUID", func(t *testing.T) {
		h := NewUserGRPCHandler(resolver, &testUserService{})
		_, err := h.ForceUserPasswordChange(ctx, &authv1.ForceUserPasswordChangeRequest{TenantUuid: tenantUUID.String(), UserUuid: "bad", Force: true})
		if code := status.Code(err); code != codes.InvalidArgument {
			t.Errorf("expected InvalidArgument, got %v", code)
		}
	})

	t.Run("assignRoles tenant resolver error", func(t *testing.T) {
		errResolver := &testUserTenantResolver{getByUUIDFn: func(ctx context.Context, tuuid uuid.UUID) (*TenantServiceDataResult, error) {
			return nil, errors.New("tenant")
		}}
		h := NewUserGRPCHandler(errResolver, &testUserService{})
		_, err := h.AssignUserRoles(ctx, &authv1.AssignUserRolesRequest{TenantUuid: tenantUUID.String(), UserUuid: userUUID.String(), RoleUuids: []string{roleUUID.String()}})
		if code := status.Code(err); code != codes.Internal {
			t.Errorf("expected Internal, got %v", code)
		}
	})

	t.Run("removeUserRole tenant resolver error", func(t *testing.T) {
		errResolver := &testUserTenantResolver{getByUUIDFn: func(ctx context.Context, tuuid uuid.UUID) (*TenantServiceDataResult, error) {
			return nil, errors.New("tenant")
		}}
		h := NewUserGRPCHandler(errResolver, &testUserService{})
		_, err := h.RemoveUserRole(ctx, &authv1.RemoveUserRoleRequest{TenantUuid: tenantUUID.String(), UserUuid: userUUID.String(), RoleUuid: roleUUID.String()})
		if code := status.Code(err); code != codes.Internal {
			t.Errorf("expected Internal, got %v", code)
		}
	})

	t.Run("grpcPagination zero", func(t *testing.T) {
		dto := grpcPagination(&authv1.Pagination{Page: 0, Limit: 0})
		if dto.Page != 1 || dto.Limit != pagination.DefaultPageSize {
			t.Errorf("expected defaults, got %+v", dto)
		}
	})

	t.Run("grpcUUID empty", func(t *testing.T) {
		_, err := grpcUUID("", "test")
		if code := status.Code(err); code != codes.InvalidArgument {
			t.Errorf("expected InvalidArgument, got %v", code)
		}
	})

	t.Run("listUsers with email filter", func(t *testing.T) {
		svc := &testUserService{
			getFn: func(ctx context.Context, filter UserServiceGetFilter) (*UserServiceGetResult, error) {
				if filter.Email == nil || *filter.Email != "test@example.com" {
					t.Error("expected email filter")
				}
				return &UserServiceGetResult{Data: []UserServiceDataResult{userResult}, Total: 1, Page: 1, Limit: 10, TotalPages: 1}, nil
			},
		}
		h := NewUserGRPCHandler(resolver, svc)
		_, err := h.ListUsers(ctx, &authv1.ListUsersRequest{TenantUuid: tenantUUID.String(), Email: "test@example.com"})
		if err != nil {
			t.Fatal(err)
		}
	})

	t.Run("listUsers with phone filter", func(t *testing.T) {
		svc := &testUserService{
			getFn: func(ctx context.Context, filter UserServiceGetFilter) (*UserServiceGetResult, error) {
				if filter.Phone == nil || *filter.Phone != "+123" {
					t.Error("expected phone filter")
				}
				return &UserServiceGetResult{Data: []UserServiceDataResult{userResult}, Total: 1, Page: 1, Limit: 10, TotalPages: 1}, nil
			},
		}
		h := NewUserGRPCHandler(resolver, svc)
		_, err := h.ListUsers(ctx, &authv1.ListUsersRequest{TenantUuid: tenantUUID.String(), Phone: "+123"})
		if err != nil {
			t.Fatal(err)
		}
	})

	t.Run("listUsers pagination validation error", func(t *testing.T) {
		svc := &testUserService{}
		h := NewUserGRPCHandler(resolver, svc)
		_, err := h.ListUsers(ctx, &authv1.ListUsersRequest{TenantUuid: tenantUUID.String(), Pagination: &authv1.Pagination{Page: -1}})
		if err == nil {
			t.Fatal("expected error")
		}
	})

	t.Run("getUser invalid UUID", func(t *testing.T) {
		h := NewUserGRPCHandler(resolver, &testUserService{})
		_, err := h.GetUser(ctx, &authv1.GetUserRequest{TenantUuid: tenantUUID.String(), UserUuid: "bad"})
		if code := status.Code(err); code != codes.InvalidArgument {
			t.Errorf("expected InvalidArgument, got %v", code)
		}
	})

	t.Run("updateUser invalid UUID", func(t *testing.T) {
		h := NewUserGRPCHandler(resolver, &testUserService{})
		_, err := h.UpdateUser(ctx, &authv1.UpdateUserRequest{TenantUuid: tenantUUID.String(), UserUuid: "bad"})
		if code := status.Code(err); code != codes.InvalidArgument {
			t.Errorf("expected InvalidArgument, got %v", code)
		}
	})

	t.Run("setUserStatus invalid UUID", func(t *testing.T) {
		h := NewUserGRPCHandler(resolver, &testUserService{})
		_, err := h.SetUserStatus(ctx, &authv1.SetUserStatusRequest{TenantUuid: tenantUUID.String(), UserUuid: "bad", Status: "inactive"})
		if code := status.Code(err); code != codes.InvalidArgument {
			t.Errorf("expected InvalidArgument, got %v", code)
		}
	})

	t.Run("verifyEmail invalid UUID", func(t *testing.T) {
		h := NewUserGRPCHandler(resolver, &testUserService{})
		_, err := h.VerifyUserEmail(ctx, &authv1.VerifyUserEmailRequest{TenantUuid: tenantUUID.String(), UserUuid: "bad"})
		if code := status.Code(err); code != codes.InvalidArgument {
			t.Errorf("expected InvalidArgument, got %v", code)
		}
	})

	t.Run("verifyPhone invalid UUID", func(t *testing.T) {
		h := NewUserGRPCHandler(resolver, &testUserService{})
		_, err := h.VerifyUserPhone(ctx, &authv1.VerifyUserPhoneRequest{TenantUuid: tenantUUID.String(), UserUuid: "bad"})
		if code := status.Code(err); code != codes.InvalidArgument {
			t.Errorf("expected InvalidArgument, got %v", code)
		}
	})

	t.Run("completeAccount invalid UUID", func(t *testing.T) {
		h := NewUserGRPCHandler(resolver, &testUserService{})
		_, err := h.CompleteUserAccount(ctx, &authv1.CompleteUserAccountRequest{TenantUuid: tenantUUID.String(), UserUuid: "bad"})
		if code := status.Code(err); code != codes.InvalidArgument {
			t.Errorf("expected InvalidArgument, got %v", code)
		}
	})

	t.Run("deleteUser invalid UUID", func(t *testing.T) {
		h := NewUserGRPCHandler(resolver, &testUserService{})
		_, err := h.DeleteUser(ctx, &authv1.DeleteUserRequest{TenantUuid: tenantUUID.String(), UserUuid: "bad"})
		if code := status.Code(err); code != codes.InvalidArgument {
			t.Errorf("expected InvalidArgument, got %v", code)
		}
	})

	t.Run("listUserRoles invalid UUID", func(t *testing.T) {
		h := NewUserGRPCHandler(resolver, &testUserService{})
		_, err := h.ListUserRoles(ctx, &authv1.ListUserRolesRequest{TenantUuid: tenantUUID.String(), UserUuid: "bad"})
		if code := status.Code(err); code != codes.InvalidArgument {
			t.Errorf("expected InvalidArgument, got %v", code)
		}
	})

	t.Run("listUserRoles tenant error", func(t *testing.T) {
		errResolver := &testUserTenantResolver{getByUUIDFn: func(ctx context.Context, tuuid uuid.UUID) (*TenantServiceDataResult, error) {
			return nil, errors.New("tenant")
		}}
		h := NewUserGRPCHandler(errResolver, &testUserService{})
		_, err := h.ListUserRoles(ctx, &authv1.ListUserRolesRequest{TenantUuid: tenantUUID.String(), UserUuid: userUUID.String()})
		if code := status.Code(err); code != codes.Internal {
			t.Errorf("expected Internal, got %v", code)
		}
	})

	t.Run("listUserIdentities invalid UUID", func(t *testing.T) {
		h := NewUserGRPCHandler(resolver, &testUserService{})
		_, err := h.ListUserIdentities(ctx, &authv1.ListUserIdentitiesRequest{TenantUuid: tenantUUID.String(), UserUuid: "bad"})
		if code := status.Code(err); code != codes.InvalidArgument {
			t.Errorf("expected InvalidArgument, got %v", code)
		}
	})

	t.Run("listUserIdentities tenant error", func(t *testing.T) {
		errResolver := &testUserTenantResolver{getByUUIDFn: func(ctx context.Context, tuuid uuid.UUID) (*TenantServiceDataResult, error) {
			return nil, errors.New("tenant")
		}}
		h := NewUserGRPCHandler(errResolver, &testUserService{})
		_, err := h.ListUserIdentities(ctx, &authv1.ListUserIdentitiesRequest{TenantUuid: tenantUUID.String(), UserUuid: userUUID.String()})
		if code := status.Code(err); code != codes.Internal {
			t.Errorf("expected Internal, got %v", code)
		}
	})

	t.Run("assignUserRoles invalid UUID", func(t *testing.T) {
		h := NewUserGRPCHandler(resolver, &testUserService{})
		_, err := h.AssignUserRoles(ctx, &authv1.AssignUserRolesRequest{TenantUuid: tenantUUID.String(), UserUuid: "bad", RoleUuids: []string{roleUUID.String()}})
		if code := status.Code(err); code != codes.InvalidArgument {
			t.Errorf("expected InvalidArgument, got %v", code)
		}
	})

	t.Run("removeUserRole invalid user UUID", func(t *testing.T) {
		h := NewUserGRPCHandler(resolver, &testUserService{})
		_, err := h.RemoveUserRole(ctx, &authv1.RemoveUserRoleRequest{TenantUuid: tenantUUID.String(), UserUuid: "bad", RoleUuid: roleUUID.String()})
		if code := status.Code(err); code != codes.InvalidArgument {
			t.Errorf("expected InvalidArgument, got %v", code)
		}
	})

	t.Run("removeUserRole invalid role UUID", func(t *testing.T) {
		h := NewUserGRPCHandler(resolver, &testUserService{})
		_, err := h.RemoveUserRole(ctx, &authv1.RemoveUserRoleRequest{TenantUuid: tenantUUID.String(), UserUuid: userUUID.String(), RoleUuid: "bad"})
		if code := status.Code(err); code != codes.InvalidArgument {
			t.Errorf("expected InvalidArgument, got %v", code)
		}
	})
}
