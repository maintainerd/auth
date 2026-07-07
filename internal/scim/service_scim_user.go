package scim

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/maintainerd/maintainerd-auth/internal/platform/apperror"
	"github.com/maintainerd/maintainerd-auth/internal/user"
	"go.opentelemetry.io/otel"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type SCIMUserService interface {
	ListUsers(ctx context.Context, tenantID int64, startIndex, count int, filter string) (*SCIMListResponse, error)
	GetUser(ctx context.Context, userID string, tenantID int64) (*SCIMUserResource, error)
	CreateUser(ctx context.Context, tenantID int64, req *SCIMUserCreateRequest) (*SCIMUserResource, error)
	UpdateUser(ctx context.Context, userID string, tenantID int64, req *SCIMUserUpdateRequest) (*SCIMUserResource, error)
	PatchUser(ctx context.Context, userID string, tenantID int64, req *SCIMPatchRequest) (*SCIMUserResource, error)
	DeleteUser(ctx context.Context, userID string, tenantID int64) error
}

type scimUserService struct {
	userSvc          user.UserService
	profileSvc       user.ProfileService
	userRepo         user.UserRepository
	userIdentityRepo user.UserIdentityRepository
	db               *gorm.DB
}

func NewSCIMUserService(db *gorm.DB, userSvc user.UserService, profileSvc user.ProfileService, userRepo user.UserRepository, userIdentityRepo user.UserIdentityRepository) SCIMUserService {
	return &scimUserService{
		db:               db,
		userSvc:          userSvc,
		profileSvc:       profileSvc,
		userRepo:         userRepo,
		userIdentityRepo: userIdentityRepo,
	}
}

func (s *scimUserService) ListUsers(ctx context.Context, tenantID int64, startIndex, count int, filter string) (*SCIMListResponse, error) {
	ctx, span := otel.Tracer("scim").Start(ctx, "SCIMUserService.ListUsers")
	defer span.End()

	page := 1
	if startIndex > 1 && count > 0 {
		page = ((startIndex - 1) / count) + 1
	}

	result, err := s.userSvc.Get(ctx, user.UserServiceGetFilter{
		TenantID:  tenantID,
		Page:      page,
		Limit:     count,
		SortBy:    "created_at",
		SortOrder: "asc",
	})
	if err != nil {
		return nil, apperror.NewInternal("list scim users", err)
	}

	resources := make([]SCIMUserResource, 0, len(result.Data))
	for _, u := range result.Data {
		resources = append(resources, s.toSCIMUser(ctx, &u))
	}

	return &SCIMListResponse{
		Schemas:      []string{SCIMListResponseSchema},
		TotalResults: int(result.Total),
		StartIndex:   max(startIndex, 1),
		ItemsPerPage: count,
		Resources:    resources,
	}, nil
}

func (s *scimUserService) GetUser(ctx context.Context, userID string, tenantID int64) (*SCIMUserResource, error) {
	ctx, span := otel.Tracer("scim").Start(ctx, "SCIMUserService.GetUser")
	defer span.End()

	userUUID, err := uuid.Parse(userID)
	if err != nil {
		return nil, apperror.NewNotFoundWithReason("user not found")
	}

	u, err := s.userSvc.GetByUUID(ctx, userUUID, tenantID)
	if err != nil {
		return nil, apperror.NewInternal("get scim user", err)
	}
	if u == nil {
		return nil, apperror.NewNotFoundWithReason("user not found")
	}

	resource := s.toSCIMUser(ctx, u)
	return &resource, nil
}

