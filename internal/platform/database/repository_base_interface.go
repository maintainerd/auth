package database

type BaseRepositoryMethods[T any] interface {
	Create(entity *T) (*T, error)
	CreateOrUpdate(entity *T) (*T, error)
}

// PaginationResult holds paginated data and meta
type PaginationResult[T any] struct {
	Data       []T
	Total      int64
	Page       int
	Limit      int
	TotalPages int
	NextCursor *int64
}
