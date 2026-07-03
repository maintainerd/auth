package user

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/maintainerd/maintainerd-auth/internal/platform/apperror"
	authv1 "github.com/maintainerd/maintainerd-auth/internal/platform/gen/go/maintainerd/auth"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type UserProfileGRPCHandler struct {
	authv1.UnimplementedUserProfileServiceServer
	tenantResolver TenantResolver
	profileService ProfileService
}

func NewUserProfileGRPCHandler(tenantResolver TenantResolver, profileService ProfileService) *UserProfileGRPCHandler {
	return &UserProfileGRPCHandler{tenantResolver: tenantResolver, profileService: profileService}
}

func (h *UserProfileGRPCHandler) ListUserProfiles(ctx context.Context, req *authv1.ListUserProfilesRequest) (*authv1.ListUserProfilesResponse, error) {
	if _, err := h.resolveTenant(ctx, req.GetTenantUuid()); err != nil {
		return nil, err
	}
	userUUID, err := grpcUUID(req.GetUserUuid(), "User UUID")
	if err != nil {
		return nil, err
	}
	dto := grpcPagination(req.GetPagination())
	filter := ProfileFilterDTO{
		FirstName:            optionalStr(req.GetFirstName()),
		LastName:             optionalStr(req.GetLastName()),
		Email:                optionalStr(req.GetEmail()),
		Phone:                optionalStr(req.GetPhone()),
		City:                 optionalStr(req.GetCity()),
		Country:              optionalStr(req.GetCountry()),
		PaginationRequestDTO: dto,
	}
	if req.IsDefault != nil {
		filter.IsDefault = req.IsDefault
	}
	if err := filter.Validate(); err != nil {
		return nil, apperror.ToGRPCError(apperror.NewValidation(err.Error()))
	}
	result, err := h.profileService.GetAll(ctx, userUUID, filter.FirstName, filter.LastName, filter.Email, filter.Phone, filter.City, filter.Country, filter.IsDefault, filter.Page, filter.Limit, filter.SortBy, filter.SortOrder)
	if err != nil {
		return nil, apperror.ToGRPCError(err)
	}
	rows := make([]*authv1.UserProfile, len(result.Data))
	for i := range result.Data {
		rows[i] = toUserProfileProto(&result.Data[i])
	}
	return &authv1.ListUserProfilesResponse{Profiles: rows, Page: grpcPageProto(result.Total, result.Page, result.Limit, result.TotalPages)}, nil
}

func (h *UserProfileGRPCHandler) GetUserProfile(ctx context.Context, req *authv1.GetUserProfileRequest) (*authv1.GetUserProfileResponse, error) {
	userUUID, profileUUID, err := h.profileRequestIDs(ctx, req.GetTenantUuid(), req.GetUserUuid(), req.GetProfileUuid())
	if err != nil {
		return nil, err
	}
	result, err := h.profileService.GetByUUID(ctx, profileUUID, userUUID)
	if err != nil {
		return nil, apperror.ToGRPCError(err)
	}
	return &authv1.GetUserProfileResponse{Profile: toUserProfileProto(result)}, nil
}

func (h *UserProfileGRPCHandler) CreateUserProfile(ctx context.Context, req *authv1.CreateUserProfileRequest) (*authv1.CreateUserProfileResponse, error) {
	if _, err := h.resolveTenant(ctx, req.GetTenantUuid()); err != nil {
		return nil, err
	}
	userUUID, err := grpcUUID(req.GetUserUuid(), "User UUID")
	if err != nil {
		return nil, err
	}
	dto := profileRequestDTOFromCreate(req)
	birthdate, err := validatedProfileBirthdate(dto)
	if err != nil {
		return nil, err
	}
	result, err := h.profileService.CreateOrUpdateSpecificProfile(ctx, uuid.New(), userUUID, dto.FirstName, dto.MiddleName, dto.LastName, dto.Suffix, dto.DisplayName, dto.Bio, birthdate, dto.Gender, dto.Phone, dto.Email, dto.Address, dto.City, dto.Country, dto.Timezone, dto.Language, dto.ProfileURL, dto.Metadata)
	if err != nil {
		return nil, apperror.ToGRPCError(err)
	}
	return &authv1.CreateUserProfileResponse{Profile: toUserProfileProto(result)}, nil
}

