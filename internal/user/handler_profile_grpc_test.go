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
	"google.golang.org/protobuf/types/known/structpb"
)

func TestUserProfileGRPCHandler_RPCS(t *testing.T) {
	ctx := context.Background()
	tenantUUID := uuid.New()
	userUUID := uuid.New()
	profileUUID := uuid.New()
	now := time.Now()
	resolver := &testUserTenantResolver{}
	result := &ProfileServiceDataResult{
		ProfileUUID: profileUUID,
		FirstName:   "Jane",
		IsDefault:   true,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	t.Run("list success preserves false is_default filter", func(t *testing.T) {
		isDefault := false
		svc := &mockProfileService{
			getAllFn: func(gotUserUUID uuid.UUID, firstName, lastName, email, phone, city, country *string, gotIsDefault *bool, page, limit int, sortBy, sortOrder string) (*ProfileServiceListResult, error) {
				if gotUserUUID != userUUID {
					t.Fatalf("expected user UUID %s, got %s", userUUID, gotUserUUID)
				}
				if gotIsDefault == nil || *gotIsDefault {
					t.Fatalf("expected explicit false is_default filter, got %v", gotIsDefault)
				}
				if page != 2 || limit != 5 || sortBy != "first_name" || sortOrder != "asc" {
					t.Fatalf("unexpected pagination: page=%d limit=%d sort_by=%q sort_order=%q", page, limit, sortBy, sortOrder)
				}
				return &ProfileServiceListResult{Data: []ProfileServiceDataResult{*result}, Total: 1, Page: page, Limit: limit, TotalPages: 1}, nil
			},
		}
		h := NewUserProfileGRPCHandler(resolver, svc)

		res, err := h.ListUserProfiles(ctx, &authv1.ListUserProfilesRequest{
			TenantUuid: tenantUUID.String(),
			UserUuid:   userUUID.String(),
			IsDefault:  &isDefault,
			Pagination: &authv1.Pagination{Page: 2, Limit: 5, SortBy: "first_name", SortOrder: "asc"},
		})

		if err != nil {
			t.Fatal(err)
		}
		if len(res.Profiles) != 1 || res.Profiles[0].ProfileUuid != profileUUID.String() {
			t.Fatalf("unexpected profiles response: %+v", res.Profiles)
		}
	})

	t.Run("get success", func(t *testing.T) {
		svc := &mockProfileService{
			getByUUIDFn: func(gotProfileUUID, gotUserUUID uuid.UUID) (*ProfileServiceDataResult, error) {
				if gotProfileUUID != profileUUID || gotUserUUID != userUUID {
					t.Fatalf("unexpected IDs: profile=%s user=%s", gotProfileUUID, gotUserUUID)
				}
				return result, nil
			},
		}
		h := NewUserProfileGRPCHandler(resolver, svc)

		res, err := h.GetUserProfile(ctx, profileIDRequest(tenantUUID, userUUID, profileUUID))

		if err != nil {
			t.Fatal(err)
		}
		if res.Profile.ProfileUuid != profileUUID.String() {
			t.Fatalf("expected profile UUID %s, got %s", profileUUID, res.Profile.ProfileUuid)
		}
	})

	t.Run("create success validates and maps body", func(t *testing.T) {
		metadata, _ := structpb.NewStruct(map[string]any{"team": "core"})
		svc := &mockProfileService{
			createOrUpdateSpecificFn: func(gotProfileUUID, gotUserUUID uuid.UUID, firstName string, middleName, lastName, suffix, displayName, bio *string, birthdate *time.Time, gender *string, phone, email, address *string, city, country *string, timezone, language *string, profileURL *string, gotMetadata map[string]any) (*ProfileServiceDataResult, error) {
				if gotProfileUUID == uuid.Nil {
					t.Fatal("expected generated profile UUID")
				}
				if gotUserUUID != userUUID || firstName != "Jane" {
					t.Fatalf("unexpected create args: user=%s first_name=%q", gotUserUUID, firstName)
				}
				if birthdate == nil || birthdate.Format("2006-01-02") != "1990-01-25" {
					t.Fatalf("expected parsed birthdate, got %v", birthdate)
				}
				if gotMetadata["team"] != "core" {
					t.Fatalf("expected metadata to map through, got %#v", gotMetadata)
				}
				return result, nil
			},
		}
		h := NewUserProfileGRPCHandler(resolver, svc)

		_, err := h.CreateUserProfile(ctx, &authv1.CreateUserProfileRequest{
			TenantUuid: tenantUUID.String(),
			UserUuid:   userUUID.String(),
			FirstName:  "Jane",
			Birthdate:  "1990-01-25",
			Gender:     "female",
			Email:      "jane@example.com",
			Country:    "PH",
			ProfileUrl: "https://example.com/avatar.png",
			Metadata:   metadata,
		})

		if err != nil {
			t.Fatal(err)
		}
	})

	t.Run("update success", func(t *testing.T) {
		svc := &mockProfileService{
			createOrUpdateSpecificFn: func(gotProfileUUID, gotUserUUID uuid.UUID, firstName string, middleName, lastName, suffix, displayName, bio *string, birthdate *time.Time, gender *string, phone, email, address *string, city, country *string, timezone, language *string, profileURL *string, metadata map[string]any) (*ProfileServiceDataResult, error) {
				if gotProfileUUID != profileUUID || gotUserUUID != userUUID || firstName != "Jane" {
					t.Fatalf("unexpected update args: profile=%s user=%s first_name=%q", gotProfileUUID, gotUserUUID, firstName)
				}
				return result, nil
			},
		}
		h := NewUserProfileGRPCHandler(resolver, svc)

		_, err := h.UpdateUserProfile(ctx, &authv1.UpdateUserProfileRequest{
			TenantUuid:  tenantUUID.String(),
			UserUuid:    userUUID.String(),
			ProfileUuid: profileUUID.String(),
			FirstName:   "Jane",
		})

		if err != nil {
			t.Fatal(err)
		}
	})

	t.Run("set default success", func(t *testing.T) {
		svc := &mockProfileService{
			setDefaultFn: func(gotProfileUUID, gotUserUUID uuid.UUID) (*ProfileServiceDataResult, error) {
				if gotProfileUUID != profileUUID || gotUserUUID != userUUID {
					t.Fatalf("unexpected set default args: profile=%s user=%s", gotProfileUUID, gotUserUUID)
				}
				return result, nil
			},
		}
		h := NewUserProfileGRPCHandler(resolver, svc)

		_, err := h.SetDefaultUserProfile(ctx, &authv1.SetDefaultUserProfileRequest{TenantUuid: tenantUUID.String(), UserUuid: userUUID.String(), ProfileUuid: profileUUID.String()})

		if err != nil {
			t.Fatal(err)
		}
	})

	t.Run("delete success", func(t *testing.T) {
		svc := &mockProfileService{
			deleteByUUIDFn: func(gotProfileUUID, gotUserUUID uuid.UUID) (*ProfileServiceDataResult, error) {
				if gotProfileUUID != profileUUID || gotUserUUID != userUUID {
					t.Fatalf("unexpected delete args: profile=%s user=%s", gotProfileUUID, gotUserUUID)
				}
				return result, nil
			},
		}
		h := NewUserProfileGRPCHandler(resolver, svc)

		_, err := h.DeleteUserProfile(ctx, &authv1.DeleteUserProfileRequest{TenantUuid: tenantUUID.String(), UserUuid: userUUID.String(), ProfileUuid: profileUUID.String()})

		if err != nil {
			t.Fatal(err)
		}
	})

	t.Run("invalid uuid returns invalid argument", func(t *testing.T) {
		h := NewUserProfileGRPCHandler(resolver, &mockProfileService{})

		_, err := h.GetUserProfile(ctx, &authv1.GetUserProfileRequest{TenantUuid: tenantUUID.String(), UserUuid: "bad", ProfileUuid: profileUUID.String()})

		if code := status.Code(err); code != codes.InvalidArgument {
			t.Fatalf("expected InvalidArgument, got %v", code)
		}
	})

	t.Run("invalid profile body returns invalid argument", func(t *testing.T) {
		h := NewUserProfileGRPCHandler(resolver, &mockProfileService{})

		_, err := h.CreateUserProfile(ctx, &authv1.CreateUserProfileRequest{TenantUuid: tenantUUID.String(), UserUuid: userUUID.String(), FirstName: "", Birthdate: "25-01-1990"})

		if code := status.Code(err); code != codes.InvalidArgument {
			t.Fatalf("expected InvalidArgument, got %v", code)
		}
	})

	t.Run("service error returns internal", func(t *testing.T) {
		svc := &mockProfileService{
			getAllFn: func(userUUID uuid.UUID, firstName, lastName, email, phone, city, country *string, isDefault *bool, page, limit int, sortBy, sortOrder string) (*ProfileServiceListResult, error) {
				return nil, errors.New("db")
			},
		}
		h := NewUserProfileGRPCHandler(resolver, svc)

		_, err := h.ListUserProfiles(ctx, &authv1.ListUserProfilesRequest{TenantUuid: tenantUUID.String(), UserUuid: userUUID.String()})

		if code := status.Code(err); code != codes.Internal {
			t.Fatalf("expected Internal, got %v", code)
		}
	})
}

func profileIDRequest(tenantUUID, userUUID, profileUUID uuid.UUID) *authv1.GetUserProfileRequest {
	return &authv1.GetUserProfileRequest{
		TenantUuid:  tenantUUID.String(),
		UserUuid:    userUUID.String(),
		ProfileUuid: profileUUID.String(),
	}
}