func (s *scimUserService) CreateUser(ctx context.Context, tenantID int64, req *SCIMUserCreateRequest) (*SCIMUserResource, error) {
	ctx, span := otel.Tracer("scim").Start(ctx, "SCIMUserService.CreateUser")
	defer span.End()

	var email *string
	for _, e := range req.Emails {
		if email == nil {
			email = &e.Value
		}
		if e.Primary {
			v := e.Value
			email = &v
			break
		}
	}

	var phone *string
	for _, p := range req.PhoneNumbers {
		if phone == nil {
			phone = &p.Value
		}
		if p.Primary {
			v := p.Value
			phone = &v
			break
		}
	}

	status := "active"
	if req.Active != nil && !*req.Active {
		status = "inactive"
	}

	u, err := s.userSvc.Create(ctx, req.UserName, email, phone, "", status, datatypes.JSON("{}"), "", uuid.Nil)
	if err != nil {
		return nil, err
	}

	// Resolve the integer UserID from the returned UUID.
	usr, err := s.userRepo.FindByUUID(u.UserUUID)
	if err != nil {
		return nil, apperror.NewInternal("resolve scim user id", err)
	}

	externalID := req.ExternalID
	if externalID != nil && *externalID != "" {
		// Set users.external_id on the user row.
		if dbErr := s.db.WithContext(ctx).Model(&user.User{}).Where("user_uuid = ?", u.UserUUID).
			Update("external_id", *externalID).Error; dbErr != nil {
			return nil, apperror.NewInternal("set scim external_id", dbErr)
		}
	}

	// Record a SCIM provisioning identity so the user is traceable to the
	// SCIM endpoint. The sub is the external_id when provided, falling back to
	// the user UUID — SCIM requires every user to have at least one identity.
	now := time.Now()
	scimSrc := "scim"
	identity := &user.UserIdentity{
		TenantID:           tenantID,
		UserID:             usr.UserID,
		Provider:           "scim",
		Sub: func() string {
			if externalID != nil && *externalID != "" {
				return *externalID
			}
			return u.UserUUID.String()
		}(),
		Metadata:          datatypes.JSON("{}"),
		ProvisioningSource: &scimSrc,
		JITProvisionedAt:   &now,
	}
	if _, repoErr := s.userIdentityRepo.Create(identity); repoErr != nil {
		return nil, apperror.NewInternal("create scim provisioning identity", repoErr)
	}

	if req.Name != nil || req.DisplayName != nil {
		var firstName string
		var middleName, lastName, dn *string
		if req.Name != nil {
			firstName = req.Name.GivenName
			if req.Name.MiddleName != "" {
				middleName = &req.Name.MiddleName
			}
			if req.Name.FamilyName != "" {
				lastName = &req.Name.FamilyName
			}
		}
		dn = req.DisplayName
		if _, err := s.profileSvc.CreateOrUpdateProfile(ctx, u.UserUUID, firstName, middleName, lastName, dn, nil, nil, nil, nil, nil, nil, nil); err != nil {
			span.RecordError(err)
		}
	}

	return s.GetUser(ctx, u.UserUUID.String(), tenantID)
}

func (s *scimUserService) UpdateUser(ctx context.Context, userID string, tenantID int64, req *SCIMUserUpdateRequest) (*SCIMUserResource, error) {
	ctx, span := otel.Tracer("scim").Start(ctx, "SCIMUserService.UpdateUser")
	defer span.End()

	userUUID, err := uuid.Parse(userID)
	if err != nil {
		return nil, apperror.NewNotFoundWithReason("user not found")
	}

	existing, err := s.userSvc.GetByUUID(ctx, userUUID, tenantID)
	if err != nil {
		return nil, apperror.NewInternal("get scim user", err)
	}
	if existing == nil {
		return nil, apperror.NewNotFoundWithReason("user not found")
	}

	username := existing.Username
	if req.UserName != nil && *req.UserName != "" {
		username = *req.UserName
	}

	email := &existing.Email
	if req.Emails != nil {
		for _, e := range req.Emails {
			if e.Primary {
				v := e.Value
				email = &v
				break
			}
		}
		if len(req.Emails) > 0 && email == &existing.Email {
			email = &req.Emails[0].Value
		}
	}

	phone := &existing.Phone
	if len(req.PhoneNumbers) > 0 {
		phone = &req.PhoneNumbers[0].Value
	}

	status := existing.Status
	if req.Active != nil {
		if *req.Active {
			status = "active"
		} else {
			status = "inactive"
		}
	}

	u, err := s.userSvc.Update(ctx, userUUID, tenantID, username, email, phone, status, existing.Metadata, uuid.Nil)
	if err != nil {
		return nil, err
	}

	if req.Name != nil || req.DisplayName != nil {
		var firstName string
		var middleName, lastName, dn *string
		if req.Name != nil {
			firstName = req.Name.GivenName
			if req.Name.MiddleName != "" {
				middleName = &req.Name.MiddleName
			}
			if req.Name.FamilyName != "" {
				lastName = &req.Name.FamilyName
			}
		}
		dn = req.DisplayName
		if _, err := s.profileSvc.CreateOrUpdateProfile(ctx, userUUID, firstName, middleName, lastName, dn, nil, nil, nil, nil, nil, nil, nil); err != nil {
			span.RecordError(err)
		}
	}

	return s.GetUser(ctx, u.UserUUID.String(), tenantID)
}