func (h *UserProfileGRPCHandler) UpdateUserProfile(ctx context.Context, req *authv1.UpdateUserProfileRequest) (*authv1.UpdateUserProfileResponse, error) {
	userUUID, profileUUID, err := h.profileRequestIDs(ctx, req.GetTenantUuid(), req.GetUserUuid(), req.GetProfileUuid())
	if err != nil {
		return nil, err
	}
	dto := profileRequestDTOFromUpdate(req)
	birthdate, err := validatedProfileBirthdate(dto)
	if err != nil {
		return nil, err
	}
	result, err := h.profileService.CreateOrUpdateSpecificProfile(ctx, profileUUID, userUUID, dto.FirstName, dto.MiddleName, dto.LastName, dto.Suffix, dto.DisplayName, dto.Bio, birthdate, dto.Gender, dto.Phone, dto.Email, dto.Address, dto.City, dto.Country, dto.Timezone, dto.Language, dto.ProfileURL, dto.Metadata)
	if err != nil {
		return nil, apperror.ToGRPCError(err)
	}
	return &authv1.UpdateUserProfileResponse{Profile: toUserProfileProto(result)}, nil
}

func (h *UserProfileGRPCHandler) SetDefaultUserProfile(ctx context.Context, req *authv1.SetDefaultUserProfileRequest) (*authv1.SetDefaultUserProfileResponse, error) {
	userUUID, profileUUID, err := h.profileRequestIDs(ctx, req.GetTenantUuid(), req.GetUserUuid(), req.GetProfileUuid())
	if err != nil {
		return nil, err
	}
	result, err := h.profileService.SetDefaultProfile(ctx, profileUUID, userUUID)
	if err != nil {
		return nil, apperror.ToGRPCError(err)
	}
	return &authv1.SetDefaultUserProfileResponse{Profile: toUserProfileProto(result)}, nil
}

func (h *UserProfileGRPCHandler) DeleteUserProfile(ctx context.Context, req *authv1.DeleteUserProfileRequest) (*authv1.DeleteUserProfileResponse, error) {
	userUUID, profileUUID, err := h.profileRequestIDs(ctx, req.GetTenantUuid(), req.GetUserUuid(), req.GetProfileUuid())
	if err != nil {
		return nil, err
	}
	result, err := h.profileService.DeleteByUUID(ctx, profileUUID, userUUID)
	if err != nil {
		return nil, apperror.ToGRPCError(err)
	}
	return &authv1.DeleteUserProfileResponse{Profile: toUserProfileProto(result)}, nil
}

func (h *UserProfileGRPCHandler) profileRequestIDs(ctx context.Context, tenantUUID string, userUUID string, profileUUID string) (uuid.UUID, uuid.UUID, error) {
	if _, err := h.resolveTenant(ctx, tenantUUID); err != nil {
		return uuid.Nil, uuid.Nil, err
	}
	parsedUserUUID, err := grpcUUID(userUUID, "User UUID")
	if err != nil {
		return uuid.Nil, uuid.Nil, err
	}
	parsedProfileUUID, err := grpcUUID(profileUUID, "Profile UUID")
	if err != nil {
		return uuid.Nil, uuid.Nil, err
	}
	return parsedUserUUID, parsedProfileUUID, nil
}

func (h *UserProfileGRPCHandler) resolveTenant(ctx context.Context, tenantUUID string) (*TenantServiceDataResult, error) {
	parsed, err := grpcUUID(tenantUUID, "Tenant UUID")
	if err != nil {
		return nil, err
	}
	result, err := h.tenantResolver.GetByUUID(ctx, parsed)
	if err != nil {
		return nil, apperror.ToGRPCError(err)
	}
	return result, nil
}

