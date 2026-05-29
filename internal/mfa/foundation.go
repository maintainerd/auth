package mfa

import (
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/maintainerd/auth/internal/platform/database"
	"github.com/maintainerd/auth/internal/platform/pagination"
	"gorm.io/gorm"
)

type BaseRepository[T any] = database.BaseRepository[T]
type BaseRepositoryMethods[T any] = database.BaseRepositoryMethods[T]
type PaginationResult[T any] = database.PaginationResult[T]

type PaginationRequestDTO = pagination.PaginationRequestDTO
type PaginatedResponseDTO[T any] = pagination.PaginatedResponseDTO[T]
type SuccessResponseDTO = pagination.SuccessResponseDTO

const (
	SortOrderAsc  = pagination.SortOrderAsc
	SortOrderDesc = pagination.SortOrderDesc
)

func NewBaseRepository[T any](db any, uuidFieldName, idFieldName string) *database.BaseRepository[T] {
	return database.NewBaseRepository[T](db.(*gorm.DB), uuidFieldName, idFieldName)
}

func parsePaginationQuery(r *http.Request) pagination.PaginationRequestDTO {
	return pagination.ParseQuery(r)
}

// ---------------------------------------------------------------------------
// Local aggregate structs
// ---------------------------------------------------------------------------

type User struct {
	UserID            int64
	UserUUID          uuid.UUID
	Email             string
	Username          string
	IsTOTPEnabled     bool
	IsWebAuthnEnabled bool
	MFAEnabledAt      *time.Time
	CreatedAt         time.Time
	UpdatedAt         time.Time
}

func (User) TableName() string { return "users" }

// ---------------------------------------------------------------------------
// Consumer repository interface
// ---------------------------------------------------------------------------

type UserRepository interface {
	BaseRepositoryMethods[User]
	WithTx(tx *gorm.DB) UserRepository
}