func (s *scimUserService) PatchUser(ctx context.Context, userID string, tenantID int64, req *SCIMPatchRequest) (*SCIMUserResource, error) {
	ctx, span := otel.Tracer("scim").Start(ctx, "SCIMUserService.PatchUser")
	defer span.End()

	userUUID, err := uuid.Parse(userID)
	if err != nil {
		return nil, apperror.NewNotFoundWithReason("user not found")
	}

	existing, err := s.userSvc.GetByUUID(ctx, userUUID, tenantID)
	if err != nil {
		return nil, apperror.NewInternal("get scim user", err)
	}
	if existing == nil {
		return nil, apperror.NewNotFoundWithReason("user not found")
	}

	username := existing.Username
	email := &existing.Email
	phone := &existing.Phone
	status := existing.Status

	for _, op := range req.Operations {
		switch strings.ToLower(op.Op) {
		case "replace":
			if op.Path != nil {
				s.applyPatchReplace(op, &username, email, phone, &status)
			} else {
				var full SCIMUserResource
				if err := json.Unmarshal(op.Value, &full); err == nil {
					username = full.UserName
					if len(full.Emails) > 0 {
						email = &full.Emails[0].Value
					}
					if len(full.PhoneNumbers) > 0 {
						phone = &full.PhoneNumbers[0].Value
					}
					status = "active"
					if !full.Active {
						status = "inactive"
					}
				}
			}
		case "active":
			var active bool
			if err := json.Unmarshal(op.Value, &active); err == nil {
				if active {
					status = "active"
				} else {
					status = "inactive"
				}
			}
		}
	}

	u, err := s.userSvc.Update(ctx, userUUID, tenantID, username, email, phone, status, existing.Metadata, uuid.Nil)
	if err != nil {
		return nil, err
	}

	return s.GetUser(ctx, u.UserUUID.String(), tenantID)
}

func (s *scimUserService) DeleteUser(ctx context.Context, userID string, tenantID int64) error {
	ctx, span := otel.Tracer("scim").Start(ctx, "SCIMUserService.DeleteUser")
	defer span.End()

	userUUID, err := uuid.Parse(userID)
	if err != nil {
		return apperror.NewNotFoundWithReason("user not found")
	}

	_, err = s.userSvc.DeleteByUUID(ctx, userUUID, tenantID, uuid.Nil)
	if err != nil {
		return err
	}
	return nil
}

func (s *scimUserService) toSCIMUser(ctx context.Context, u *user.UserServiceDataResult) SCIMUserResource {
	resource := SCIMUserResource{
		Schemas:  []string{SCIMUserSchema},
		ID:       u.UserUUID.String(),
		UserName: u.Username,
		Active:   u.Status == "active",
	}

	if u.ExternalID != nil && *u.ExternalID != "" {
		resource.ExternalID = u.ExternalID
	}

	if u.Email != "" {
		resource.Emails = []SCIMEmail{
			{Value: u.Email, Type: "work", Primary: true},
		}
	}

	if u.Phone != "" {
		resource.PhoneNumbers = []SCIMPhoneNumber{
			{Value: u.Phone, Type: "mobile", Primary: true},
		}
	}

	resource.Meta = &SCIMMeta{
		ResourceType: "User",
		Created:      u.CreatedAt.Format(time.RFC3339),
		LastModified: u.UpdatedAt.Format(time.RFC3339),
		Location:     "/scim/v2/Users/" + u.UserUUID.String(),
	}

	return resource
}

func (s *scimUserService) applyPatchReplace(op SCIMPatchOperation, username *string, email, phone, status *string) {
	if op.Path == nil {
		return
	}
	switch *op.Path {
	case "userName":
		var v string
		if json.Unmarshal(op.Value, &v) == nil {
			*username = v
		}
	case "active":
		var v bool
		if json.Unmarshal(op.Value, &v) == nil {
			if v {
				*status = "active"
			} else {
				*status = "inactive"
			}
		}
	}
}