func profileRequestDTOFromCreate(req *authv1.CreateUserProfileRequest) ProfileRequestDTO {
	return ProfileRequestDTO{
		FirstName:   req.GetFirstName(),
		MiddleName:  optionalStr(req.GetMiddleName()),
		LastName:    optionalStr(req.GetLastName()),
		Suffix:      optionalStr(req.GetSuffix()),
		DisplayName: optionalStr(req.GetDisplayName()),
		Bio:         optionalStr(req.GetBio()),
		Birthdate:   optionalStr(req.GetBirthdate()),
		Gender:      optionalStr(req.GetGender()),
		Phone:       optionalStr(req.GetPhone()),
		Email:       optionalStr(req.GetEmail()),
		Address:     optionalStr(req.GetAddress()),
		City:        optionalStr(req.GetCity()),
		Country:     optionalStr(req.GetCountry()),
		Timezone:    optionalStr(req.GetTimezone()),
		Language:    optionalStr(req.GetLanguage()),
		ProfileURL:  optionalStr(req.GetProfileUrl()),
		Metadata:    profileStructToMap(req.GetMetadata()),
	}
}

func profileRequestDTOFromUpdate(req *authv1.UpdateUserProfileRequest) ProfileRequestDTO {
	return ProfileRequestDTO{
		FirstName:   req.GetFirstName(),
		MiddleName:  optionalStr(req.GetMiddleName()),
		LastName:    optionalStr(req.GetLastName()),
		Suffix:      optionalStr(req.GetSuffix()),
		DisplayName: optionalStr(req.GetDisplayName()),
		Bio:         optionalStr(req.GetBio()),
		Birthdate:   optionalStr(req.GetBirthdate()),
		Gender:      optionalStr(req.GetGender()),
		Phone:       optionalStr(req.GetPhone()),
		Email:       optionalStr(req.GetEmail()),
		Address:     optionalStr(req.GetAddress()),
		City:        optionalStr(req.GetCity()),
		Country:     optionalStr(req.GetCountry()),
		Timezone:    optionalStr(req.GetTimezone()),
		Language:    optionalStr(req.GetLanguage()),
		ProfileURL:  optionalStr(req.GetProfileUrl()),
		Metadata:    profileStructToMap(req.GetMetadata()),
	}
}

func validatedProfileBirthdate(dto ProfileRequestDTO) (*time.Time, error) {
	if err := dto.Validate(); err != nil {
		return nil, apperror.ToGRPCError(apperror.NewValidation(err.Error()))
	}
	if dto.Birthdate == nil || *dto.Birthdate == "" {
		return nil, nil
	}
	parsed, _ := time.Parse("2006-01-02", *dto.Birthdate)
	return &parsed, nil
}

func profileStructToMap(s *structpb.Struct) map[string]any {
	if s == nil {
		return nil
	}
	return s.AsMap()
}

func profileMapToStruct(m map[string]any) *structpb.Struct {
	if m == nil {
		return nil
	}
	result, _ := structpb.NewStruct(m)
	return result
}

func toUserProfileProto(result *ProfileServiceDataResult) *authv1.UserProfile {
	if result == nil {
		return nil
	}
	profile := &authv1.UserProfile{
		ProfileUuid: result.ProfileUUID.String(),
		FirstName:   result.FirstName,
		MiddleName:  stringValue(result.MiddleName),
		LastName:    stringValue(result.LastName),
		Suffix:      stringValue(result.Suffix),
		DisplayName: stringValue(result.DisplayName),
		Bio:         stringValue(result.Bio),
		Gender:      stringValue(result.Gender),
		Phone:       stringValue(result.Phone),
		Email:       stringValue(result.Email),
		Address:     stringValue(result.Address),
		City:        stringValue(result.City),
		Country:     stringValue(result.Country),
		Timezone:    stringValue(result.Timezone),
		Language:    stringValue(result.Language),
		ProfileUrl:  stringValue(result.ProfileURL),
		IsDefault:   result.IsDefault,
		Metadata:    profileMapToStruct(result.Metadata),
		CreatedAt:   timestamppb.New(result.CreatedAt),
		UpdatedAt:   timestamppb.New(result.UpdatedAt),
	}
	if result.Birthdate != nil {
		profile.Birthdate = result.Birthdate.Format("2006-01-02")
	}
	return profile
}

func stringValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}
